# examples/

A backend to test against, and worked flows for both runners.

```
examples/
  example_app/          A real Django/DRF backend. noCRUD dogfoods itself on it in CI.
  example-runners/
    python-impl/        Example flows for the Python runner
    go-impl/            Example flows for the Go runner
```

## Read this before you spend time here

**This directory is largely an artifact of how noCRUD worked before the skill
existed.** The original path into the tool was: read the example app, read the
example flows, copy the pieces you understood into your own runner, adapt them
by hand. That's a real amount of work, and these examples existed to make it
survivable.

That is no longer the main path. `.claude/skills/nocrud-scaffold/` does that work
directly against *your* backend — it reads your routes and rules, picks the
runner, writes the flows, and wires up config and auth. You do not need to
understand the example app to get noCRUD running on your project.

So treat this directory as:

- **Reference**, when you want to see what a good flow looks like before writing
  your own — especially the multi-user business-logic ones, which are the part
  worth copying the *shape* of.
- **A demo**, when you want to watch the thing actually run before pointing it at
  something you care about.
- **CI's backend**, which is the job it does that nothing else can: the workflows
  run both runners against `example_app` on every PR, which is what keeps the
  examples compiling and the runners honest.

It is *not* the recommended way to onboard. If you're here to get noCRUD working
on your own project, use the skill instead.

## If you do want to run the examples

**Work outside the repo.** Trying the examples means copying flow files into a
runner's `flows/` directory, and possibly editing config — none of which is
change you want tangled up with the repo you may later want to send a PR from.
Give yourself a scratch directory somewhere else entirely:

```bash
mkdir -p ~/nocrud-trial && cd ~/nocrud-trial
cp -r /path/to/noCRUD/go .                       # or python
cp -r /path/to/noCRUD/examples/example_app .
cp /path/to/noCRUD/examples/example-runners/go-impl/flows/*.go ./go/flows/
```

Point the runner at the app and run it:

```bash
cd go
export NOCRUD_APP_DIR=~/nocrud-trial/example_app
export DB_USER=postgres DB_PASS=postgres DB_HOST=localhost DB_PORT=5432
export DJANGO_KEY=dev-insecure-key
go run ./cmd/nocrud -l
go run ./cmd/nocrud -coll
```

`NOCRUD_APP_DIR` is what makes this work from anywhere — both runners default to
the in-repo `examples/example_app`, and the env var overrides that. The Python
runner uses `APP_DIR` in `config.py` for the same purpose.

Two things you'll hit either way:

- **The example app needs Python 3.11.** `example_app/requirements.txt` is pinned
  to a set verified there, and CI uses it. On 3.12+ you'll need
  `psycopg2-binary>=2.9.10` and `setuptools<81` (drf_yasg imports
  `pkg_resources`, which newer setuptools dropped).
- **Postgres must be reachable as a superuser**, because parallel mode creates
  and drops a database per flow.

If you'd rather not work outside the repo, the copy targets
(`go/flows/*.go`, `python/flows/auto_registered/`) are gitignored, so a trial run
in place won't dirty the tree. Working outside is still cleaner.
