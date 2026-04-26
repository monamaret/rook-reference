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

---

## Feature Focus

_To be detailed in scoping. Proposed areas:_

- `guides-service` Firestore schema: guides collection (space, title, slug, body, actions, author, version, published status, timestamps)
- `guides-service` fetch endpoints: `GET /guides` (list, filterable by space/tag), `GET /guides/{id}` (full guide content)
- YAML action block specification: schema for embedded action types — links, shell commands, checklists, references to other guides
- `rook-cli` guide list TUI: browsable index of published guides for the current space with title, author, and last-updated
- `rook-cli` guide reader TUI: full-screen glamour-rendered Markdown view with lipgloss navigation chrome (breadcrumb, scroll position, keybindings footer)
- YAML action parser in `rook-cli`: extract and render action blocks as interactive elements (e.g. copyable commands, clickable links)
- Guide caching: store fetched guide content locally in the XDG cache directory with TTL-based invalidation
- `rook guide list` and `rook guide read {id|slug}` CLI entrypoints

---

## Dependencies

- PRD008 v0.7 complete — launcher integration and space context required; guide tile in launcher must be wired up
- PRD004 v0.3 complete — space-scoped data access pattern established

---

## Out of Scope for This Release

- Guide authoring, editing, or publishing — read-only access only
- Server-side full-text search — local filter by title and tag only
- Guide versioning or diff view — always shows the current published version
- Interactive action execution (e.g. running embedded shell commands) — actions are rendered but not executed
- Guide access control beyond space-level membership

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
