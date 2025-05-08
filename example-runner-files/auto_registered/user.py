from utils.api_client import APIClient
from utils.fixtures import get_fixture_by_index
from utils.crud import UpdateDetails, crud_exec
from utils.common import setup
from utils.misc import random_string


def crud_user_flow():
    api = setup()
    update_details: UpdateDetails = {
        "fieldName": "phone",
        "length": 49,
        "newValue": "5551234567",
    }
    crud_res = crud_exec("users", api, create, update_details)
    return crud_res


def create(api: APIClient):
    """
    Create the user.
    """
    user = get_fixture_by_index("users.json", 0)
    user["username"] = random_string(
        10
    )  # since the users fixtures is likely already loaded
    res = api.create_object("users", user)
    user_id = res["id"]
    return user_id
