# Persisted timings & regression tracking

Every request noCRUD makes is already timed (the `@with_perf` decorators on
`APIClient` print `Create: 12.30 ms`, etc.). The `--perf` flag additionally
**saves** those timings so you can compare runs over time — e.g. after a code
change, or as a CI gate — and see how your API's latency moved.

## Collecting timings

Add `--perf` to any run:

```bash
python noCRUD.py -crud --perf            # parallel
python noCRUD.py -crud --serial --perf   # serial (app already running)
python noCRUD.py -f actor --perf         # a single flow
```

This writes one file per flow to `perf/runs/<run_id>/<flow>.ndjson`. Each line is
one timed request:

```json
{"run_id": "...", "git_sha": "a18a15f", "ts": "...", "flow": "actor", "op": "create_object", "endpoint": "actor", "ms": 12.3}
```

At the end of the run it prints a table comparing this run to the baseline (if
one exists). Nothing is collected unless `--perf` is passed, so normal runs are
unchanged.

## Comparing & baselines

The comparison groups samples by `(flow, op, endpoint)` and reports the change in
mean latency.

```bash
python perf_report.py                 # compare the latest run vs baseline
python perf_report.py --run <run_id>  # compare a specific run
python perf_report.py --threshold 25  # regression threshold in % (default 20)
python perf_report.py --metric p95    # gate on p95 instead of mean (also p99)
python perf_report.py --set-baseline  # promote latest (or --run) to baseline
```

`perf_report.py` exits non-zero when any key regresses past the threshold, so it
can gate CI directly.

### Which metric?

Each key reports `count`, `mean`, `p95`, and `p99`. Gate on `mean` by default.
Percentiles (`--metric p95`/`p99`) catch tail regressions and need many samples
per key to be stable — with one request per op (plain CRUD) p95/p99 just equal
that single sample, and the report says so. They come into their own when a flow
hits an endpoint many times, or with a load/hammer flow.

## What's committed vs. transient

- `perf/baseline.ndjson` — **commit this**. It's the reference you compare
  against. Update it deliberately with `--set-baseline` when a new level of
  performance is the intended one.
- `perf/runs/` — transient per-run artifacts, git-ignored. In CI, upload the run
  dir as a build artifact if you want to keep the raw numbers.

## Typical CI shape

```bash
python noCRUD.py -crud --perf     # collect (also prints the diff)
python perf_report.py --threshold 20   # non-zero exit fails the job on a regression
```

There are two workflows in this repo:

- **Consumer template** —
  [`example-runner-files/ci/perf-regression.yml`](../../example-runner-files/ci/perf-regression.yml).
  Copy it into **your** project's `.github/workflows/` (the repo where you cloned
  noCRUD next to your app). Perf steps are wired; backend bring-up is marked
  `ADAPT`. It does not run in the noCRUD repo itself.
- **Live example-app CI** —
  [`.github/workflows/example-app-perf.yml`](../../.github/workflows/example-app-perf.yml)
  runs noCRUD against the bundled `example_app` on every PR (a working,
  end-to-end reference of this whole page). It runs without a committed baseline,
  so it's green and non-flaky; to make it a real gate, capture a baseline from a
  CI run's artifact and commit it as `python/perf/baseline.ndjson`.

## Notes / limits

- Comparison gates on **mean** by default; `--metric p95`/`p99` gate on the tail
  (see "Which metric?" above).
- `op` is the `APIClient` method name (`create_object`, `get_object_by_id`, …)
  and `endpoint` is the resource passed to it — together they distinguish the
  four CRUD operations on the same resource.
- Timings are wall-clock and machine-dependent; compare runs from the same
  environment (that's the point of a committed baseline + a CI runner).
