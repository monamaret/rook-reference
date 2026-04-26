# Rook v0.8 — Guides Reader

**ID:** PRD009  
**Version:** v0.8  
**Status:** Stub  
**Date:** 2026-04-25  
**Author:** Mona Maret  
**Parent:** [PRD001 — Rook Project Overview for PoC](PRD001-rook-overview-v1.0.md)  

---

## Overview

This release introduces the `guides-service` and the `rook-cli` guide reader, enabling users to browse and read structured guides published within their space. Guides are Markdown documents with a YAML action block extension that embeds structured metadata — links, commands, checklists — parsed and rendered by the CLI. The reader uses glamour for Markdown rendering and lipgloss for styled navigation chrome. This release is read-only; guide authoring and publishing are delivered in v0.9.

Offline reading is a first-class capability in this release: users can explicitly save guides to a local flat-file store and read them with no network connection, using the same sync-state model established by the stash feature (v0.5).

---

## Feature Focus

_To be detailed in scoping. Proposed areas:_

- `guides-service` Firestore schema: guides collection (space, title, slug, body, actions, author, version, published status, timestamps)
- `guides-service` fetch endpoints: `GET /guides` (list, filterable by space/tag), `GET /guides/{id}` (full guide content)
- YAML action block specification: schema for embedded action types — links, shell commands, checklists, references to other guides
- `rook-cli` guide list TUI: browsable index of published guides for the current space with title, author, and last-updated
- `rook-cli` guide reader TUI: full-screen glamour-rendered Markdown view with lipgloss navigation chrome (breadcrumb, scroll position, keybindings footer)
- YAML action parser in `rook-cli`: extract and render action blocks as interactive elements (e.g. copyable commands, clickable links)
- Guide TTL cache: store fetched guide content in the XDG cache directory with TTL-based invalidation — a performance layer for online reads only
- Offline guide store: persistent flat-file store in `<storage-dir>/guides/saved/<space-id>/<guide-id>/`; separate from the TTL cache; survives cache eviction
- Sync-state tracking per saved guide: `synced`, `stale`, `unavailable` — mirrors the stash sync-state model
- `rook guide list` TUI: `📥` badge for locally saved guides
- `rook guide list`, `rook guide read {id|slug}`, `rook guide save {id|slug}`, `rook guide saved`, `rook guide remove {id|slug}`, and `rook guide pull [id|slug]` CLI entrypoints
- Offline fallback in `rook guide read`: render from local store with `⚠ Offline` notice when server unreachable; helpful error if guide not saved
- Silent background pull on launcher startup: re-fetch saved guides if server version is newer; failures logged only

---

## Dependencies

- PRD008 v0.7 complete — launcher integration and space context required; guide tile in launcher must be wired up
- PRD004 v0.3 complete — space-scoped data access pattern established
- PRD006 v0.5 complete — sync-state model and background sync conventions established by stash sync

---

## Out of Scope for This Release

- Guide authoring, editing, or publishing — read-only access only
- Server-side full-text search — local filter by title and tag only
- Guide versioning or diff view — always shows the current published version
- Interactive action execution (e.g. running embedded shell commands) — actions are rendered but not executed
- Guide access control beyond space-level membership
- Auto-save on read — saving is always explicit (`rook guide save`) or pull-driven (`rook guide pull`)
- Conflict resolution — read-only; stale copy replaced on pull, no merge required
- Offline publishing or authoring — guide publish requires a live server connection

---

## Open Questions

_To be resolved during scoping._

- What is the canonical YAML action block syntax — embedded as a fenced code block with a `yaml-action` language tag, or as a custom Markdown extension?
- Should glamour use a built-in theme or a custom lipgloss-based theme to ensure consistent styling with the rest of the TUI?
- How should the guide reader handle guides that reference other guides (internal links) — open in-reader, push to a navigation stack, or open in the list view?

---

## Notes

_Space for any early design notes, constraints, or decisions relevant to this release._

- The YAML action block schema should be finalized before the reader is implemented, as it will be the same format the guide builder (v0.9) produces — misalignment here creates rework.
- glamour's default word wrap width should respect the terminal width; test on narrow terminals (80 col) as well as wide ones.
- Guide slugs should be unique per space and URL-safe; the service should enforce uniqueness at write time (relevant for v0.9 publish).
- Cache invalidation should be based on the guide's `updatedAt` timestamp fetched from the list endpoint, avoiding unnecessary full-body fetches.
- The offline store uses flat files — not SQLite. Guides are read-only for readers (no dirty state, no local editing), so a flat directory is sufficient, has no new dependencies, and is directly inspectable.
- `rook guide pull` staleness check: compare local `synced_at` vs server `published_at` from GET /guides list — one lightweight call, full re-fetch only when stale.
- Offline store path `<storage-dir>/guides/saved/<space-id>/<guide-id>/` follows the same convention as stash.
