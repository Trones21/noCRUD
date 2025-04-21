import argparse
import os
import importlib.util
import inspect
from typing import Dict, Callable
import time
import traceback
import requests
import io
from contextlib import redirect_stdout
from multiprocessing import Pool

### Local Utils
from utils.provisioning import provision_env_for_flow
from utils.db_client import DBClient
from utils.printing import format_crud_print, print_group_separator, print_warn
from utils.decorators import with_stack_trace

##### Import Manually Registered Flows
from flows import example_api_client, example_biz_logic
from flows.crud import actor, episode

### Manually Registered

# Dictionary of request flows
REQUEST_FLOWS = {
    "example_api_client": example_api_client.exec,
    "example_biz_logic": example_biz_logic.exec,
}

# Dictionary of crud flows - should not be multi-user or multi endpoint (except creating prerequisite objects)
CRUD_FLOWS = {"episode": episode.crud, "actor": actor.crud}


def main():
    # Avoiding manual registration but flow functions must end in _flow
    collectedFlows: dict = {}
    try:
        collectedFlows = collect_flows_by_folder("flows/oneoff")
    except Exception as e:
        print("Error collecting flows:", e)

    parser = argparse.ArgumentParser(description="Run backend user flow simulations.")
    parser.add_argument(
        "-s",
        "--serial",
        action="store_true",
        help="Run flows serially",
    )
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument(
        "-f",
        "--flows",
        nargs="+",  # Allow multiple flows
        choices=list(REQUEST_FLOWS.keys())
        + list(CRUD_FLOWS.keys())
        + list(collectedFlows.keys()),
        help="The name(s) of the flow(s) to run",
    )
    group.add_argument(
        "-req",
        "--request_flows",
        action="store_true",
        help="Run all request flows",
    )

    group.add_argument(
        "-crud",
        "--crud",
        action="store_true",
        help="Run all crud flows",
    )
    group.add_argument(
        "-l",
        "--list",
        action="store_true",
        help="List all flows",
    )

    args = parser.parse_args()

    #####################################
    # Options that dont run any flows
    #####################################
    if args.list:
        print("\nCRUD_FLOWS:")
        print(*(f"\n{k}: {v}" for k, v in CRUD_FLOWS.items()))
        print("\n\nREQUEST FLOWS:")
        print(*(f"\n{k}: {v}" for k, v in REQUEST_FLOWS.items()))
        print("\n\nCOLLECTED FLOWS:")
        print(*(f"\n{k}: {v}" for k, v in collectedFlows.items()))
        return

    #####################################
    # "Main"
    #####################################

    # Run Migrations just to ensure we have a clean env
    start_time = time.perf_counter()
    print_group_separator("Initial Setup")
    db = DBClient()
    db.reset()

    is_parallel = True
    if args.serial:
        is_parallel = False

    # Filter and Run
    print_group_separator("Run Flows")
    flows_to_run = []
    allFlows = {**REQUEST_FLOWS, **CRUD_FLOWS, **collectedFlows}
    if args.request_flows:
        flows_to_run = REQUEST_FLOWS.keys()
        print(f"Flows to run: {flows_to_run}")
        request_flows_runner(flows_to_run, allFlows)

    if args.crud:
        flows_to_run = CRUD_FLOWS.keys()
        print(f"Flows to run: {flows_to_run}")
        print(is_parallel)
        crud_flows_runner(flows_to_run, allFlows, parallel=is_parallel)

    if args.flows:
        flows_to_run = args.flows  # Add explicitly specified flows
        print(f"Flows to run: {flows_to_run}")
        request_flows_runner(flows_to_run, allFlows)

    end_time = time.perf_counter()
    print("=" * 80)
    print(f"\nTest runner took: {end_time - start_time:.6f} seconds")


def run_isolated_flow(flow_name_and_func):
    flow_name, flow_function = flow_name_and_func

    provision_env_for_flow(flow_name)

    buffer = io.StringIO()
    try:
        with redirect_stdout(buffer):
            print(f"Running flow: {flow_name}\n{'-' * 60}")
            res = flow_function()
            formatted = format_crud_print(res)
            print("\n")
            print(formatted)
            print("\n")
        return flow_name, formatted, buffer.getvalue()
    except requests.exceptions.ConnectionError:
        msg = (
            f"\nFlow '{flow_name}' failed: Unable to connect to the backend. "
            f"Is it running?\n"
        )
        return flow_name, msg.strip(), buffer.getvalue() + msg
    except Exception as e:
        tb = traceback.format_exc()
        err = f"\nFlow '{flow_name}' failed with error: {e}\nStack trace:\n{tb}"
        return flow_name, f"Fail: {e}", buffer.getvalue() + err


def crud_flows_runner(flowsToRun, allFlows, parallel=True):
    """Run Flows with output that expects CRUD hashmap returned from each flow"""

    # Pair flow names with functions so we can parallelize
    flow_items = [(name, allFlows[name]) for name in flowsToRun]

    if parallel:
        with Pool() as pool:
            results = pool.map(run_isolated_flow, flow_items)
    else:
        results = crud_flows_runner_serial(flowsToRun, allFlows)

    # Output and summary collection
    crud_results = {}
    for flow_name, formatted_result, captured_output in results:
        print(captured_output)
        crud_results[flow_name] = formatted_result

    # Print test case Results
    print_group_separator("Results Summary")
    max_length = max(len(k) for k in crud_results.keys())
    for k, v in crud_results.items():
        print(f"{k.ljust(max_length)}: {v}")


def crud_flows_runner_serial(flowsToRun, allFlows):
    """Run Flows with output that expects CRUD results returned from each flow"""
    results = []
    for flow_name in flowsToRun:
        flow_function = allFlows[flow_name]
        buffer = io.StringIO()
        try:
            with redirect_stdout(buffer):
                print(f"Running flow: {flow_name}\n{'-' * 60}")
                res = flow_function()
                formatted = format_crud_print(res)
                print("\n")
                print(formatted)
                print("\n")
            results.append((flow_name, formatted, buffer.getvalue()))
        except requests.exceptions.ConnectionError:
            msg = f"\nFlow '{flow_name}' failed: Unable to connect to the backend. Is it running?\n"
            results.append((flow_name, msg.strip(), buffer.getvalue() + msg))
        except Exception as e:
            tb = traceback.format_exc()
            err = f"\nFlow '{flow_name}' failed with error: {e}\nStack trace:\n{tb}"
            results.append((flow_name, f"Fail: {e}", buffer.getvalue() + err))
    return results


def request_flows_runner(flowstoRun, allFlows):
    """Run Flows with the standard request flows style output"""
    test_case_results = {}
    for flow_name in flowstoRun:
        flow_function = allFlows[flow_name]
        try:
            print(f"Running flow: {flow_name}\n{'-' * 60}")
            test_case_results[flow_name] = flow_function()
            print(f"\nFlow '{flow_name}' completed successfully. \n")
        except requests.exceptions.ConnectionError:
            print(
                f"\nFlow '{flow_name}' failed: Unable to connect to the backend. Is it running? \n"
            )
        except Exception as e:
            test_case_results[flow_name] = f"Fail: {e}"
            print(f"\nFlow '{flow_name}' failed with error: {e} \n")
            print("Stack trace:")
            print(traceback.format_exc())

    # Print test case Results
    print_group_separator("Results Summary")
    # Determine the max length for entity names to align columns properly
    max_length = max(len(k) for k in test_case_results.keys())

    for k, v in test_case_results.items():
        print(f"{k.ljust(max_length)}: {v}")


@with_stack_trace
def collect_flows_by_folder(relativePath: str) -> Dict[str, Callable]:
    """
    Collects flows from a specific folder. Function names must end in _flow.
    The key will be the function name, and the value will be the function itself.
    """
    flows = {}
    folderToScan = os.path.join(os.path.dirname(__file__), relativePath)

    for filename in os.listdir(folderToScan):
        if filename.endswith(".py") and filename != "__init__.py":
            module_name = filename[:-3]  # Remove .py extension
            module_path = os.path.join(folderToScan, filename)

            spec = importlib.util.spec_from_file_location(module_name, module_path)
            if spec and spec.loader:
                module = importlib.util.module_from_spec(spec)
                spec.loader.exec_module(module)

                for name, obj in inspect.getmembers(module, inspect.isfunction):
                    if name.endswith("_flow"):
                        flows[name] = obj

    if len(flows) == 0:
        print_warn(
            f"Automatic registration: ${folderToScan} found, but no functions ending in _flow found. COLLECTED_FLOWS will be empty"
        )
    return flows


if __name__ == "__main__":
    main()
