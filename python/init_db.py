from utils.db_client import DBClient
from os import getenv

## This is actually not really related to the test runner
## it's just a quick way to create any DB with the DB_NAME env var

client = DBClient()
client.createDB(getenv("DB_NAME"))
