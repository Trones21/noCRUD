import requests
from utils.fixtures import get_fixture_by_index
from utils.decorators import with_perf, with_stack_trace
import json
import os

from utils.misc import random_string


def login(username, password):
    """Log in a user and return their token."""
    headers = {"Content-Type": "application/json"}
    response = requests.post(
        f"http://localhost:{os.getenv('APP_PORT')}/api/login/",
        json={"username": username, "password": password},
        headers=headers,
    )
    response.raise_for_status()
    print(f"API Client Created With {username}")
    return response.cookies


def new_api_client_with_user_via_fixture(user_fixture_index):
    """Creates an APIClient with the user"""
    users = get_fixture_by_index("users.json", user_fixture_index)
    creatingUser = users[user_fixture_index]["fields"]["username"]
    unhashed_password = users[user_fixture_index]["unhashed_pass"]
    api = APIClient()
    api.login(creatingUser, unhashed_password)
    return api


def new_api_client_with_new_user_via_fixture(user_fixture_index):
    """Creates an APIClient with a new user (retrieved via fixture). Note that this doesnt currently do any alteration,
    so this will fail if you've already loaded the fixtures and you have some unique constraints
    """
    users = get_fixture_by_index("users.json", user_fixture_index)
    creatingUser = users[user_fixture_index]["fields"]["username"]
    unhashed_password = users[user_fixture_index]["unhashed_pass"]
    api = APIClient()
    api.login(creatingUser, unhashed_password)
    return api


def new_api_client_with_new_random_user():
    """Creates an APIClient with a new randomly generated user"""
    unhashed_pass = random_string(12)
    creatingUser = {
        "username": random_string(12),
        "email": f"{random_string(12)}@example.com",
        "password": unhashed_pass,
    }

    api = APIClient()
    user_res = api.create_object("users", creatingUser)
    creatingUser = user_res.get("username", creatingUser["username"])
    api.login(creatingUser, unhashed_pass)

    return api


class APIClient:
    def __init__(self, base_url=None):
        port = os.getenv("APP_PORT", "8000")
        self.base_url = base_url or f"http://localhost:{port}/api"
        self.session = requests.Session()
        self.username = ""

    def login(self, username, password):
        """Log in a user and set cookies and X-CSRFToken"""
        headers = {"Content-Type": "application/json"}
        response = self.session.post(
            f"{self.base_url}/login/",
            json={"username": username, "password": password},
            headers=headers,
        )
        self.username = username
        response.raise_for_status()

        # Extract cookies and CSRF token
        csrf_token = response.cookies.get("csrftoken")
        if csrf_token:
            # Add CSRF token to session headers
            self.session.headers.update({"X-CSRFToken": csrf_token})

        print(f"\nAPI Client Created With {username}")
        return response.cookies

    # Django standard DRF CRUD format
    @with_perf("Create:")
    @with_stack_trace
    def create_object(self, endpoint, data):
        """Create an object at the specified endpoint and return the response."""
        response = self.session.post(f"{self.base_url}/{endpoint}/", json=data)
        print_on_fail_status(response)
        response.raise_for_status()
        print(f"Object created: {response.json()}")
        return response.json()

    @with_perf("Get:")
    @with_stack_trace
    def get_object_by_id(self, endpoint, object_id=None, silent=False):
        """Read an object by ID or get all objects at the specified endpoint."""
        url = f"{self.base_url}/{endpoint}/"
        if object_id:
            url += f"{object_id}/"
        response = self.session.get(url)
        print_on_fail_status(response)
        response.raise_for_status()
        if silent != True:
            print(f"Objects retrieved: {response.json()}")
        return response.json()

    @with_perf("Update:")
    @with_stack_trace
    def update_object_by_id(self, endpoint, object_id, data):
        """Update an object by ID at the specified endpoint."""
        response = self.session.put(
            f"{self.base_url}/{endpoint}/{object_id}/", json=data
        )
        print_on_fail_status(response)
        response.raise_for_status()
        print(f"Object updated: {response.json()}")
        return response.json()

    @with_perf("Delete:")
    @with_stack_trace
    def delete_object_by_id(self, endpoint, object_id):
        """Delete an object by ID at the specified endpoint."""
        url = f"{self.base_url}/{endpoint}/"
        if object_id:
            url += f"{object_id}/"
        response = self.session.delete(url)
        print_on_fail_status(response)
        response.raise_for_status()
        print(f"Object with ID {object_id} deleted successfully.")

    # Freeform endpoint - use when the url/endpoint you need doesnt follow DRF (this is common for "actions")

    @with_perf("GET:")
    @with_stack_trace
    def get(self, endpoint, silent=False):
        """Sends GET request the specified endpoint."""
        response = self.session.get(f"{self.base_url}/{endpoint}")
        print_on_fail_status(response)
        response.raise_for_status()
        if silent != True:
            print(f"{self.username}: object/s retrieved: {response.json()}")
        return response.json()

    @with_perf("POST:")
    @with_stack_trace
    def post(self, endpoint, data=None, silent=False):
        """Sends POST request to the specified endpoint."""
        response = self.session.post(f"{self.base_url}/{endpoint}/", json=data)
        print_on_fail_status(response)
        response.raise_for_status()
        if silent != True:
            print(f"{self.username}: object/s created (POST): {response.json()}")
        return response.json()

    @with_perf("PUT:")
    @with_stack_trace
    def put(self, endpoint, data=None, silent=False):
        """Sends PUT request to the specified endpoint."""
        response = self.session.put(f"{self.base_url}/{endpoint}/", json=data)
        print_on_fail_status(response)
        response.raise_for_status()
        if silent != True:
            print(f"{self.username}: object/s updated (PUT): {response.json()}")
        return response.json()

    @with_perf("DELETE:")
    @with_stack_trace
    def delete(self, endpoint, silent=False):
        """Sends DELETE request to the specified endpoint."""
        response = self.session.delete(f"{self.base_url}/{endpoint}")
        print_on_fail_status(response)
        response.raise_for_status()
        print(f"{self.username}: object/s deleted: {response}")

    # Cleanup methods dont raise on error (you might just run this to ensure that a table is empty before you write to it, but that of course would raise an error, which we dont want to see b/c it doesnt matter)
    def cleanup_delete(self, endpoint, silent=True):
        """Delete an object by ID at the specified endpoint."""
        url = f"{self.base_url}/{endpoint}"
        response = self.session.delete(url)
        if silent != True:
            return response

    def get_user_id_via_username(self, username):
        """Username is unique in backend so we dont currently need to handle
        a case where multiple users are returned from the get_object call"""
        res = self.get(f"users?username={username}", silent=True)
        return res["results"][0]["id"]

    def get_user_via_username(self, username):
        """Get user"""
        res = self.get(f"users?username={username}", silent=True)
        return res["results"][0]


def print_on_fail_status(response):
    if response.status_code > 299:
        print(
            f"🔴 ERROR {response.status_code}: {response.text}"
        )  # Log the exact response
        try:
            error_data = response.json()
            print(
                "🔍 Error Details:", json.dumps(error_data, indent=2)
            )  # Pretty-print JSON response
        except json.JSONDecodeError:
            print("⚠️ Response is not JSON:", response.text)
