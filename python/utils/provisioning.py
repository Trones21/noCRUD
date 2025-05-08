import os
import socket
import subprocess
import sys
import threading
import time
from utils.db_client import DBClient
from config import APP_DIR


def provision_env_for_flow(flow_name):
    # You can derive a unique DB name and port from the flow name or hash
    port = find_open_port()
    db_name = f"noCRUD_p{port}_{flow_name}"

    # Update environment variables for the subprocess
    os.environ["DB_NAME"] = db_name
    os.environ["APP_PORT"] = str(port)

    # create db
    db_client = DBClient(admin_mode=True)
    db_client.createDB(db_name)

    # Make migrations and migrate
    run_mgmt_command_quietly(args=["makemigrations"], cwd=APP_DIR, env=os.environ)
    print(f"✅ Migration created for DB: {db_name}")

    run_mgmt_command_quietly(args=["migrate"], cwd=APP_DIR, env=os.environ)
    print(f"✅ Migration succeeded for DB: {db_name}")

    # Ensure the db we just created is the same as what settings.py will use
    db_match_check(db_name)

    # start django
    backend_proc = start_backend_subprocess(port)

    env = {
        "DB_NAME": db_name,
        "APP_PORT": port,
        "client": db_client,
        "proc": backend_proc,
    }

    return env


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
        stdout=subprocess.DEVNULL,
        stderr=subprocess.PIPE,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(
            f"Command failed: {' '.join(args)}\nStderr:\n{result.stderr.strip()}"
        )


def db_match_check(db_created_by_runner):
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
        ["python", "manage.py", "runserver", f"0.0.0.0:{port}"],
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
    time.sleep(3)
    return proc


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
