from utils.crud import UpdateDetails, crud_exec, simple_create
from utils.common import setup


def crud():
    api = setup()
    update_details: UpdateDetails = {
        "fieldName": "first_name",
        "length": 49,
        "newValue": "WW",
    }
    crud_res = crud_exec(
        "character",
        api,
        simple_create(api, "character", 0, "id", "characters.json"),
        update_details,
    )
    return crud_res


## YOu dont necessarily need to use your own create function (if the logic is super simple), there is a generic one in utils.crud
# def create(api: APIClient):
#     """
#     Create the character.
#     """
#     obj = get_fixture_by_index("characters.json", 0)
#     res = api.create_object("character", obj)
#     obj_id = res["id"]
#     return obj_id
