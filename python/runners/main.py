from functools import partial
from multiprocessing import Pool
from runners.parallel import run_isolated_flow
from runners.serial import crud_flows_runner_serial, request_flows_runner_serial
from utils.printing import (
    format_crud_print,
    format_req_flow_print,
    print_group_separator,
)


def crud_flows_runners(flows, parallel=True):
    """Run Flows with output that expects CRUD hashmap returned from each flow"""
    if parallel:
        print("Parallel Run Begin \n")
        run_isolated_crud_flow = partial(
            run_isolated_flow, print_formatter=format_crud_print
        )
        with Pool() as pool:
            buffered_results = pool.map(run_isolated_crud_flow, flows)
            print_buffered_results(buffered_results)
    else:
        serial_summary = crud_flows_runner_serial(flows)
        print_unbuffered_results(serial_summary)


def request_flows_runners(flows, parallel=True):
    if parallel:
        print("Parallel Run Begin \n")
        run_isolated_req_flow = partial(
            run_isolated_flow, print_formatter=format_req_flow_print
        )
        with Pool() as pool:
            buffered_results = pool.map(run_isolated_req_flow, flows)
            print_buffered_results(buffered_results)

    else:
        serial_summary = request_flows_runner_serial(flows)
        print_unbuffered_results(serial_summary)


def print_buffered_results(results):
    # Output and summary collection
    formatted_results = {}
    for flow_name, formatted_result, captured_output in results:
        print(captured_output)
        formatted_results[flow_name] = formatted_result

    # Summary
    print_group_separator("Results Summary")
    max_length = max(len(k) for k in formatted_results.keys())
    for k, v in formatted_results.items():
        print(f"{k.ljust(max_length)}: {v}")


def print_unbuffered_results(results):
    # We only print the summary b/c serial mode prints in real time
    print_group_separator("Results Summary")
    for name, result in results.items():
        print(f"{name}: {result}")
