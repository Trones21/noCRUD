from utils.api_client import APIClient
from utils.fixtures import get_fixture_by_index
from utils.crud import UpdateDetails, crud_exec
from utils.common import setup


def crud_universe_flow():
    api = setup()
    update_details: UpdateDetails = {
        "fieldName": "name",
        "length": 49,
        "newValue": "A whole new world",
    }
    crud_res = crud_exec("universe", api, create, update_details)
    return crud_res


def create(api: APIClient):
    """
    Create the universe.
    """
    # Create a universe
    obj = get_fixture_by_index("universes.json", 0)
    res = api.create_object("universe", obj)
    universe_id = res["id"]
    return universe_id
