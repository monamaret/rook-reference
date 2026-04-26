---
status: proposed
date: 2026-04-25
decision-makers: Mona Maret
consulted: Mona Maret
informed: Mona Maret
---

# rook-cli Features and UX Architecture

## Context and Problem Statement

The system architecture ADR ([`2026-04-25-rook-reference-system-architecture.md`](2026-04-25-rook-reference-system-architecture.md)) establishes `rook-cli` as a local-first Go binary with a Bubble Tea TUI that syncs lazily with `rook-server`. This ADR defines the full feature set, UX model, navigation architecture, and rendering strategy for `rook-cli` in enough detail for implementation to begin.

What are the features, UX flows, component boundaries, and rendering approach for `rook-cli`?

## Decision Drivers

- The TUI must run entirely locally — the user may be offline for most of their usage
- The CLI must be useful before any server connection is established
- Navigation must be simple enough for a solo developer to implement as a PoC
- All local content must be stored as human-readable flat files (`.md` + `.json`)
- The Charmbracelet ecosystem is the canonical toolkit — prefer its libraries over custom implementations
- Server connectivity is ambient and advisory, never a blocker for local workflows
- Spaces are the primary organizational boundary — all server-side data is segregated by space with no cross-space access
- The architecture must not foreclose adding new guides or services to the wishlist without CLI changes

## Considered Options

- **Launcher model** — home screen is a root launcher; apps take over the terminal while active and return to launcher on exit
- **Persistent shell model** — a status bar or split-pane is always visible while apps run in a content area
- **Full-screen per-app model** — each feature is a fully independent binary invoked by the CLI

## Decision Outcome

Chosen model: **Launcher**, because it is the simplest model to implement correctly in Bubble Tea, gives each feature full terminal real estate, and avoids the complexity of persistent layout management across app boundaries.

---

## Feature Specifications

### 1. First-Run Setup Flow

Triggered when no local config is detected. A Bubble Tea interactive flow that:

- Checks for an existing SSH key; if none, generates one and displays the public key for the user to register with a server admin
- Prompts for one or more `rook-server` addresses to configure
- Prompts for the flat-file storage directory (default: `$XDG_CONFIG_HOME/rook/storage/`; fully configurable via `storage-dir` in config)
- Writes config to `$XDG_CONFIG_HOME/rook/config.json`
- On completion, drops the user into the launcher home screen

The setup flow is a first-class Bubble Tea experience — not a shell script or prompt sequence.

**Directory layout (defaults):**
```
$XDG_CONFIG_HOME/rook/
├── config.json          # application config
└── storage/             # flat-file data root (configurable via storage-dir)
    ├── stash/<space-id>/
    ├── messages/<space-id>/<conversation-id>/
    └── cache/           # wishlist ACL cache, space membership
```

**Platform notes:**
- **macOS**: `$XDG_CONFIG_HOME` defaults to `~/.config`; storage at `~/.config/rook/storage/`
- **Linux**: same XDG defaults apply
- **Windows**: users run `rook-cli` under WSL; XDG defaults apply within the WSL environment, giving `~/.config/rook/` in the WSL home. No container or special Windows build is required.

### 2. Launcher Home Screen

The root of the CLI. Always the entry and exit point for all other features.

**Layout:**
- Server status pane: one row per configured server, showing reachability as an ambient indicator (e.g., 🟢 / 🔴 emoji or lipgloss-styled badge). Status is updated via a short-lived poll at launch and on manual refresh — not continuous background polling.
- Local feature shortcuts: document stash, search (always available offline)
- Connected feature shortcuts: messaging, guides (greyed out or hidden when server unreachable)
- Space selector: visible **only** when the user belongs to multiple spaces on a connected server — shows one row per space with an emoji indicator for the currently active space; hidden entirely for single-space users

**Behaviour:**
- Selecting a local feature opens it full-screen; exit returns to launcher
- Selecting a server feature initiates the `rook auth user@server` challenge-response flow if not already authenticated for that session, then opens the feature full-screen
- Server reachability indicators do not block launcher interaction — the user can open local features immediately

### 3. Authentication and Server Handshake

Subcommand: `rook auth user@server`

- Uses HTTPS challenge-response with the user's SSH private key: fetches a nonce from `GET /auth/challenge`, signs it locally, submits to `POST /auth/verify`, receives an opaque session token
- Session token is cached in-memory for the duration of the CLI session and attached as `Authorization: Bearer` on all subsequent HTTPS requests; re-auth is required on next launch
- App/space discovery via authenticated HTTP endpoints on `user-service`: `GET /spaces` and `GET /spaces/{space-id}/apps`; response is space-scoped and group-ACL-filtered — the user sees only apps their group has permission to access
- All requests include `X-Rook-Space-ID` header so the server can resolve space membership alongside identity in a single call
- If the user belongs to multiple spaces on a server, the launcher home screen shows the space selector with an emoji indicator for the active space; switching spaces re-scopes app discovery and all connected features to the selected space
- See [`2026-04-25-ssh-auth-identity-chain-and-cloud-run-topology.md`](2026-04-25-ssh-auth-identity-chain-and-cloud-run-topology.md) for the full auth protocol specification

### 4. Spaces, Groups, and Access Control

- A `rook-server` hosts one or more **spaces**
- A user may belong to multiple spaces; membership is managed by a space admin
- All data within a space is fully segregated — no cross-space data access of any kind
- Within a space, users are assigned to a **group** at onboarding time by the admin
- Each group has a per-app ACL that determines which wishlist apps are visible and accessible
- The wishlist a user sees = apps available in their space ∩ apps their group's ACL permits
- Space membership and ACL are server-authoritative — the CLI reads them at sync time and caches them locally in `.json`

### 5. Document Stash

The primary offline feature. Available without any server connection.

- **Create/edit**: opens the user's `$EDITOR` (respects the `EDITOR` / `VISUAL` environment variable); TUI suspends while editor is active, resumes on exit
- **View**: renders markdown via `charmbracelet/glow`; inline in the TUI, not a separate process
- **Browse**: file list view with metadata (title, last modified, sync state, tags if present in frontmatter)
- **Sync**: user-initiated; pushes/pulls changes to/from the server document stash service; last-write-wins on conflict
- **Share**: when synced to a server, a document can be shared with other users in the same space via permission entry in its `.json` metadata
- **Local storage**: `<storage-dir>/stash/<space-id>/` — one directory per space, flat `.md` + `.json` pairs (default: `$XDG_CONFIG_HOME/rook/storage/stash/<space-id>/`)
- **Offline**: all create/edit/view/browse operations work with no server connection; sync and share require connection

### 6. Search

Available in two modes depending on connectivity:

- **Local search** (always available): full-text search across local flat-file stash using filename, frontmatter, and content; results rendered with match highlighting
- **Server search** (when connected): extends search to server-synced documents the user has access to within their current space, including documents shared with them by other users
- Search is unified — a single search view shows local and server results together, with provenance indicated (local / server / shared)
- Server-side search queries Firestore collections scoped to the user's current space and documents shared with them

### 7. Messaging

Async, pull-only. Full protocol model in [`2026-04-25-real-time-messaging-protocol.md`](2026-04-25-real-time-messaging-protocol.md).

**CLI features:**
- **Conversation list**: shows all conversations (1:1 and 1:many) the user participates in, with unread indicators derived from last sync metadata
- **Thread view**: renders conversation history from local flat files using `charmbracelet/glow` for markdown content
- **Compose**: opens `$EDITOR` for message composition; on save, message is queued for send on next sync
- **Sync**: user-initiated from the conversation list or thread view; pulls new messages, pushes queued outbound messages
- **New conversation**: user selects participants from a space member list; participant set is fixed at creation — users cannot be added to an existing conversation
- **Notification indicators**: unread counts shown in the launcher home screen after a sync, without entering the messaging view
- **Local storage**: `<storage-dir>/messages/<space-id>/<conversation-id>/` — `.md` for content, `.json` for metadata (participants, timestamps, sync state) (default: `$XDG_CONFIG_HOME/rook/storage/messages/`)
- **Sync modes**: `sync`, `no-sync`, `selective-unsync`, `admin-silent-sync` — per-conversation, user-configurable

### 8. Guides

Wishlist-accessible TUI micro-apps. Each guide is a self-contained Bubble Tea application with its own:

- **Markdown content** (`.md`) — rendered via `charmbracelet/glamour` with custom styling
- **Lipgloss style files** (`.yml`) — per-guide custom theming via `charmbracelet/lipgloss`
- **YAML config** (`.yml`) — defines interactive elements (buttons, navigation, actions) beyond static content
- **Ownership**: any authenticated space user can create a guide; the creator has admin rights over it (edit, delete, style, configure). Server-owned guides are created and managed by the space admin account.
- **Access**: controlled by space group ACL — a guide appears in wishlist only for groups that have access
- **CLI behaviour**: selecting a guide from the wishlist launches it full-screen; exit returns to the launcher

Guides are intended for tutorials, explainers, reference documentation, onboarding flows, and admin notices — persistent, styled, interactive documents rather than ephemeral messages.

#### Guide Builder

Guides are authored entirely within `rook-cli` via a dedicated guide builder TUI, accessible from the launcher. The builder manages the full guide lifecycle:

**Authoring:**
- **New guide**: prompts for guide name and description; scaffolds a local guide directory at `<storage-dir>/guides/drafts/<guide-id>/` with stub `.md`, lipgloss `.yml`, and YAML config files
- **Edit content**: opens the guide's `.md` file(s) in `$EDITOR`; resumes builder on exit
- **Edit styles**: opens the lipgloss `.yml` file in `$EDITOR`
- **Edit config**: opens the YAML config in `$EDITOR`
- **Preview**: renders the guide full-screen using the local draft assets — exactly as it will appear to users after publish; exit returns to builder
- **Local draft storage**: `<storage-dir>/guides/drafts/<guide-id>/` — all assets as flat files until published

**Validation (run automatically on publish, available on demand):**
- YAML config is parsed and validated against the guide config schema — missing required fields, unknown keys, and malformed action references are reported as errors
- Lipgloss style file is parsed and validated — invalid property names or values are flagged
- Markdown content is checked for broken internal links (references to other guide pages or sections)
- Validation results are shown inline in the builder TUI with per-file error detail; publish is blocked until all errors are resolved

**Publish:**
- Runs validation; if any errors exist, publish is blocked and errors are displayed
- On success, packages the guide assets and uploads to the guides service on `rook-server`
- Guide becomes visible in the wishlist for groups the creator has granted access to
- Published guide assets are stored in Firestore (metadata + content) by the guides service
- After publish, the local draft is retained as the editable source of truth; subsequent edits and re-publishes follow the same flow

**Manage (for guide owners):**
- **Unpublish**: removes the guide from the wishlist and Firestore; local draft is preserved
- **Update access**: configure which space groups can see the guide in their wishlist
- **Delete**: removes the guide from Firestore and optionally the local draft

#### Offline Reading

Users can save published guides for offline reading using an explicit save command. Saved guides are stored in `<storage-dir>/guides/saved/<space-id>/<guide-id>/` as flat files (same convention as the stash and messages stores).

- **Save**: `rook guide save {id|slug}` — downloads the full guide bundle and writes it to the offline store
- **List saved**: `rook guide saved` — list TUI showing all locally saved guides with sync-state icons (`✓ synced`, `↑ stale`, `? unavailable`)
- **Remove**: `rook guide remove {id|slug}` — deletes the guide from the offline store (with confirmation prompt)
- **Pull**: `rook guide pull [id|slug]` — re-fetches saved guides from the server if a newer version exists; sets `unavailable` on error without losing local copy
- **Offline read fallback**: `rook guide read` checks the local store when the server is unreachable; renders with an `⚠ Offline` notice; shows a helpful error if the guide is not saved
- **List badge**: `rook guide list` annotates saved guides with a `📥` badge
- **Background pull**: on launcher startup, a silent goroutine re-fetches all saved guides for the current space if newer server versions exist; failures are logged only, never shown in the UI
- **Sync state**: `synced` | `stale` | `unavailable` — mirrors the stash sync-state model

### 9. Rendering and Styling Stack

| Purpose | Library |
|---------|---------|
| Markdown viewing (stash, messages) | `charmbracelet/glow` |
| Markdown rendering with custom themes (guides) | `charmbracelet/glamour` |
| TUI component styling (layout, colour, borders) | `charmbracelet/lipgloss` |
| TUI component primitives (lists, inputs, viewports) | `charmbracelet/bubbles` |
| TUI application framework | `charmbracelet/bubbletea` |
| SSH key loading and request signing (HTTPS challenge-response auth) | `golang.org/x/crypto/ssh` |

---

## Implementation Plan

- **Affected paths**:
  - `rook-cli/main.go` — entry point; detects first-run, launches setup flow or launcher
  - `rook-cli/setup/` — first-run Bubble Tea setup flow
  - `rook-cli/launcher/` — home screen model (server status, shortcuts, space selector)
  - `rook-cli/auth/` — HTTPS challenge-response flow, SSH key signing, session token cache
  - `rook-cli/stash/` — document stash: browse, view (glow), edit ($EDITOR handoff), sync
  - `rook-cli/search/` — unified local + server search
  - `rook-cli/messaging/` — conversation list, thread view, compose, sync
  - `rook-cli/guides/` — guide loader, glamour renderer, lipgloss style loader, YAML action parser
  - `rook-cli/guides/builder/` — guide builder TUI (new, edit, preview, validate, publish, manage)
  - `rook-cli/guides/validator/` — YAML config schema validator, lipgloss style validator, markdown link checker
  - `rook-cli/spaces/` — space selector, HTTP-based space/app discovery, membership and ACL cache
  - `rook-cli/config/` — local config read/write (`$XDG_CONFIG_HOME/rook/config.json`)
- **Dependencies**:
  - `charmbracelet/bubbletea` — TUI framework
  - `charmbracelet/bubbles` — list, textarea, viewport, spinner, textinput components
  - `charmbracelet/lipgloss` — styling
  - `charmbracelet/glamour` — markdown rendering for guides
  - `charmbracelet/glow` — markdown viewing for stash and messages
  - `golang.org/x/crypto/ssh` — SSH key loading and signing for HTTPS challenge-response auth
- **Patterns to follow**:
  - Each major feature area is its own Bubble Tea `Model` with its own `Update`/`View`; the launcher composes them
  - Online and offline logic are strictly separated within each feature model
  - `$EDITOR` handoff uses `tea.ExecProcess` to suspend the TUI, launch the editor, and resume cleanly
  - All local state is flat files under `<storage-dir>/`; no embedded database
  - Space directory structure is always `<storage-dir>/<feature>/<space-id>/` to ensure segregation
  - Config file is always at `$XDG_CONFIG_HOME/rook/config.json`; storage root is always read from `storage-dir` in config
  - Space and app list is fetched from `GET /spaces` and `GET /spaces/{space-id}/apps` after each successful auth, cached locally in `<storage-dir>/cache/spaces.json`, and refreshed on each new session
- **Patterns to avoid**:
  - Do not background-poll servers — all network activity is user- or event-initiated
  - Do not share state between spaces in any local data structure
  - Do not embed a markdown editor in the TUI — always delegate to `$EDITOR`
  - Do not hardcode server addresses or space IDs — always read from config
  - Do not mix guide rendering logic with stash/message rendering — guides have their own style pipeline
- **Configuration** (`$XDG_CONFIG_HOME/rook/config.json`):
  - SSH key path
  - Configured server addresses (one or more)
  - Default space per server (if user belongs to multiple)
  - `storage-dir` — root path for all flat-file storage (default: `$XDG_CONFIG_HOME/rook/storage/`; fully configurable)
- **Migration steps**: N/A — greenfield

### Verification

- [ ] First-run setup flow: detects missing config, generates SSH key, prompts for server address and storage directory, writes `$XDG_CONFIG_HOME/rook/config.json`, drops to launcher
- [ ] Config is written to `$XDG_CONFIG_HOME/rook/config.json`; flat files are written under `storage-dir`, not the home directory
- [ ] `storage-dir` override: setting a custom path in config redirects all flat-file writes to that location
- [ ] Binary builds and runs from source with `go build` on macOS and Linux
- [ ] Binary builds and runs correctly under WSL on Windows; XDG paths resolve within the WSL environment
- [ ] Launcher displays server reachability indicators (🟢/🔴) for each configured server on open
- [ ] Launcher is fully interactive while server status check is pending — no blocking
- [ ] `rook auth user@server` completes the HTTPS challenge-response flow and the CLI receives a session token; subsequent requests to all services succeed with that token
- [ ] User with multiple spaces sees a space selector with emoji active indicator on the launcher home screen; switching space re-scopes wishlist and connected features
- [ ] User with a single space sees no space selector on the launcher home screen
- [ ] Document stash: create, browse, and view a `.md` file with no server connection
- [ ] Document stash: `$EDITOR` opens on edit, TUI resumes cleanly on exit
- [ ] Document stash: sync pushes a local document to the server; last-write-wins on conflict
- [ ] Document stash: shared document is accessible to another user in the same space
- [ ] Search: local search returns results from `<storage-dir>/stash/` with match highlighting
- [ ] Search: server search returns shared documents from the current space when connected
- [ ] Messaging: conversation list shows unread counts after sync without entering a thread
- [ ] Messaging: compose opens `$EDITOR`; saved message is sent on next sync
- [ ] Messaging: new conversation with fixed participant set is created; participants cannot be added post-creation
- [ ] Messaging: `no-sync` mode message never leaves the local store
- [ ] Guides: selecting a guide from wishlist launches it full-screen with correct glamour styling
- [ ] Guides: exiting a guide returns to the launcher home screen
- [ ] Guide builder: new guide scaffolds `.md`, lipgloss `.yml`, and YAML config stubs in `<storage-dir>/guides/drafts/<guide-id>/`
- [ ] Guide builder: edit content opens `$EDITOR`; builder resumes cleanly on exit
- [ ] Guide builder: preview renders the guide full-screen using local draft assets; exit returns to builder
- [ ] Guide builder: validation catches malformed YAML config and reports per-field errors before publish
- [ ] Guide builder: validation catches invalid lipgloss style properties and reports them before publish
- [ ] Guide builder: publish is blocked when validation errors are present
- [ ] Guide builder: successful publish uploads guide assets to the guides service and guide appears in space wishlist
- [ ] Guide builder: unpublish removes the guide from Firestore and wishlist; local draft is preserved
- [ ] Guide builder: local draft persists after publish as the editable source of truth for future updates
- [ ] Space segregation: data in `<storage-dir>/<feature>/<space-a>/` is never accessible from space-b context
- [ ] All server addresses and space IDs are read from `$XDG_CONFIG_HOME/rook/config.json` — no hardcoded values

## More Information

### Interaction with other ADRs

- **Refines**: [`2026-04-25-rook-reference-system-architecture.md`](2026-04-25-rook-reference-system-architecture.md) — this ADR supersedes the `rook-cli` section of the system architecture ADR in detail. The system architecture ADR should be updated to: (1) remove "private message service" as a separate service (collapsed into messaging), (2) rename "bulletin board service" to "guides service".
- **Depends on**: [`2026-04-25-real-time-messaging-protocol.md`](2026-04-25-real-time-messaging-protocol.md) — messaging sync model, local flat-file format, and sync modes.
- **Server-side search**: unblocked — storage decision resolved (Firestore). Server-side search queries Firestore collections scoped to the user's space and shared document permissions.

### Deferred Decisions

**1. Local config path** ✅ Resolved
Config is stored at `$XDG_CONFIG_HOME/rook/config.json`. Flat-file storage is under `$XDG_CONFIG_HOME/rook/storage/` by default, with a `storage-dir` config key allowing full relocation. No data is written to the home directory root. XDG was chosen over `~/.rook/` to correctly separate config from data and follow platform conventions on all supported environments (macOS, Linux, WSL).

**2. `charmbracelet/charm`, `wish`, and `wishlist` evaluation** ✅ Resolved
`charmbracelet/charm` is archived (March 2025) and must not be used as a backend primitive. `charmbracelet/wish` and `charmbracelet/wishlist` are not used anywhere in the system — server auth and app discovery are handled over HTTPS (see [`2026-04-25-ssh-auth-identity-chain-and-cloud-run-topology.md`](2026-04-25-ssh-auth-identity-chain-and-cloud-run-topology.md)). Server-side file store and user management are implemented using **Google Cloud Firestore** (`cloud.google.com/go/firestore`). The CLI-side file store remains plain flat files as specified.

**3. Guide distribution** ✅ Resolved
Guides are authored and published entirely within `rook-cli` via a dedicated guide builder TUI. No separate admin tooling or server-side filesystem access is required. The builder scaffolds local draft assets (`.md`, lipgloss `.yml`, YAML config) in `<storage-dir>/guides/drafts/<guide-id>/`, provides `$EDITOR` handoff for authoring, a full-screen preview, inline validation (YAML schema, lipgloss properties, markdown links), and a publish command that uploads validated assets to the guides service on `rook-server`. Publish is blocked until validation passes. Local drafts are retained as the editable source of truth after publish.

**4. Space selector UX** ✅ Resolved
The space selector is embedded directly in the launcher home screen and is **only shown when the user belongs to multiple spaces** on a connected server — one row per space, with an emoji indicator identifying the currently active space. Single-space users see no selector at all; no unnecessary chrome is introduced. Switching spaces from the launcher re-scopes the wishlist and all connected features to the selected space.
