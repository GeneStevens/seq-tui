# Seq TUI Gameplay Constitution

## Purpose

The Seq TUI exists to evaluate **spatial playability** and **sense of place** in a zone,
without introducing gameplay mechanics or diverging from backend truth.

The TUI is not a gameplay engine.
The TUI is not a simulation.
The TUI is not authoritative.

The TUI is an **observational and input surface** only.

---

## Core Principles

### 1. Backend Authority
All gameplay logic lives in the backend.

The TUI:
- must not compute combat outcomes
- must not simulate encounters
- must not derive game state
- must not predict events

The TUI only renders backend truth and sends player intent.

---

### 2. Thin Client
The TUI must remain a thin client.

Allowed:
- rendering
- input handling
- layout
- presentation logic

Not allowed:
- gameplay rules
- scheduling logic
- combat logic
- encounter logic
- movement rules

---

### 3. Shared Gameplay Model
RT and ASYNC modes must not diverge in the TUI.

The TUI must:
- display both modes consistently
- avoid mode-specific gameplay behavior
- avoid special-case UI logic that alters interpretation

The backend defines differences.
The TUI only displays them.

---

### 4. Spatial Presence Over Mechanics
The TUI exists to answer:

"Does the world feel like a place worth inhabiting?"

Therefore prioritize:
- movement
- spatial awareness
- encounter presence
- zone exploration
- clearing areas
- moving deeper

Avoid:
- abilities
- classes
- inventory
- progression systems
- skill trees

---

### 5. No Client-Side Simulation
The TUI must not:
- simulate mobs
- simulate movement
- simulate cooldowns
- simulate timers

All timing must come from backend truth.

---

### 6. Deterministic Rendering
Given the same backend state,
the TUI must render the same output.

No randomness in rendering.
No stochastic UI behavior.

---

### 7. Minimalism
The TUI must evolve in small, bounded steps.

Each addition must:
- preserve thin-client discipline
- preserve backend authority
- preserve shared gameplay model

---

## Non-Goals

The TUI is not:
- a full MMO client
- a replacement for seq-web
- a gameplay sandbox
- a mechanics experimentation layer

---

## Success Criteria

The TUI succeeds if a player can:

- move through a zone
- observe mobs
- encounter threats
- clear areas
- retreat safely
- move deeper
- feel progression through space

Without adding gameplay complexity.