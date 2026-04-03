# Claude Code Mission Guidelines

## Purpose

Missions must be:

- bounded
- small
- deterministic
- UI-only

---

## Mission Rules

Each mission must:

1. Affect only seq-tui repo
2. Avoid backend changes
3. Avoid gameplay logic
4. Avoid large refactors
5. Be independently testable

---
## Mission Tracking

All missions MUST be stored in:

~/src/seq/seq-notes/missions/

The supervising agent must:

- create a mission file before implementation
- follow existing mission naming conventions
- update MISSION_INDEX.md when appropriate
- keep missions small and bounded

seq-tui repository must not store mission files.

---

## Reporting

After completing a mission, the agent must:

- provide a concise console summary
- optionally generate a report file

Report files should be written to:

~/src/seq/seq-notes/reports/

If reports directory does not exist, it may be created.

Reports must be:
- markdown
- concise
- drag-and-drop friendly
- easy to paste into ChatGPT

---

## Allowed Work

- rendering
- layout
- input handling
- ASCII map display
- panels
- status views

---

## Not Allowed

- combat logic
- movement rules
- simulation
- encounter logic
- scheduling logic

---

## Mission Size

Each mission should:

- take < 1 hour
- change limited files
- be reviewable easily

---

## Mission Structure

Each mission should specify:

- Goal
- Scope
- Constraints
- Acceptance Criteria
- Non-Goals

---

## Example Mission

Goal: Render static ASCII map

Scope:
- fixed grid
- no backend calls

Non-goals:
- movement
- mobs
- interaction