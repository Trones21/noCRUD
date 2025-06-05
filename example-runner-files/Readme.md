## Setting up the Example Runner

The `example-runner-files/` directory contains example flows you can copy into your runner implementation.

> ⚠️ **IMPORTANT:**
> These example files are designed to break if used directly.
> They contain _intentionally broken imports_.
> This is expected — once copied into the proper runner directories, all imports will resolve correctly.

---

### 🧪 Example: CRUD Flows

To load the CRUD example flows, copy them into your flow directory:

All commands assume your `pwd` is the noCRUD project root

```bash
cp ./example-runner-files/crud/* ./python/flows/crud/
```

This places the flow files into `./python/flows/crud/`, where the runner expects to find CRUD definitions.

---

### ⚙️ Configuration

Set Envrionment variables

```bash
source ./example_app/backend_env.sh
```

`config.py` is already pre-configured to match the example app structure.

---

### 🧰 Registering Flows Manually

If you're sticking with **manual registration** (recommended when starting out), you’ll need to import and register the flows:

Lets use the actor and tag flow as an example:

```python
# noCRUD.py

##### Import Manually Registered Flows
from flows.crud import actor
from flows.crud import tag

### Manually Registered

# Dictionary of request flows
REQUEST_FLOWS = {}

# Dictionary of crud flows - should not be multi-user or multi-endpoint (except for prerequisites)
CRUD_FLOWS = {
    "actor": actor.crud,
    "tag": tag.crud,
}
```

We should now see our flows listed when running -l/--list or -h/--help

```bash
python noCRUD.py -l
```

---

### Choose a Provisioning Method

Since the runner is parallel by default, we need each flow to provision its own db and app.

```text
             ┌───────────────────┐     ┌──────────────────┐
        -->  │ App :1 Port 8001  │ --> │ DB: noCRUD_p8001 │
       /     ├───────────────────┤     ├──────────────────┤
Runner --->  │ App :2 Port 8002  │ --> │ DB: noCRUD_p8002 │
       \     ├───────────────────┤     ├──────────────────┤
        -->  │ App :3 Port 8003  │ --> │ DB: noCRUD_p8003 │
             └───────────────────┘     └──────────────────┘
```

There are currently 3 pre-built provisioning methods. Open `provisioning.py` and look at `provision_env_for_flow`

One of the following calls needs to be uncommented. I recommend `provision_django_env_using_migrate` because there is no additional setup.

```python
    # return provision_django_env_direct_via_sql(flow_name)
    # return provision_django_env_using_migrate(flow_name)
    # return provision_django_env_direct_via_template_db(flow_name)
```

<details>
    I created the other two provsioning methods due to the overhead that running migrations for each flow introduced. I documented the performance difference in a blog article [here](https://thomasrones.com/projects/open-source-contributions/no-crud/b07177ba-890b-4dac-bfc9-770f92997f6f)

    [Youtube series](https://www.youtube.com/@not_only_CRUD) telling this optimization story is coming soon!

    noCRUD is intended to be a scaffold for you. So let's say you have enough flows that the performance difference matters to you and you prefer to create & migrate the DB with `createdb -T <db_name>`, and you dont want to manually create the db and run the migrations each time. Well in this case you would simply place the template db creation and migration in the single threaded startup portion of the application, before each flow kicks off its own thread. Definitely going to make another youtube video detailing this as well

</details>

> If this is your first time running the flows, pay attention to the output! I've added stack traces and error messages to assist when something may be wrong with the setup. Additionally, you can check out the `Common Issues` markdown in the docs folder.

### ▶️ Running the Flows

To run all registered CRUD flows:

```bash
python noCRUD.py -crud
```

To run only the actor flow:

```bash
python noCRUD.py -f actor
```

---

### ⚡ Execution Modes

The test runner supports both **parallel** and **serial** modes:

| Mode     | DB_USER            | DB_NAME Format          | Backend Started? | Notes                           | Runner Output                      | Backend Instance Output                                  |
| -------- | ------------------ | ----------------------- | ---------------- | ------------------------------- | ---------------------------------- | -------------------------------------------------------- |
| Serial   | use backend_env.sh | use backend_env.sh      | Already running  | Uses `backend_env.sh`           | Buffered to avoid log interleaving | Prefixed with `[Django:<port>]` but outut is interleaved |
| Parallel | `postgres`         | `nocrud_p<port>_<flow>` | Spawned per flow | Uses `provision_env_for_flow()` | Realtime                           | Shown in your other terminal                             |

#### ✅ Parallel (default)

Each flow spins up its own isolated instance of `example_app` on a random port.
You **do not need** to start `example_app` manually.

#### 🐢 Serial

In this mode, you **must** start `example_app` yourself in a separate terminal.
Make sure **both terminals** have the correct environment variables loaded:

```bash
source <path>/example_app/backend_env.sh
```

```bash
python manage.py runserver 8000
```

Run with the `--serial` flag:

```bash
python noCRUD.py -crud --serial
```
