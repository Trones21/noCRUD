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

This skill supersedes the template generators that ship with each runner
(`python/create_crud_flow.py`, `go/cmd/createcrudflow`) — both pre-agentic. Do
the comprehension they couldn't: read the backend, infer the fields /
dependencies / rules, and write the flows yourself.

## Before you start

- **Ask which runner to scaffold: Go or Python.** Both are complete and at
  parity — see [Choosing the runner](#choosing-the-runner). Ask before
  generating anything; it changes every file you are about to write.
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

## Choosing the runner

`python/` and `go/` are both complete and do the same things — CRUD checks,
multi-user business-logic flows, per-flow isolated provisioning, parallel runs,
persisted timings. Their perf files share a schema and op names, so a baseline
captured by one can be compared by the other. Ask the user which they want.

**The backend's language is not the answer.** noCRUD speaks HTTP, so either
runner tests either backend — the Python runner against a Go API is a perfectly
normal setup. What should decide it is which language the *team will write and
maintain flows in*, since the flows are the thing they'll live with. Say this
when you ask, because "my backend is Go, so use the Go runner" is the assumption
people arrive with.

Tiebreakers if they have no preference:

- **Python** is the older implementation and the one the docs lead with. Prefer
  it if the team is not already fluent in Go.
- **Go** gives flows-as-goroutines rather than processes, so parallel runs cost
  the runner almost nothing, and it produces a single static binary — worth
  something in CI. Registration is automatic (`init()`), so there is no
  central list to keep in sync.

Once chosen, everything below has a per-runner form. Read the chosen runner's
`Readme.md` (`go/Readme.md` or `python/docs/README.md`) before generating.

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

Copy the chosen runner (`python/` or `go/`) to where the user wants it, or point
its config at the app in place. Then, per runner:

| Step        | Python                                        | Go                                                        |
| ----------- | --------------------------------------------- | --------------------------------------------------------- |
| Config      | `config.py`: `APP_DIR`, `FIXTURES_PATH`       | `config/config.go`, or the `NOCRUD_APP_DIR` / `NOCRUD_FIXTURES_PATH` env vars |
| Auth        | `utils/api_client.py::APIClient.login`        | `utils/apiclient/apiclient.go::Client.Login`               |
| Flows go in | `flows/`                                      | `flows/`                                                   |

**Adapting the auth handshake is the one piece you almost always must edit.**
Both bundled clients assume Django session + CSRF via `/api/login/`. Match the
project's real login endpoint, token vs. session, and header format. USAGE.md
flags this as the expected per-project customization.

If the runner was copied out of the noCRUD repo, note that the Go module path
travels with it: either keep the module name and import paths as they are, or
rename the module in `go.mod` and update imports to match. Mismatched import
paths are the first thing that breaks a copied Go runner.

## Phase 3 — Generate flows

For each endpoint in the route inventory, generate a **CRUD flow**, a
dependency-aware **fixture** so the object actually validates, and its
**registration**. Reuse the create function of each dependency rather than
hardcoding IDs — that is what makes them reusable as object builders. Infer a
sensible update field/value from the schema; don't ask.

| Piece        | Python                                                             | Go                                                                    |
| ------------ | ------------------------------------------------------------------ | --------------------------------------------------------------------- |
| File         | `flows/crud/<model>.py`, defining `crud()`                          | `flows/<model>.go`                                                     |
| Entry point  | `crud_exec(endpoint, api, create, update_details)`                  | `crud.Exec(c, api, endpoint, create, crud.UpdateDetails{...})`         |
| No-dep create| `simple_create`                                                     | `crud.SimpleCreate(endpoint, fixture, index, "id")`                    |
| Setup        | `setup()` from `utils/common.py`                                    | `c.Setup()`                                                            |
| Registration | Add to `CRUD_FLOWS` in `noCRUD.py`, or auto-register (see `example-runner-files/auto_registered/`) | `func init() { nocrud.RegisterCRUD("<model>", crud<Model>Flow) }` — automatic, no central list |

Working examples of every one of these live in `example-runner-files/` —
`crud/` and `auto_registered/` for Python, `go/flows/` for Go. Read the ones
matching the chosen runner before writing your own.

For the rules inventory, generate **business-logic flows**: multi-endpoint,
often multi-user sequences that assert a rule and its expected failures — e.g.
create as user A → read as user B expect 403 → grant permission as A → read as B
expect 200. This is the highest-value output and the part a template generator
never could produce — bias toward covering the real rules, not just happy-path
CRUD.

| Piece            | Python                                              | Go                                                                |
| ---------------- | --------------------------------------------------- | ----------------------------------------------------------------- |
| File             | `flows/confirm-business-logic/<rule>.py`            | `flows/<rule>.go`                                                  |
| Registration     | `REQUEST_FLOWS`                                     | `nocrud.Register(nocrud.Flow{Kind: nocrud.Request, Doc: "...", ...})` |
| A second user    | `new_api_client_*` in `api_client.py`               | `c.NewRandomUserClient()`                                          |
| Failure signal   | Let the exception raise                             | Return the error, or `nocrud.Must(v, err)` for setup steps         |
| Expected failure | Assert on the raised status                         | `nocrud.ExpectStatus(c, desc, 403, fn)` / `nocrud.ExpectFail(c, desc, fn)` |

Request timings are collected automatically in both — `@with_perf` decorators on
the Python `APIClient`, and inside the Go client's request path.

**Assert the rule, not a technicality.** The trap, and it produces a flow that
passes while proving nothing: to test that an edit is refused, send only the
field being changed — PATCH, not PUT. A PUT with a partial body is rejected for
the fields it *left out*, so the request fails validation before the rule is
ever consulted, and the flow would pass just as green against a backend with no
rule at all. This is a real bug that shipped in the bundled examples. Prefer
`ExpectStatus` over `ExpectFail` wherever the correct status is knowable, for
the same reason: "it failed somehow" is a much weaker claim than it looks.

## Phase 4 — Report and offer to run

- Run the coverage check — `python model_coverage_check.py` or
  `go run ./cmd/modelcoverage` — to report which models/endpoints got flows and
  which didn't. Report gaps **honestly**: a silently skipped endpoint reads as
  covered when it isn't.
- Confirm the flows are registered before claiming they exist. In Go this is
  one command, `go run ./cmd/nocrud -l`, and it catches a flow file that
  compiles but never registered itself.
- Summarize what you generated and where.
- Offer to run the flows. Run only if the user says yes.

  | | Everything | One flow | Against an app you started |
  | --- | --- | --- | --- |
  | Python | `python noCRUD.py -crud` | `-f <flow>` | add `--serial` |
  | Go | `go run ./cmd/nocrud -crud` | `-f <flow>` | add `--serial` |

## Framework Adapter

Everything above is HTTP and framework-agnostic. Only these three concerns are
per-framework. **This table is the source of truth for framework support — keep
it honest as adapters are added.**

Both runners implement Django/DRF, in the same three places:

| Concern         | Django/DRF (implemented)                                             | Python file              | Go file                              | Other frameworks |
| --------------- | ------------------------------------------------------------------- | ------------------------ | ------------------------------------ | ---------------- |
| DB provisioning | psql; per-flow DB via migrate, or bitwise template copy (`-T`)       | `utils/provisioning.py`  | `utils/provisioning/provisioning.go` | Not implemented — generate it (below) |
| Start the app   | `manage.py runserver <port>` per flow                                | `utils/provisioning.py`  | `utils/provisioning/provisioning.go` | Not implemented — generate it (below) |
| Auth handshake  | login → session + CSRF token                                         | `utils/api_client.py`    | `utils/apiclient/apiclient.go`       | Edit it for the project's auth |

### Generating an adapter for a new framework

When the target backend isn't Django, **write the adapter** rather than telling
the user it's unsupported. The interface is small, and the same shape in both
runners — provision, clean up, authenticate.

**Python** (`utils/provisioning.py`):

- `provision_env_for_flow(flow_name) -> dict` — create an isolated DB, load the
  schema / run migrations, start the app on a free port (use `find_open_port()`
  and `wait_for_backend_to_listen_on_port()`), and return
  `{"DB_NAME": ..., "APP_PORT": ..., "client": <db_client>, "proc": <process>}`.
- `cleanup_env(env)` — terminate `env["proc"]` and drop `env["DB_NAME"]` unless
  `env["persist_db"]` is set (the runner sets `persist_db=True` on failure so the
  DB can be inspected).
- `APIClient` in `utils/api_client.py` — the auth handshake and request methods.

**Go** (`utils/provisioning/provisioning.go`) — same contract, typed:

- `ProvisionEnvForFlow(ctx, flowName, out) (*nocrud.Env, error)` — same job,
  returning an `*nocrud.Env` with `DBName`, `AppPort`, `DBConfig`, `Admin` and
  `Proc` set. Use `FindOpenPort()` and `WaitForBackendToListen()`.
- `CleanupEnv(ctx, env, out)` — stop `env.Proc` and drop `env.DBName` unless
  `env.PersistDB` is set.
- `Client.Login` in `utils/apiclient/apiclient.go` — the auth handshake.
- Either replace those two functions, or leave them alone and pass your own as
  `runners.Options{Provision: ..., Cleanup: ...}` — they're function fields, so
  a new adapter needs no edit to the runner itself.

Model the new adapter on the existing `provision_django_env_*` /
`ProvisionDjangoEnv*` functions. What changes per framework:

- **DB provisioning** — the app's own migration tool, not `manage.py`. E.g. Go
  backends use goose / golang-migrate / gorm auto-migrate; run that against the
  fresh DB. The bitwise template-DB trick (`createdb -T`) is framework-agnostic
  and worth reusing once a migrated template exists.
- **Start the app** — `go run ./...` or a built binary on `$APP_PORT` instead of
  `manage.py runserver`.
- **Auth handshake** — match the real login (JWT bearer, session cookie, API
  key). **This is the fiddly part: you need to know which headers dev expects**
  (CSRF, `Authorization`, content-type, cookie names). Ask the user for these or
  read them from the backend's auth middleware — don't guess silently.

Confirm the app's DB engine, migration tool, run command, and auth scheme with
the user before writing the adapter. After writing it, run one flow to prove the
provision → request → cleanup loop works end to end before generating the rest.

Keep the Framework Adapter table above updated when you add one.

## Notes

- Request timings are collected per request in both runners. They print inline,
  and with the `--perf` flag they are also persisted and compared against a
  baseline for regression tracking — see `python/docs/PERF.md`. The two write
  the same schema and the same op names, so a baseline captured by one runner
  can be compared by the other.
- The runner flags are the same in both: `-f`/`--flows`, `-crud`,
  `-req`/`--request_flows`, `-coll`/`--collected`, `-l`/`--list`,
  `-s`/`--serial`, `--perf`. Invocation differs — `python noCRUD.py <flags>`
  vs `go run ./cmd/nocrud <flags>`.
