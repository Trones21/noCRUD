from utils.api_client import APIClient, new_api_client_with_new_random_user
from utils.common import setup
from utils.fixtures import get_fixture_by_index


# Pitches cannot be edited after they have upvotes or comments
def exec():
    first_user: APIClient = setup()
    pitch = get_fixture_by_index("backstory_pitches.json", 1)
    pitchRes = first_user.create_object("pitch", pitch)

    second_user: APIClient = new_api_client_with_new_random_user()
    comment = get_fixture_by_index("pitch_comments.json", 1)
    # comment =
    comment["pitch"] = pitchRes["id"]
    second_user.create_object("pitch_comment", comment)
