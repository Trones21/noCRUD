import argparse
import time

### Local Utils
from runners.main import crud_flows_runners, request_flows_runners
from utils.db_client import DBClient
from utils.printing import print_group_separator
from utils.collector import collect_flows_by_folder
##### Import Manually Registered Flows

### Manually Registered

# Dictionary of request flows
REQUEST_FLOWS = {}

# Dictionary of crud flows - should not be multi-user or multi endpoint (except creating prerequisite objects)
CRUD_FLOWS = {}


def main():
    # Automatic registration - Currently flow functions must end in _flow to be collected
    collectedFlows: dict = {}
    try:
        collectedFlows = collect_flows_by_folder("flows/auto_registered")
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
        "-coll",
        "--collected",
        action="store_true",
        help="Run all collected flows",
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

    start_time = time.perf_counter()

    is_parallel = True
    if args.serial:
        is_parallel = False
        # This is only done on the serial side - provision_env takes care of this in parallel mode
        print_group_separator("Initial Setup")
        db = DBClient()
        db.reset()

    # Filter and Run
    print_group_separator("Run Flows")
    flows_to_run = []
    allFlows = {**REQUEST_FLOWS, **CRUD_FLOWS, **collectedFlows}

    if args.request_flows:
        flows_to_run = REQUEST_FLOWS.keys()
        print(f"Flows to run: {flows_to_run}")
        flows = [(name, allFlows[name]) for name in flows_to_run]
        request_flows_runners(flows, parallel=is_parallel)

    if args.crud:
        flows_to_run = CRUD_FLOWS.keys()
        print(f"Flows to run: {flows_to_run}")
        flows = [(name, allFlows[name]) for name in flows_to_run]
        crud_flows_runners(flows, parallel=is_parallel)

    if args.collected:
        flows_to_run = collectedFlows.keys()
        print(f"Flows to run: {flows_to_run}")
        flows = [(name, allFlows[name]) for name in flows_to_run]
        request_flows_runners(flows, parallel=is_parallel)

    if args.flows:
        flows_to_run = args.flows  # Add explicitly specified flows
        print(f"Flows to run: {flows_to_run}")
        flows = [(name, allFlows[name]) for name in flows_to_run]
        request_flows_runners(flows, parallel=is_parallel)

    end_time = time.perf_counter()
    print("=" * 80)
    print(f"\nTest runner took: {end_time - start_time:.6f} seconds")


# @with_stack_trace
# def collect_flows_by_folder(relativePath: str) -> Dict[str, Callable]:
#     """
#     Collects flows from a specific folder. Function names must end in _flow.
#     The key will be the function name, and the value will be the function itself.
#     """
#     flows = {}
#     folderToScan = os.path.join(os.path.dirname(__file__), relativePath)
#     project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "."))


#     for filename in os.listdir(folderToScan):
#         if filename.endswith(".py") and filename != "__init__.py":
#             module_path = os.path.join(folderToScan, filename)
#             module_name = get_module_name_from_path(module_path, project_root)

#             spec = importlib.util.spec_from_file_location(module_name, module_path)
#             if spec and spec.loader:
#                 module = importlib.util.module_from_spec(spec)
#                 sys.modules[module_name] = module
#                 spec.loader.exec_module(module)

#                 for name, obj in inspect.getmembers(module, inspect.isfunction):
#                     if name.endswith("_flow"):
#                         flows[name] = obj

#     if len(flows) == 0:
#         print_warn(
#             f"Automatic registration: {folderToScan} found, but no functions ending in _flow found. COLLECTED_FLOWS will be empty"
#         )
#     return flows


# def get_module_name_from_path(module_path: str, project_root: str) -> str:
#     """
#     Converts a module file path into a dotted Python module path,
#     e.g., '.../flows/auto_registered/actor.py' → 'flows.auto_registered.actor'
#     """
#     rel_path = os.path.relpath(module_path, start=project_root)
#     no_ext = os.path.splitext(rel_path)[0]
#     return no_ext.replace(os.path.sep, ".")


# @with_stack_trace
# def collect_flows_by_folder_old(relativePath: str) -> Dict[str, Callable]:
#     """
#     Collects flows from a specific folder. Function names must end in _flow.
#     The key will be the function name, and the value will be the function itself.
#     """
#     flows = {}
#     folderToScan = os.path.join(os.path.dirname(__file__), relativePath)

#     for filename in os.listdir(folderToScan):
#         if filename.endswith(".py") and filename != "__init__.py":
#             module_name = filename[:-3]  # Remove .py extension
#             module_path = os.path.join(folderToScan, filename)

#             spec = importlib.util.spec_from_file_location(module_name, module_path)
#             if spec and spec.loader:
#                 module = importlib.util.module_from_spec(spec)
#                 spec.loader.exec_module(module)

#                 for name, obj in inspect.getmembers(module, inspect.isfunction):
#                     if name.endswith("_flow"):
#                         flows[name] = obj

#     if len(flows) == 0:
#         print_warn(
#             f"Automatic registration: ${folderToScan} found, but no functions ending in _flow found. COLLECTED_FLOWS will be empty"
#         )
#     return flows


if __name__ == "__main__":
    main()
