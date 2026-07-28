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

## 2. Persisted timings for CI/CD (regression tracking)

Today timings are captured per request (`@with_perf` in `utils/decorators.py`)
but only **printed**. To catch performance regressions across runs we want to
save them.

**Sketch:**

- Route `@with_perf` output through the collector into structured records:
  `{flow, endpoint, method, ms, timestamp, git_sha}`.
- Write newline-delimited JSON (or CSV) under `perf/` — git-diffable and easy to
  publish as a CI artifact.
- On each run, optionally diff against a baseline (previous run, or a committed
  baseline file) and flag regressions past a threshold (e.g. p95 +20%).
- In CI: run flows → write the perf file → compare to baseline → warn/fail on
  regression. So after a code change you can see exactly how timings moved.

Touches `decorators.py`, `collector.py`, and runner output. Contained feature.

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

## 4. Go runner parity

The Go **runner** (`go/`) lags the Python one significantly. Separate track from
the Go *adapter* above (which is about testing a Go backend from the Python
runner). Goal: bring the Go runner up to feature parity with Python.
