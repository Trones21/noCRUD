from utils.api_client import APIClient
from utils.fixtures import get_fixture_by_index
from utils.crud import UpdateDetails, crud_exec
from utils.common import setup
from flows.crud.tag_category import create as create_tag_category


def crud():
    api = setup()
    update_details: UpdateDetails = {
        "fieldName": "name",
        "length": 49,
        "newValue": "newtagval",
    }
    crud_res = crud_exec("tag", api, create, update_details)
    return crud_res


def create(api: APIClient):
    """
    Create the tag.
    """
    tag_category_id = create_tag_category(api)

    tag = get_fixture_by_index("tags.json", 0)
    tag["tag_category"] = tag_category_id
    res = api.create_object("tag", tag)
    tag_id = res["id"]
    return tag_id
