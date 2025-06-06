# noCRUD Initial Video Plan

Post on Youtube and then cross post to other platforms.

**Record each video in one go, no practice runs. Just keep recording. Resist the urge to stop, delete, and redo... it should really just be a process in the background that I forget is even there until I am done showing/exploring/writing/running code. **

Editing and voiceover will come later. Can always pause (add frames) and of course chop up/speed up if the narration doesn't align timewise. (Can also cut out errors I dont want to show). The point is to get lots of footage. **_The storyboard is actually DOING IT, plan for linear_**, honestly I think just running through each of these will be less effort than trying to stitch together existing clips for longform. I will invariably end up with clips I can stitch together for shortform promotional content.

Videos can be refined later. Let's just focus on getting the content out there. These are mostly tutorial/explainer.

Later "experimental" videos will follow a similar pattern in that I will keep recording for hours on end while i conduct the experiments (not worried about getting it perfect, no stopping, titling, categorizing... the chop up should be done afterwards, when I am storyboarding)

## Series: The Fundamentals

### ✅ Architectural Overview of noCRUD

- **Goal:** Provide a high-level picture of how all pieces interact
- **Key Points:**

  - Role of the runner, test flows, provision logic, and DB orchestration
  - How threads and DBs map per flow in parallel mode
  - Entry points and responsibility boundaries

- **Visuals:** Architecture diagram, example timeline of a test run

Link: `<To be added when Complete>`

### ✅ How Flows Work (The Engine of noCRUD)

- **Goal:** Explain what a flow is, how it's executed, and why it’s central to noCRUD
- **Key Points:**

  - What a flow is: a function triggered via API
  - Auto-generated views come from typed dict return
  - CRUD flows vs. business logic flows (request flows)
  - Flow discovery: manual vs. auto-registration
  - Template generator for starting new flows

- **Visuals:** flow code, terminal run, output UI, API call
- **Mentioned In:** nearly every other video

Link: `<To be added when Complete>`

### ✅ The Provisioning Method

- **Goal:** Explain the role of the `provision()` method and how it governs environment setup
- **Key Points:**

  - Port selection, DB creation, schema prep (migrations/export/import/template)
  - App startup and readiness checks
  - Hook for full customization of test environment per flow

- **Mentioned In:** Optimization and Customization videos

Link: `<To be added when Complete>`

### Setting up the Example App

    1. Start from the beginning open browser, go to repo, then readme
    2. open terminal, clone, open vs code in directory
    3. Docs on one side of the screen, vs code on the other.
    4. open top level dir for a second to explain what they are, then follow along with the docs

Link: `<To be added when Complete>`

---

## Series: ⚡ Provisioning Optimization Story

> "Run it lean. Run it fast. Run it right."

_This series will end up more as edutainment. While this does go deep into provisioning, I have purposely added a separate video on provisioning in the fundamentals series, b/c this optimization series is really more of a story._

### 1. DB Provisioning via Migrations

- Traditional method using Django-style migrations

Link: `<To be added when Complete>`

### 2. via Schema Export / Import

- Use `pg_dump -s` and `psql` for quick schema setup

Link: `<To be added when Complete>`

### 3. via DB Copy with `createdb --template`

- Fastest way to duplicate DB

Link: `<To be added when Complete>`

### 4. Optimizing DB Match Check

- The DB Match validator (e.g., avoid slow fallback)

Link: `<To be added when Complete>`

---

## Series: 🧩 Customization (Making It Yours)

> "Adapt noCRUD to fit your workflow."

### Moving `createdb` Into Single-Threaded Startup

- Embed template copy logic directly into your app

Link: `<To be added when Complete>`

### Keeping the DB After Flow Success

- Defaults to dropping on success; show how to persist

Link: `<To be added when Complete>`

---

## Series: Architectural components in more depth Videos

### Parallel vs Serial Mode

- Parallel: no infra needed, fully automatic
- Serial: user is responsible for spinning up app + DB

Link: `<To be added when Complete>`

### Manual vs Auto Flow Registration

- How noCRUD discovers flows
- Use cases for both

Link: `<To be added when Complete>`

### CRUD vs Request Flows

- When to return typed dict vs plain dict
- Impacts UI generation

Link: `<To be added when Complete>`

---

## Series: 🐛 Debugging & Behavior Explanations

### DB Mismatch Error

- Common scenario when DB name diverges from config

Link: `<To be added when Complete>`

### Slow Fallback (Can't Load `settings.py`)

- Triggers manage.py shell workaround (adds \~700ms per flow)

Link: `<To be added when Complete>`

### Flow Fail = DB Stays / Flow Succeeds = DB Dropped

- Default lifecycle explanation + how to override

Link: `<To be added when Complete>`

### How I Avoid Port Collision

_Possibly end up as non-linear, more edutainment_

- An overview of the function and why it works
- Could dive as deep as we want

Link: `<To be added when Complete>`

---

## Series: Building Flows

### Build a CRUD flow from scratch

- Start with the standard "flows are the building blocks... each is a series of API calls... for an in depth guide to flows check out `this fundamental video`".
- In this video I'll be using parallel mode, which provisions the app and db for me, for an overview of parallel v. serial mode, check out `this video`.
- Build the flow

  - Ensure i show how this works with fixtures:
    - For login via the setup method (common.py)
      - The importance of unhashed pass
    - and for the crud entity itself

- Dont forget to register the flow (show -l to verify)
- Run this specific flow with -f `<flow name>`
- Run crud flows (-crud)
- You can also reuse flows (Possibly show if this isnt too long, otherwise reference the other video)
- Mention the template generator at the end

Link: `<To be added when Complete>`

### Using The crud flow template generator

Link: `<To be added when Complete>`

### Flow Reuseability (Importing into another flow)

- I think I modified the collector so that auto generated flows are simple to import, but should double check to see if there is a difference vs. manually registered flows and be sure to show both

Link: `<To be added when Complete>`
