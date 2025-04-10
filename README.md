Note: Code coming soon, I'm just working on the parallelization, and some examples

## 🔧 noCRUD — A CLI Tool for Simulating Real-World API Workflows

`noCRUD` (short for “Not Only CRUD”) is a test runner + backend flow simulator designed to:

- ✅ Run standard **CRUD checks** on API endpoints
- 🚦 Simulate **full user workflows**, such as login, approvals, and nested object creation
- 🔍 Verify **business logic and rule enforcement**, including invalid flows and expected failures
- 📦 Reuse `create()` flows to **seed the database** with dependency-aware objects
- 🧪 Reduce repetitive UI testing by interacting with the backend directly

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

Need to create a `TestPlan` but it depends on a `Program`, `Phase`, and `Discipline`?

Rather than loading raw fixtures, `noCRUD` lets you:
- Reuse object builders that encapsulate dependency logic
- Avoid hardcoding IDs or managing inconsistent test data
- Seed a DB programmatically with fully valid objects

---

### 🧪 Example Use Cases

- Validate all CRUD endpoints after a schema change
- Simulate a full mission approval workflow
- Seed a dev or staging DB with real object graphs
- Debug a failing frontend flow by replicating it in CLI
- Confirm that rule violations (e.g., missing dependencies) fail as expected

---

### 🚀 Philosophy

> This isn’t just a test runner.  
> It’s a developer’s toolbox for interacting with your backend in the same way your **users** and **frontend** do — just without the mouse.
