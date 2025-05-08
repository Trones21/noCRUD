import os
import importlib.util
import inspect
import sys
from typing import Dict, Callable

from utils.printing import print_warn
from utils.decorators import with_stack_trace


@with_stack_trace
def collect_flows_by_folder(relativePath: str) -> Dict[str, Callable]:
    """
    Collects flows from a specific folder. Function names must end in _flow.
    The key will be the function name, and the value will be the function itself.
    """
    flows = {}
    project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
    folder_to_scan = os.path.join(project_root, relativePath)
    print("root", project_root, "toScan", folder_to_scan)

    for filename in os.listdir(folder_to_scan):
        if filename.endswith(".py") and filename != "__init__.py":
            module_path = os.path.join(folder_to_scan, filename)
            module_name = get_module_name_from_path(module_path, project_root)

            spec = importlib.util.spec_from_file_location(module_name, module_path)
            if spec and spec.loader:
                module = importlib.util.module_from_spec(spec)
                sys.modules[module_name] = module
                spec.loader.exec_module(module)

                for name, obj in inspect.getmembers(module, inspect.isfunction):
                    if name.endswith("_flow"):
                        flows[name] = obj

    if len(flows) == 0:
        print_warn(
            f"Automatic registration: {folder_to_scan} found, but no functions ending in _flow found. COLLECTED_FLOWS will be empty"
        )
    return flows


def get_module_name_from_path(module_path: str, project_root: str) -> str:
    """
    Converts a module file path into a dotted Python module path,
    e.g., '.../flows/auto_registered/actor.py' → 'flows.auto_registered.actor'
    """
    rel_path = os.path.relpath(module_path, start=project_root)
    no_ext = os.path.splitext(rel_path)[0]
    return no_ext.replace(os.path.sep, ".")
