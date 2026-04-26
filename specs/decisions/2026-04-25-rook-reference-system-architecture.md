---
status: proposed
date: 2026-04-25
decision-makers: Mona Maret
consulted: Mona Maret
informed: Mona Maret
---

# Rook Reference System Architecture

## Context and Problem Statement

The Rook reference project is a proof-of-concept implementation of a private social media server with offline sync capabilities. It must define a coherent architecture spanning: a cloud-hosted Go server, a local terminal-first CLI, offline-capable document management, real-time communication, and user identity management — while remaining approachable as a reference implementation for users, admins, and contributors.

What architecture, component boundaries, protocol choices, and infrastructure strategy should govern the Rook system?

## Decision Drivers

- Must be a self-contained proof-of-concept — not production-hardened
- Must support offline-first document workflows in the CLI with lazy sync to the server
- Must support multi-user document sharing with per-file permissions
- Must support async messaging (1:1 and 1:many) via a custom pull-based protocol — see [`2026-04-25-real-time-messaging-protocol.md`](2026-04-25-real-time-messaging-protocol.md)
- Must be extensible — the initial service set must not foreclose adding new services later
- Architecture must remain viable if the deployment target changes from Cloud Run to Kubernetes
- All document content must be written and stored in human-readable formats (markdown, JSON)

## Considered Options

- REST/HTTP vs. gRPC for inter-service communication — resolved: gRPC with Protobuf (see resolved question 1)
- Static site renderer for `rook-docs` — resolved: Hugo + Hextra (see resolved question 2)
- Database engine per service — resolved: Google Cloud Firestore per service (see resolved question 3)

## Decision Outcome

Chosen architecture: **three-component system** — `rook-server` (Go microservices), `rook-cli` (Go TUI), and `rook-docs` (static site) — because it cleanly separates runtime concerns, allows the CLI to work offline, and keeps the reference implementation focused and demonstrable.

### System Components

#### rook-server

- **Language**: Go
- **Deployment**: Google Cloud Run (initial target); architecture must remain portable to Kubernetes
- **Architecture**: Service-oriented — each feature area is a discrete Go binary deployed as its own service. The initial service set is not exhaustive; the architecture must not foreclose adding new services.
- **Initial services**:
  - **Messaging service** — async 1:1 and 1:many conversations (private messages are 1:1 conversations; not a separate service) via custom pull-based HTTP protocol (see [`2026-04-25-real-time-messaging-protocol.md`](2026-04-25-real-time-messaging-protocol.md))
  - **Guides service** — space-scoped TUI micro-apps (tutorials, explainers, admin notices) accessible via wishlist; replaces "bulletin board" (see [`2026-04-25-rook-cli-features-and-ux-architecture.md`](2026-04-25-rook-cli-features-and-ux-architecture.md))
  - **Document/stash service** — store, retrieve, and permission-control markdown documents
  - **User management service** — manages user records, spaces, groups, and per-app ACL in Firestore. `charmbracelet/wish` handles the SSH handshake and key validation at the server boundary; it does not manage or store keys. Key generation happens locally in `rook-cli`; key registration is performed by a server admin via `rook-server-cli` (see [`2026-04-25-rook-server-admin-cli.md`](2026-04-25-rook-server-admin-cli.md)).
- **Inter-service communication**: **gRPC with Protobuf** — unary RPCs only; Go stubs generated via `buf`; OIDC bearer tokens for Cloud Run service-to-service auth. See [`2026-04-25-grpc-inter-service-communication.md`](2026-04-25-grpc-inter-service-communication.md).
- **Data layer**: Each service owns its own **Google Cloud Firestore** collection namespace. No shared collections across services. Go client: `cloud.google.com/go/firestore`. Kubernetes portability of the data layer is explicitly a non-goal at this deployment scale.
- **Auth**: SSH handshake and key validation via `charmbracelet/wish` (SSH server middleware); service/app routing via `charmbracelet/wishlist`. Key generation is local to `rook-cli`; key storage is in Firestore (`users/{id}/keys/`); key registration is an admin operation via `rook-server-cli`.
- **Architecture diagrams**: component relationship overview (all three binaries, personas, protocols, data boundaries) — see [`../architecture/component-overview.md`](../architecture/component-overview.md); per-service detail, service responsibilities, Firestore layout, and gRPC call flows — see [`../architecture/grpc-call-flows.md`](../architecture/grpc-call-flows.md)

#### rook-cli

- **Language**: Go
- **TUI framework**: `charmbracelet/bubbletea` (Elm architecture — `Model`/`Update`/`View`); runs entirely locally
- **Architecture**: Hybrid — local-first launcher model; server features sync lazily on user initiative. Each major feature is a full-screen Bubble Tea app that launches from and returns to the launcher.
- **Navigation**: Launcher home screen with server reachability indicators, local feature shortcuts (always available), connected feature shortcuts (greyed/hidden when offline), and a space selector (shown only when the user belongs to multiple spaces). Features open full-screen; exit always returns to launcher.
- **Auth**: `rook ssh user@server` subcommand; `charmbracelet/wish` for SSH key auth, `charmbracelet/wishlist` for space-scoped, ACL-filtered app discovery. Auth state is session-scoped; re-auth is required on next launch.
- **Spaces, groups, and ACL**: A server hosts one or more spaces; a user may belong to multiple spaces. All data is fully segregated by space — no cross-space access. Within a space, users are assigned to a group; each group has a per-app ACL controlling which wishlist apps are accessible. Space membership and ACL are server-authoritative, cached locally in `.json`.
- **Local storage**: Flat files — `.md` for content, `.json` for metadata; no embedded database. Config at `$XDG_CONFIG_HOME/rook/config.json`; flat-file data root at `$XDG_CONFIG_HOME/rook/storage/` by default (fully relocatable via `storage-dir` config key). Space data is always under `<storage-dir>/<feature>/<space-id>/`.
- **Features**: Document stash (offline create/edit/view/browse; online sync and share), unified local+server search, async pull-only messaging, and space-scoped guides. A dedicated guide builder TUI enables guide authoring, preview, validation, and publish entirely within `rook-cli` — no separate admin tooling required.
- **Sync conflict resolution**: Last-write-wins. Concurrent editing is not an expected use case.
- **Distribution**: Build from source. macOS and Linux run natively; Windows users run under WSL. No packaged binary distribution for the reference implementation.
- **Excluded dependency**: `charmbracelet/charm` is archived (March 2025) and must not be used. Server-side user management and file store use Google Cloud Firestore; CLI-side storage is plain flat files.
- **Full feature specification**: see [`2026-04-25-rook-cli-features-and-ux-architecture.md`](2026-04-25-rook-cli-features-and-ux-architecture.md)

#### rook-docs

- **Purpose**: Documentation site targeting three personas — users, admins, and contributors
- **Content format**: Markdown (content must be writable and readable without a static site renderer)
- **Renderer**: **Hugo** with the **[Hextra](https://github.com/imfing/hextra)** theme. Content-renderer separation is a hard constraint — all docs content must remain readable as plain markdown without running Hugo.

#### rook-server-cli

- **Purpose**: Server admin tooling — a standalone Go binary for operations that require direct authority over `user-service`: SSH key registration, user management, and space/group administration.
- **Language**: Go
- **Auth**: Pre-shared admin token (`ROOK_ADMIN_TOKEN` env var); attached as a gRPC metadata bearer token on every call. Validated by a unary interceptor on `user-service` before any `AdminService` RPC handler runs.
- **Interface**: gRPC to `user-service` via a dedicated `AdminService` proto (`rook-server/proto/admin/v1/admin.proto`). Admin RPCs are never exposed to `rook-cli` or end users.
- **Initial commands**: `user register-key`, `user list`, `user add-to-space`, `user remove-from-space`, `user set-group`, `space create`, `space list`, `space members`
- **Distribution**: Build from source (`go build` in `rook-server/admin-cli/`). Run by the server admin from any machine with network access to `user-service`.
- **Full specification**: see [`2026-04-25-rook-server-admin-cli.md`](2026-04-25-rook-server-admin-cli.md)

### Consequences

- Good, because the CLI is fully usable offline for core document workflows
- Good, because Cloud Run scales to zero, minimizing cost for a PoC deployment
- Good, because the Charmbracelet ecosystem provides consistent SSH auth, TUI widgets, and service routing
- Good, because the custom async messaging protocol is designed around the actual CLI usage model — pull-only, offline-first, no persistent connections
- Good, because per-service data ownership prevents tight coupling and allows independent evolution
- Good, because flat-file local storage is transparent and inspectable without tooling
- Good, because markdown-first docs content is portable regardless of renderer choice
- Bad, because multiple Cloud Run services increases operational complexity vs. a monolith
- Bad, because hybrid online/offline sync (even with last-write-wins) requires careful state tracking in the CLI

## Implementation Plan

*To be detailed per-service as specs are written. The following captures system-wide patterns.*

- **Affected paths**:
  - `rook-server/` — one `go.mod` per service binary; each service is an independent Go module under `rook-server/<service-name>/`
  - `rook-server/admin-cli/` — `rook-server-cli` binary; one `go.mod`; talks gRPC to `user-service` via `AdminService`
  - `rook-server/proto/admin/v1/admin.proto` — `AdminService` definition; generated stubs in `rook-server/gen/go/admin/v1/`
  - `rook-cli/` — single Go module, Bubble Tea application
  - `rook-docs/` — Hugo site with Hextra theme; markdown content under `rook-docs/content/`
- **Dependencies**:
  - Server: `charmbracelet/wish`, `charmbracelet/wishlist`, `google.golang.org/grpc`, `cloud.google.com/go/firestore`
  - CLI: `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss`, `charmbracelet/glamour`, `charmbracelet/glow`, `charmbracelet/wish`, `charmbracelet/wishlist`
  - Inter-service: `google.golang.org/grpc` + `google.golang.org/protobuf` (see [`2026-04-25-grpc-inter-service-communication.md`](2026-04-25-grpc-inter-service-communication.md))
  - Docs: Hugo + Hextra theme (`github.com/imfing/hextra`)
- **Patterns to follow**:
  - Each `rook-server` service is an independent Go binary with its own `main.go` and data store
  - CLI uses Bubble Tea `Model`/`Update`/`View` for all interactive screens
  - Online and offline logic are separated into distinct Bubble Tea models, composed at the top level
  - Auth always flows through `charmbracelet/wish` — no bespoke auth mechanisms
  - Service endpoints are always environment-variable-driven — no hardcoded URLs
  - Local CLI state is always flat files (`.md` + `.json`) — no embedded DB for the reference impl
- **Patterns to avoid**:
  - Do not mix online and offline logic in the same Bubble Tea model
  - Do not hardcode Cloud Run service URLs
  - Do not share a database across services
  - Do not introduce packaging or distribution tooling — users build from source
- **Configuration**:
  - Server services: URLs via environment variables (one per service); no hardcoded Cloud Run URLs; `ROOK_ADMIN_TOKEN` on `user-service` for admin RPC auth
  - `rook-server-cli`: `USER_SERVICE_ADDR` (gRPC target), `ROOK_ADMIN_TOKEN` (must match `user-service`)
  - CLI (`$XDG_CONFIG_HOME/rook/config.json`): SSH key path, configured server addresses, default space per server, `storage-dir` (flat-file storage root; defaults to `$XDG_CONFIG_HOME/rook/storage/`)
- **Migration steps**: N/A — greenfield

### Verification

- [ ] Each `rook-server` service builds and starts independently (`go build ./...` in each service directory)
- [ ] `rook-server-cli` builds from source with `go build` in `rook-server/admin-cli/`
- [ ] `rook-server-cli user register-key` registers a public key via `user-service`; user can subsequently authenticate with `rook ssh`
- [ ] `AdminService` RPCs on `user-service` reject calls with a missing or incorrect `ROOK_ADMIN_TOKEN` with gRPC `UNAUTHENTICATED`
- [ ] `rook-cli` builds from source with `go build` on macOS and Linux; builds and runs correctly under WSL on Windows
- [ ] CLI config is written to `$XDG_CONFIG_HOME/rook/config.json`; flat-file data is written under `storage-dir`, not the home directory root
- [ ] `storage-dir` override: setting a custom path in config redirects all flat-file writes to that location
- [ ] CLI document stash creates, reads, and edits a `.md` file offline — no server required
- [ ] CLI syncs a local document to the server and the server reflects the update
- [ ] Last-write-wins: editing a document locally then syncing overwrites the server version
- [ ] User auth via SSH key succeeds end-to-end through `charmbracelet/wish`
- [ ] Space-filtered wishlist: user sees only apps their group's ACL permits in their current space
- [ ] User with multiple spaces sees the space selector on the launcher home screen; single-space user does not
- [ ] Space data segregation: data under `<storage-dir>/<feature>/<space-a>/` is never accessible from a space-b context
- [ ] User A cannot access User B's private document via the document service
- [ ] Guide builder: publish uploads assets to the guides service; guide appears in space wishlist for permitted groups
- [ ] `charmbracelet/charm` is not imported anywhere in the codebase
- [ ] `rook-docs` content is readable as plain markdown without running any renderer
- [ ] All service endpoint URLs are read from environment variables — no hardcoded values in source

## Pros and Cons of the Options

### REST/HTTP for inter-service communication

- Good, because it is natively supported by Cloud Run and Kubernetes with no extra tooling
- Good, because it is simple to debug (curl, standard HTTP tooling)
- Good, because Go's `net/http` stdlib is sufficient — no additional dependency
- Bad, because no enforced schema contract between services (must use OpenAPI or similar to compensate)

### gRPC for inter-service communication

- Good, because Protobuf schemas enforce a typed contract between services
- Good, because efficient binary serialization is appropriate for high-frequency internal calls
- Good, because gRPC is well-supported in Kubernetes service meshes
- Bad, because it adds a code-generation step (`protoc`) and a non-trivial dependency
- Bad, because local development and debugging is more complex than plain HTTP

## More Information

### Resolved Questions

1. **Inter-service communication protocol** ✅ Resolved — **gRPC with Protobuf**. See [`2026-04-25-grpc-inter-service-communication.md`](2026-04-25-grpc-inter-service-communication.md). All inter-service calls use generated Go stubs via `buf`; unary RPCs only; OIDC bearer tokens for Cloud Run service-to-service auth; no raw HTTP between services. Call flow diagrams and the `UserService` RPC reference: [`../architecture/grpc-call-flows.md`](../architecture/grpc-call-flows.md).

2. **Static site renderer for `rook-docs`** ✅ Resolved — **Hugo** with the **[Hextra](https://github.com/imfing/hextra)** theme. Hugo is fast, widely used, and has no runtime dependency. Hextra provides a docs-optimised layout with sidebar navigation and persona-based content organisation out of the box. Hard constraint upheld: all content remains readable as plain markdown.

3. **Per-service database engine** ✅ Resolved — **Google Cloud Firestore** (`cloud.google.com/go/firestore`). Each service owns its own collection namespace. No shared collections. See messaging protocol ADR deferred decision 2 for full rationale.

4. **`rook-server` module structure** ✅ Resolved — **one `go.mod` per service binary**, each under `rook-server/<service-name>/`. Services share no module-level code; shared types (generated Protobuf stubs) live in `rook-server/gen/` as a separate module imported by each service.

5. **`charmbracelet/charm` dependency** ✅ Resolved — **do not use**. `charmbracelet/charm` is archived (March 2025). Server-side file store and user management use **Google Cloud Firestore** (`cloud.google.com/go/firestore`). CLI-side storage is plain flat files. No charm dependency is required anywhere in the codebase.

6. **Guide distribution model** ✅ Resolved — Guides are authored and published entirely within `rook-cli` via a dedicated **guide builder TUI**. No separate admin tooling or server-side filesystem access is required. The builder scaffolds local draft assets (`.md`, lipgloss `.yml`, YAML config) under `<storage-dir>/guides/drafts/<guide-id>/`, provides `$EDITOR` handoff for authoring, full-screen preview, inline validation, and a publish command that uploads validated assets to the guides service on `rook-server`. Publish is blocked until validation passes. Local drafts are retained as the editable source of truth after publish.

7. **Space selector UX** ✅ Resolved — The space selector is embedded in the launcher home screen and shown **only when the user belongs to multiple spaces** on a connected server. Single-space users see no selector. Switching spaces re-scopes the wishlist and all connected features to the selected space.

8. **Local config path** ✅ Resolved — CLI config is stored at `$XDG_CONFIG_HOME/rook/config.json`. Flat-file storage is under `$XDG_CONFIG_HOME/rook/storage/` by default, with a `storage-dir` config key allowing full relocation. No data is written to the home directory root. XDG conventions apply on macOS, Linux, and WSL.

9. **SSH key registration model** ✅ Resolved — Key generation is local to `rook-cli` (first-run setup flow). Key storage is in Firestore (`users/{id}/keys/` subcollection), owned by `user-service`. `charmbracelet/wish` handles SSH handshake and key validation only — it does not manage or store keys. Registration is an explicit admin operation: the user provides their public key to a server admin out-of-band; the admin registers it using `rook-server-cli user register-key`. See [`2026-04-25-rook-server-admin-cli.md`](2026-04-25-rook-server-admin-cli.md) for the full admin CLI specification and key registration call flow.

### Non-Goals (Explicit — PoC Scope Only)

- No web UI
- No mobile clients
- No federation with external servers
- No production hardening (rate limiting, HA, advanced observability) beyond what is needed to demonstrate the concept
- No packaged binary distribution (Homebrew, apt, etc.) — users build from source; Windows users use WSL
- No multi-tenancy beyond the user permission model on documents
