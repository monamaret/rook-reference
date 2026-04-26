# Rook v0.9 — Guide Builder

**ID:** PRD010  
**Version:** v0.9  
**Status:** Stub  
**Date:** 2026-04-25  
**Author:** Mona Maret  
**Parent:** [PRD001 — Rook Project Overview for PoC](PRD001-rook-overview-v1.0.md)  

---

## Overview

This release delivers the guide authoring surface: a `rook-cli` guide builder TUI that allows authorized users to create, edit, preview, validate, and publish guides to their space. The `guides-service` gains a publish endpoint with server-side validation of the guide schema and YAML action blocks. The builder completes the guides feature loop — combined with the reader from v0.8, guides become a fully editable and publishable knowledge format within Rook.

---

## Feature Focus

_To be detailed in scoping. Proposed areas:_

- `rook-cli` guide builder TUI: new guide wizard (title, slug, tags), inline Markdown editor, and guide management list (edit, preview, delete, publish, unpublish)
- Live preview pane in the builder: split view or toggle between edit and glamour-rendered preview without leaving the TUI
- YAML action block editor: structured form or fenced block assistant for inserting and editing action blocks within guide body
- Client-side guide validation: check required fields, slug format, action block schema, and Markdown structure before publish
- `guides-service` publish endpoint: `POST /guides` (create) and `PUT /guides/{id}` (update/republish) with server-side validation mirroring client-side rules
- `guides-service` unpublish endpoint: `PATCH /guides/{id}/status` — toggle published/draft state without deleting content
- Authorization: only space members with an `author` or `admin` role may publish; `guides-service` enforces this
- `rook guide new`, `rook guide edit {id|slug}`, and `rook guide publish {id|slug}` CLI entrypoints

---

## Dependencies

- PRD009 v0.8 complete — guides-service, Firestore schema, YAML action block spec, and guide reader TUI all required
- PRD004 v0.3 complete — space membership and role model required for publish authorization

---

## Out of Scope for This Release

- Guide versioning with history and rollback — publish overwrites the current version only
- Collaborative co-authoring or real-time editing
- Image or binary asset embedding in guides
- Guide import/export (e.g. from external Markdown files) beyond `$EDITOR` passthrough
- Admin-level guide management across spaces — builders manage guides within their own space only

---

## Open Questions

_To be resolved during scoping._

- What roles within a space are permitted to publish guides — is `author` a distinct role from `member`, or is publish permission a space-level flag on any member?
- Should the builder support draft saving (local, unpublished) so authors can iterate before publishing, and if so, how does draft state interact with the guides-service schema?
- Is a split-view editor (edit + preview side-by-side) feasible within typical terminal widths, or should preview be a toggle/separate screen?

---

## Notes

_Space for any early design notes, constraints, or decisions relevant to this release._

- Server-side validation in the publish endpoint must be the authoritative check; client-side validation is a UX convenience and must not be the sole gatekeeper.
- The guide builder's Markdown editor should wrap `$EDITOR` as a fallback (consistent with the stash editor pattern) so authors can use their preferred tool.
- Slug uniqueness enforcement at publish time must return a clear error message with a suggested alternative slug to reduce friction for authors.
- The unpublish flow should confirm intent with a prompt before sending the request, since readers will immediately lose access to the guide.
