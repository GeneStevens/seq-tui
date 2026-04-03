# Seq TUI Architectural Position

## System Overview

The Seq ecosystem consists of:

- seq (backend authority)
- seqcli (scripted client)
- seq-web (graphical client)
- seq-tui (terminal client)

All clients are thin and share backend truth.

```
        seq backend
             │
 ┌───────────┼───────────┐
 seqcli    seq-web     seq-tui
```

## Role of seq-tui

seq-tui is:
- interactive spatial client
- terminal-based
- thin
- non-authoritative

seq-tui is not:
- a gameplay engine
- a simulation layer
- a rules interpreter

---

## Communication Model

seq-tui communicates with backend via:
- HTTP endpoints
- existing observability surfaces
- existing intent submission endpoints

seq-tui must not require new gameplay endpoints.

---

## State Ownership

Backend owns:
- player state
- zone state
- mob state
- encounter state
- combat outcomes

seq-tui owns:
- rendering layout
- input mapping
- screen refresh

---

## Mode Handling

RT and ASYNC:
- share identical rendering pipeline
- differ only in backend timing
- must not diverge in UI logic

---

## Instance Awareness

seq-tui must:
- support canonical instance identity
- avoid implicit instance selection
- rely on backend routing

---

## Evolution Constraints

seq-tui may evolve:
- rendering sophistication
- layout improvements
- input ergonomics

seq-tui must not evolve:
- gameplay semantics
- rule interpretation
- state derivation

---

## Agent Workflow Discipline

Development is performed via:

- mission file
- Claude Code execution
- result review
- bounded iteration

All changes must follow this workflow.

Direct large-scale implementation without mission definition is not allowed.