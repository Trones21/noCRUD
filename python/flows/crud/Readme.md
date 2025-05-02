This is where you should place all your crud flows, assuming you are doing manual registration and have not done any modifications to `noCRUD.py`.

# Automatic Registration of CRUD Flows

If you want to do automatic registration of crud flows, then `noCRUD.py` will need some modification. Automatic registration is the very first step done when you run the program:

```python
# Avoiding manual registration but flow functions must end in _flow
    collectedFlows: dict = {}
    try:
        collectedFlows = collect_flows_by_folder("flows/crud") # make sure this points to the folder with your crud flows
    except Exception as e:
        print("Error collecting flows:", e)
```

The following section basically maps the flags to the flows you want to run.

```python
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
        crud_flows_runner(flows_to_run, allFlows, parallel=is_parallel)

    if args.flows:
        flows_to_run = args.flows  # Add explicitly specified flows
        print(f"Flows to run: {flows_to_run}")
        request_flows_runner(flows_to_run, allFlows)
```

There are a few different approaches at this point, but just make sure you are sending the correct dictionary of flows to the crud flows runner. The crud_flows_runner is needed to get the CRUD styled ouput

```python
    if args.crud:
        flows_to_run = collectedFlows.keys()
        print(f"Flows to run: {flows_to_run}")
        crud_flows_runner(flows_to_run, allFlows, parallel=is_parallel)
```

# Modifying the Flow Collector (Registrar)

Modify the collector if you wanted to have some different logic for determining which functions should be collected as flows

```python
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
```
