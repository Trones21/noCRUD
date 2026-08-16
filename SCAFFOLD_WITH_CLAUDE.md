# Scaffolding noCRUD with Claude Code

This repo ships a **Claude Code skill** that sets noCRUD up against your backend
for you — it discovers your REST endpoints and generates flows (both CRUD and
multi-user business-logic flows) so you don't have to write them by hand.

You do **not** need to have used a skill (or Claude Code) before. This page walks
through it from zero.

## What's a skill, in one paragraph?

A skill is a set of instructions that Claude Code loads automatically when your
request matches what the skill is for. You don't install or run anything — you
just ask in plain English, and if the request fits, Claude picks the skill up and
follows it. The skill for this repo lives in `.claude/skills/nocrud-scaffold/`,
so it travels with the repo: clone noCRUD, and Claude already knows how to wire
it into your project.

## One-time setup

1. **Install Claude Code** (Anthropic's CLI). See the docs at
   https://code.claude.com/docs — the `claude` command is what you'll run.
2. **Clone noCRUD next to the project you want to test**, as sibling folders:

   ```text
   my-workspace/
   ├── my-backend/      ← your existing app, in any language
   └── noCRUD/          ← this repo, cloned alongside it
   ```

3. **Open Claude Code from the folder that contains both** (`my-workspace/`
   above), so it can see your backend and the noCRUD skill at the same time:

   ```bash
   cd my-workspace
   claude
   ```

## Your backend's language doesn't matter

Worth saying up front, because it's the most common wrong assumption people
arrive with: **the language your backend is written in has nothing to do with
which runner you use.**

noCRUD is a series of API calls. That's the whole thing. It talks to your backend
exactly the way your frontend does — HTTP in, JSON out — so it neither knows nor
cares what's on the other end. A Go backend tested by the Python runner is a
completely normal setup. So is a Java backend tested by the Go runner.

There are two runners, `python/` and `go/`, and they are **complete and at
parity** — same CRUD checks, same multi-user business-logic flows, same isolated
database and app per flow, same parallel execution, same persisted timings. Their
timing files even share a schema, so a baseline captured by one can be compared
by the other.

So pick on the only basis that actually matters: **which language your team wants
to write and maintain flows in.** The flows are what you'll live with. The skill
will ask you which one you want.

### The one thing that is specific to your app

The runner has to be able to authenticate against *your* backend, and that part
nobody can guess. Login endpoint, session vs. bearer token vs. API key, which
headers you expect, cookie names, CSRF handling — all of it is yours.

That's what the API client is for, and adapting it is an expected step, not a
workaround:

| Runner   | File to adapt                                |
| -------- | -------------------------------------------- |
| Python   | `utils/api_client.py` — `APIClient.login`    |
| Go       | `utils/apiclient/apiclient.go` — `Client.Login` |

The skill does this for you as part of setup — it reads your auth middleware, or
asks you when it can't tell. But it's the piece most likely to need your eyes
afterward, so if a freshly scaffolded run fails, check there first. Both bundled
clients ship assuming Django session + CSRF, because that's what the example app
uses.

Beyond auth, two more pieces are specific to your *framework* rather than your
app — creating an isolated database per flow, and starting your app on a port.
Those are implemented for Django/DRF today; see "Good to know" below.

## Using it

Just ask. Any of these will trigger the skill:

- "Set up noCRUD for my backend."
- "Scaffold noCRUD flows for `my-backend`."
- "Generate API flows for this project with noCRUD."

Claude will then, roughly:

1. **Discover your backend** — find your routes and rules. If your app exposes an
   OpenAPI schema (e.g. `/api/schema/`) or a browsable API root, it uses that;
   otherwise it reads your source. It shows you an inventory first so you can
   catch anything it missed.
2. **Wire noCRUD in** — ask which runner you want, copy it into place, point its
   config at your app (`config.py` for Python, `config/config.go` or the
   `NOCRUD_*` env vars for Go), and adapt the API client to your login/auth.
3. **Generate flows** — CRUD flows per endpoint, plus multi-user business-logic
   flows that assert your real rules (e.g. "user B can't read user A's object
   until A grants access").
4. **Report coverage** — tell you which endpoints got flows and which didn't,
   and **offer** to run them. It won't run anything until you say so.

## Good to know

- **Django/DRF is the only fully-supported framework today** — meaning the
  *provisioning* pieces, not the flows. Spinning up an isolated database per
  flow and starting your app are the two things noCRUD can't do over HTTP, and
  they're implemented for Django. For other frameworks Claude still generates
  every flow and will write the adapter, but that adapter is new code and
  deserves a read. This is a framework question, not a language one — see above.
  Current status lives in the "Framework Adapter" table in
  `.claude/skills/nocrud-scaffold/SKILL.md`.
- **Both runners are complete.** Neither is a preview or a port-in-progress. Pick
  by team preference, not maturity.
- **Nothing runs automatically.** The skill generates and reports; running the
  flows is always your call.
- **Review before you run.** Treat the generated flows and the inventory as a
  first draft to check, not a black box — especially the business-logic flows,
  which encode your rules.

## If you'd rather do it by hand

The skill is a convenience layer over the manual path. Everything it does you can
still do yourself — see [`USAGE.md`](./USAGE.md) and
[`examples/example-runners/Readme.md`](./examples/example-runners/Readme.md).
