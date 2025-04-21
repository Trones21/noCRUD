from utils.api_client import APIClient
from utils.fixtures import get_fixture_by_index
from utils.crud import UpdateDetails, crud_exec
from utils.common import setup


def crud():
    api = setup()
    update_details: UpdateDetails = {
        "fieldName": "title",
        "length": 49,
        "newValue": None,
    }
    crud_res = crud_exec("episodes", api, create, update_details)
    return crud_res


def create(api: APIClient):
    """
    Create the episode.
    """
    # Create a episode
    obj = get_fixture_by_index("episodes.json", 0)
    res = api.create_object("episodes", obj)
    episode_id = res["id"]
    return episode_id
