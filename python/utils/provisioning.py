import os
import socket
import subprocess
import sys
import threading
import time
from typing import Dict
from utils.db_client import DBClient
from config import APP_DIR
from pathlib import Path
import runpy


def provision_env_for_flow(flow_name) -> Dict:
    """
    This function provisions the environment for a given flow.
    Customize this per project.

    It is expected to:
    - Create an isolated DB
    - Run migrations or load schema
    - Start the backend on a unique port

    Returns:
        env = {
            "DB_NAME": db_name,
            "APP_PORT": port,
            "client": db_client,
            "proc": backend_proc,
        }
    """
    # Project-specific logic goes here:
    # set os environment variables to be injected into the process
    # create database
    # start server, etc.
    # See existing provision funcs for examples
    # raise NotImplementedError("Implement this function in your project.")
    # return provision_django_env_direct_via_sql(flow_name)
    return provision_django_env_using_migrate(flow_name)
    # return provision_django_env_direct_via_template_db(flow_name)


# =================================================
#  Pre-built Provision Funcs
# =================================================


def provision_django_env_using_migrate(flow_name):
    # You can derive a unique DB name and port from the flow name or hash
    port = find_open_port()
    db_name = f"noCRUD_p{port}_{flow_name}"

    # Update environment variables for the subprocess
    os.environ["DB_NAME"] = (
        db_name  # the key must match the os env var that settings.py looks to for getting the datbase name, the value is the db we just created
    )
    os.environ["APP_PORT"] = str(port)

    # create db
    db_client = DBClient(admin_mode=True)
    db_client.createDB(db_name)

    # Check for migrations file
    # Django Migrations do not have a --noinput option, so this will hang if makemigrations needs input (which it frequently does)
    # Therefore we want to actually run the migrations externally, or beforehand
    ensure_migrations_exist(APP_DIR)
    print("🛡️  MIGRATION CHECK PASSED — Found existing migration files.")

    run_mgmt_command_quietly(
        args=["migrate"],
        cwd=APP_DIR,
        env=os.environ,
    )
    print(f"✅  Migration succeeded for DB: {db_name}")

    # Ensure the db we just created is the same as what settings.py will use
    db_match_check(db_name, f"{APP_DIR}/example/settings.py")

    # start django
    backend_proc = start_backend_subprocess(port)

    env = {
        "DB_NAME": db_name,
        "APP_PORT": port,
        "client": db_client,
        "proc": backend_proc,
    }

    return env


def provision_django_env_direct_via_sql(flow_name):
    port = find_open_port()
    db_name = f"noCRUD_p{port}_{flow_name}"

    os.environ.update(
        {
            "DB_NAME": db_name,  # the key must match the os env var that settings.py looks to for getting the datbase name, the value is the db we just created
            "APP_PORT": str(port),
            "PGPASSWORD": "postgres",
        }
    )

    db_client = DBClient(admin_mode=True)
    db_client.createDB(db_name)

    ensure_schema_definition_exists(APP_DIR)

    subprocess.run(
        [
            "psql",
            "-U",
            "postgres",
            "-h",
            "0.0.0.0",
            "-d",
            db_name,
            "-f",
            f"{APP_DIR}/schema.sql",
        ],
        check=True,
        stdout=subprocess.DEVNULL,
    )

    # Ensure the db we just created is the same as what settings.py will use
    db_match_check(db_name, f"{APP_DIR}/settings.py")

    backend_proc = start_backend_subprocess(port)

    return {
        "DB_NAME": db_name,
        "APP_PORT": port,
        "client": db_client,
        "proc": backend_proc,
    }


def provision_django_env_direct_via_template_db(flow_name):
    port = find_open_port()
    db_name = f"noCRUD_p{port}_{flow_name}"

    os.environ.update(
        {
            "DB_NAME": db_name,  # the key must match the os env var that settings.py looks to for getting the datbase name, the value is the db we just created
            "APP_PORT": str(port),
            "PGPASSWORD": "postgres",
        }
    )

    provision_db_from_template(db_name, "template_db")

    db_client = DBClient(admin_mode=True)

    # Ensure the db we just created is the same as what settings.py will use
    db_match_check(db_name, f"{APP_DIR}/settings.py")

    backend_proc = start_backend_subprocess(port)

    return {
        "DB_NAME": db_name,
        "APP_PORT": port,
        "client": db_client,
        "proc": backend_proc,
    }


# =================================================
#  Helpers
# =================================================


def find_open_port():
    """Finds an available port by letting the OS assign one temporarily."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("", 0))  # 0 tells OS to find an available port
        return s.getsockname()[1]


def run_mgmt_command_quietly(args, cwd, env):
    result = subprocess.run(
        ["python", "manage.py"] + args,
        cwd=str(cwd),
        env=env,
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.PIPE,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(
            f"Command failed: {' '.join(args)}\nStderr:\n{result.stderr.strip()}"
        )


def ensure_schema_definition_exists(app_dir, schema_filename="schema.sql"):
    """
    Ensures that a schema.sql file exists in the given app directory.

    Args:
        app_dir (str or Path): The root directory of the app
        schema_filename (str): The name of the schema file to look for

    Raises:
        FileNotFoundError: If the schema file does not exist
    """
    schema_path = Path(app_dir) / schema_filename

    if not schema_path.exists():
        raise FileNotFoundError(
            f"❌ Schema definition not found. Expacted at: {schema_path}\n"
            "Please generate it using:\n"
            "    pg_dump -U postgres -h 0.0.0.0 --schema-only --no-owner --no-privileges <db_to_dump> > schema.sql\n"
            "Or switch to a provision method that uses Django migrations."
        )

    print(f"📄 Schema file found: {schema_path}")


def ensure_migrations_exist(apps_dir: Path):
    """
    Check if at least one migration exists (besides __init__.py) in each app's migrations/ folder.
    """
    apps_with_no_migrations = []

    for app_path in apps_dir.iterdir():
        migrations_dir = app_path / "migrations"
        if migrations_dir.is_dir():
            py_files = [
                f for f in migrations_dir.glob("*.py") if f.name != "__init__.py"
            ]
            if not py_files:
                apps_with_no_migrations.append(app_path.name)

    if apps_with_no_migrations:
        raise RuntimeError(
            f"Missing migration files in: {', '.join(apps_with_no_migrations)}"
        )


def provision_db_from_template(new_db, template_db="test_template"):
    """Creates a new database by cloning a template DB using createdb -T."""
    try:
        subprocess.run(
            ["createdb", "-h", "0.0.0.0", "-U", "postgres", "-T", template_db, new_db],
            check=True,
        )
        print(f"✅ Created DB from template: {new_db}")
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"❌ Failed to create DB: {e}")

    return new_db


def db_match_check(db_created_by_runner, settings_dot_py_path):
    try:
        db_match_check_via_settings(db_created_by_runner, settings_dot_py_path)
    except Exception as e:
        print(
            "🔄 Fast DB match check aborted: could not read settings.py. \n",
            "Falling back to slower DB match check (manage.py shell)\n",
            f"Inner exception was: \n→ {e}",
        )
        db_match_check_slow(db_created_by_runner)  # uses manage.py shell


def db_match_check_via_settings(db_created_by_runner, settings_path):
    """Faster than launching shell via manage.py"""
    # Run Django settings as a standalone script to get actual values
    context = runpy.run_path(settings_path)

    try:
        db_name = context["DATABASES"]["default"]["NAME"]
    except KeyError as e:
        raise RuntimeError(f"Missing DATABASES setting: {e}")

    if db_name != db_created_by_runner:
        raise RuntimeError(
            f"[SETTINGS MISMATCH] settings.py resolves DB name to '{db_name}', "
            f"but runner created '{db_created_by_runner}'"
        )

    print(f"✅ settings.py resolves DB to expected value: {db_name}")


def db_match_check_slow(db_created_by_runner):
    result = subprocess.run(
        [
            "python",
            "manage.py",
            "shell",
            "-c",
            "from django.db import connection; print(connection.settings_dict['NAME'])",
        ],
        cwd=str(APP_DIR),
        env={**os.environ},
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )

    if result.returncode != 0:
        raise RuntimeError(f"Failed to query current DB:\n{result.stderr.strip()}")

    db_that_backend_is_using = result.stdout.strip().splitlines()[-1]

    if db_that_backend_is_using != db_created_by_runner:
        raise RuntimeError(
            f"""[DB MISMATCH] Django is using DB '{db_that_backend_is_using}', expected to use the db created by the runner '{db_created_by_runner}'.
            Please ensure that provision_env_for_flow and settings.py use the same environment variable for the database name. 
            Settings.py MUST use an environment variable because the database name is programmatically generated"""
        )
    else:
        print(f"✅ Django is using expected DB: {db_that_backend_is_using}")


def start_backend_subprocess(port):
    proc = subprocess.Popen(
        ["python", "manage.py", "runserver", "--noreload", f"0.0.0.0:{port}"],
        cwd=APP_DIR,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        bufsize=1,
    )

    def stream_output():
        for line in proc.stdout:
            print(f"[Django:{port}] {line}", end="", file=sys.__stdout__)

    threading.Thread(target=stream_output, daemon=True).start()
    wait_for_backend_to_listen_on_port(port)
    return proc


def wait_for_backend_to_listen_on_port(port, host="0.0.0.0", timeout=10):
    """Blocks until a port is actively listening on the given host."""
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            with socket.create_connection((host, port), timeout=0.25):
                return True  # Success: something is listening
        except (ConnectionRefusedError, OSError):
            time.sleep(0.1)
    raise TimeoutError(f"Timeout waiting for server to listen on port {port}")


def cleanup_env(env):
    """Terminates the backend process and drops the DB"""
    try:
        env["proc"].terminate()
        if env["persist_db"]:
            print(f"Persisting db {env['DB_NAME']}")
        else:
            db_client = DBClient(admin_mode=True)
            db_client.dropDB(env["DB_NAME"])
    except Exception as e:
        print(f"⚠️ Cleanup warning: {e}")
