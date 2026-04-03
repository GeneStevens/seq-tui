# ADR-0001: Thin Client Principle

## Status
Accepted

## Context

Seq architecture requires:

- backend authoritative gameplay
- multiple thin clients
- shared gameplay model

seq-tui must not diverge from this model.

Without constraints, UI clients tend to:

- add simulation
- add inferred state
- embed gameplay logic

This would fragment gameplay semantics.

## Decision

seq-tui will be a thin client.

seq-tui will:

- render backend truth only
- send player intent only
- avoid gameplay interpretation

## Consequences

Positive:

- consistent gameplay across clients
- backend remains authoritative
- deterministic behavior
- simpler architecture

Negative:

- UI must tolerate backend latency
- fewer client-side conveniences

## Enforcement

All contributions must:

- avoid gameplay logic
- avoid simulation
- avoid derived rules