from logging import fatal
import copy
from utils.common import setup
from utils.misc import random_string
from utils.api_client import APIClient
from utils.fixtures import get_fixture_by_index
from flows.crud.backstory import BackstoryCreateRes, create as create_backstory
from utils.decorators import with_perf

# backstories migt not be the best object for this, but at least I've got the structure  

def large_obj_flow():
    api: APIClient = setup()
    # This object have many dependant objects, so use the create method in backstory crud
    id = create_backstory(api)
    backstory = api.get_object_by_id("backstory", id)
    # Now prep an update
    bigObj = ""
    for idx in range(8):
        bigObj += random_string(10000)

    print(len(bigObj))
    backstory["comments"] = bigObj
    api.update_object_by_id("backstory", backstory["id"], backstory)

    return True


def stress_backstory_flow():
    api: APIClient = setup()
    bs = get_backstory_template(api)
    bses = makeRandomBackstories(bs, 50)
    create_single_objects(api, bses)
    return


def run_perf_batch(api: APIClient, templateObj, count):
    objs = []
    # prep objects
    objs = makeRandomBackstories(templateObj, count)



@with_perf("Create Batch Objects: ")
def create_batch_objects(api, points):
    fatal("Not implemented yet")


@with_perf("Create Objs")
def create_single_objects(api: APIClient, bs, silent=True):
    count = 0
    size = len(bs)
    for b in bs:
        api.post("backstory", b, True)
        print(f"{count} of {size}")
        count += 1
    return


def get_backstory_template(api: APIClient):
    # Backstories have many dependant objects, so use the create method in backstory crud
    ids: BackstoryCreateRes = create_backstory(api)
    backstory = get_fixture_by_index("backstory.json", 0)
    backstory["discipline"] = ids["film_id"]
    backstory["character"] = ids["character_id"]
    return backstory


def makeRandomBackstories(backstory, count):
    bs = []
    for n in range(count):
        newObj = copy.deepcopy(backstory)
        newObj["backstories"] = random_string(30)
        bs.append(newObj)
    return bs
