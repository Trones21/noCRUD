from utils.api_client import login, create_object
from utils.fixtures import get_fixture

def exec():
    """This example uses requests without a session
    but the same thing can be done using sessions. 
    By using a session, the APIClient will set the X-CSRFToken on login for you"""
    
    #Login
    users = get_fixture("users.json")
    user = users[0]
    cookies = login(user['fields']['username'], user['fields']['password'])
    
    #Set headers for subsequent requests
    csrftoken = cookies.get("csrftoken")
    headers = {"Content-Type": "application/json",
               "X-CSRFToken": csrftoken
               }
    
    #PK-ToDo - update

    return "Success"
    