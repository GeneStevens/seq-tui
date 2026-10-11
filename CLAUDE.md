# CLAUDE.md — Seq TUI Execution Contract

This file defines how Claude Code (the Builder) must operate when working in the `seq-tui` repository.

This repository is a **thin client**.

---

# 1. Core Identity

You are working on the **tui-client project**.

This repository:
- Renders backend state
- Displays gameplay information
- Sends explicit player intents

It is NOT authoritative.

The backend owns all gameplay truth.   

---

# 2. Absolute Invariants

## 2.1 Thin Client Doctrine

The TUI must:

- Render backend state
- Display event streams
- Show derived presentation only

The TUI must NOT:

- Simulate gameplay
- Predict outcomes
- Infer rules
- Maintain authoritative state

---

## 2.2 No Gameplay Logic

You must NOT implement:

- combat logic
- XP calculation
- loot rules
- threat logic
- encounter rules

If it belongs in gameplay → it belongs in backend

---

## 2.3 Backend Truth Only

All displayed values must come from:

- API responses
- Encounter summaries
- Gameplay status endpoints

Do not compute or infer missing data.

---

# 3. Execution Rules

## 3.1 Scope Discipline

You may only touch:

- `seq-tui`

You must NOT touch:

- `seq-backend`
- `seq-notes`

---

## 3.2 Rendering Responsibility

The TUI is responsible for:

- layout
- formatting
- readability
- emphasis of events

It is NOT responsible for:

- correctness of gameplay logic
- resolving contradictions
- “fixing” backend behavior

---

## 3.3 Event Stream Discipline

- Event stream must reflect backend timeline
- No deduplication that changes meaning
- No synthetic events
- No reordering

---

## 3.4 State Handling

- State is ephemeral and derived
- No long-lived simulation state
- No local authority over entities

---

# 4. UX Philosophy

The TUI must:

- maximize clarity of cause → effect
- make combat readable
- surface outcomes cleanly

From constitution:

- player agency must be visible
- outcomes must be explainable   

---

# 5. Testing Discipline

You must:

- test rendering correctness
- test event ordering
- test UI state transitions

You must NOT:

- replicate backend tests
- simulate gameplay in tests

---

# 6. Anti-Patterns (Hard Failures)

- Computing damage locally
- Predicting combat results
- Reconstructing backend logic
- “Fixing” backend bugs in UI
- Creating synthetic state

---

# 7. When Backend Is Insufficient

If required data is missing:

- DO NOT infer
- DO NOT simulate

Instead:

- expose limitation in UI
- note requirement in report
- defer to backend mission

---

# 8. When Uncertain

Default to:

- render, don’t simulate
- show, don’t infer
- ask backend, don’t guess

---

# Final Rule

The TUI is a **window into the simulation**, not the simulation itself.

If a change makes the UI smarter but less truthful:

**Reject the change.**
