from logging import raiseExceptions
from utils.api_client import APIClient
from utils.fixtures import get_fixture, get_fixture_by_index


def exec():
    """By using sessions, the APIClient will set the X-CSRFToken on login for you"""
    # Login
    # There is also a setup method in utils.common that wraps this, it's used in the biz_logic example
    users = get_fixture("users.json") # Ensure you set your fixtures path in fixtures.py
    creatingUser = users[0]['fields']['username']
    firstUser = APIClient()
    firstUser.login(creatingUser, "demo")

    # Create a Film
    filmObj = get_fixture_by_index("films.json", 0) # Pulls fields key directly from the object at param index
    film = firstUser.create_object("programs", filmObj)
    # Switch User
    secondUser = APIClient()
    updatingUser= users[1]['fields']['username']
    secondUser.login(updatingUser, "demo")

    # Update the Film
    film['title'] = "I updated the object"
    secondUser.update_object_by_id("films", film['id'], film)

    # Get the Program
    updatedfilm = secondUser.get_object_by_id("films", film['id'])

    # ===== Running some assertions ==== #
    # Store failed assertions
    failed_assertions = []

    # Perform assertions
    if updatedfilm['originated_by'] != creatingUser:
        failed_assertions.append(
            f"originated_by: ({updatedfilm['originated_by']}) does not match expected: ({creatingUser})")

    if updatedfilm['last_updated_by'] != updatingUser:
        failed_assertions.append(
            f"last_updated_by expected {updatingUser} but is {updatedfilm['last_updated_by']} ")

    if len(updatedfilm['title']) == 0:
        failed_assertions.append("Assertion failed: Banner message should have been populated on the update, but is empty")

    # Print results
    if failed_assertions:
        raise Exception("Assertions failed: ".join(failed_assertions))
    else:
        print("All assertions passed!")
    
    return "Success"

