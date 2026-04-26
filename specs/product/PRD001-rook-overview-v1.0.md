# Rook — Project Overview for PoC

**ID:** PRD001  
**Version:** v1.0 (PoC complete)  
**Status:** Draft  
**Date:** 2026-04-25  
**Author:** Mona Maret  
**Scope:** North-star reference for rook-server · rook-cli through PoC completion · Per-release detail in PRD002–PRD011

---

## 1. Purpose and Problem Statement

Rook is a private, self-hosted collaboration platform for small teams that prioritize local-first workflows and data ownership. It gives teams a shared space for async messaging and a document stash — both usable entirely offline — that sync on demand with a central server.

The core problem Rook solves: existing collaboration tools assume persistent connectivity and store all data on third-party infrastructure. Rook's users want to own their data, work from the terminal, and not require a live server connection for everyday work.

---

## 2. Target Users

| Persona | Description |
|---|---|
| **Space member** | Developer or collaborator. Works locally in the TUI, reads and sends messages, manages their document stash, syncs on their own schedule. |
| **Space admin** | Manages user onboarding (key registration), space membership, group assignments, and app visibility ACLs. Operates primarily through the admin CLI. |
| **Server operator** | Deploys and operates `rook-server` on Cloud Run (PoC) or Kubernetes (future). Manages Firestore, service accounts, and infrastructure secrets. |

---

## 3. Product Overview

Rook has two first-class deliverables:

### 3.1 rook-server

A collection of independently deployable Go microservices hosted on Cloud Run. Each service has a single responsibility and is stateless — all persistent state lives in Firestore.

| Service | Role |
|---|---|
| **user-service** | Identity, auth, session management, space membership, group ACLs |
| **stash-service** | Document storage and retrieval, scoped by space |
| **messaging-service** | Async message store and per-conversation sync endpoint |
| **guides-service** | Serves structured guides and onboarding content to authorized users |

Inter-service communication uses gRPC over HTTPS with OIDC-authenticated service accounts. End-user requests arrive via HTTPS only — there is no raw TCP exposure.

### 3.2 rook-cli

A local-first Go binary with a Bubble Tea TUI. It is the only client for rook-server. The CLI must be useful before any server is reachable — offline access to the document stash and message history is a first-class requirement, not a degraded mode.

The CLI is structured as a **launcher**: a home screen that provides full-screen access to each feature app. The user navigates into a feature, works, and returns to the launcher. The launcher is always the entry and exit point.

---

## 4. Feature Requirements

### 4.1 First-Run Setup

**Trigger:** No local config detected on launch.  
**Owner:** rook-cli

- Detect whether the user has an existing SSH key; if not, generate one and display the public key for registration with a server admin.
- Prompt for one or more `rook-server` addresses, the flat-file storage directory (default: `$XDG_CONFIG_HOME/rook/storage/`), and any user preferences.
- Write config to `$XDG_CONFIG_HOME/rook/config.json`.
- Implemented as a first-class Bubble Tea interactive flow — not a shell-script prompt sequence.
- On completion, drop the user into the launcher home screen.

**Acceptance criteria:**
- [ ] Running `rook` with no config initiates the setup flow.
- [ ] SSH key is generated if none exists; public key is displayed for admin registration.
- [ ] Config is written to the correct XDG path; subsequent launches skip setup.
- [ ] WSL (Linux XDG paths) is the supported Windows path — no native Windows build required.

---

### 4.2 Launcher Home Screen

**Owner:** rook-cli

The root of the CLI. Provides:

- **Server status pane** — one row per configured server with a reachability indicator (🟢 / 🔴). Polled once at launch and on manual refresh; not continuous background polling.
- **Local feature shortcuts** — document stash, search. Always available regardless of server state.
- **Connected feature shortcuts** — messaging, guides. Greyed out or hidden when server is unreachable.
- **Space selector** — visible only when the user belongs to multiple spaces. Shows one row per space with an indicator for the active space. Hidden for single-space users.

**Acceptance criteria:**
- [ ] Local features are accessible immediately on launch, before any server reachability check completes.
- [ ] Server status indicators are non-blocking.
- [ ] Space selector is hidden for single-space users.
- [ ] Switching the active space re-scopes all connected features.

---

### 4.3 Authentication

**Subcommand:** `rook auth user@server`  
**Owner:** rook-cli + user-service

- HTTPS challenge-response using the user's SSH private key as the credential.
- Flow: `GET /auth/challenge` → sign nonce locally → `POST /auth/verify` → opaque session token.
- Nonces are single-use, 60-second TTL; stored in Firestore with a TTL policy that auto-deletes within 72 hours.
- Session token is in-memory only for the process lifetime; not written to disk. Re-auth required on next launch.
- All subsequent requests carry `Authorization: Bearer <token>` and `X-Rook-Space-ID`.
- App/space discovery via `GET /spaces` and `GET /spaces/{space-id}/apps` after successful auth — ACL-filtered by the user's group.

**Acceptance criteria:**
- [ ] `rook auth user@server` completes the challenge-response flow and caches the token in-memory.
- [ ] Authenticated requests include both `Authorization` and `X-Rook-Space-ID` headers.
- [ ] Session token is never written to disk.
- [ ] An expired or invalid token triggers re-auth, not a crash.
- [ ] Nonces cannot be replayed — a second use returns `401`.
- [ ] Firestore TTL policy is in place on `auth/nonces/{nonce}.expires_at`.

---

### 4.4 Spaces, Groups, and Access Control

**Owner:** user-service

- A `rook-server` instance hosts one or more **spaces**; all data is fully segregated by space.
- Users belong to one or more spaces; membership is managed by a space admin.
- Within a space, each user belongs to a **group**. Groups control which apps (guides, services) are visible via ACL.
- No cross-space data access of any kind.

**Acceptance criteria:**
- [ ] A user can only access data in spaces they are a member of.
- [ ] A user sees only the apps their group has ACL permission to access.
- [ ] Switching the active space in the launcher re-scopes all queries, including `X-Rook-Space-ID` on every request.

---

### 4.5 Document Stash

**Owner:** rook-cli + stash-service

- A per-user, space-scoped collection of markdown documents and files.
- Files are stored locally as flat files (`.md` content + `.json` metadata, including sync state).
- Sync is **user-initiated** — users choose when to push local changes to the server and when to pull remote changes.
- Sync modes:
  - `sync` — content is synced to the server.
  - `no-sync` — content remains local only.
  - `selective-unsync` — user explicitly removes specific documents from server storage.
  - `admin-silent-sync` — server-side sync initiated by admin (must be disclosed in documentation).
- Search is local-only, operating over the flat-file store.

**Local storage layout:**
```
$XDG_CONFIG_HOME/rook/storage/
├── stash/<space-id>/        # user's documents for each space
├── messages/<space-id>/
│   └── <conversation-id>/  # flat-file message history
└── cache/                  # space membership + app discovery cache
```

**Acceptance criteria:**
- [ ] User can create, edit, and view stash documents entirely offline.
- [ ] User can initiate a sync to push/pull stash changes to/from stash-service.
- [ ] Sync state per document is visible in the TUI.
- [ ] Search operates on local content only, without a server request.
- [ ] Admin silent sync is disclosed clearly in user-facing documentation.

---

### 4.6 Async Messaging

**Owner:** rook-cli + messaging-service

- Conversations are 1:1 (direct) or 1:many (named rooms/channels), scoped to a space.
- Messages are ephemeral records — the conversation is not the artifact. Documents and files generated in discussion are persistent and live in the stash.
- History is stored locally as flat files with the same `.md` + `.json` structure as the stash.
- Sync is pull-only and user-initiated. There is no real-time push delivery — if both users have the CLI open, a message is not delivered to the recipient until they sync.
- Notifications (unread counts, new message indicators) are delivered as metadata in the next sync response — no persistent connection or push channel required.
- The CLI may display notification indicators in the launcher after a sync.

**Acceptance criteria:**
- [ ] User can read, compose, and view conversation history entirely offline using cached flat files.
- [ ] Sync with messaging-service retrieves new messages and pushes any unsynced outbound messages.
- [ ] Unread count indicators are shown in the launcher after a successful sync.
- [ ] Message artifacts (documents, files) are saved to the stash, not stored in the messaging service.
- [ ] No persistent TCP or WebSocket connection is held to the server.

---

### 4.7 Guides

**Owner:** rook-cli + guides-service

- Structured onboarding and reference content served by guides-service.
- Access is ACL-controlled by group membership.
- The CLI renders guides as TUI-formatted markdown using the Glamour renderer.
- Guides are fetched from the server on demand; no offline caching required at PoC.

**Acceptance criteria:**
- [ ] A group-authorized user can browse and read guides from the TUI.
- [ ] A user whose group does not have guides access does not see the guides shortcut in the launcher.

---

### 4.8 Admin CLI (rook-server)

**Owner:** rook-server (user-service)

A lightweight admin surface for server operators and space admins. Exposed as HTTP endpoints on user-service, not as a separate binary for the PoC.

Key operations:
- Register a new user's SSH public key.
- Create a space; add/remove members.
- Assign users to groups; update group ACLs for app access.
- List active sessions; revoke a session token.
- View space membership.

**Acceptance criteria:**
- [ ] A space admin can register a user's public key and assign space/group membership.
- [ ] A space admin can revoke a session token; the revoked token is rejected on the next request.
- [ ] Admin endpoints require an admin-scoped credential — they are not accessible with a user session token.

---

## 5. Non-Functional Requirements

| Requirement | Target |
|---|---|
| **Offline usability** | 100% of local features (stash read/write, message history, search) must work with no network. |
| **Server dependency** | Connected features (messaging sync, guides, stash sync) degrade gracefully — the launcher remains usable and local features are unaffected. |
| **Deployment target (PoC)** | Google Cloud Run (stateless, scale-to-zero). No persistent TCP. |
| **Portability** | Architecture must not foreclose migration to Kubernetes without re-architecture. |
| **Data storage** | All user data is stored as human-readable flat files locally (`.md` + `.json`). No binary formats. |
| **Security** | Session tokens never written to disk. Nonces single-use. All service-to-service calls OIDC-authenticated. No secrets in CLI config. |
| **Platform** | macOS, Linux, WSL. No native Windows build. |
| **CLI latency** | Launcher home screen must render within 500ms with no server interaction. |

---

## 6. Out of Scope

The following are explicitly deferred and must not be designed into the PoC architecture:

- Real-time/push message delivery (WebSocket, SSE, persistent connections).
- A web or mobile client — rook-cli is the only client.
- IRC compatibility or any non-Rook client interoperability.
- File attachment storage beyond markdown and `.json` metadata.
- Federation or multi-server spaces.
- Native Windows build (WSL is the supported path).
- Public user registration — all users are manually onboarded by a space admin.

---

## 7. Roadmap and Release Planning

### Milestone 0 — PoC (current)

**Goal:** Prove the core architecture end-to-end with a single space, single admin, and two to three users.

| Area | Deliverable |
|---|---|
| rook-server | user-service: key registration, challenge-response auth, session management, space membership, ValidateSession RPC |
| rook-server | stash-service: store and retrieve documents, space-scoped |
| rook-server | messaging-service: store and retrieve messages, per-conversation sync |
| rook-server | guides-service: serve static guide content, ACL-gated by group |
| rook-cli | First-run setup flow |
| rook-cli | Launcher home screen with server status, local/connected feature shortcuts |
| rook-cli | `rook auth` — challenge-response auth flow |
| rook-cli | Document stash: create, view, sync |
| rook-cli | Messaging: conversation list, thread view, compose, sync |
| rook-cli | Guides: browse and read |
| Infrastructure | Cloud Run deployment for all four services |
| Infrastructure | Firestore schema and TTL policies (nonces) |
| Infrastructure | OIDC service-to-service auth |

**PoC success criteria:**
- A space admin can register a user via the admin CLI.
- A registered user can run `rook`, complete setup, authenticate, and access messaging, stash, and guides — all in a single session.
- All four services run on Cloud Run and communicate over gRPC/HTTPS.
- Local features remain available when the server is unreachable.

---

### Milestone 1 — Hardening

**Goal:** Make the PoC stable and safe enough for regular daily use by a real team.

| Area | Deliverable |
|---|---|
| Auth | Session token expiry and graceful re-auth flow in the TUI (no crash, prompt to re-authenticate) |
| Messaging | Conflict resolution for concurrent edits to synced conversation history |
| Stash | Conflict resolution for concurrent stash document edits |
| Stash | Selective-unsync and admin-silent-sync modes |
| Admin CLI | Session revocation; list active sessions per space |
| Observability | Structured logging and basic error reporting across all services |
| Security | Audit logging for admin operations (key registration, group changes, session revocation) |
| Documentation | Admin operations guide; user-facing disclosure of admin-silent-sync |

---

### Milestone 2 — Multi-Space and Access Polish

**Goal:** Support real-world team topologies with multiple spaces and non-trivial group ACLs.

| Area | Deliverable |
|---|---|
| rook-cli | Space selector UX — full multi-space experience; space switching without re-auth |
| rook-server | Multi-space membership per user; cross-space admin delegation |
| rook-server | Group ACL editor in admin CLI |
| Guides | Guides versioning — serve different guide content to different groups |
| rook-cli | Notification persistence across launches (unread state survives process exit) |
| Platform | Linux package (`.deb`, `.rpm`) and Homebrew tap for macOS |

---

### Milestone 3 — Portability and Operations

**Goal:** Support deployment outside Cloud Run; make the system observable and operable in production.

| Area | Deliverable |
|---|---|
| Infrastructure | Kubernetes manifests for all services (Deployment + Service + Ingress) |
| Infrastructure | Helm chart for operator-managed deployment |
| rook-server | Health and readiness endpoints on all services |
| Observability | OpenTelemetry tracing across all gRPC and HTTP boundaries |
| rook-server | Rate limiting on auth endpoints (challenge and verify) |
| rook-cli | `rook update` — self-update from a configured release source |

---

### Post-v1 (Backlog, Not Committed)

- End-to-end encryption for stash documents and message history.
- File attachment support beyond markdown.
- Federation or multi-server space peering.
- `charmbracelet/charm` encrypted file store evaluation as a stash primitive.
- Real-time notification delivery (if demand warrants the infrastructure complexity).

---

## 8. Open Questions

| # | Question | Owner | Resolution Target |
|---|---|---|---|
| 1 | What is the intended maximum number of users per space for the PoC? Informs Firestore read/write cost estimates. | Mona Maret | Milestone 0 |
| 2 | Should guides content be version-controlled in the repo (static) or editable by admins at runtime (dynamic)? | Mona Maret | Milestone 0 |
| 3 | Is `charmbracelet/charm`'s encrypted file store a viable primitive for stash, or is the custom flat-file model preferred long-term? | Mona Maret | Milestone 1 |
| 4 | What is the expected latency tolerance for stash/message sync? Informs whether a loading indicator or background sync is required in the TUI. | Mona Maret | Milestone 1 |

---

## 9. References

| Document | Path |
|---|---|
| System Architecture ADR | `specs/decisions/2026-04-25-rook-reference-system-architecture.md` |
| rook-cli Features and UX ADR | `specs/decisions/2026-04-25-rook-cli-features-and-ux-architecture.md` |
| Auth and Cloud Run Topology ADR | `specs/decisions/2026-04-25-ssh-auth-identity-chain-and-cloud-run-topology.md` |
| Real-Time Messaging Protocol ADR | `specs/decisions/2026-04-25-real-time-messaging-protocol.md` |
| gRPC Inter-Service Communication ADR | `specs/decisions/2026-04-25-grpc-inter-service-communication.md` |
| Admin CLI ADR | `specs/decisions/2026-04-25-rook-server-admin-cli.md` |
| Component Overview | `specs/architecture/component-overview.md` |
| gRPC Call Flows | `specs/architecture/grpc-call-flows.md` |
