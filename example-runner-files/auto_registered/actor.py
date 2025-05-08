from utils.crud import UpdateDetails, crud_exec, simple_create
from utils.common import setup


def crud_actor_flow():
    api = setup()
    update_details: UpdateDetails = {
        "fieldName": "first_name",
        "length": 49,
        "newValue": "Bill",
    }
    crud_res = crud_exec(
        "actor",
        api,
        lambda api: simple_create(api, "actor", 0, "id", "actors.json"),
        update_details,
    )
    return crud_res


## You dont necessarily need to use your own create function (if the logic is super simple), there is a generic one in utils.crud
# but this syntax isnt the nicest is you arent used to it (lambda or partial from functools)
# def create(api: APIClient):
#     """
#     Create the actor.
#     """
#     obj = get_fixture_by_index("actors.json", 0)
#     res = api.create_object("actor", obj)
#     actor_id = res["id"]
#     return actor_id
