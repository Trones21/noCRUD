from utils.crud import UpdateDetails, crud_exec
from utils.common import setup
from utils.api_client import APIClient
from utils.fixtures import get_fixture_by_index
from flows.crud.production import create as create_production


def crud():
    api = setup()
    update_details: UpdateDetails = {
        "fieldName": "name",
        "length": 49,
        "newValue": "WW",
    }
    crud_res = crud_exec(
        "character",
        api,
        create,
        update_details,
    )
    return crud_res


def create(api: APIClient):
    """
    Create the character.
    """
    production_id = create_production(api)

    obj = get_fixture_by_index("characters.json", 0)
    obj["productions"] = [production_id]
    res = api.create_object("character", obj)
    obj_id = res["id"]
    return obj_id
