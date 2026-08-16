from functools import partial
from multiprocessing import Pool
import time

from runners.parallel import run_isolated_flow
from runners.serial import crud_flows_runner_serial, request_flows_runner_serial
from utils import results
from utils.printing import (
    format_crud_print,
    format_req_flow_print,
    print_group_separator,
)


# Each runner prints exactly what it always did, and also returns a normalized
# list of flow records so `--json` has something structured to write. Printing
# and reporting stay separate on purpose: the terminal output is for a person
# mid-run, the records are for whatever reads the run afterwards.


def crud_flows_runners(flows, parallel=True, kind="crud"):
    """Run Flows with output that expects CRUD hashmap returned from each flow"""
    return _run(flows, parallel, kind, format_crud_print, crud_flows_runner_serial)


def request_flows_runners(flows, parallel=True, kind="request"):
    """Run Flows with the standard request-flow output.

    `kind` is what the record is labelled with, and it is a parameter because
    this runner is also the path for -coll and -f. Those flows have no declared
    kind in the Python runner — unlike Go, where every flow registers one — so
    labelling them "request" would put a crud/request split into the results
    that nobody actually declared.
    """
    return _run(flows, parallel, kind, format_req_flow_print, request_flows_runner_serial)


def _run(flows, parallel, kind, print_formatter, serial_runner):
    if parallel:
        print("Parallel Run Begin \n")
        run_one = partial(run_isolated_flow, print_formatter=print_formatter)
        with Pool() as pool:
            buffered_results = pool.map(run_one, flows)
            print_buffered_results(buffered_results)
        return [
            results.flow_record(name, kind, formatted, ms)
            for name, formatted, _output, ms in buffered_results
        ]

    serial_summary = serial_runner(flows)
    print_unbuffered_results(serial_summary)
    return [
        results.flow_record(name, kind, formatted, ms)
        for name, (formatted, ms) in serial_summary.items()
    ]


def print_buffered_results(results):
    # Output and summary collection
    formatted_results = {}
    for flow_name, formatted_result, captured_output, _elapsed_ms in results:
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
    for name, (result, _elapsed_ms) in results.items():
        print(f"{name}: {result}")


def wall_ms(started):
    return (time.perf_counter() - started) * 1000
