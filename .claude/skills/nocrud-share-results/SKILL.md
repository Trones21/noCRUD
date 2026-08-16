---
name: nocrud-share-results
description: >
  Build an anonymized summary of a noCRUD run and help the user submit it as a
  public run report, so the project can see what people point noCRUD at and at
  what scale. Use when the user wants to "share my noCRUD results", "submit a
  run report", "send my run stats", or asks how to contribute run data.
---

# Sharing a noCRUD run

The project wants to know what noCRUD gets pointed at in the wild — which
runner, what kind of backend, how many flows, how long it took. None of that
requires knowing anything about the user's application, and this skill exists to
keep it that way.

## The rule that matters

**Build the payload from the allowlist below. Never from a denylist.**

This is not a style preference. A denylist ("strip flow names, strip URLs") is
wrong the moment a future schema version adds a field nobody thought to strip,
and the failure mode is publishing a stranger's internal model names to a public
issue tracker. Start from nothing, add only what is listed, and if you find
yourself wanting to include something not on the list, ask the user instead of
deciding.

**The user must see the exact payload and agree before anything is submitted.**
No auto-submitting, no "I'll open the issue for you" without showing the JSON
first. This is a public, permanent, outward-facing action.

## Why not just upload results.json

`results.json` (what `--json` writes) is a **local** artifact and is not safe to
share as-is. It contains:

- `flows[].name` — usually endpoint names, which usually *are* the domain model
  (`claim`, `patient`, `invoice`, `prescription`)
- `flows[].summary` and `flows[].error` — free text that can quote field names,
  validation messages, and URLs
- `runner_sha` — noCRUD's commit, harmless, but people confuse it with theirs

The same applies, more so, to `perf/runs/*.ndjson`: every record carries `flow`
and `endpoint`. Never suggest attaching either file.

## Phase 1 — Get a run

If there's no `results.json`, ask the user to produce one. Nothing is collected
from scratch; this only ever summarizes a run that already happened.

```bash
go run ./cmd/nocrud -coll --json results.json      # Go
python noCRUD.py -coll --json results.json         # Python
```

## Phase 2 — Build the payload

Read `results.json` and construct **exactly** this shape. Every field is either
copied from the results file, computed from it, or asked of the user.

```json
{
  "schema": "nocrud.run-report/v1",
  "runner": "go",
  "runner_sha": "139c0b8",
  "mode": "parallel",
  "jobs": 8,
  "wall_ms": 4021,
  "flows": { "total": 9, "passed": 9, "failed": 0, "crud": 7, "request": 2 },
  "flow_ms": { "min": 1249, "median": 1395, "max": 2170 },
  "backend": { "language": "go", "framework": "chi", "database": "postgres" },
  "env": { "os": "linux", "arch": "amd64", "cpus": 12 }
}
```

| Field | Source | Notes |
| --- | --- | --- |
| `runner` | results `runner` | `go` or `python` |
| `runner_sha` | results `runner_sha` | noCRUD's commit. Include only if the user confirms the runner is unmodified upstream; drop it otherwise, since a sha for a fork says nothing |
| `mode`, `jobs`, `wall_ms` | results | as-is |
| `flows.*` | **counts computed** from `flows[]` | counts only — never the array |
| `flow_ms.*` | computed from `flows[].ms` | min / median / max. Not per-flow, which would let someone line timings up against a guessed model |
| `backend.*` | **ask the user** | free text they choose. Not derivable from results.json, and not worth guessing |
| `env.*` | ask, or read from the machine | os / arch / cpu count |

Anything not in that table does not go in. In particular: no `flows[]` array, no
names, no summaries, no error strings, no paths, no hostnames, no ports, no
database names, no git remote, no username.

**On failures:** report `flows.failed` as a number. Do not include the error
text. If the user *wants* to report a failure, that's a bug report — a different,
better-suited thing — and you should offer to help write one instead.

## Phase 3 — Show it, then submit

1. Print the JSON in full and ask the user to read it. Say plainly that it will
   be public and permanent.
2. Only after they agree, help them open the issue. Either:
   - Print the JSON and the URL
     `https://github.com/Trones21/noCRUD/issues/new?template=run-report.md&labels=run-report`
     and let them paste it, **or**
   - If `gh` is installed and authenticated, offer to run
     `gh issue create --repo Trones21/noCRUD --label run-report --title "[run] ..." --body-file <file>`
     — offer, and let them confirm the body first.
3. If they decline at any point, stop. Don't ask twice. A declined share is a
   perfectly good outcome and the payload is theirs to keep.

Title format: `[run] <runner> / <framework> / <n> flows` — e.g.
`[run] go / chi / 9 flows`.

## If the user is on a fork or a private backend they can't discuss

That's fine and common. `backend.*` is optional — a report with `runner`,
counts, and timings is still useful. Say so rather than pressing for detail.

## Notes

- This is opt-in and always will be. Neither runner phones home, has telemetry,
  or writes anything outside the directory it was told to. Nothing here changes
  that — a person runs this skill and clicks submit, or nothing happens.
- If the user asks you to automate submission on a schedule or in CI, say no and
  explain: the review step is the entire privacy control, and a cron job cannot
  read a payload and decide it's fine to publish.
