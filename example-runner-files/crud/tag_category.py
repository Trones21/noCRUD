from utils.api_client import APIClient
from utils.fixtures import get_fixture_by_index
from utils.crud import UpdateDetails, crud_exec
from utils.common import setup


def crud():
    api = setup()
    update_details: UpdateDetails = {
        "fieldName": "name",
        "length": 49,
        "newValue": "new_tagcategory_val",
    }
    crud_res = crud_exec("tag_category", api, create, update_details)
    return crud_res


def create(api: APIClient):
    """
    Create the tag_category.
    """
    tag_category = get_fixture_by_index("tag_categories.json", 0)
    res = api.create_object("tag_category", tag_category)
    tag_category_id = res["id"]
    return tag_category_id
