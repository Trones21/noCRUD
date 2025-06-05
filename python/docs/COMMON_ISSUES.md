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
