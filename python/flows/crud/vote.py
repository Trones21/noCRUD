from utils.api_client import APIClient
from utils.fixtures import get_fixture_by_index
from utils.crud import UpdateDetails, crud_exec
from utils.common import setup
from flows.crud.pitch import create as create_pitch


def crud():
    api = setup()
    update_details: UpdateDetails = {
        "fieldName": "value",
        "length": 49,
        "newValue": -1,
    }
    crud_res = crud_exec("vote", api, create, update_details)
    return crud_res


def create(api: APIClient):
    """
    Create the vote.
    """
    pitch_id = create_pitch(api)

    vote = get_fixture_by_index("votes.json", 0)
    vote["pitch"] = pitch_id
    res = api.create_object("vote", vote)
    vote_id = res["id"]
    return vote_id
