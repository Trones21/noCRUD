### Parallel runs

#### Port NNNNN was already taken before this flow's backend could start

```text
Port 45077 was already taken before this flow's backend could start.
The runner reserved this port and released it only to hand it over, so something
else grabbed it in that instant — another flow, or any outbound connection on
this machine (they draw from the same ephemeral range).
```

Rerun. This is rare by construction and a retry should clear it.

**Why it's reported rather than retried automatically.** The alternative to
failing here is running the flow against whatever else is on that port — a
different app, on a different database. That is how it used to behave: the
runner asked the OS for a free port, let go of it, spent several seconds
creating a database and running migrations, and only then started the app. If
anything took the port in the meantime, the app exited with "That port is
already in use", the *other* process answered the readiness probe, and the flow
ran to completion against someone else's data. It usually still reported a pass.

Two changes close that off, and both are covered by
`utils/provisioning_test.py`:

- `reserve_open_port()` **holds** the port until the moment the app takes it, so
  the window shrinks from seconds to microseconds.
- `assert_port_not_taken()` runs at handover, before the app is spawned — when
  nothing of ours can be answering yet, so anything that does answer is somebody
  else.

If you see this repeatedly, something on the machine is aggressively consuming
ephemeral ports; `cat /proc/sys/net/ipv4/ip_local_port_range` and check what
else is running.

#### Backend for port NNNNN exited before it started listening

The app died on startup and the last of its output follows the message. Common
causes are a settings error, a missing migration, or the port case above. This
used to sit through the full timeout and then report only that nothing ever
listened.

### New Project Setup

Error:

#### DB MISMATCH

```text
[DB MISMATCH] Django is using DB 'example', expected to use the db created by the runner 'api_runner_p60463_tag'.
            Please ensure that provision_env_for_flow and settings.py use the same environment variable for the database name.
            Settings.py MUST use an environment variable because the database name is programmatically generated
```

Open settings.py and ensure that the database name is being pulled from an environment variable:

```python
DATABASE_CONFIGURATION = {
    "ENGINE": "django.db.backends.postgresql",
    "NAME": environ.get("DB_NAME"),
    "USER": environ.get("DB_USER"),
    "PASSWORD": environ.get("DB_PASS"),
    "HOST": environ.get("DB_HOST"),
    "PORT": "5432",
}
```

and that it is the same envrionment variable that the provisioning method expects:

```python
def provision_django_env_using_migrate(flow_name):
    # You can derive a unique DB name and port from the flow name or hash
    port = find_open_port()
    db_name = f"noCRUD_p{port}_{flow_name}"

    # Update environment variables for the subprocess
    os.environ["DB_NAME"] = (
        db_name  # the key must match the os env var that settings.py looks to for getting the datbase name, the value is the db we just created
    )
    os.environ["APP_PORT"] = str(port)
```

and of course that it is set:

```text
vu@_:~/gh/noCRUD/python
$ env | grep DB
DB_PORT=5432
DB_USER=postgres
DB_HOST=localhost
DB_NAME=example
DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus
DB_PASS=postgres
```

It's expected the the value of the env var is different, you just need to ensure that the key is the same (since we are setting thre value with `os.environ`)
