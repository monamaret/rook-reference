# Rook v0.6 — Messaging Service

**ID:** PRD007  
**Version:** v0.6  
**Status:** Stub  
**Date:** 2026-04-25  
**Author:** Mona Maret  
**Parent:** [PRD001 — Rook Project Overview for PoC](PRD001-rook-overview-v1.0.md)  

---

## Overview

This release introduces the `messaging-service` backend and the `rook-cli` messaging TUI, supporting conversations and threaded messages within a space. The service provides endpoints for storing and retrieving messages, backed by a Firestore schema designed for thread-organized conversations. The CLI surfaces a conversation list, a thread view, and a compose interface. As with the stash in v0.4, messaging is offline-first in this release — the TUI reads and writes to a local message store only, with sync deferred to v0.7.

---

## Feature Focus

_To be detailed in scoping. Proposed areas:_

- `messaging-service` Firestore schema: conversations collection (space, participants, metadata) and messages sub-collection (author, body, timestamp, threadRef)
- `messaging-service` store endpoint: `POST /conversations/{id}/messages` — append a message to a conversation
- `messaging-service` retrieve endpoints: `GET /conversations`, `GET /conversations/{id}`, `GET /conversations/{id}/messages`
- Space-scoped conversation visibility: conversations belong to a space; all space members can read
- `rook-cli` conversation list TUI: list of conversations with participant names, last message preview, and timestamp
- `rook-cli` thread view TUI: scrollable message history with author attribution and timestamps
- `rook-cli` compose interface: single-line or multi-line message input with send confirmation
- Local message store: SQLite cache in XDG data directory, structured to match the Firestore schema for easy sync in v0.7

---

## Dependencies

- PRD006 v0.5 complete — sync patterns and local store conventions from stash inform the messaging local store design

---

## Out of Scope for This Release

- Real-time message delivery — no WebSocket or SSE push; messages are fetched on demand
- Message sync between local store and messaging-service — deferred to v0.7
- Unread counts or notification badges in the launcher — deferred to v0.7
- Reactions, edits, or deletions of sent messages
- Direct messages between individuals — conversations are space-scoped group threads only

---

## Open Questions

_To be resolved during scoping._

- Are conversations created explicitly by users, or do they emerge implicitly from message threads — and who can create a new conversation within a space?
- Should the messaging local store schema be identical to the stash local store (same SQLite model patterns) for consistency, or does messaging have different access patterns that warrant a different structure?
- What is the message size limit, and does the service need to enforce pagination on `GET /conversations/{id}/messages` from the start?

---

## Notes

_Space for any early design notes, constraints, or decisions relevant to this release._

- The thread view TUI should handle long conversations gracefully — virtual scrolling or paginated fetch will be needed; design with this in mind even if full pagination is deferred.
- Message author attribution should use the identity's `displayName` resolved from the user-service; avoid embedding raw key fingerprints in the message view.
- The local store should record a `fetchedAt` cursor per conversation so that v0.7 sync can request only messages newer than the last fetch.
- Compose input should support at minimum a `ctrl+enter` / `enter` send convention and `esc` to cancel, consistent with the stash editor keybindings.
