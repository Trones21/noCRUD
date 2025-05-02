from contextlib import redirect_stdout
import io
import traceback
import requests
from utils.printing import format_crud_print, print_group_separator


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
