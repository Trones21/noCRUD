from .db_client import DBClient
from .fixtures import get_fixture_by_index
from .api_client import APIClient


def setup():
    db = DBClient()
    db.clearDataExceptMinimalFixtures()

    # Login
    user = get_fixture_by_index("users.json", 0)
    creatingUser = user["username"]
    api = APIClient()
    api.login(creatingUser, "demo")
    return api
