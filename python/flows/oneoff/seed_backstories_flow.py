from utils.common import setup
# from flows.crud.backstories import create as create_backstories

# Example that shows how you can seed the db with data (where there are dependent objects, and with real API calls rather than fixture insertion) 
def seed_backstories_flow():
    api = setup()
    
