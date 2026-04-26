---
status: proposed
date: 2026-04-25
decision-makers: Mona Maret
consulted: Mona Maret
informed: Mona Maret
---

# Guides Offline Store: Flat Files vs SQLite

## Context and Problem Statement

The guides reader (PRD009) adds offline reading as a first-class capability. Saved guides must be persisted locally so they can be read without a network connection. The storage format for this offline store must be chosen.

The stash offline store (PRD005) left this as an open question and ultimately uses flat `.md` + `.json` pairs. The guides offline store faces the same decision but with a simpler access pattern: guides are read-only for readers, so there is no local editing, no dirty state, and no conflict resolution.

## Decision Drivers

- No new Go module dependencies should be introduced unless justified
- Local data must be human-readable and directly inspectable (project-wide constraint from the system architecture ADR)
- The access pattern is simple: list saved guides, read a guide by ID/slug, check if a guide is saved, upsert on save, delete on remove
- Guides are read-only for readers — no local mutations after save

## Considered Options

- **Option A: SQLite** — single database file; queryable; requires `mattn/go-sqlite3` (CGo) or `modernc.org/sqlite` (pure Go, larger binary)
- **Option B: Flat files** — one directory per guide with `meta.json` + asset files; no dependencies; directly inspectable

## Decision Outcome

**Chosen: Option B — Flat files.**

Because guides are read-only for readers, the access pattern never requires cross-guide queries, aggregations, or transactions. All operations (`list`, `get`, `upsert`, `delete`) map directly onto `os.ReadDir`, file reads, atomic directory write (temp dir + `os.Rename`), and `os.RemoveAll`. No new dependencies. Consistent with the project-wide flat-file convention.

SQLite would be appropriate if offline search across saved guides were in scope — it is not (explicitly out of scope in PRD009).

## Store Layout

```
<storage-dir>/guides/saved/<space-id>/<guide-id>/
├── meta.json       # metadata + sync state
├── content.md      # guide body
├── style.yml       # lipgloss theme
└── config.yml      # navigation + actions config
```

### `meta.json` schema

```json
{
  "id": "<uuid>",
  "space_id": "<space-id>",
  "slug": "getting-started",
  "title": "Getting Started with Rook",
  "description": "...",
  "version": "1.0.0",
  "owner_id": "<user-id>",
  "published_at": "2026-04-25T12:00:00Z",
  "saved_at": "2026-04-26T09:00:00Z",
  "synced_at": "2026-04-25T12:00:00Z",
  "sync_state": "synced"
}
```

### `sync_state` values

| Value | Meaning |
|---|---|
| `synced` | Local copy matches server version (`synced_at == server published_at`) |
| `stale` | Server has a newer `published_at`; re-fetch needed |
| `unavailable` | Last pull failed; local copy retained but may be stale |

## Consequences

- No new module dependencies
- Consistent with stash and messages flat-file conventions
- Directly inspectable by users and operators
- If cross-guide full-text search is added in a future release, a migration to SQLite (or an index file) would be needed — acceptable given this is explicitly out of scope for the PoC
