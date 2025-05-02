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

### Config

Set `APP_DIR` and `FIXTURES_PATH` in config.py

### Create a crud flow via the template

Example: Create a crud flow for a `book` entity/url. On the PUT call, update the title field value to `newtitle`.
Please ensure that:

- the url is correct,
- the fixture exists (and the name matches),
- and the field exists on the model

```bash
python create_crud_flow.py book -uf title -ufv newtitle
```

Non crud flows are usually based on some kind of business logic and therefore don't have a standard template. Take a look at the examples for ideas.

Possible Upgrade (Not Impemented): Add flag to create pre-req objects (as seen in production.py)

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

Perfect — here's a drop-in replacement that keeps your structure and improves clarity and flow while including the caveat rewrite:

---

### Available Flows

There are two ways to add flows:

- **Manual Registration** _(Recommended when starting out)_
- **Automatic Registration** _(Better once you're familiar with the tool)_

You can check which flows have been registered — without running them — using:

```bash
python noCRUD.py -l
# or
python noCRUD.py --list
```

This helps you confirm registration worked as expected, especially when using automatic collection.

---

### 🔧 Manual Registration

This is the most explicit and beginner-friendly method. You add your flows directly to the appropriate dictionary in `noCRUD.py`:

```python
# noCRUD.py
CRUD_FLOWS = {
    "actor": actor.crud,
    "character": character.crud,
}
```

✅ **Benefits**:

- Easy to trace what's being run
- Encourages understanding of how runners work
- Ideal for debugging and learning

---

### ⚙️ Automatic Registration

Once you're more comfortable with the project, you can reduce boilerplate by automatically collecting flows from a folder:

```python
collect_flows_by_folder("flows/someFolder")
```

This will register any function in that folder that ends with `_flow`.

✅ **Benefits**:

- Great for scaling or generated tests
- Keeps the main file (`noCRUD.py`) lean
- Useful for prototyping and quick iteration

⚠️ **Caveat**:
It's easy to assume a flow was collected when it wasn’t. Use the `--list` flag to verify what was actually registered. This avoids surprises when running a batch of flows.

Also, be careful to pair flows with the correct runner (e.g., don’t run CRUD flows through the request runner unless you want raw output).

---

### Different Execution Paths Depending on the Flag / Group of Flows

IN `noCRUD.py` you may notice that different flags use different runners:

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

```text
episode     : C:✔ R:✔ U:✔ D:✔
actor       : C:✔ R:✔ U:✔ D:✔
character   : C:✔ R:✔ U:✔ D:✔
```

### Debugging

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

Running the `universe` flow:

```bash
python noCRUD.py -f universe
```

#### Output:

```text
Running flow: universe
------------------------------------------------------------
All tables cleared (except django required)
adding fixtures

API Client Created With var_undecided
Object created: {'id': 2, 'name': 'Breaking Bad Universe', 'description': 'Includes Breaking Bad, Better Call Saul, etc.'}
Create: 66.00 ms

Objects retrieved: {'id': 2, 'name': 'Breaking Bad Universe', 'description': 'Includes Breaking Bad, Better Call Saul, etc.'}
Get: 70.22 ms

Get: 71.04 ms

Object updated: {'id': 2, 'name': 'A whole new world', 'description': 'Includes Breaking Bad, Better Call Saul, etc.'}
Update: 80.30 ms

Object with ID 2 deleted successfully.
Delete: 86.06 ms

```

#### Output on Failure:

- When a unexpected failure occurs, we've included the stack trace

```text
Running flow: character
------------------------------------------------------------
All tables cleared (except django required)
adding fixtures

API Client Created With var_undecided
🔴 ERROR 400: {"production":["Invalid pk \"1\" - object does not exist."]}
🔍 Error Details: {
  "production": [
    "Invalid pk \"1\" - object does not exist."
  ]
}
[DEBUG] Exception raised in create_object

Flow 'character' failed with error: 400 Client Error: Bad Request for url: http://localhost:8000/api/character/
Stack trace:
Traceback (most recent call last):
  File "/home/thomasrones/gh/noCRUD/python/noCRUD.py", line 201, in crud_flows_runner_serial
    res = flow_function()
  File "/home/thomasrones/gh/noCRUD/python/flows/crud/character.py", line 15, in crud
    simple_create(api, "character", 0, "id", "characters.json"),
  File "/home/thomasrones/gh/noCRUD/python/utils/crud.py", line 12, in simple_create
    res = api.create_object(endpoint, obj)
  File "/home/thomasrones/gh/noCRUD/python/utils/decorators.py", line 17, in wrapper
    result = func(*args, **kwargs)
  File "/home/thomasrones/gh/noCRUD/python/utils/decorators.py", line 40, in wrapper
    return func(*args, **kwargs)
  File "/home/thomasrones/gh/noCRUD/python/utils/api_client.py", line 97, in create_object
    response.raise_for_status()
  File "/usr/lib/python3/dist-packages/requests/models.py", line 943, in raise_for_status
    raise HTTPError(http_error_msg, response=self)
requests.exceptions.HTTPError: 400 Client Error: Bad Request for url: http://localhost:8000/api/character/
```

#### Output Summary (CRUD Flows)

```text
================================================================================
                                Results Summary
================================================================================
episode   : Fail: 400 Client Error: Bad Request for url: http://localhost:8000/api/episode/
actor     : C:✔ R:✔ U:✔ D:✔
character : Fail: 400 Client Error: Bad Request for url: http://localhost:8000/api/character/
production: C:✔ R:✔ U:✔ D:✔
universe  : C:✔ R:✔ U:✔ D:✔
================================================================================
```

---

## ⚠️ A Note on Django Fixtures & Passwords

If you're loading users via fixtures in Django, remember this:

### **Django expects passwords to be pre-hashed.**

If you load a user fixture with a plaintext password like `"fixture_pass"`, authentication will fail because Django won't be able to match the unhashed string against its expected hash format.

---

### ✅ Two Ways to Handle It

#### 1. **Create users programmatically**

Use Django’s built-in `create_user()` to ensure the password is hashed:

```python
User.objects.create_user(username="foo", password="fixture_pass")
```

I would recommend dropping this in the `common.py` file, maybe just replace the `setup()` function or create a similar one.

#### 2. **Keep the unhashed password in the fixture (out-of-band)**

If you want to stick with fixtures, store the unhashed password at the top level:

```json
{
  "model": "api.users",
  "pk": 1,
  "unhashed_pass": "fixture_pass",
  "fields": {
    "username": "var_undecided",
    "first_name": "var",
    "last_name": "Undecided",
    "password": "pbkdf2_sha256$260000$..."
  }
}
```

Then return that field in your fixture utility:

```python
def get_fixture_by_index(filename, index):
    entry = get_fixture(filename)[index]
    fields = entry["fields"]
    if "unhashed_pass" in entry:
        fields["unhashed_pass"] = entry["unhashed_pass"]
    return fields
```

And use it in your setup:

```python
def setup():
    db = DBClient()
    db.clearDataExceptMinimalFixtures()

    user = get_fixture_by_index("users.json", 0)
    api = APIClient()
    api.login(user["username"], user["unhashed_pass"])

    return api
```

---

### 🧠 Why This Matters

This keeps your fixtures compatible with Django's expectations **and** your test code clean and DRY — no need to hardcode passwords in multiple places or hash them on load.

---

## Status

### In Progress

- Paralell Mode:

  - crud runner: Complete and tested!
  - request runner: not started
  - flow runner: not started

- Minor refactors

### To Do

- Decide how I want to do the backend output/logging for parallelized flows. The logs are currently mixed, but I do get the port...

  ```shell
  [Django:54947] Outgoing Response: 200
  [Django:43445] [02/May/2025 20:30:50] "GET /api/production/1/ HTTP/1.1" 200 213
  [Django:54947] [02/May/2025 20:30:50] "PUT /api/universe/1/ HTTP/1.1" 200 97
  [Django:50147] Outgoing Response: 201
  ```

  - By showing something it at least gives the user confidence that things are running as expected, so definitely not planning on removing setup and teardown, e.g.
    ✅ Database 'nocrud_p43445_production' created.
    ✅ Database 'nocrud_p54947_universe' created.
    ...
    🗑️ Database 'nocrud_p50147_episode' dropped.
    🗑️ Database 'nocrud_p37497_pitch' dropped.

  - but for the actual server side logging (the requests).. I'm not sure i can come to a good decision

  - I'm currently thinking one file per isolated flow... then a frontend could easily pick these up... but the way I envisioned this tool working is:
    1. Run batch - see what failed
    2. Choose an individual flow that failed and debug
    3. Run batch, move to working on another flow that failed
    - This might even clear up multiple. For example, you might see that a very fundamental entity failed and you know other entities depend on that one, so fixing that one causes the others to succeed as well

## Contributing

- Ensure new flows are properly documented.
- Follow existing patterns for utility functions.
- Test all changes before submitting.
