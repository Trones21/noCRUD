from .db_client import DBClient
from .fixtures import get_fixture_by_index
from .api_client import APIClient


def setup():
    db = DBClient()
    db.clearDataExceptMinimalFixtures()

    # Login
    user = get_fixture_by_index("users.json", 0)
    api = APIClient()
    api.login(user["username"], user["unhashed_pass"])
    return api
