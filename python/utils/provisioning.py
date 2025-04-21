import os
import socket
from utils.db_client import DBClient


def provision_env_for_flow(flow_name):
    # You can derive a unique DB name and port from the flow name or hash
    i = int("".join(filter(str.isdigit, flow_name)) or "0")  # crude fallback
    db_name = f"nocrud_{i}"
    port = find_open_port()

    # Update environment variables for the subprocess
    os.environ["POSTGRES_DB_NAME"] = db_name
    os.environ["APP_PORT"] = str(port)

    # create db
    dbClient = DBClient()
    # start django


def find_open_port():
    """Finds an available port by letting the OS assign one temporarily."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("", 0))  # 0 tells OS to find an available port
        return s.getsockname()[1]
