---
status: proposed
date: 2026-04-25
decision-makers: Mona Maret
consulted: Mona Maret
informed: Mona Maret
---

# Guides Service Architecture

## Context and Problem Statement

The system architecture ADR ([`2026-04-25-rook-reference-system-architecture.md`](2026-04-25-rook-reference-system-architecture.md)) and the rook-cli UX ADR ([`2026-04-25-rook-cli-features-and-ux-architecture.md`](2026-04-25-rook-cli-features-and-ux-architecture.md)) both reference a `guides-service` and a guide builder TUI without fully specifying the service boundary, storage model, asset schema, authorship model, access control, or the contract between the CLI builder and the server.

This ADR defines all aspects of the guides service: what a guide is, how it is authored, validated, stored, published, fetched, and access-controlled — from the guide builder in `rook-cli` through to `guides-service` on `rook-server`.

## Decision Drivers

- Guides must be authored and published entirely within `rook-cli` — no server-side filesystem access or separate tooling required
- Guide rendering must be fully local after fetch — no streaming or per-page server requests during reading
- Access control must be consistent with the rest of the system: group ACL via `user-service`, defence-in-depth checked at fetch time
- The guide asset format must be human-readable, flat-file-friendly, and renderable by the Charmbracelet stack
- `guides-service` must be stateless — all persistent state in Firestore; compatible with Cloud Run scale-to-zero
- Guide ownership (create, edit, unpublish, delete) belongs to the guide's creator; space admin is a superuser over all space guides
- The server must validate all uploaded assets before storing — the CLI validator is not the only gate

## Considered Options

### Guide storage model
- **Option A: Firestore documents only** — all guide assets stored as fields in Firestore documents (metadata, markdown content, style YAML, config YAML inline)
- **Option B: Firestore metadata + Cloud Storage blobs** — metadata in Firestore; large assets (`.md`, `.yml`) in Cloud Storage objects
- **Option C: Firestore metadata + Firestore subcollection assets** — metadata doc + `assets/` subcollection where each file is a document

### Guide asset schema
- **Option A: Monolithic bundle** — all assets in a single JSON/YAML envelope uploaded and fetched as one unit
- **Option B: Per-file assets** — each guide file (`.md`, lipgloss `.yml`, YAML config) stored and fetched individually

### Guide authorship model
- **Option A: Admin-only authoring** — only the space admin account can publish guides
- **Option B: Any authenticated space member** — any user can author and publish a guide; ACL controls visibility

## Decision Outcome

**Guide storage model:** Option C — Firestore metadata document + `assets/` subcollection. Keeps all state in Firestore (no additional GCP service to provision for the PoC), supports document-level security rules, and avoids the 1 MiB Firestore document size limit by splitting assets into individual documents. Cloud Storage (Option B) is the natural migration path if guide assets grow beyond document size limits.

**Guide asset schema:** Option A — monolithic bundle. The CLI always needs all assets to render a guide; fetching them individually adds round-trips with no benefit. The bundle is a single JSON envelope; the server stores it decomposed into the `assets/` subcollection and re-assembles it on fetch.

**Guide authorship model:** Option B — any authenticated space member can author and publish guides. This makes guides a first-class collaborative tool rather than an admin-only broadcast channel. The space admin retains superuser rights over all space guides.

---

## What Is a Guide

A guide is a self-contained, space-scoped, interactive markdown document authored and published by a `rook-cli` user. It is intended for tutorials, reference documentation, onboarding flows, and admin notices — persistent, styled content rather than ephemeral messages.

A guide consists of three asset files:

| Asset | Format | Purpose |
|---|---|---|
| `content.md` | Markdown | The guide's readable content. Rendered via `charmbracelet/glamour` with custom theming. |
| `style.yml` | YAML (lipgloss schema) | Per-guide custom theming: colours, borders, padding, typography. Applied at render time by the CLI's lipgloss style loader. |
| `config.yml` | YAML (guide config schema) | Interactive elements: navigation structure, buttons, action bindings. Parsed by the CLI's YAML action parser. |

All three files are required. The CLI scaffolds stubs for each on guide creation.

---

## Guide Lifecycle

```
[new]
  │
  ▼
[draft] ──────────────────────────────────────► [draft]
  │  (edit content / style / config in $EDITOR)    ▲
  │                                                 │
  ▼                                                 │ (edit + re-publish)
[validated]                                         │
  │                                                 │
  ▼                                                 │
[published] ◄────────────────────────────────────────
  │
  ├── unpublish ──► [unpublished] (local draft preserved)
  │
  └── delete ──► [deleted from Firestore] (local draft optionally removed)
```

- **Draft** — assets exist only in `<storage-dir>/guides/drafts/<guide-id>/` on the author's machine. Not visible to any other user.
- **Validated** — the CLI has run all validation checks against the local draft and found no errors. A transient state; validation is re-run on each publish attempt.
- **Published** — assets have been uploaded to `guides-service` and stored in Firestore. The guide appears in the app list for groups the creator has granted access to. The local draft is retained as the editable source of truth.
- **Unpublished** — the guide has been removed from Firestore by the owner. Local draft is preserved; the guide can be re-published.
- **Deleted** — the guide has been removed from Firestore. The owner is prompted whether to also delete the local draft.

---

## Guide Author and Ownership

- Any authenticated space member can create, publish, and manage their own guides.
- The creating user is recorded as the guide **owner** at publish time.
- Only the owner (or a space admin) can edit metadata (title, description, access), re-publish, unpublish, or delete a guide.
- The space admin account has superuser rights over all guides in the space — can unpublish or delete any guide without being the owner.
- Server admin endpoints for guide management require an admin-scoped credential (same as the admin CLI).

---

## Access Control

Guide visibility is controlled by the group ACL system defined in [`2026-04-25-rook-reference-system-architecture.md`](2026-04-25-rook-reference-system-architecture.md).

- At publish time, the owner specifies which space groups can see the guide. The default is the owner's own group.
- The guide's ACL is stored in its Firestore `meta` document as a list of `group_id` values.
- `user-service`'s `GET /spaces/{space-id}/apps` endpoint filters the app list by the requesting user's group — the user only sees guides their group has ACL access to. This is the **primary access gate**.
- `guides-service` performs a **defence-in-depth** `CheckAppAccess(user_id, space_id, guide_id)` gRPC call to `user-service` on every guide fetch and publish. If `CheckAppAccess` returns `DENIED`, the service returns HTTP 403 even if the guide was somehow listed.
- The guide ACL is not embedded in the session token — it is always resolved server-side at request time.

---

## Guide Asset Schema

### `content.md`

Standard Markdown. No special syntax extensions. Internal links to other sections within the same guide use standard Markdown anchor syntax (`[text](#section-id)`). References to other guides are not supported in the PoC — a guide is a self-contained document.

Maximum size: 512 KiB (enforced at upload by `guides-service`).

### `style.yml`

A YAML file conforming to the lipgloss style schema. Defines visual properties applied to the guide's glamour renderer at render time.

```yaml
# style.yml — example
theme: dark           # base glamour theme: "dark" | "light" | "notty"
heading:
  foreground: "#FF6E6E"
  bold: true
code_block:
  background: "#1A1A2E"
  foreground: "#E0E0FF"
  margin: 1
link:
  foreground: "#6EC6FF"
  underline: true
```

The CLI's lipgloss style loader applies these values to the glamour renderer configuration before rendering `content.md`. Unknown keys are rejected by the validator.

Maximum size: 64 KiB (enforced at upload).

### `config.yml`

A YAML file defining the guide's interactive elements and navigation structure.

```yaml
# config.yml — example
id: "onboarding-guide-001"
title: "Getting Started with Rook"
description: "A step-by-step onboarding guide for new space members."
version: "1.0.0"
navigation:
  type: linear          # "linear" | "menu"
  pages:
    - id: "intro"
      title: "Introduction"
      content_file: "content.md"   # single-file guide; multi-page not in PoC scope
actions:
  - id: "open-stash"
    label: "Open the Stash"
    type: "navigate"
    target: "stash"     # navigates to the stash feature on exit
  - id: "copy-key"
    label: "Copy SSH Public Key"
    type: "copy"
    value: "{{ssh_public_key}}"  # CLI-resolved template variable
```

**Supported navigation types (PoC):**
- `linear` — single content file; the guide is one document with optional action buttons at the bottom.
- `menu` — a top-level menu with named sections; each section renders a portion of `content.md` using heading anchors. Multi-file page sets are post-PoC.

**Supported action types (PoC):**
- `navigate` — on activation, the guide exits and the launcher navigates to the specified feature (`stash`, `messaging`, `launcher`).
- `copy` — copies a value to the clipboard. Supports the `{{ssh_public_key}}` template variable (resolved by the CLI at render time).

**Supported template variables (PoC):**
- `{{ssh_public_key}}` — the authenticated user's SSH public key in OpenSSH format.
- `{{user_id}}` — the authenticated user's ID.
- `{{space_id}}` — the active space ID.

Maximum size: 64 KiB (enforced at upload).

---

## CLI: Guide Builder TUI

The guide builder is a Bubble Tea application accessible from the launcher. It manages the full guide lifecycle on the author's local machine.

### Local Draft Storage

```
<storage-dir>/guides/drafts/<guide-id>/
├── content.md      # guide content
├── style.yml       # lipgloss theme
└── config.yml      # navigation + actions
```

The `<guide-id>` is a UUID generated by the CLI at guide creation time. It is used as the Firestore document ID after publish.

### Builder States and Transitions

| State | Description |
|---|---|
| **Home** | List of local draft guides + published guides owned by the user. Entry point to all other builder states. |
| **New guide** | Prompts for title and description; scaffolds draft directory with stub files; transitions to Home. |
| **Edit** | Shows the three asset files; each opens in `$EDITOR` via `tea.ExecProcess`; returns to Edit on exit. |
| **Preview** | Renders the guide full-screen using local draft assets — identical to the reader experience. Exit returns to Edit. |
| **Validate** | Runs all validators against local draft; displays per-file errors inline. Available on demand from Edit; runs automatically on Publish. |
| **Publish** | Runs validation; if clean, uploads bundle to `guides-service`; shows result. Blocked if validation errors exist. |
| **Manage** | Available for published guides owned by the user: update access (group ACL), unpublish, delete. |

### Validation Rules

The CLI validator enforces these rules before publish. The server re-validates the same rules on upload.

**`config.yml` validation:**
- Required fields: `id`, `title`, `description`, `version`, `navigation`.
- `id` must be a valid UUID.
- `navigation.type` must be `linear` or `menu`.
- Each action `type` must be a known type (`navigate`, `copy`).
- `navigate` actions must reference a known target (`stash`, `messaging`, `launcher`).
- Template variable references must use only known variables (`{{ssh_public_key}}`, `{{user_id}}`, `{{space_id}}`).
- No unknown top-level keys.

**`style.yml` validation:**
- `theme` must be `dark`, `light`, or `notty`.
- All property names must be in the allowed lipgloss property set.
- Colour values must be valid hex (`#RRGGBB`) or named glamour palette values.
- No unknown top-level keys.

**`content.md` validation:**
- Internal anchor links (`[text](#section-id)`) must resolve to headings present in the document.
- No validation of external URLs (not followed or checked).

---

## guides-service: Server-Side Specification

### HTTP Endpoints

All endpoints require `Authorization: Bearer <session-token>` and `X-Rook-Space-ID: <space-id>`.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/guides` | Session token | List all guides visible to the requesting user in the active space (ACL-filtered). Returns metadata only — no assets. |
| `GET` | `/guide/{guide-id}` | Session token | Fetch a guide bundle (all three assets + metadata). Defence-in-depth `CheckAppAccess` required. |
| `POST` | `/guide` | Session token | Publish a new guide or re-publish an existing one. Owner or space admin only. Full server-side validation before write. |
| `PATCH` | `/guide/{guide-id}/access` | Session token (owner or admin) | Update the guide's group ACL. |
| `DELETE` | `/guide/{guide-id}` | Session token (owner or admin) | Unpublish and delete a guide from Firestore. |

**Admin-only endpoints** (require admin-scoped credential, not a user session token):

| Method | Path | Description |
|---|---|---|
| `GET` | `/admin/guides` | List all guides in a space regardless of ACL. |
| `DELETE` | `/admin/guide/{guide-id}` | Delete any guide in a space. |

### Request: Publish (`POST /guide`)

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Getting Started with Rook",
  "description": "A step-by-step onboarding guide for new space members.",
  "version": "1.0.0",
  "access": ["group-id-1", "group-id-2"],
  "assets": {
    "content_md": "<base64-encoded content.md>",
    "style_yml": "<base64-encoded style.yml>",
    "config_yml": "<base64-encoded config.yml>"
  }
}
```

- `id` is the UUID generated by the CLI at guide creation. On re-publish, the existing Firestore document is overwritten.
- `access` is the list of `group_id` values that can see this guide. Must be non-empty. All group IDs are validated against the space's group list before write.
- All asset fields are required and base64-encoded. The server decodes, validates, and stores them.

### Response: Guide Bundle (`GET /guide/{guide-id}`)

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Getting Started with Rook",
  "description": "A step-by-step onboarding guide for new space members.",
  "version": "1.0.0",
  "owner_id": "user-abc123",
  "space_id": "space-xyz",
  "published_at": "2026-04-25T12:00:00Z",
  "assets": {
    "content_md": "<base64-encoded content.md>",
    "style_yml": "<base64-encoded style.yml>",
    "config_yml": "<base64-encoded config.yml>"
  }
}
```

### Server-Side Validation

`guides-service` re-runs the full validation suite on upload — identical rules to the CLI validator. This is not optional; the CLI validator is a convenience layer, not the sole gate.

If validation fails, the service returns HTTP 422 with a structured error body listing per-file errors, identical in format to the CLI's validation output. The guide is not written to Firestore on any validation failure.

### Firestore Schema

```
guides/
└── {space_id}/
    └── {guide_id}/
        ├── meta (doc)
        │   ├── title: string
        │   ├── description: string
        │   ├── version: string
        │   ├── owner_id: string
        │   ├── space_id: string
        │   ├── access: [group_id, ...]   # group ACL
        │   └── published_at: timestamp
        └── assets/ (subcollection)
            ├── content_md (doc) { data: string }
            ├── style_yml   (doc) { data: string }
            └── config_yml  (doc) { data: string }
```

Assets are stored as plain strings (decoded from base64 on upload). No binary storage. The `assets/` subcollection sidesteps Firestore's 1 MiB document size limit — each asset document can be up to 1 MiB independently.

### gRPC Dependencies

`guides-service` makes two gRPC calls to `user-service` on every authenticated request:

1. `ValidateSession(token, space_id)` → `{user_id, group}` — called by the shared `SessionAuthMiddleware`; provides identity and space membership.
2. `CheckAppAccess(user_id, space_id, guide_id)` → `AccessDecision` — defence-in-depth ACL check on guide fetch and publish. Not called for `/guides` list (the list is already ACL-filtered by `ValidateSession` + `GET /spaces/{id}/apps`).

Both calls use OIDC-authenticated service accounts as specified in [`2026-04-25-grpc-inter-service-communication.md`](2026-04-25-grpc-inter-service-communication.md).

---

## Implementation Plan

### Affected Paths

**guides-service (rook-server):**
- `rook-server/guides-service/main.go` — service entry point; Cloud Run HTTP server; middleware registration
- `rook-server/guides-service/handlers/guides.go` — `GET /guides`, `GET /guide/{id}`, `POST /guide`, `PATCH /guide/{id}/access`, `DELETE /guide/{id}`
- `rook-server/guides-service/handlers/admin.go` — admin-scoped guide management endpoints
- `rook-server/guides-service/store/firestore.go` — Firestore read/write for guide metadata and assets
- `rook-server/guides-service/validator/` — server-side asset validation (mirrors CLI validator rules)
- `rook-server/guides-service/middleware/auth.go` — `SessionAuthMiddleware` shared with other services

**rook-cli:**
- `rook-cli/guides/` — guide loader, glamour renderer, lipgloss style loader, YAML action parser
- `rook-cli/guides/builder/` — guide builder TUI (new, edit, preview, validate, publish, manage)
- `rook-cli/guides/validator/` — YAML config schema validator, lipgloss style validator, markdown link checker
- `rook-cli/guides/client.go` — HTTP client for `GET /guides`, `GET /guide/{id}`, `POST /guide`, `PATCH`, `DELETE`
- `rook-cli/guides/store/store.go` — flat-file offline store: Save, Remove, Get, List, ListIDs; atomic write via temp dir + os.Rename
- `rook-cli/guides/sync/pull.go` — pull logic: compare synced_at vs server published_at, selective re-fetch, sync_state update
- `rook-cli/guides/tui/saved_list.go` — `rook guide saved` list TUI with sync-state icons
- `rook-cli/cmd/guide_save.go` — `rook guide save` command
- `rook-cli/cmd/guide_saved.go` — `rook guide saved` command
- `rook-cli/cmd/guide_remove.go` — `rook guide remove` command
- `rook-cli/cmd/guide_pull.go` — `rook guide pull` command

**rook-cli (modifications to existing files):**
- `rook-cli/guides/client.go` — add sentinel network-error type for offline fallback detection
- `rook-cli/guides/reader.go` — offline fallback path; render ⚠ Offline banner from local store
- `rook-cli/guides/tui/guide_list.go` — 📥 badge for saved guides
- `rook-cli/launcher/startup.go` — silent background guide pull goroutine on session start

### Dependencies

| Package | Used by | Purpose |
|---|---|---|
| `charmbracelet/glamour` | rook-cli | Markdown rendering with custom theming |
| `charmbracelet/lipgloss` | rook-cli | Per-guide style application |
| `charmbracelet/bubbletea` | rook-cli | Guide builder and reader TUI |
| `charmbracelet/bubbles` | rook-cli | List, viewport, textinput components |
| `gopkg.in/yaml.v3` | rook-cli, guides-service | YAML config and style parsing/validation |
| `google.com/cloud/firestore` | guides-service | Firestore read/write |
| `google.golang.org/grpc` | guides-service | gRPC client for user-service calls |

### Constraints

- Guide IDs are UUIDs generated by the CLI at creation time — the server must accept the CLI-provided ID, not generate its own
- The server must not partially write a guide bundle — Firestore writes for `meta` and all `assets/` documents must succeed atomically (use a Firestore batch write)
- `GET /guides` returns metadata only — asset documents are not read for the list view
- The CLI must never render a guide asset fetched from the server without first decoding and validating the asset locally — the server is trusted but the client must not blindly execute arbitrary `config.yml` actions
- Action types not recognised by the CLI must be silently ignored, not cause a crash — forward compatibility

### Do Not

- Do not store guide assets in Cloud Storage for the PoC — Firestore subcollections are sufficient and avoid a second GCP dependency
- Do not allow cross-space guide references — a guide is fully scoped to its space
- Do not allow multi-file (multi-page) guides in the PoC — `navigation.type: linear` with a single `content.md` is the supported model
- Do not allow guide content to embed external images — the TUI renderer does not support remote image fetch
- Do not expose the raw Firestore document structure in the HTTP API — always use the JSON bundle schema defined above

---

## Verification

- [ ] `POST /guide` with valid assets and a valid session token stores the guide in Firestore and returns `201` with the `guide_id`
- [ ] `POST /guide` with a malformed `config.yml` returns `422` with per-file validation errors; nothing is written to Firestore
- [ ] `POST /guide` with a group ID not present in the space returns `422`; nothing is written
- [ ] `GET /guides` returns only guides whose ACL includes the requesting user's group
- [ ] `GET /guide/{id}` for a guide the user has access to returns the full bundle with all three assets
- [ ] `GET /guide/{id}` for a guide the user does not have ACL access to returns `403`, even if the guide exists
- [ ] `PATCH /guide/{id}/access` by the guide owner updates the ACL; subsequent `GET /guides` reflects the change for affected groups
- [ ] `PATCH /guide/{id}/access` by a user who is not the owner and not a space admin returns `403`
- [ ] `DELETE /guide/{id}` by the owner removes the guide from Firestore; `GET /guide/{id}` returns `404`
- [ ] Guide builder: new guide scaffolds `content.md`, `style.yml`, and `config.yml` stubs in `<storage-dir>/guides/drafts/<uuid>/`
- [ ] Guide builder: preview renders the guide full-screen using local draft assets; exit returns to the builder
- [ ] Guide builder: validate reports per-file errors for a malformed `config.yml`; publish is blocked
- [ ] Guide builder: successful publish calls `POST /guide` and the guide appears in `GET /guides` for the owner's group
- [ ] Guide builder: unpublish calls `DELETE /guide/{id}`; the guide no longer appears in `GET /guides`
- [ ] Guide builder: re-publish after edit calls `POST /guide` with the same `guide_id`; the Firestore document is updated in place
- [ ] CLI reader: a `navigate` action in `config.yml` causes the launcher to navigate to the correct feature on guide exit
- [ ] CLI reader: an unknown action type in `config.yml` is silently ignored; the guide renders without error

---

## References

| Document | Path |
|---|---|
| System Architecture ADR | `specs/decisions/2026-04-25-rook-reference-system-architecture.md` |
| rook-cli Features and UX ADR | `specs/decisions/2026-04-25-rook-cli-features-and-ux-architecture.md` |
| gRPC Inter-Service Communication ADR | `specs/decisions/2026-04-25-grpc-inter-service-communication.md` |
| Auth and Cloud Run Topology ADR | `specs/decisions/2026-04-25-ssh-auth-identity-chain-and-cloud-run-topology.md` |
| Component Overview | `specs/architecture/component-overview.md` |
| gRPC Call Flows | `specs/architecture/grpc-call-flows.md` |
