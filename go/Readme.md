# noCRUD — Go runner

At parity with the Python runner: CRUD checks, multi-user business-logic flows,
an isolated database and backend per flow, parallel runs, and persisted request
timings. The perf files use the same schema and the same op names as the Python
runner, so a baseline captured by either can be compared by the other.

Pick the implementation your team would rather write flows in. The flows read
about the same either way; what differs is underneath (see
[Differences from the Python runner](#differences-from-the-python-runner)).

## Quick start, against the bundled example_app

The example app is Django, so its dependencies come first:

```bash
python -m venv .venv && source .venv/bin/activate
pip install -r ../examples/example_app/requirements.txt
```

Point the runner at a postgres it can create databases in, and give Django its
key:

```bash
export DB_USER=postgres DB_PASS=postgres DB_HOST=localhost DB_PORT=5432
export DJANGO_KEY=dev-insecure-key
```

The example flows live outside the runner so `flows/` stays yours. Copy them in:

```bash
cp ../examples/example-runners/go-impl/flows/*.go ./flows/
go run ./cmd/nocrud -l          # confirm they registered
go run ./cmd/nocrud -coll       # run every one of them
```

Nothing else to start. In the default parallel mode each flow provisions its own
database and its own backend, runs, and tears both down again:

```
actor                        : C:✔ R:✔ U:✔ D:✔
character                    : C:✔ R:✔ U:✔ D:✔
pitch                        : C:✔ R:✔ U:✔ D:✔
pitch_lock_after_interactions: pitch locked after interaction, as expected
production                   : C:✔ R:✔ U:✔ D:✔
seed_pitches                 : seeded 5 pitches (with their characters, productions and universes)
universe                     : C:✔ R:✔ U:✔ D:✔
users                        : C:✔ R:✔ U:✔ D:✔
vote                         : C:✔ R:✔ U:✔ D:✔
```

## Layout

| Path                  | What it is                                                          |
| --------------------- | ------------------------------------------------------------------- |
| `nocrud/`             | The vocabulary flows are written in: `Flow`, `Ctx`, `Env`, the registry, `Must` / `ExpectFail` / `ExpectStatus` |
| `flows/`              | **Yours.** Ships empty; flow files register themselves from `init()` |
| `runners/`            | Executes flows, serial or parallel, and prints the summary          |
| `config/`             | The one package to edit when dropping this next to your own app     |
| `utils/apiclient/`    | Talks to the backend the way your frontend does. One client per user |
| `utils/crud/`         | The create → read → update → delete check                           |
| `utils/provisioning/` | Per-flow database and backend. **The framework-specific part**      |
| `utils/dbclient/`     | Postgres: create, drop, clear, reset, load fixtures                 |
| `utils/fixtures/`     | Reads the app's fixture json                                        |
| `utils/perf/`         | Records and compares request timings                                |
| `utils/jsonx/`        | Path-addressed JSON (`res.Get("results.0.id")`) without ceremony    |
| `cmd/`                | The commands below                                                  |

## Commands

```bash
go run ./cmd/nocrud          # the runner
go run ./cmd/modelcoverage   # which models have a CRUD flow and which don't
go run ./cmd/createcrudflow  # generate a CRUD flow file
go run ./cmd/perfreport      # compare a run against the baseline, or set one
go run ./cmd/initdb          # create and migrate a database
```

### Runner flags

| Flag                        | Effect                                                     |
| --------------------------- | ---------------------------------------------------------- |
| `-f`, `--flows`             | Run named flows. Repeat the flag or comma-separate          |
| `-crud`                     | Run every CRUD flow                                         |
| `-req`, `--request_flows`   | Run every request flow                                      |
| `-coll`, `--collected`      | Run everything registered                                   |
| `-l`, `--list`              | List registered flows and exit                              |
| `-s`, `--serial`            | Run against a backend you started yourself, printing live   |
| `-j`, `--jobs`              | Max flows at once in parallel mode (default GOMAXPROCS)     |
| `--perf`                    | Persist timings and compare against the baseline            |
| `--threshold`, `--metric`   | Regression gate: percent, and `mean` / `p95` / `p99`        |
| `--json <path>`             | Also write machine-readable results (see below)             |

One of `-f`, `-crud`, `-req`, `-coll` or `-l` is required — there is no default
that runs something you didn't ask for.

`-j` caps concurrency for the sake of the *backend and database*, not the
runner. Flows are goroutines here, so the runner itself is nearly free; each
concurrent flow is another app process and another database, and that is what a
small box runs out of.

### Machine-readable results

`--json <path>` writes what the run did, in a schema shared with the Python
runner (`nocrud.results/v1`):

```json
{
  "schema": "nocrud.results/v1",
  "runner": "go", "mode": "parallel", "jobs": 12,
  "wall_ms": 4021.5, "passed": true,
  "flows": [
    { "name": "actor", "kind": "crud", "ok": true, "ms": 1249.4,
      "summary": "C:✔ R:✔ U:✔ D:✔" }
  ]
}
```

Useful for a CI step that wants to key off a specific flow rather than parse
terminal output. `ok` reflects whether the flow *raised* — a CRUD flow reporting
`U:✘` did not raise, so it stays `ok: true` with the tick visible in `summary`.

**This file is local and is not anonymized.** Flow names are usually your
endpoint names, and endpoint names are usually your domain model. If you want to
share run data with the project, use the `nocrud-share-results` skill, which
builds a separate payload from a field allowlist and shows it to you first —
don't upload this file or a `perf/runs/*.ndjson`.

## Writing a flow

A flow is a function and an `init()` that registers it. Drop the file in
`flows/` and there is nothing else to wire up:

```go
package flows

import (
	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/crud"
)

func init() { nocrud.RegisterCRUD("actor", crudActorFlow) }

func crudActorFlow(c *nocrud.Ctx) (any, error) {
	api, err := c.Setup()
	if err != nil {
		return nil, err
	}
	return crud.Exec(c, api, "actor",
		crud.SimpleCreate("actor", "actors.json", 0, "id"),
		crud.UpdateDetails{Field: "first_name", NewValue: "Bill"},
	)
}
```

`c.Setup()` clears the database to a known state and hands back a logged-in
client. `crud.Exec` runs all four operations and reports them as one line.

The more valuable flows are the ones a CRUD check can't reach — a rule that only
exists in the interaction between two users:

```go
func init() {
	nocrud.Register(nocrud.Flow{
		Name: "pitch_lock_after_interactions",
		Kind: nocrud.Request,
		Doc:  "A pitch cannot be edited once someone has commented on it",
		Fn:   pitchLockAfterInteractionsFlow,
	})
}
```

Inside, `c.NewRandomUserClient()` gets you a second user with their own session,
and the expectation helpers say what should happen:

- `nocrud.Must(v, err)` — fail the flow here. For the setup steps, so the object
  graph a rule needs doesn't bury the rule under error handling.
- `nocrud.ExpectFail(c, "editing a locked pitch", fn)` — assert something is
  refused, without pinning the status.
- `nocrud.ExpectStatus(c, "another user editing it", 403, fn)` — assert *which*
  refusal. Prefer this when you know the right status; it catches a rule enforced
  in the wrong layer (a 500 where a 403 belongs, a 404 hiding a 403).

One caution worth internalising, because it is the easy way to write a flow that
passes for the wrong reason: `ExpectFail` accepts *any* failure. If you PUT a
partial body to assert a permission rule, the request fails validation for the
fields you left out and never reaches the rule — and the flow would pass against
a backend with no rule at all. PATCH the one field you mean to change.

## Adapting it to your own backend

Everything above `utils/provisioning` is plain HTTP and framework-agnostic.
Three things are not:

| Concern         | Where                              | What to change                                              |
| --------------- | ---------------------------------- | ----------------------------------------------------------- |
| Auth handshake  | `utils/apiclient/apiclient.go`     | `Login` — token vs session, header names, the login path     |
| DB provisioning | `utils/provisioning/provisioning.go` | Your migration tool instead of `manage.py migrate`         |
| Starting the app| `utils/provisioning/provisioning.go` | Your run command instead of `manage.py runserver`          |

`Login` is the one almost every project has to touch. That is the point of
shipping source rather than a library: open the file and make it do what your
backend expects.

Django is what's implemented today. Three provisioning strategies are built in,
selected with `NOCRUD_PROVISION`:

| Value                 | How the schema gets there    | Trade                                |
| --------------------- | ---------------------------- | ------------------------------------ |
| `migrate` *(default)* | `manage.py migrate`          | Slowest, but needs no setup          |
| `sql`                 | `psql -f schema.sql`         | Faster; keep `schema.sql` current    |
| `template`            | `createdb -T <template>`     | Fastest; build the template first    |

## Configuration

Everything resolves from the environment, so the same binary works from a shell,
from CI and from an editor without edits. Defaults are in `config/config.go`.

| Variable                | Default                        | What                              |
| ----------------------- | ------------------------------ | --------------------------------- |
| `NOCRUD_APP_DIR`        | `../example_app`               | Root of the app under test        |
| `NOCRUD_FIXTURES_PATH`  | `<app>/api/fixtures`           | Fixture json                      |
| `NOCRUD_RUNNER_DIR`     | discovered via `go.mod`        | Where `perf/` is written          |
| `NOCRUD_SETTINGS_PATH`  | `<app>/<app>/settings.py`      | For the DB match check            |
| `NOCRUD_PROVISION`      | `migrate`                      | `migrate` / `sql` / `template`    |
| `NOCRUD_TEMPLATE_DB`    | `template_db`                  | Template mode only                |
| `DB_NAME` … `DB_PORT`   | `postgres` / `localhost` / `5432` | Connection, as the app reads it |
| `APP_PORT`              | `8000`                         | Serial mode only                  |

A compiled binary can be invoked from anywhere, so unlike Python — which leans
on `__file__` — the runner directory has to be discovered by walking up to the
`go.mod`, or set with `NOCRUD_RUNNER_DIR`.

## Differences from the Python runner

Same behaviour, different mechanics. Worth knowing when reading the source:

- **Parallelism is goroutines, not processes.** Python uses a
  `multiprocessing.Pool`; here the runner is nearly free and the concurrency
  limit exists to protect the backend and database.
- **Per-flow state is passed, not global.** Python can keep the port, the output
  stream and the perf collector in `os.environ` and module globals because each
  flow is its own process. Goroutines share those, so they live on `Ctx` and
  `Env` instead, and `Env.Environ()` converts back into real environment
  variables for the subprocesses that still read them.
- **`Must` stands in for a bare exception.** A Python flow fails by letting an
  exception escape. `Must` panics with a `*FlowPanic`, which the runner unwraps
  so it reports as a plain failure — while an accidental panic still gets its
  stack trace.
- **Output is buffered in parallel mode.** Each flow's log is printed in one
  piece when it finishes, so concurrent flows don't interleave. Print through
  `c.Printf`, not `fmt.Println`, or it lands out of order on the terminal.

## Testing the runner itself

```bash
go test ./...
```

The suite covers the runner end to end against a fake DRF backend in
`internal/testutil` — registry, runner, context, API client, CRUD helpers, perf
collection and the summary. Only provisioning needs the real thing. The dbclient
tests use postgres when one is reachable and skip themselves when it isn't, so
`go test ./...` is green either way — CI provides one so they actually run.
