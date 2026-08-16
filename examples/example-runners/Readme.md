# example-runners/

Worked example flows, one set per implementation. They run against
[`../example_app`](../example_app).

| Directory                        | For                | Start here                                        |
| -------------------------------- | ------------------ | ------------------------------------------------- |
| [`python-impl/`](./python-impl/) | the `python/` runner | [`python-impl/Readme.md`](./python-impl/Readme.md) |
| [`go-impl/`](./go-impl/)         | the `go/` runner     | [`go-impl/Readme.md`](./go-impl/Readme.md)         |

Both sets cover the same ground against the same backend — CRUD flows over every
endpoint, a dependency chain built without hardcoded ids, a multi-user rule, and
a seeding flow — so they're also the most direct way to compare how the two
runners read.

## Why these files don't resolve in place

Neither set is inside its runner, and that's deliberate: `flows/` ships empty in
both runners because flows are *yours*. The examples only become real code once
copied in, which is what CI does before compiling them — and that copy step is
what keeps them from rotting against the runner.

The consequence is that your editor will complain about them where they sit. The
two runners get there differently:

- **Python** — the imports are *intentionally* broken, and always were. They
  resolve once the files are in `python/flows/`.
- **Go** — the imports are already correct. The files just aren't part of the Go
  module (`go.mod` lives at `go/`, and this directory is outside it), so gopls
  has nothing to resolve `github.com/Trones21/noCRUD/go/nocrud` against until you
  copy them into `go/flows/`.

Neither is a bug, and neither means the example is stale — CI compiles both sets
on every PR.

## Before you invest time here

See [`../Readme.md`](../Readme.md): if your goal is getting noCRUD running on
your own backend, the `nocrud-scaffold` skill does that directly and you can skip
this directory. These examples are reference and demo, not the onboarding path
they used to be.
