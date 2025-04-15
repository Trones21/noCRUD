import random
import string
from utils.api_client import APIClient


# Produce a random string of N chars
def random_string(length):
    return "".join(random.choices(string.ascii_letters + string.digits, k=length))


# If i did want a random string with varying length I would want to ensure
# its never too small if used on a field with a unique constraint


def get_signature_id_by_user_id(user_id, signatureBlock):
    """SignatureBlock should be for a single entity name/id combo, therefore users should never appear more than once in the array"""
    for signature in signatureBlock:
        if signature["user_id"] == user_id:
            return signature["id"]


def get_user_id_via_username(apiclient: APIClient, username):
    """Username is unique in backend so we dont currently need to handle a case where multiple users are returned from the get_object call"""
    res = apiclient.get(f"users?username={username}", silent=True)
    return res["results"][0]["id"]
