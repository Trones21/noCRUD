from utils.db_client import DBClient
import random
import string


def random_db_name():
    return "testdb_" + "".join(random.choices(string.ascii_lowercase, k=6))


def test_create_and_drop():
    client = DBClient()
    db_name = random_db_name()

    print(f"\n🔧 Testing DB lifecycle: {db_name}")

    client.dropDB(db_name)  # cleanup if left over from a crash
    client.createDB(db_name)  # should succeed
    client.createDB(db_name)  # should warn: already exists
    client.dropDB(db_name)  # should clean up

    print("✅ test_create_and_drop passed\n")


if __name__ == "__main__":
    test_create_and_drop()
