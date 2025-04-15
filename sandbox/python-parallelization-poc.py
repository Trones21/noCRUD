from multiprocessing import Pool
import time
import random

# Approach (For Actual Implementation)
# 1 process per flow
# 1 app per flow == vary port 
# 1 db* per flow == vary db name
    # *(not multiple db instances, but db objects in postgres... still only one pg instance running)

# POC of the parallelization just for fun 
# ChatGPT was able to create this in a minute, so just wanted to play with it before wiring things up and making it real
# the other thing here is that we might need to rethink printing to the console 
def run_dummy_flow(i):
    t0 = time.time()
    # Simulate work: db setup, migrations, app start, tests...
    sleep_time = random.uniform(4.5, 5.5)  # simulate ~5s setup
    time.sleep(sleep_time)

    print(f"[flow {i}] completed in {time.time() - t0:.2f}s")
    return i

def run_benchmark(n_flows=500, parallelism=12):
    print(f"\nRunning {n_flows} flows with parallelism={parallelism}")
    start = time.time()

    with Pool(processes=parallelism) as pool:
        results = pool.map(run_dummy_flow, range(n_flows))

    duration = time.time() - start
    print(f"\n✅ Completed {n_flows} flows in {duration:.2f} seconds")

if __name__ == "__main__":
    run_benchmark(n_flows=500, parallelism=12)
