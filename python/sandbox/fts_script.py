import time
import random
import string
from sqlalchemy import create_engine, text

# This file is actually not connected to the runner at all (you would just run fts_script.py to run this), its just here to show a potential structure you might use

# Connect to PostgreSQL
DB_URL = "postgresql://postgres:postgres@localhost:5432/testdb"
engine = create_engine(DB_URL)

NUM_ROWS = 1_000_000  # rows
WORDS = [
    "performance",
    "optimization",
    "postgresql",
    "full-text",
    "search",
    "index",
    "query",
    "speed",
    "ranking",
    "database",
]


def random_text():
    return " ".join(
        random.choices(
            WORDS
            + ["".join(random.choices(string.ascii_lowercase, k=5)) for _ in range(3)],
            k=10,
        )
    )


with engine.begin() as conn:
    conn.execute(text("DROP TABLE IF EXISTS posts"))
    conn.execute(
        text("""
        CREATE TABLE posts (
            id SERIAL PRIMARY KEY,
            title TEXT,
            body TEXT
        )
    """)
    )

    # Insert random data
    insert_query = "INSERT INTO posts (title, body) VALUES (:title, :body)"
    data = [{"title": random_text(), "body": random_text()} for _ in range(NUM_ROWS)]
    print("inserting")
    conn.execute(text(insert_query), data)

    # Create indexes
    conn.execute(
        text(
            "CREATE INDEX idx_posts_fts ON posts USING GIN(to_tsvector('english', title || ' ' || body))"
        )
    )


# Run benchmarks
def run_query(query):
    with engine.begin() as conn:
        start = time.time()
        conn.execute(text(query))
        return time.time() - start


query_ilike = "SELECT * FROM posts WHERE title ILIKE '%performance%'"
query_fts = "SELECT * FROM posts WHERE to_tsvector('english', title || ' ' || body) @@ to_tsquery('performance')"
query_fts_ranked = """
    SELECT * FROM posts
    WHERE to_tsvector('english', title || ' ' || body) @@ to_tsquery('performance')
    ORDER BY ts_rank(to_tsvector('english', title || ' ' || body), to_tsquery('performance')) DESC
"""

print("running benchmarks")
# Measure execution times
results = {
    "ILIKE": run_query(query_ilike),
    "FTS": run_query(query_fts),
    "FTS (Ranked)": run_query(query_fts_ranked),
}

print("Benchmark Results:")
for method, time_taken in results.items():
    print(f"{method}: {time_taken:.4f} seconds")
