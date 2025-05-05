import traceback
import requests
from utils.printing import format_crud_print


# These runners execute each flow in real-time (no output buffering),
# and returns result as a dictionary for easier programmatic access or reporting.
# If you need buffered output (e.g. for CI or replayable logs), see the parallel runner pattern.


def crud_flows_runner_serial(flows):
    """Run CRUD flows serially with real-time output, returns {flow_name: result}"""
    results = {}
    for flow_name, flow_function in flows:
        print(f"Running flow: {flow_name}\n{'-' * 60}")
        try:
            res = flow_function()
            formatted = format_crud_print(res)
            print("\n" + formatted + "\n")
            results[flow_name] = formatted
        except requests.exceptions.ConnectionError:
            msg = f"\nFlow '{flow_name}' failed: Unable to connect to the backend.\n"
            print(msg)
            results[flow_name] = msg.strip()
        except Exception as e:
            tb = traceback.format_exc()
            err = f"\nFlow '{flow_name}' failed: {e}\nStack trace:\n{tb}"
            print(err)
            results[flow_name] = f"Fail: {e}"
    return results


def request_flows_runner_serial(flows):
    """Run Flows with the standard request flows style output"""
    test_case_results = {}
    for flow_name, flow_function in flows:
        print(flow_name, flow_function)

    for flow_name, flow_function in flows:
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
    return test_case_results
