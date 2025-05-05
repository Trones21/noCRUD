## Setting up the Example Runner

The `example-runner-files/` directory contains example flows you can copy into your runner implementation.

> ⚠️ **IMPORTANT:**
> These example files are designed to break if used directly.
> They contain _intentionally broken imports_.
> This is expected — once copied into the proper runner directories, all imports will resolve correctly.

---

### 🧪 Example: CRUD Flows

To load the CRUD example flows, copy them into your flow directory:

```bash
# From the noCRUD project root
cp ./example-runner-files/crud/* ./python/flows/crud/
```

This places the flow files into `./python/flows/crud/`, where the runner expects to find CRUD definitions.

---

### ⚙️ Configuration

No setup required — `config.py` is already pre-configured to match the example app structure.

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
