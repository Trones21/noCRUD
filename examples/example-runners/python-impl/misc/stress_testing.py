from utils.common import setup
from utils.db_client import DBClient
from python.scripts.flows.sandbox.backstory_testing import (
    get_backstory_template,
    makeRandomBackstories,
    create_single_objects,
)
from utils.printing import print_group_separator


def single_obj_hammer_flow():
    api = setup()
    db = DBClient()
    point = get_backstory_template(api)
    sizes = [100]
    for idx, size in enumerate(sizes):
        print_group_separator(f"Size: {size}")
        objs = makeRandomBackstories(point, size)
        create_single_objects(api, objs)
        db.clearTable("backstory")
    return
