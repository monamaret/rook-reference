# Rook v0.4 — Stash Service

**ID:** PRD005  
**Version:** v0.4  
**Status:** Stub  
**Date:** 2026-04-25  
**Author:** Mona Maret  
**Parent:** [PRD001 — Rook Project Overview for PoC](PRD001-rook-overview-v1.0.md)  

---

## Overview

This release delivers the `stash-service` backend and the `rook-cli` stash TUI, enabling users to create, view, and edit stash documents locally. The stash-service exposes CRUD endpoints backed by Firestore, and the CLI provides an interactive document editor and list view. Sync between the CLI and the service is explicitly deferred — in this release the TUI operates against a local store only, establishing the UX and data model before the sync layer is added in v0.5.

---

## Feature Focus

_To be detailed in scoping. Proposed areas:_

- `stash-service` Firestore schema: document collection with owner, space, content, tags, created/updated timestamps, and sync-state fields
- `stash-service` CRUD endpoints: `POST /stash`, `GET /stash`, `GET /stash/{id}`, `PATCH /stash/{id}`, `DELETE /stash/{id}`
- Space-scoped document visibility: documents are readable by all members of the owning space
- `rook-cli` stash TUI: document list view with search/filter, document detail view, and inline editor
- Local stash store: SQLite or flat-file cache in the XDG data directory for offline-first operation
- Document content format: Markdown body with a YAML front-matter header for title, tags, and metadata
- `rook stash new`, `rook stash list`, `rook stash edit {id}`, and `rook stash delete {id}` CLI entrypoints
- TUI keyboard navigation conventions: consistent with launcher patterns established in v0.3

---

## Dependencies

- PRD004 v0.3 complete — space identity, authenticated HTTP client, and launcher shell required

---

## Out of Scope for This Release

- Sync between local stash store and stash-service — all edits are local-only
- Conflict resolution or merge logic for concurrent edits
- Attachment or binary file support — text/Markdown documents only
- Full-text search backed by the server — local filter only
- Stash sharing or access control beyond space-level membership

---

## Open Questions

_To be resolved during scoping._

- Should the local stash store be SQLite (queryable, structured) or a flat directory of Markdown files (portable, inspectable)? What are the tradeoffs for the sync model in v0.5?
- What is the maximum document size, and should the service enforce a limit at the API layer?
- How should the inline editor handle unsaved changes when the user navigates away — autosave, prompt, or discard?

---

## Notes

_Space for any early design notes, constraints, or decisions relevant to this release._

- The stash document schema should include a `syncState` field (e.g. `local`, `synced`, `dirty`, `conflict`) from the start, even though sync is not implemented until v0.5, to avoid a schema migration.
- The CLI editor should wrap `$EDITOR` as a fallback for users who prefer an external editor over the bubbletea inline editor.
- CRUD endpoint authorization must verify that the requesting identity is a member of the document's owning space; return 403 for cross-space access attempts.
- Consider whether `DELETE` is a hard delete or a soft delete with a `deleted` flag — soft delete is safer for future sync semantics.
