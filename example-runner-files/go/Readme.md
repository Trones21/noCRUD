# Example flows — Go runner

Flows that run against the bundled `example_app`. Unlike the Python examples in
the folder above, these are **not** intentionally broken: the import paths are
already correct, they just aren't part of the module until you copy them in.

All commands assume your `pwd` is the noCRUD project root.

```bash
cp ./example-runner-files/go/flows/*.go ./go/flows/
cd go
go run ./cmd/nocrud -l      # confirm they registered
go run ./cmd/nocrud -crud   # run them
```

## What's in here

| File                               | Kind    | Shows                                                    |
| ---------------------------------- | ------- | -------------------------------------------------------- |
| `actor.go`                         | CRUD    | The whole flow for a standalone object — six lines        |
| `universe.go`                      | CRUD    | A hand-written create function, reusable by other flows   |
| `production.go`                    | CRUD    | One dependency: universe → production                     |
| `character.go`                     | CRUD    | A many-to-many field                                      |
| `pitch.go`                         | CRUD    | A four-deep chain, no hardcoded ids                       |
| `vote.go`                          | CRUD    | Five deep, and a numeric update value                     |
| `user.go`                          | CRUD    | Working around a unique constraint the fixtures hold      |
| `pitch_lock_after_interactions.go` | Request | Two users, and a rule that must refuse the second edit    |
| `seed_pitches.go`                  | Request | The same builders used to fill a database instead         |

Read them in that order — each one adds a single idea to the one before.

## Setup

The runner needs to know where the app is and how to reach postgres. For the
bundled example app the defaults are already right; you only need the database
variables:

```bash
source ./example_app/backend_env.sh
```

Parallel mode (the default) provisions an app and a database per flow, so
there's nothing to start. For serial mode, start `example_app` yourself first —
see [`go/Readme.md`](../../go/Readme.md).
