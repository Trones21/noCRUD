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
   ├── my-backend/      ← your existing app (Django/DRF for now)
   └── noCRUD/          ← this repo, cloned alongside it
   ```

3. **Open Claude Code from the folder that contains both** (`my-workspace/`
   above), so it can see your backend and the noCRUD skill at the same time:

   ```bash
   cd my-workspace
   claude
   ```

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
2. **Wire noCRUD in** — copy the runner into place, set `config.py`, and adapt
   the `APIClient` to your login/auth.
3. **Generate flows** — CRUD flows per endpoint, plus multi-user business-logic
   flows that assert your real rules (e.g. "user B can't read user A's object
   until A grants access").
4. **Report coverage** — tell you which endpoints got flows and which didn't,
   and **offer** to run them. It won't run anything until you say so.

## Good to know

- **Django/DRF is the only fully-supported framework today.** noCRUD is just
  making HTTP calls, so the flows themselves are framework-agnostic — but three
  pieces are framework-specific (spinning up an isolated DB per flow, starting
  your app, and the auth handshake). Those are implemented for Django; for other
  frameworks Claude can still generate flows but will flag what needs writing.
  The current status lives in the "Framework Adapter" table in
  `.claude/skills/nocrud-scaffold/SKILL.md`.
- **Nothing runs automatically.** The skill generates and reports; running the
  flows is always your call.
- **Review before you run.** Treat the generated flows and the inventory as a
  first draft to check, not a black box — especially the business-logic flows,
  which encode your rules.

## If you'd rather do it by hand

The skill is a convenience layer over the manual path. Everything it does you can
still do yourself — see [`USAGE.md`](./USAGE.md) and
[`example-runner-files/Readme.md`](./example-runner-files/Readme.md).
