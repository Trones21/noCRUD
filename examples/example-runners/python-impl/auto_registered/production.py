from utils.api_client import APIClient
from utils.fixtures import get_fixture_by_index
from utils.crud import UpdateDetails, crud_exec
from utils.common import setup
from flows.auto_registered.universe import create as create_universe


def crud_production_flow():
    api = setup()
    update_details: UpdateDetails = {
        "fieldName": "title",
        "length": 49,
        "newValue": "Reconstructing Goodman",
    }
    crud_res = crud_exec("production", api, create, update_details)
    return crud_res


def create(api: APIClient):
    """
    Create the production.
    """
    # First create a universe (not strictly necessary in this example, but this shows the technique)
    universe_id = create_universe(api)

    # Create a production
    obj = get_fixture_by_index("productions.json", 0)
    obj["universe"] = universe_id
    res = api.create_object("production", obj)
    production_id = res["id"]
    return production_id
