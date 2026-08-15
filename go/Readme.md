# noCRUD — Go implementation

A complete Go runner: CRUD checks, multi-user request flows, per-flow isolated
app + database provisioning, parallel execution, and persisted request timings
with regression comparison.

**Status: at parity with the Python runner.** Everything `python/` does, this
does. The differences that remain are the ones Go forces (see
[Differences from the Python runner](#differences-from-the-python-runner)); none
of them lose a feature.

```bash
cd go
go run ./cmd/nocrud -l          # what's registered
go run ./cmd/nocrud -crud       # run every CRUD flow, in parallel
go run ./cmd/nocrud -f actor -s # one flow, against an app you started
go test ./...                   # the runner's own tests
```

Requires Go 1.24+ and a reachable postgres. The only dependency is
[pgx](https://github.com/jackc/pgx) (what `psycopg` is to the Python runner).

---

## Layout

Same shape as `python/`, so the two can be read side by side.

| Go                             | Python                       | What it is                                       |
| ------------------------------ | ---------------------------- | ------------------------------------------------ |
| `cmd/nocrud/`                  | `noCRUD.py`                  | Entry point: flags, flow selection, summary       |
| `cmd/perfreport/`              | `perf_report.py`             | Timing comparison and baselines                   |
| `cmd/createcrudflow/`          | `create_crud_flow.py`        | Generates a CRUD flow file                        |
| `cmd/modelcoverage/`           | `model_coverage_check.py`    | Which models have no flow                         |
| `cmd/initdb/`                  | `init_db.py`                 | Create a database named by `DB_NAME`              |
| `config/`                      | `config.py`                  | Where your app and fixtures live                  |
| `nocrud/`                      | —                            | `Ctx`, `Env`, flow registry, `Must`, assertions   |
| `runners/`                     | `runners/`                   | Serial and parallel execution                     |
| `utils/apiclient/`             | `utils/api_client.py`        | One logged-in user's HTTP session                 |
| `utils/crud/`                  | `utils/crud.py`              | The create/read/update/delete check               |
| `utils/dbclient/`              | `utils/db_client.py`         | Database back door                                |
| `utils/fixtures/`              | `utils/fixtures.py`          | Reading the app's test data                       |
| `utils/perf/`                  | `utils/perf.py`              | Persisted request timings                         |
| `utils/printing/`              | `utils/printing.py`          | Output formatting                                 |
| `utils/provisioning/`          | `utils/provisioning.py`      | Per-flow app + database                           |
| `utils/jsonx/`                 | — (Python has dicts)         | Ergonomic access to decoded JSON                  |
| `flows/`                       | `flows/`                     | **Your flows.** Ships empty on purpose.           |

---

## Writing a flow

A flow is a function that gets a `*nocrud.Ctx` and returns whatever belongs in
the summary. It registers itself from `init()`, so adding a file to the `flows`
package is the whole of the wiring — the Go equivalent of the Python runner's
folder collector.

A CRUD flow for an object with no dependencies:

```go
package flows

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

`c.Setup()` resets the data to the minimal fixtures and returns a client logged
in as the first fixture user — the counterpart of `utils/common.py`'s `setup()`.

### Objects that need other objects first

Creating the object is the part that varies, so it's a function
(`crud.CreateFunc`). A create function that builds a dependency chain is
reusable: other flows call it instead of inventing their own graph, and nothing
hardcodes an id.

```go
func createProduction(c *nocrud.Ctx, api *apiclient.Client) (jsonx.Value, error) {
	universeID, err := createUniverse(c, api) // → the flow one level down
	if err != nil {
		return universeID, err
	}

	obj, err := fixtures.GetByIndex("productions.json", 0)
	if err != nil {
		return jsonx.Invalid(err), err
	}
	obj["universe"] = universeID.Raw()

	res, err := api.CreateObject("production", obj)
	if err != nil {
		return res, err
	}
	return crud.ID(res, "production", "id")
}
```

The same functions seed a database — see `seed_pitches.go` in the examples.

### Multi-user flows

Each `*apiclient.Client` is one user with its own cookie jar, so a multi-user
flow is just several of them:

```go
author, _ := c.Setup()                  // fixture user
critic, _ := c.NewRandomUserClient()    // a brand new registered user

pitch := nocrud.Must(author.CreateObject("pitch", body))
nocrud.Must(critic.CreateObject("pitch_comment", comment))

// Half of what a flow proves is that the backend says no at the right moment.
err := nocrud.ExpectStatus(c, "editing a locked pitch", 403, func() error {
	_, err := author.UpdateObjectByID("pitch", pitch.Get("id").ID(), edit)
	return err
})
```

`nocrud.ExpectFail` is the looser form, when any failure will do.

### Errors: return them, or `Must` them

A Python flow lets exceptions fly and the runner turns them into a traceback and
a red mark. `(value, error)` on every step is the Go way, but it doubles the
length of a flow that has nothing useful to do with the error.

`nocrud.Must` gives you the Python ergonomics without giving up Go's: it panics
with the error, the runner recovers it, and the flow reads as one line per step.
Handle errors normally where the flow actually branches on them.

```go
pitch := nocrud.Must(author.CreateObject("pitch", body))
```

Either way the flow is reported as failed, with a stack trace — and an accidental
panic (nil map, index out of range) is caught too, so one bad flow never takes
the run down with it.

---

## Running

```bash
go run ./cmd/nocrud -crud                 # every CRUD flow
go run ./cmd/nocrud -req                  # every request flow
go run ./cmd/nocrud -coll                 # everything registered
go run ./cmd/nocrud -f actor -f universe  # named flows (or -f actor,universe)
go run ./cmd/nocrud -l                    # list without running
```

| Flag                       | Meaning                                                        |
| -------------------------- | -------------------------------------------------------------- |
| `-s`, `--serial`           | Run serially against an app you started yourself                |
| `--perf`                   | Persist timings and compare them against the baseline           |
| `-j`, `--jobs`             | Max flows at once in parallel mode (default `GOMAXPROCS`)       |
| `--threshold`, `--metric`  | Passed to the end-of-run perf comparison                        |

The runner exits non-zero if any flow failed, so it gates CI directly.

### Parallel (default)

Each flow gets its own database and its own backend on its own port. You don't
start anything yourself.

```text
             ┌───────────────────┐     ┌──────────────────┐
        -->  │ App :1 Port 8001  │ --> │ DB: noCRUD_p8001 │
       /     ├───────────────────┤     ├──────────────────┤
Runner --->  │ App :2 Port 8002  │ --> │ DB: noCRUD_p8002 │
       \     ├───────────────────┤     ├──────────────────┤
        -->  │ App :3 Port 8003  │ --> │ DB: noCRUD_p8003 │
             └───────────────────┘     └──────────────────┘
```

Each flow's output is buffered and printed in one piece, so the logs don't
interleave. The backend's own log lines go straight to the terminal as they
happen, prefixed with the port.

A flow that fails keeps its database, so there's something left to inspect.
Clean them up with `python/drop_dbs_by_pattern.sh`.

Which provisioning method runs is set by `NOCRUD_PROVISION`:

| Value               | How the schema gets in       | Setup needed                       |
| ------------------- | ---------------------------- | ---------------------------------- |
| `migrate` (default) | `manage.py migrate`          | none                               |
| `sql`               | `psql -f schema.sql`         | keep `schema.sql` current          |
| `template`          | `createdb -T <template>`     | build the template before the run  |

If your backend isn't Django, `ProvisionEnvForFlow` in
`utils/provisioning/provisioning.go` is the only thing to replace — everything
above it is framework-agnostic HTTP.

#### Ports

A flow's port is **reserved and held** (`ReserveOpenPort`) until the instant the
app binds it, not merely looked up and let go. Asking the OS for a free port and
releasing it leaves a window in which the port belongs to nobody — and in the
migrate path that window is several seconds wide, since the database is created
and migrated before the app ever starts.

At handover the runner also checks nothing beat it to the port
(`AssertPortNotTaken`), which it can only know *before* the app is spawned:
afterwards, a process that already owns the port answers the readiness probe
just as convincingly as your own app would. Without that check a collision
doesn't fail — it runs the flow against another app and another database and
usually reports a pass.

See `python/docs/COMMON_ISSUES.md` for what the resulting errors mean.

### Serial

Start the app yourself, then:

```bash
source ../example_app/backend_env.sh
go run ./cmd/nocrud -crud --serial
```

Serial mode resets the database the app is already using, and prints in real
time.

---

## Configuration

`config/` resolves everything, and each value has an environment override:

| Variable                | Default                                | What it is                    |
| ----------------------- | -------------------------------------- | ----------------------------- |
| `NOCRUD_RUNNER_DIR`     | nearest directory with a `go.mod`       | where `perf/` is written      |
| `NOCRUD_APP_DIR`        | `<runner>/../example_app`               | the app under test            |
| `NOCRUD_FIXTURES_PATH`  | `<app>/api/fixtures`                    | the fixture json              |
| `NOCRUD_SETTINGS_PATH`  | `<app>/<app>/settings.py`               | for the DB match check        |
| `NOCRUD_PROVISION`      | `migrate`                               | provisioning method           |

The database connection comes from the same variables the app uses: `DB_NAME`,
`DB_USER`, `DB_PASS`, `DB_HOST`, `DB_PORT`.

---

## Timings and regressions

Identical to the Python feature, and **interchangeable with it**: same NDJSON
schema, same run layout, same operation names. A baseline captured by either
runner can be compared by the other, including with `python/perf_report.py`.

```bash
go run ./cmd/nocrud -crud --perf        # collect, and print the diff
go run ./cmd/perfreport --metric p95    # gate on the tail instead of the mean
go run ./cmd/perfreport --set-baseline  # promote this run
```

See [`python/docs/PERF.md`](../python/docs/PERF.md) — all of it applies here.

---

## Differences from the Python runner

Same features throughout. These are the places Go forced a different shape, and
what each one buys.

**Flows take a `*nocrud.Ctx`.** The Python runner isolates flows with
processes, so a flow can read `os.environ["APP_PORT"]` and print to stdout and
still be talking about its own backend. Go runs flows as goroutines in one
process, where those globals are shared — so everything per-flow (which port to
call, which database to reset, where output goes, where timings are collected)
travels in a `Ctx`. That is what makes `-j` possible at all, and it means
provisioning never mutates the process environment.

**Registration is always explicit.** Go has no runtime import, so there is no
folder collector. An `init()` in the flow's own file does the same job with the
same amount of typing — and a duplicate flow name panics at startup rather than
silently shadowing.

**Flags come before positional arguments** in `createcrudflow`, and `-f` is
repeated or comma-separated rather than variadic. Go's `flag` package stops
parsing at the first non-flag argument.

**`-coll`/`--collected` runs everything registered.** With no collector there is
no manual/collected split to preserve; `-crud` and `-req` still select by kind.

**Failures come back as errors, not exit calls.** Where the Python
`verifyAllTablesCleared` calls `sys.exit(1)`, the Go one returns an error, so
the failure is attributed to the flow that caused it while the others carry on.

**A flow that returns nothing prints `ok`** rather than `None`.

**`jsonx.Value` wraps decoded JSON.** `res["results"][0]["id"]` is one
expression in Python and a pile of type assertions in Go;
`res.Get("results.0.id").ID()` is the one-liner back. Lookups never panic.

---

## Tests

```bash
go test ./...          # unit + end-to-end, no backend needed
go test -race ./...
```

The suite runs the whole stack — registry, runner, context, API client, CRUD
helpers, perf collection, summary — against a fake DRF backend
(`internal/testutil`) that enforces CSRF and session auth the way the real one
does. The database tests skip themselves when there's no postgres to talk to.

---

## Examples

Ready-made flows for the bundled `example_app` are in
[`example-runner-files/go/flows/`](../example-runner-files/go/flows/). Copy them
in and run:

```bash
cp ../example-runner-files/go/flows/*.go ./flows/
go run ./cmd/nocrud -l
```
