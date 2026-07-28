---
name: nocrud-scaffold
description: >
  Scaffold noCRUD flows for a backend project — CRUD flows AND multi-step /
  multi-user business-logic flows — by discovering the backend's REST endpoints
  and rules, then wiring noCRUD in to run against it. Use when noCRUD has been
  cloned alongside a project and the user wants to "set up noCRUD", "scaffold
  noCRUD", "generate flows for my backend", "scaffold API tests", or "wire
  noCRUD into this project".
---

# noCRUD Scaffolding

The user has a backend and wants noCRUD to exercise it. Your job is to go from
"here is my backend" to "runnable flows against it" with no manual templating.

noCRUD talks to the backend the same way a frontend does — **it is just HTTP
calls** — so the core of this work is framework-agnostic. Only three things are
framework-specific, and they are isolated in the [Framework Adapter](#framework-adapter)
section: provisioning an isolated database per flow, starting the application,
and the authentication handshake. Everything else is HTTP.

This skill supersedes `python/create_crud_flow.py` (a pre-agentic template
generator). Do the comprehension it couldn't: read the backend, infer the
fields / dependencies / rules, and write the flows yourself.

## Before you start

- Confirm noCRUD lives **alongside** the target project (sibling directories),
  and identify which directory is the backend to test.
- Confirm the backend's framework. **Django/DRF is the only framework with a
  working adapter today** (see the table below). For anything else, tell the
  user what's missing before proceeding — you can still generate flows, but the
  DB-provisioning and app-startup pieces will need to be written.
- Read `USAGE.md` and `example-runner-files/Readme.md` in the noCRUD repo for
  the runner's conventions before generating anything.
- **Do not auto-run the flows.** Generate, report, and *offer* to run them. Only
  run if the user asks.

## Phase 1 — Discover the backend

Produce two inventories and show them to the user before generating anything —
this is the checkpoint where they catch a missed endpoint or a wrong dependency.

**Route inventory:** path, HTTP method, auth required, request/response shape,
and dependencies (what must exist before this object can be created).

**Rules inventory:** the business logic worth asserting — permissions, who can
do what and when it should fail, validation, state transitions, and custom
actions.

Gather these from the best available source, in order, and merge:

1. **A live route/schema endpoint (best).** If the app runs or you can start it:
   - An **OpenAPI schema** (`/api/schema/`, `/openapi.json`, `/swagger.json`) is
     the goldmine — paths, methods, and field shapes in one place (drf-spectacular
     / drf-yasg expose this).
   - A **DRF browsable API root** (DefaultRouter lists registered routes).
2. **The source code.** URL conf and routers give routes; serializers and models
   give field shapes and dependencies; permission classes, serializer
   `validate_*`, model constraints, signals, and `@action` methods give the
   business logic. The schema rarely expresses *rules* — mine the code for those.
3. **Merge:** schema for endpoint/field shape, code for the rules the schema
   can't express.

Write the merged inventory to a scratch markdown and show the user.

## Phase 2 — Wire noCRUD into place

- Copy the `python/` runner to where the user wants it, or point config at it.
- Set `config.py`: `APP_DIR` (path to the app) and `FIXTURES_PATH`.
- **Adapt `utils/api_client.py::APIClient` to the project's auth.** The bundled
  one assumes Django session + CSRF via `/api/login/`. Match the project's real
  login endpoint, token vs. session, and header format. USAGE.md flags this as
  the expected per-project customization — it is the one piece you almost always
  must edit.

## Phase 3 — Generate flows

For each endpoint in the route inventory:

- **CRUD flow** — `flows/crud/<model>.py` with a `crud()` that calls
  `crud_exec(endpoint, api, create, update_details)`. Use `simple_create` when
  the object has no dependencies; write a custom `create` when it does. Infer a
  sensible `UpdateDetails` field/value from the schema — don't ask.
- **Fixture** — a dependency-aware fixture so the object actually validates.
  Reuse the `create()` flows of dependencies rather than hardcoding IDs.
- **Registration** — add the flow to `CRUD_FLOWS` in `noCRUD.py`, or use
  auto-registration (see `example-runner-files/auto_registered/`).

For the rules inventory, generate **business-logic flows** under
`flows/confirm-business-logic/`: multi-endpoint, often multi-user sequences that
assert a rule and its expected failures — e.g. create as user A → read as user B
expect 403 → grant permission as A → read as B expect 200. Build extra API
clients with different user creds (see `new_api_client_*` in `api_client.py`).
Register these in `REQUEST_FLOWS`. This is the highest-value output and the part
a template generator never could produce — bias toward covering the real rules,
not just happy-path CRUD.

Match the existing runner idioms: assertions raise (a raised exception stops the
flow and is the failure signal — see `crud_exec`), and request timings are
printed automatically via the `@with_perf` decorators on `APIClient`.

## Phase 4 — Report and offer to run

- Run `python/model_coverage_check.py` (or reason it through) to report which
  models/endpoints got flows and which didn't. Report gaps **honestly** — a
  silently skipped endpoint reads as covered when it isn't.
- Summarize what you generated and where.
- Offer to run the flows (`python noCRUD.py -crud`, or `-f <flow>` for one, or
  add `--serial` if the app is already running). Run only if the user says yes.

## Framework Adapter

Everything above is HTTP and framework-agnostic. Only these three concerns are
per-framework. **This table is the source of truth for framework support — keep
it honest as adapters are added.**

| Concern         | Django/DRF (implemented)                                             | Other frameworks |
| --------------- | ------------------------------------------------------------------- | ---------------- |
| DB provisioning | psql; per-flow DB via migrate, or bitwise template copy (`-T`). See `utils/provisioning.py` | Not implemented — must be written |
| Start the app   | `manage.py runserver <port>` per flow (`provisioning.py`)           | Not implemented — must be written |
| Auth handshake  | `APIClient` login → session + CSRF token (`utils/api_client.py`)     | Edit `APIClient` for the project's auth |

To support a new framework, fill in exactly these three rows — the discovery and
generation phases don't change.

## Notes

- The **Go** runner lags the Python one and has no scaffolding path yet; this
  skill targets the Python runner. Bringing Go to parity is separate work.
- Request timings are collected and printed per request (via `@with_perf`) but
  not yet aggregated or persisted.
