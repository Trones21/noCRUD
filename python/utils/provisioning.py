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
    db_name = f"nocrud_p{port}_{flow_name}"

    # Update environment variables for the subprocess
    os.environ["DB_NAME"] = db_name
    os.environ["APP_PORT"] = str(port)

    # create db
    db_client = DBClient(admin_mode=True)
    db_client.createDB(db_name)

    # Run Migrations
    subprocess.run(
        ["python", "manage.py", "migrate"],
        cwd=APP_DIR,
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        env={**os.environ, "DB_NAME": db_name},
    )

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
        db_client = DBClient(admin_mode=True)
        db_client.dropDB(env["DB_NAME"])
    except Exception as e:
        print(f"⚠️ Cleanup warning: {e}")
