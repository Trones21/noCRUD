# noCRUD Roadmap

Directions we want to take the tool. Rough design sketches, not commitments.

## 1. Framework adapters beyond Django

noCRUD is just HTTP calls; only three things are framework-specific — DB
provisioning, app startup, and the auth handshake (the "Framework Adapter" in
`.claude/skills/nocrud-scaffold/SKILL.md`).

**Approach:** rather than hand-writing an adapter per framework, the
`nocrud-scaffold` skill *generates* the adapter for the target backend against a
small contract (`provision_env_for_flow` / `cleanup_env` / `APIClient`). We may
still bundle a few reference adapters (Django done; Go next) as starting points.

**The hard part** is auth: knowing which headers dev expects (CSRF vs. bearer
token vs. API key, cookie names, content-type). That has to come from the user
or the backend's auth middleware — it can't be reliably guessed.

- [x] Django/DRF adapter (implemented in `utils/provisioning.py`)
- [x] Skill can generate an adapter for an unsupported framework
- [ ] Reference Go adapter (goose/golang-migrate or gorm; `go run`/binary; JWT)
- [ ] Reference adapters for other common stacks as they come up

## 2. Persisted timings for CI/CD (regression tracking) — implemented

Timings are captured per request (`@with_perf` in `utils/decorators.py`). They
still print inline, and now they can also be **persisted and compared** to catch
regressions. See `python/docs/PERF.md` for usage.

- [x] `--perf` flag on `noCRUD.py` persists timings to
      `perf/runs/<run_id>/<flow>.ndjson` as records
      `{run_id, git_sha, ts, flow, op, endpoint, ms}` (one file per flow, so
      parallel mode has no write contention).
- [x] `perf_report.py` aggregates a run by `(flow, op, endpoint)` and diffs it
      against a committed `perf/baseline.ndjson`, flagging regressions past a
      threshold and exiting non-zero (CI gate). `--set-baseline` promotes a run.
- [x] Percentiles (p95/p99) alongside mean; `perf_report.py --metric` gates on
      the chosen metric.
- [x] Consumer CI template — `example-runner-files/ci/perf-regression.yml`, which
      the consuming repo copies into its own `.github/workflows/` (run flows
      `--perf`, fail on regression, upload the run dir as an artifact).
- [x] Live dogfood CI — `.github/workflows/example-app-perf.yml` runs noCRUD
      against the bundled example_app on every PR (verified end to end on
      Python 3.11 + Django 4.2; pinned in `example_app/requirements.txt` and
      `python/requirements.txt`). Runs baseline-less (green, non-flaky); commit a
      CI-captured baseline to turn it into a real gate.

## 3. Performance / load characterization mode

A use case from the start: point noCRUD at an endpoint and **hammer it** — N
concurrent or for T seconds — to see requests/sec on a given instance size and
characterize latency under load. A third mode alongside CRUD and Request Flow.

**Sketch:**

- New runner (e.g. `runners/load.py`) + a flag: target endpoint, concurrency,
  duration; report req/sec, latency percentiles, error rate.
- Reuses `APIClient`, so auth and object-builders come for free.

**Why not just use k6/vegeta/locust?** You can — and AI can generate those
configs easily. noCRUD's edge is that it reuses the *same* auth and
dependency-aware object builders, so you can load-test an endpoint that needs a
fully-built object graph to exist first, without re-plumbing any of it.

## 4. Go runner parity — implemented

The Go **runner** (`go/`) is now at feature parity with the Python one. Separate
track from the Go *adapter* above (which is about testing a Go backend from the
Python runner).

- [x] CRUD flows, request flows, flow registry, `--list`
- [x] Serial and parallel runners, with per-flow output buffering
- [x] Per-flow app + database provisioning (migrate / schema.sql / template db),
      DB match check, cleanup that keeps a failing flow's database
- [x] `APIClient` with sessions and CSRF, multi-user flows, expected-failure
      assertions
- [x] Fixture loading, object builders, database back door
- [x] Persisted timings + regression report, using the **same NDJSON schema and
      op names as Python**, so a baseline is portable between the two
- [x] The supporting scripts: `perfreport`, `createcrudflow`, `modelcoverage`,
      `initdb`
- [x] Test suite covering the runner itself, end to end, against a fake DRF
      backend — no Django or postgres required to run it

The design differences Go forced (a `Ctx` per flow instead of process globals,
`init()` registration instead of a folder collector) are listed at the bottom of
[`go/Readme.md`](./go/Readme.md).

Still open:

- [ ] Load/hammer mode (item 3 above) in both runners
- [ ] A Go *adapter* — the reference for testing a Go backend (item 1 above)
