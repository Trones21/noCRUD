from .fixtures import get_fixture, get_fixture_by_index
from .api_client import APIClient
from .misc import random_string
from typing import Callable, TypedDict, Optional


def simple_create(
    api: APIClient, endpoint: str, fixtureIndex: int, idField: str, fixtureName: str
):
    """Assumes that the object is not dependent on other obejcts (and needs no other modification before request is sent)"""
    obj = get_fixture_by_index(fixtureName, fixtureIndex)
    res = api.create_object(endpoint, obj)
    return res[idField]


def read(api: APIClient, endpoint, id):
    api.get_object_by_id(endpoint, id)
    return True


class UpdateDetails(TypedDict):
    fieldName: str
    length: int
    newValue: Optional[str]


def update(api: APIClient, endpoint, id, updateDetails: UpdateDetails):
    obj = api.get_object_by_id(endpoint, id, True)
    if updateDetails["newValue"] is not None:
        expected = updateDetails["newValue"]
    else:
        expected = random_string(updateDetails["length"])
    obj[updateDetails["fieldName"]] = expected
    res = api.update_object_by_id(endpoint, id, obj)
    actual = res[updateDetails["fieldName"]]
    if expected == actual:
        return True
    print("Expected: ", expected, " Actual: ", actual)
    return False


def delete(api: APIClient, endpoint, id):
    api.delete_object_by_id(endpoint, id)
    return True


def crud_exec(
    endpoint: str, api: APIClient, createFunc: Callable[[APIClient], str], updateDetails
):
    crud = {}
    id = createFunc(api)
    crud["create"] = (
        True  # we would have gotten a stack trace and stopped execution if there was a failure so if we make it this far then we know its a success
    )
    crud["read"] = read(api, endpoint, id)
    crud["update"] = update(api, endpoint, id, updateDetails)
    crud["delete"] = delete(api, endpoint, id)
    return crud


def simpleGetAndCreate(api: APIClient, entityName, returnVal, id_field_name):
    returnValOptions = ["object", "id"]
    # if returnVal

    disciplines = get_fixture("disciplines.json")
    obj = disciplines[0]["fields"]
    res = api.create_object("disciplines", obj)
    discipline_id = res["discipline_id"]


# Creates an object and all prerequisite objects
def createPreRequisiteObjs(entityName):
    EntityFlows = {}
