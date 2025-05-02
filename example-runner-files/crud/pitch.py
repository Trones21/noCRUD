from utils.api_client import APIClient
from utils.fixtures import get_fixture_by_index
from utils.crud import UpdateDetails, crud_exec
from utils.common import setup
from flows.crud.character import create as create_character


def crud():
    api = setup()
    update_details: UpdateDetails = {
        "fieldName": "pitch_text",
        "length": 49,
        "newValue": "updated_text",
    }
    crud_res = crud_exec("pitch", api, create, update_details)
    return crud_res


def create(api: APIClient):
    """
    Create the pitch.
    """
    character_id = create_character(api)

    pitch = get_fixture_by_index("backstory_pitches.json", 0)
    pitch["character"] = character_id
    res = api.create_object("pitch", pitch)
    pitch_id = res["id"]
    return pitch_id
