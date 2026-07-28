## 🔧 noCRUD — A CLI Tool for Simulating Real-World API Workflows

`noCRUD` (short for “Not Only CRUD”) is a test runner + backend flow simulator designed to:

- ✅ Run standard **CRUD checks** on API endpoints
- 🚦 Simulate **full workflows**
  - And this is multi user!! So you can create multiple API Clients with different user creds, and then do something like:
    - POST as user A ->
    - GET as user B (expect fail) ->
    - PUT as user A (update perms so other users can read) ->
    - GET as user C (Expect success)
- 🔍 Verify **business logic and rule enforcement**, including invalid flows and expected failures
- 📦 Reuse `create()` flows to **seed the database** with dependency-aware objects
- 🧪 Reduce repetitive UI testing by interacting with the backend directly

---

### 🤖 Set it up with Claude Code

This repo ships a Claude Code skill that discovers your backend's endpoints and
generates flows for you — no manual templating. Clone noCRUD alongside your
project and just ask ("set up noCRUD for my backend"). See
[`SCAFFOLD_WITH_CLAUDE.md`](./SCAFFOLD_WITH_CLAUDE.md) for a from-zero walkthrough.

---

### 💡 Two Modes of Use

1. **CRUD Mode**

   - Run structured Create → Read → Update → Delete tests per endpoint
   - Output color-coded summaries and logs
   - Parallelized by default

2. **Request Flow Mode**
   - Simulate more complex workflows involving multiple endpoints
   - Assert business rules, expected failures, approvals, etc.
   - Great for reproducing bugs or automating common flows

---

### 🔁 Reusable Object Builders

Need to create a `Trip` but it depends on a `Driver`, `Passenger`, and `Vehicle`?

Rather than loading raw fixtures, `noCRUD` lets you:

- Reuse object builders that encapsulate dependency logic
- Avoid hardcoding IDs or managing inconsistent test data
- Seed a DB programmatically with fully valid objects (using the actual API! (but also a backdoor DBClient if you wish to))

---

### 🧪 Example Use Cases

- Validate all CRUD endpoints after a schema change
- Simulate a full multi-user workflow.
- Seed a dev or staging DB with real object graphs
- Debug a failing frontend flow by replicating it in CLI
- Confirm that rule violations (e.g., missing dependencies) fail as expected

---

### 🚀 Philosophy

> This isn’t just a test runner.  
> It’s a developer’s toolbox for interacting with your backend in the same way your **users** and **frontend** do — just without the mouse.

### Architecture of any Test Runner

I've added this section so that `noCRUD.py` will make sense to people who havent written test runners before

1. Get the set of tests to be run
2. Run tests and collect results
3. Print results

Steps 2 & 3 are not completely mutually exclusive... I like to print a few specific things while running the test and then just print a summary at the end. It's also completely up to you to decide what to bubble up to the results.

## Status

See the readme in the individual implementation folders (python, go) for implementation specific status.
