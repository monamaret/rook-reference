# Rook v0.5 — Stash Sync

**ID:** PRD006  
**Version:** v0.5  
**Status:** Stub  
**Date:** 2026-04-25  
**Author:** Mona Maret  
**Parent:** [PRD001 — Rook Project Overview for PoC](PRD001-rook-overview-v1.0.md)  

---

## Overview

This release connects the local stash store to the `stash-service`, enabling push/pull synchronization of documents between the CLI and the server. Sync state is tracked per document so the CLI can identify local-only, server-only, dirty, and conflicted documents. The `rook-cli` sync UX surfaces sync status in the stash list view and provides explicit push/pull commands. This release makes the stash a genuinely collaborative, persistent feature rather than a local scratchpad.

---

## Feature Focus

_To be detailed in scoping. Proposed areas:_

- Sync engine in `rook-cli`: per-document sync state machine (`local`, `synced`, `dirty`, `conflict`, `deleted`)
- `rook stash push` command: upload locally dirty documents to stash-service, update sync state on success
- `rook stash pull` command: fetch documents from stash-service that are absent or newer than the local copy
- `rook stash sync` command: bidirectional push+pull in one operation with conflict detection
- Conflict detection strategy: last-write-wins by default for the PoC, with conflict flagging for manual resolution
- Sync state display in stash list TUI: icons or indicators for dirty, synced, conflict, and local-only documents
- Background sync on launcher startup: silent pull of latest documents when session is valid
- `stash-service` pagination: `GET /stash` must support cursor-based pagination for large document sets

---

## Dependencies

- PRD005 v0.4 complete — local stash store, CRUD endpoints, and stash TUI required

---

## Out of Scope for This Release

- Real-time or event-driven sync — sync is always user-initiated or triggered on startup
- Operational transform or CRDT-based merge — conflict resolution is flag-and-defer for the PoC
- Sync of deleted documents across devices beyond soft-delete propagation
- Bandwidth optimization (diffing, compression) — full document payloads for the PoC
- Sync of attachments or binary content

---

## Open Questions

_To be resolved during scoping._

- What constitutes a conflict — is it simply a server `updatedAt` timestamp newer than the local `syncedAt`, or does it require field-level diffing?
- Should `rook stash sync` be fully automatic (e.g. run on every `rook stash list`) or always explicit — what is the right default for a PoC?
- How should the CLI handle a sync failure mid-operation (e.g. partial push) — rollback, retry, or leave in a dirty state with a clear error?

---

## Notes

_Space for any early design notes, constraints, or decisions relevant to this release._

- The sync state machine should be well-defined and tested in isolation before integrating with the TUI to avoid coupling UI state to sync correctness.
- Push should be idempotent: if the same document is pushed twice with the same content, the service should return success without creating a duplicate version.
- The `syncedAt` timestamp stored locally must be the server's `updatedAt` from the response, not the local clock, to avoid drift-based false conflicts.
- Consider adding a `rook stash status` command that shows a quick diff of local vs. server state without performing any writes.
