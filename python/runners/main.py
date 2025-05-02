from multiprocessing import Pool
from runners.parallel import run_isolated_flow
from runners.serial import crud_flows_runner_serial
from utils.printing import print_group_separator


def crud_flows_runner(flowsToRun, allFlows, parallel=True):
    """Run Flows with output that expects CRUD hashmap returned from each flow"""

    # Pair flow names with functions so we can parallelize
    flow_items = [(name, allFlows[name]) for name in flowsToRun]

    if parallel:
        print("Parallel Run Begin \n")
        with Pool() as pool:
            results = pool.map(run_isolated_flow, flow_items)
    else:
        results = crud_flows_runner_serial(flowsToRun, allFlows)

    # Output and summary collection
    crud_results = {}
    for flow_name, formatted_result, captured_output in results:
        print(captured_output)
        crud_results[flow_name] = formatted_result

    # Print test case Results
    print_group_separator("Results Summary")
    max_length = max(len(k) for k in crud_results.keys())
    for k, v in crud_results.items():
        print(f"{k.ljust(max_length)}: {v}")
