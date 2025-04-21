from utils.api_client import APIClient
from utils.fixtures import get_fixture_by_index
from utils.crud import UpdateDetails, crud_exec
from utils.common import setup


def crud():
    api = setup()
    update_details: UpdateDetails = {
        "fieldName": "first_name",
        "length": 49,
        "newValue": "Bill",
    }
    crud_res = crud_exec("actor", api, create, update_details)
    return crud_res


def create(api: APIClient):
    """
    Create the actor.
    """
    obj = get_fixture_by_index("actors.json", 0)
    res = api.create_object("actor", obj)
    actor_id = res["id"]
    return actor_id
