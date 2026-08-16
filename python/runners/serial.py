import time
import traceback
import requests
from utils.printing import format_crud_print
from utils import perf


# These runners execute each flow in real-time (no output buffering),
# and returns result as a dictionary for easier programmatic access or reporting.
# If you need buffered output (e.g. for CI or replayable logs), see the parallel runner pattern.
#
# Each entry is {flow_name: (formatted_result, elapsed_ms)} — the timing is what
# `--json` reports, and it is collected here rather than around the whole run so
# one slow flow is attributable.


def crud_flows_runner_serial(flows):
    """Run CRUD flows serially with real-time output, returns {flow_name: (result, ms)}"""
    results = {}
    for flow_name, flow_function in flows:
        print(f"Running flow: {flow_name}\n{'-' * 60}")
        perf.set_current_flow(flow_name)
        started = time.perf_counter()
        try:
            res = flow_function()
            formatted = format_crud_print(res)
            print("\n" + formatted + "\n")
            results[flow_name] = (formatted, _ms(started))
        except requests.exceptions.ConnectionError:
            msg = f"\nFlow '{flow_name}' failed: Unable to connect to the backend.\n"
            print(msg)
            results[flow_name] = (f"Fail: {msg.strip()}", _ms(started))
        except Exception as e:
            tb = traceback.format_exc()
            err = f"\nFlow '{flow_name}' failed: {e}\nStack trace:\n{tb}"
            print(err)
            results[flow_name] = (f"Fail: {e}", _ms(started))
        finally:
            perf.flush_flow(flow_name)
    return results


def request_flows_runner_serial(flows):
    """Run Flows with the standard request flows style output"""
    test_case_results = {}
    for flow_name, flow_function in flows:
        print(flow_name, flow_function)

    for flow_name, flow_function in flows:
        perf.set_current_flow(flow_name)
        started = time.perf_counter()
        try:
            print(f"Running flow: {flow_name}\n{'-' * 60}")
            test_case_results[flow_name] = (flow_function(), _ms(started))
            print(f"\nFlow '{flow_name}' completed successfully. \n")
        except requests.exceptions.ConnectionError:
            msg = f"Unable to connect to the backend. Is it running?"
            print(f"\nFlow '{flow_name}' failed: {msg} \n")
            test_case_results[flow_name] = (f"Fail: {msg}", _ms(started))
        except Exception as e:
            test_case_results[flow_name] = (f"Fail: {e}", _ms(started))
            print(f"\nFlow '{flow_name}' failed with error: {e} \n")
            print("Stack trace:")
            print(traceback.format_exc())
        finally:
            perf.flush_flow(flow_name)
    return test_case_results


def _ms(started):
    return (time.perf_counter() - started) * 1000
