import os
import json
from typing import Optional
import subprocess

from config import APP_DIR, FIXTURES_PATH

# global => file local
app_dir = APP_DIR
fixtures_path = FIXTURES_PATH


# Opening fixtures
def resolve_path(relative_path):
    """Resolve a file path relative to the project root."""
    currentPath = os.path.dirname(__file__)
    esperBackendRoot = os.path.dirname(os.path.dirname(currentPath))
    return os.path.join(esperBackendRoot, relative_path)


def get_fixture(filename):
    """Wrapper for open_json"""
    fixtureDir = resolve_path(fixtures_path)
    return open_json(f"{fixtureDir}/{filename}")


def get_fixture_by_index(filename, index):
    entry = get_fixture(filename)[index]
    fields = entry["fields"]
    if "unhashed_pass" in entry:
        fields["unhashed_pass"] = entry["unhashed_pass"]
    return fields


def open_json(filepath):
    """Utility to open a JSON file and return its contents."""
    if not os.path.exists(filepath):
        raise FileNotFoundError(f"JSON file not found: {filepath}")
    with open(filepath, "r") as file:
        return json.load(file)


def addFixtures(fixturesToAdd: Optional[list[str]] = None) -> None:
    print("adding fixtures")
    if fixturesToAdd is None:
        fixturesToAdd = ["users"]

    for fixture in fixturesToAdd:
        file = f"{fixtures_path}/{fixture}.json"
        result = subprocess.run(
            ["python", "manage.py", "loaddata", f"{file}"],
            cwd=app_dir,
            capture_output=True,
            text=True,
            check=False,
        )
        if len(result.stderr) > 0:
            print("Error Adding Fixture: ", result.stderr)
