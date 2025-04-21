# CLI User Flow Tool Documentation

This documentation outlines the usage and purpose of the CLI User Flow Tool for verifying backend workflows. The tool allows developers to simulate and validate end-to-end user flows against the backend API, ensuring business logic works as expected in various scenarios. Additionally, it serves as a form of executable documentation, expressing complex business logic through API calls. For example, flows can demonstrate behaviors such as User A signing an approval and User B denying it, triggering a restart of the approval process, thereby reinforcing expected API behavior.

## Overview

The CLI User Flow Tool is designed to:

- Do a basic Check CRUD status on all endpoints
- Simulate complete user workflows, such as login, object creation, and updates.
- Automate verification of backend logic (e.g., prerequisites for actions).
- Reduce repetitive manual UI testing by directly interacting with the backend.
- Validate workflows or ensure specific failures when rules are violated.

You can also pull out the create method of an object that might have a very long dependency chain (e.g. 15 other objects need to exist before you create this one), and use this to fill th db. This is a bit better than just loading the fixtures because:

1. You are making actual API calls, therefore there's no chance of loading a set of objects in an incomplete/corrupted state (assuming the endpoints are all working as expected)
2. no need to manually manage ids to ensure the relationships are correct (the flow should halde this)

## Key Features

1. **Dynamic Flow Selection**: Run specific flows based on your needs.
2. **Reusable Utilities**: Common actions like login, object creation, and updates are modularized.
3. **Flow-Oriented Output**: Clear and concise output for success and failure states.
4. **Version-Controlled Test Data**: Pull fixtures directly from the repository to maintain consistency.

## Project Folder Structure and Organization

The project is organized to separate flows and CRUD operations for modularity and scalability. Flows define specific workflows, while CRUD operations contain reusable logic for interacting with API endpoints. Most **standard CRUD operations** can use the generic utilities provided in utils.py. The crud/ folder is reserved for any special cases where additional logic is required beyond the generic implementations.

```
<test-runner-root>/
    flows/                  # Folder for flow definitions
      crud/                 # CRUD flows
      <anything you want>/  # Any folders that you instruct the test runner to collect and place in the collectedFlows dict
      example_no_sessions.py  # Example flow using raw requests found in utils.py
      example_api_client.py # Example flow using the APIClient found in utils.py
    custom_requests/        # Folder for custom requests (that dont fit the patterns provided via utils.py (or for cleanliness, e.g. needs a massive custom object that you dont want affecting readability of the flow))
    utils/                  # All the helpers - APIClient, DBClient, decorators, fixture loading, etc.
    noCRUD.py            # Entry point / main
    README.md               # Documentation for the CLI tool
<path-to-your-fixtures>/
    <entity>.json           # App fixtures
```

## Prerequisites

- Python 3.7+
- `requests` library installed:
  ```bash
  pip install requests
  ```

## How to Use

### Using the Runner

To execute a specific flow, use the following command:

```bash
python noCRUD.py -f <flow-name>
```

For example:

```bash
python noCRUD.py -f example_api_client
```

**Run multiple:**

```bash
python noCRUD.py -f example_no_sessions example_api_client
```

**Run crud flows:**

```bash
python noCRUD.py -crud / --crud
```

**Run request flows (manually registered) (not only crud, but rather for biz logic or whatever):**

```bash
python noCRUD.py -req / --request_flows
```

**Run request flows (collected):**

```bash
python noCRUD.py tbd -- this flag isn't implenmented yet, for now you'll have to just use -f with the flow name(s)
```

### 2. Available Flows

There are two ways to add flows:

- Manual Registration
- Automatic Registration

You can list registered flows with the `-l / --list` flag

#### Manual Registration

Add more flows by creating a new file in the `flows/` folder and implementing the workflow logic. Then import and register the flow in the appropriate dictionary (in noCRRUD.py)

### Automatic Registration

You can register directories to be searched with the `collect_flows_by_folder(relativePath)` function. The inclusion logic is not currently exposed as an argument, its just `name.endswith("_flow")`

Crud flows and request flows are currently manually registered only... there's a visibility/familiarity tradeoff here. I am considering making the crud flows automatically registered... but then you might not open main (noCRUD.py) as often which means you will be less familiar with how it works (and therefore it's a bit harder to bend to your will).

I understand the truth matrix of what you run, how its collected (automatic/manual) isnt MECE... but again see my point above. Everyone using this tool should be intimately familiar with the runner source code, not just their tests.

### 3. Different Execution Paths Depending on the Flag / Group of Flows

Open `noCRUD.py` and you will notice that different flags use different runners:

```python
    if args.request_flows:
        flows_to_run = REQUEST_FLOWS.keys()
        print(f"Flows to run: {flows_to_run}")
        request_flows_runner(flows_to_run, allFlows)

    if args.crud:
        flows_to_run = CRUD_FLOWS.keys()
        print(f"Flows to run: {flows_to_run}")
        crud_flows_runner(flows_to_run, allFlows)

```

The main difference between these is that CRUD flows all output a very specific format, the CRUD dict, as seen in the implementation of `crud_exec`

```python
    crud["create"] = (
        True  # we would have gotten a stack trace and stopped execution if there was a failure so if we make it this far then we know its a success
    )
    crud["read"] = read(api, endpoint, id)
    crud["update"] = update(api, endpoint, id, updateDetails)
    crud["delete"] = delete(api, endpoint, id)
    return crud
```

Therefore we have a consistent structure that our `format_crud_print` function can rely on.

Output:

```shell
<add when I have the examples finished>
```

### 4. Data

Data is primarily loaded from the `example-app/api/fixtures/` folder. However, for cases where you need to test with data not included in the fixtures (e.g., intentionally malformed objects to validate error handling), you can define such data directly in the flow function. This allows you to simulate specific backend responses or verify rule enforcement without modifying the static fixture files.

### 5. Debugging

If a flow fails or doesn’t behave as expected:

- Check the output for error messages.
- Ensure the fixtures are correctly formatted.
- Verify the backend API endpoints and payloads.

For debugging the user flow itself, attach the debugger to the process:

<details>

#### Requirements:

You must have the proper configuration found in launch.json. The location of launch.json varies depending on the workspace root (directory you opened vscode in)

If you open to `<your-machine-path/`, then your launch.json must be at `<your-machine-path/<project>/.vscode/laumch.json`

```json
{
  "configurations":[
    {
      "name": "Python Debugger: noCRUD",
      "type": "debugpy",
      "request": "launch",
      "program": "${workspaceFolder}/<path-to_runner>/noCRUD.py",
      "console": "integratedTerminal",
      "args": [""]
    },
    <additional configurations>
  ]
}
```

You should see the configuration when you navigate to the debug tab `CTRL+SHIFT+D` and open the dropdown
![Dropdown](PkToDo)

Then just select corresponding comfiguration and run `F5`

Set breakpoints as desired.

</details>

### 5. Output Example (Not exact but just to give you an idea)

Running the `entity_creation` flow:

```bash
python noCRUD.py entity_creation
```

Output:

```text
Running Flow: Entity Creation
------------------------------------------------------------
Step 1: Attempting to create Entity B (expected to fail)
✅ Expected failure: Creation of Entity B is not allowed before Entity A.

Step 2: Creating Entity A
✅ Entity A created: {"id": 1, "type": "entity_a", "name": "Required Entity"}

Step 3: Creating Entity B
✅ Entity B created: {"id": 2, "type": "entity_b", "related_to": 1}

Flow 'entity_creation' completed successfully.
```

Output on Failure:

- When a unexpected failure occurs, we've included the stack trace

```bash

```

### In Progress

- Run each entity test on a separate db and backend (allows us to parallelize)

### To Do

mktest.py to scaffold tests following different standard patterns

## Contributing

- Ensure new flows are properly documented.
- Follow existing patterns for utility functions.
- Test all changes before submitting.
