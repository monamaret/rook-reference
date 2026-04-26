# Rook v0.7 — Messaging Sync

**ID:** PRD008  
**Version:** v0.7  
**Status:** Stub  
**Date:** 2026-04-25  
**Author:** Mona Maret  
**Parent:** [PRD001 — Rook Project Overview for PoC](PRD001-rook-overview-v1.0.md)  

---

## Overview

This release connects the local message store to the `messaging-service`, enabling messages composed offline to be sent and new messages from other participants to be fetched. Unread count metadata is introduced — both stored server-side and surfaced in the `rook-cli` launcher as notification indicators on the messaging app tile. By the end of this release, messaging is a functional asynchronous communication channel between space members.

---

## Feature Focus

_To be detailed in scoping. Proposed areas:_

- Message sync engine in `rook-cli`: fetch new messages per conversation since the local `fetchedAt` cursor, send queued outbound messages
- `rook messages sync` command (or automatic sync on launcher startup and on entering the messaging TUI)
- Outbound message queue: messages composed offline are queued locally and flushed on next sync
- `messaging-service` unread count endpoint: `GET /conversations/unread` — return per-conversation unread counts for the authenticated identity
- Server-side read-state tracking: `POST /conversations/{id}/read` — mark a conversation as read up to a message ID
- Launcher notification indicators: unread badge on the messaging app tile, fetched on startup and refreshed on sync
- Conversation list sort: conversations ordered by most recent message, with unread conversations highlighted
- Error handling: failed sends are retained in the queue with status indicators; retry on next sync

---

## Dependencies

- PRD007 v0.6 complete — local message store, messaging TUI, and messaging-service endpoints required
- PRD006 v0.5 complete — sync engine patterns from stash sync applicable to messaging sync

---

## Out of Scope for This Release

- Real-time push delivery (WebSocket, SSE, or FCM) — polling/on-demand sync only
- Per-message read receipts visible to other participants — unread counts are self-only
- Notification sounds or OS-level desktop notifications
- Message search backed by the server
- Typing indicators or presence information

---

## Open Questions

_To be resolved during scoping._

- Should unread counts be computed server-side (stored in Firestore per identity) or computed client-side from the last-read cursor — which approach is simpler for the PoC?
- How should the outbound message queue handle ordering guarantees — must messages be delivered in compose order, and what happens if two queued messages fail independently?
- Should sync run automatically at a polling interval while the messaging TUI is open, or only on explicit user action for the PoC?

---

## Notes

_Space for any early design notes, constraints, or decisions relevant to this release._

- The `fetchedAt` cursor per conversation should be stored in the local SQLite store and updated atomically after each successful fetch to prevent duplicate message ingestion.
- Outbound message queue should survive process restarts — store queued messages in the local SQLite store with a `status` field (`queued`, `sent`, `failed`).
- The launcher unread badge should not make a blocking network call on startup — load cached unread counts first, refresh in the background.
- Read-state tracking (`POST /conversations/{id}/read`) should be best-effort; a failure here must not block the user from viewing messages.
