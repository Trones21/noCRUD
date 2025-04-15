import os
import json
from typing import Optional
import subprocess


# Opening fixtures
def resolve_path(relative_path):
    """Resolve a file path relative to the project root."""
    currentPath = os.path.dirname(__file__)
    esperBackendRoot = os.path.dirname(os.path.dirname(currentPath))
    return os.path.join(esperBackendRoot, relative_path)


def get_fixture(filename):
    """Wrapper for open_json"""
    fixtureDir = resolve_path("./api/fixtures/")
    return open_json(fixtureDir + filename)


def get_fixture_by_index(filename, index):
    fixtures = get_fixture(filename)
    return fixtures[index]["fields"]


def open_json(filepath):
    """Utility to open a JSON file and return its contents."""
    if not os.path.exists(filepath):
        raise FileNotFoundError(f"JSON file not found: {filepath}")
    with open(filepath, "r") as file:
        return json.load(file)


def addFixtures(fixturesToAdd: Optional[list[str]] = None) -> None:
    print("adding fixtures")
    fixturesPath = "api/fixtures"

    if fixturesToAdd is None:
        fixturesToAdd = ["", ""]

    for fixture in fixturesToAdd:
        file = f"{fixturesPath}/{fixture}.json"
        result = subprocess.run(
            ["python", "manage.py", "loaddata", f"{file}"],
            cwd="..",
            capture_output=True,
            text=True,
            check=False,
        )
        if len(result.stderr) > 0:
            print("Error Adding Fixture: ", result.stderr)
