from utils.api_client import APIClient
from utils.fixtures import get_fixture_by_index
from utils.crud import UpdateDetails, crud_exec
from utils.common import setup
from flows.crud.production import create as create_production


def crud():
    api = setup()
    update_details: UpdateDetails = {
        "fieldName": "title",
        "length": 49,
        "newValue": None,
    }
    crud_res = crud_exec("episode", api, create, update_details)
    return crud_res


def create(api: APIClient):
    """
    Create the episode.
    """
    # Create a Production
    production_id = create_production(api)

    # Create a episode
    obj = get_fixture_by_index("episodes.json", 0)
    obj["production"] = production_id
    res = api.create_object("episode", obj)
    episode_id = res["id"]
    return episode_id
