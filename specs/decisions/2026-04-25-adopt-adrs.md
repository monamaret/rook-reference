---
status: accepted
date: 2026-04-25
decision-makers: rook-project maintainers
---

# Adopt Architecture Decision Records

## Context and Problem Statement

The Rook reference implementation is a multi-component project (server, CLI) with non-trivial architectural decisions spanning infrastructure, protocol choices, and client/server boundaries. Without a lightweight mechanism to record *why* decisions were made, future contributors and coding agents will lack the context to maintain consistency or make compatible choices.

## Decision Outcome

Adopted. ADRs will be stored in `specs/decisions/` using date-prefixed filenames (`YYYY-MM-DD-short-title.md`) and the MADR-style template for decisions with multiple options, or the simple template for clear-cut decisions.

### Consequences

- Good, because decisions are self-documenting and discoverable by both humans and coding agents.
- Good, because the MADR format captures rejected alternatives, preventing the same debates from recurring.
- Bad, because it adds a small authoring overhead per decision.

## More Information

See `specs/decisions/README.md` for conventions and the full ADR index.
