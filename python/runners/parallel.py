from contextlib import redirect_stdout
import time
import io
import traceback
from utils.provisioning import cleanup_env, provision_env_for_flow
from utils.decorators import time_block
from utils import perf


def run_isolated_flow(flow_name_and_func, print_formatter=lambda x: x):
    """Run one flow in its own provisioned environment.

    Returns (flow_name, formatted_result, captured_output, elapsed_ms). The
    elapsed time covers the flow itself, not provisioning — that is the
    environment's cost rather than the flow's, and it is timed separately above.
    """
    flow_name, flow_function = flow_name_and_func
    perf.set_current_flow(flow_name)
    with time_block("provision env"):
        env = provision_env_for_flow(flow_name)
    buffer = io.StringIO()
    started = time.perf_counter()

    def elapsed_ms():
        return (time.perf_counter() - started) * 1000

    try:
        with redirect_stdout(buffer):
            print(f"Running flow: {flow_name}\n{'-' * 60}")
            res = flow_function()
            formatted = print_formatter(res)
            print("\n")
            print(formatted)
            print("\n")
            env["persist_db"] = False
        return flow_name, formatted, buffer.getvalue(), elapsed_ms()
    except Exception as e:
        env["persist_db"] = True
        tb = traceback.format_exc()
        err = f"\nFlow '{flow_name}' failed with error: {e}\nStack trace:\n{tb}"
        return flow_name, f"Fail: {e}", buffer.getvalue() + err, elapsed_ms()
    finally:
        cleanup_env(env)
        perf.flush_flow(flow_name)
