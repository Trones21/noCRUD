### Directory Structure

| Directory         | Description / Status                                                                                                               |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| `python/`         | Python Implementation - **Serial version complete, currently working on parallelization**                                          |
| `go/`             | Go Implementation - **Super Basic - still playing around with the patterns, not as feature complete as the python implementation** |
| `example-app/`    | Example Application for running tests against                                                                                      |
| `example-runner/` | This is what your runner will look like after you start building it out a bit.                                                     |

### Using this Tool

Unlike many libraries and frameworks out there, you are meant to copy the source code directly over to start. If you compare the python implementation with the example-runner, you will see that the example-runner has all the same files, it just also has the actual test files that point against the example app.

I recommend looking through the example-runner before you start building out your own runner to get an idea of some of the patterns possible.

**See the readme inside the runner for the flags and examples.**

### Notes:

There aren't many authentication options yet, each implementation only has the auth options for the application I was creating the implementation for, but that's the beauty of having the source code! Just open up the `APIClient` and modify it to meet your needs. (same go for everything else in the utils folder... this is really the core of the application)

## How To Use

I'm currently debating whether or not I should keep the implementation folder clean or not (no examples from example app, keep those only in example runner)

Since the implementation is still undergoing quite a few changes, I've decided to use the runner directly from there (which also caused me to realize that i needed to pull out some config), but soon the implementation will be "clean" and I'll just have the examples in the example_runner. To currently get the "clean" version, just copy over everything in the python folder except for the `flows` and `sandbox` folders

## In your own environment

1. Copy python folder to where you want your runner
2. Update config.py
3. Create a flow (check out the example)
4. Run!

## Using with the example app (serial runs)

Open two terminals:

- one for the example app `cd to noCRUD/example_app`
- one for the runner `cd to noCRUD/python`

### Setup and Start the example_app

- Create as postgres db with the name that matches the POSTGRES_DB_NAME env var in backend_env.sh
- Run this:

```bash
   . backend_env.sh
   bash migrate_and_load_fixtures.sh
   python manage.py runserver
```

- Now open the terminal at the runner and try one of the commands, e.g. `python noCRUD.py -f actor --serial`
  **The serial flag is important. The runner defaults to parallel and it will provision apps and dbs on its own rather than using the one you just setup**

`<Pk_ToDo: Here is a video tutorial>`

## Using with the example_app (parallelized runs)

The runner will run flows in parallel by default. You **must** pass the -s flag to run flows serially.

Run: `python noCRUD.py -crud`

### How It Works

**Note: I am first implementing for `crud_flows_runner`, but the `req_flows_runner` will be pretty much the same**

Each flow is provisioned its own app and db (object in postgres, not instance)

The runner works the same up as serial mode until `crud_flows_runner` is called:

```python
def crud_flows_runner(flowsToRun, allFlows, parallel=True):
    """Run Flows with output that expects CRUD hashmap returned from each flow"""

    # Pair flow names with functions so we can parallelize
    flow_items = [(name, allFlows[name]) for name in flowsToRun]

    if parallel:
        with Pool() as pool:
            results = pool.map(run_isolated_flow, flow_items)
    else:
        results = crud_flows_runner_serial(flowsToRun, allFlows)
```

Now taking a look at `run_isolated_flow` you will see that it calls `provision_env_for_flow`, which is found in `provisioning.py`

```python
def run_isolated_flow(flow_name_and_func):
    flow_name, flow_function = flow_name_and_func
    provision_env_for_flow(flow_name)
```

`provision_env_for_flow` creates the db and starts the app on an open port

```python
 <Pk_todo: Add when provision_env_for_flow is complete>
```
