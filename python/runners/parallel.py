from contextlib import redirect_stdout
import io
import traceback
from utils.provisioning import cleanup_env, provision_env_for_flow


def run_isolated_flow(flow_name_and_func, print_formatter=lambda x: x):
    flow_name, flow_function = flow_name_and_func
    env = provision_env_for_flow(flow_name)
    buffer = io.StringIO()
    try:
        with redirect_stdout(buffer):
            print(f"Running flow: {flow_name}\n{'-' * 60}")
            res = flow_function()
            formatted = print_formatter(res)
            print("\n")
            print(formatted)
            print("\n")
        return flow_name, formatted, buffer.getvalue()
    except Exception as e:
        tb = traceback.format_exc()
        err = f"\nFlow '{flow_name}' failed with error: {e}\nStack trace:\n{tb}"
        return flow_name, f"Fail: {e}", buffer.getvalue() + err
    finally:
        cleanup_env(env)
