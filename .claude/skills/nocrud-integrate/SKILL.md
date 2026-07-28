---
name: nocrud-integrate
description: >
  Integrate noCRUD into a backend project and generate flows for it — CRUD
  flows AND multi-step / multi-user business-logic flows — by discovering the
  backend's REST endpoints and rules. Use when noCRUD has been cloned alongside
  a project and the user wants to "set up noCRUD", "generate flows for my
  backend", "scaffold API tests", or "wire noCRUD into this project".
---

# noCRUD Integration & Flow Generation

<!-- SKETCH v0 — for review. Inline NOTE/OPEN markers are questions for the maintainer,
     not final instructions. Remove them before shipping. -->

## What this skill does

The user has a backend and wants noCRUD to exercise it. Your job is to go from
"here is my backend" to "runnable flows against it" with no manual templating.
noCRUD talks to the backend the same way a frontend does — **it is just HTTP
calls** — so the core of this skill is framework-agnostic. Only three things are
framework-specific, and they are isolated in the Framework Adapter section below:

1. Provisioning an isolated database per flow
2. Starting the application on a port
3. The authentication handshake (in `APIClient`)

This skill supersedes `python/create_crud_flow.py` (a pre-agentic template
generator). You do the comprehension it couldn't: read the backend, infer the
fields/dependencies/rules, and write the flows yourself.

## Phase 1 — Discover the backend

Goal: produce a **route inventory** (path, method, auth required, request shape,
dependencies) and a **rules inventory** (permissions, validation, state
transitions, custom actions). Try these sources in order and merge what you find:

1. **A live route/schema endpoint (best).** If the app is running or you can
   start it, look for:
   - An OpenAPI schema (`/api/schema/`, `/swagger.json`, `/openapi.json`) —
     this is the goldmine: paths, methods, and request/response field shapes in
     one place. drf-spectacular / drf-yasg expose this.
   - A DRF browsable API root (DefaultRouter lists registered routes).
   OPEN: should we prompt the user for the base URL / schema path, or probe a
   list of common ones?
2. **The source code.** URL conf and routers give routes; viewsets, serializers,
   and models give field shapes and dependencies; permission classes,
   serializer `validate_*`, model constraints, signals, and `@action` methods
   give the **business logic** worth asserting.
3. **Merge.** Use the schema for endpoint/field shape and the code for the rules
   the schema can't express (who can do what, when it should fail).

Write the merged inventory somewhere visible (e.g. a scratch markdown) and show
the user before generating — this is the checkpoint where they catch a missed
endpoint or a wrong dependency.

## Phase 2 — Wire noCRUD into place

- Copy the runner (`python/`) to where the user wants it, or point config at it.
- Set `config.py`: `APP_DIR` (path to the app) and `FIXTURES_PATH`.
- Adapt `utils/api_client.py::APIClient` to the project's auth (login endpoint,
  token vs session, header format). USAGE.md flags this as the expected
  per-project customization — it is the one piece you almost always must edit.

## Phase 3 — Generate flows

For each endpoint in the inventory:

- **CRUD flow** — a `flows/crud/<model>.py` with a `crud()` that calls
  `crud_exec(...)`, using `simple_create` or a custom `create` when there are
  dependencies. Infer a sensible `UpdateDetails` field/value from the schema
  instead of asking.
- **Fixture** — a dependency-aware fixture so the object actually validates.
  Reuse create() flows of dependencies rather than hardcoding IDs.
- **Registration** — add the flow to `CRUD_FLOWS` / `REQUEST_FLOWS` in
  `noCRUD.py` (or use auto-registration).

For the rules inventory, generate **business-logic flows** (`flows/
confirm-business-logic/`): multi-endpoint, often multi-user sequences that
assert the rule and its expected failures — e.g. create as A → read as B expect
403 → grant → read as B expect 200. This is the highest-value output and the
part a template generator never could produce.

## Phase 4 — Verify

- Run `model_coverage_check.py` (or an equivalent) to report which
  models/endpoints have flows and which don't. Report gaps honestly.
- Optionally run the flows (`python noCRUD.py -crud`) and surface results.

## Framework Adapter (the only framework-specific surface)

Everything above is HTTP. These three pieces are per-framework. **Django is the
only one implemented today; document status here as others are added.**

| Concern            | Django (implemented)                                             | Other frameworks |
| ------------------ | ---------------------------------------------------------------- | ---------------- |
| DB provisioning    | psql; per-flow DB via migrate, or bitwise template copy (`-T`)   | TODO / stub      |
| Start the app      | `manage.py runserver <port>` per flow (see `provisioning.py`)    | TODO / stub      |
| Auth handshake     | `APIClient` login → session/token                                | TODO / stub      |

NOTE: the "just make it generic" promise holds only if a new framework fills in
exactly these three rows. Keep this table honest as the source-of-truth for
implementation progress. (Go runner lags the Python one — separate parity task.)

## Non-goals / open questions for review

- OPEN: skill lives in-repo (`.claude/skills/`) so it ships when noCRUD is
  cloned alongside a project — confirm that's the intended delivery.
- OPEN: how much should we auto-run vs. stop at generated-but-not-run?
- OPEN: naming — `nocrud-integrate` vs `nocrud` vs `nocrud-scaffold`.
