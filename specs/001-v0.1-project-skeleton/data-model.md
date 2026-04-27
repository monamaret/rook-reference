# Data Model: Rook v0.1 — Project Skeleton

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)  
**Date**: 2026-04-26

---

## Overview

v0.1 introduces a single persistent entity: the **Config** file. There is no database, no Firestore, and no network-resident state in this release. All data is contained in one JSON file on the local filesystem.

---

## Entity: Config

The `Config` struct is the in-memory representation of `$XDG_CONFIG_HOME/rook/config.json`. It is loaded once at startup, used read-only for the lifetime of the process, and written atomically when the user completes first-run setup or triggers an explicit save.

### Fields

| Field | JSON key | Go type | Required | Default | Description |
|---|---|---|---|---|---|
| Servers | `servers` | `[]ServerEntry` | No | `[]` (empty) | Ordered list of configured `rook-server` addresses. Empty on first launch. |
| ActiveSpace | `active_space` | `string` | No | `""` | Space ID of the last-active space. Empty until the user authenticates and selects a space. |
| StorageDir | `storage_dir` | `string` | No | `""` | Absolute path to the flat-file store root. When empty, resolved at runtime to `$XDG_DATA_HOME/rook/storage/` (fallback: `~/.local/share/rook/storage/`). |
| FeatureFlags | `feature_flags` | `map[string]bool` | No | `{}` | Reserved for future use. Ignored in v0.1; must round-trip without data loss. |

### Nested Entity: ServerEntry

A single configured `rook-server` endpoint.

| Field | JSON key | Go type | Required | Description |
|---|---|---|---|---|
| Address | `address` | `string` | Yes | Base URL of the server (e.g., `https://rook.example.com`). Must be a valid absolute URL. |
| Alias | `alias` | `string` | No | Human-readable label for display in the launcher (e.g., `"work"`, `"personal"`). |

---

## Validation Rules

| Rule | Applies to | Behaviour on violation |
|---|---|---|
| `address` must be a non-empty absolute URL | `ServerEntry.Address` | `Load()` returns a validation error; config is not used. |
| `storage_dir`, if set, must be an absolute path | `Config.StorageDir` | `Load()` returns a validation error. |
| Unknown JSON fields | All | Silently ignored (`json:",omitempty"` not applied to decode; decoder uses `DisallowUnknownFields` is NOT set — forward-compatible). |
| Missing config file | `Load()` | Returns `(Config{}, ErrNotFound)` — not a fatal error; caller decides whether to initiate first-run setup. |

---

## State Transitions

```
[no config file]
      │
      │ first-run setup completes
      ▼
[config file written atomically]
      │
      │ rook-cli starts
      ▼
[Config loaded into memory (read-only)]
      │
      │ user updates a setting (v0.2+)
      │ OR first-run setup writes initial config (v0.2)
      ▼
[Config.Save() → temp file → os.Rename → config file updated]
```

In v0.1, `Config.Save()` exists and is tested, but the only caller path that invokes it is the stub first-run flow (writing an empty/default config). Substantive first-run setup is implemented in v0.2.

---

## File Layout

```
$XDG_CONFIG_HOME/rook/        (created by rook-cli on first save if absent)
└── config.json               (Config entity — JSON, UTF-8, no BOM)
```

Example `config.json` (after v0.1 first-run stub):

```json
{
  "servers": [],
  "active_space": "",
  "storage_dir": "",
  "feature_flags": {}
}
```

---

## Relationships

No relationships in v0.1 — Config is the only entity, and it contains no foreign keys to server-side data. `active_space` is a string reference to a space ID that will be resolved against `user-service` in v0.2.

---

## Notes for Implementers

- Use `json.MarshalIndent` with 2-space indent for human-readable on-disk JSON — easier to inspect with `cat` or in `git diff`.
- `StorageDir` resolution must happen at the call site (e.g., in `main.go` startup), not inside `Load()` — keep `Load()` pure (no env reads beyond the config file path itself).
- `FeatureFlags` must round-trip even with unknown flag keys — do not strip unknown map entries on `Save()`.
- `ErrNotFound` should be a sentinel (e.g., `var ErrNotFound = errors.New("config: file not found")`) so callers can `errors.Is` it without string comparison.
