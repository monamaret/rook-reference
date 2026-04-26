---
status: proposed
date: 2026-04-25
decision-makers: Mona Maret
consulted: Mona Maret
informed: Mona Maret
---

# Real-Time Messaging Protocol: IRC vs. Custom Async Messaging

## Context and Problem Statement

The system architecture ADR ([`2026-04-25-rook-reference-system-architecture.md`](2026-04-25-rook-reference-system-architecture.md)) provisionally named IRC as the primary real-time communication protocol. That assumption needs to be re-examined.

Rook is a **private** server — all users are authenticated Rook users connecting exclusively via `rook-cli`. The CLI is a local-first TUI application built on Bubble Tea. Users open it, work with local content (messages, documents, markdown files), sync with the server as needed, and close it. They do not hold open persistent server connections.

Given this usage model: should Rook implement IRC, or a custom async messaging system designed around the actual usage pattern?

## Decision Drivers

- `rook-cli` is the **only** client — no need for IRC client interoperability
- All users authenticate via SSH key through `charmbracelet/wish` — IRC's NICK/USER/PASS auth is a mismatch with no benefit
- The CLI is **local-first**: users open it, work offline, and sync lazily — a persistent server connection is the wrong model
- Messaging is **fully async pull-only**: the CLI pulls updates on demand; there is no polling, push, or persistent connection
- Chat history is stored in **local flat files** (same format as the document stash: `.md` + `.json`) with server sync as an option
- Messages are **ephemeral** — the conversation itself is not the artifact; documents and files generated in discussion are the persistent artifacts
- Sync behavior is user-controlled: sync, don't sync, selective unsync, and admin silent sync are all first-class options
- Cloud Run's stateless, scale-to-zero model is a constraint — persistent TCP connections (IRC) are incompatible
- `charmbracelet/charm` may provide useful primitives for user management and encrypted file store that should be evaluated before designing a custom solution

## Considered Options

- **IRC** — implement a standard IRC server; CLI connects as an IRC client
- **Custom async messaging over HTTP** — purpose-built message service with a REST or gRPC API; CLI syncs on demand

## Decision Outcome

Chosen option: **Custom async messaging over HTTP**, because:

- The usage model (local-first, offline-capable, pull-only CLI) is fundamentally incompatible with IRC's session-oriented, persistent-connection design
- All users are SSH-authenticated Rook users — there is no value in IRC's open client ecosystem
- HTTP request/response maps directly to the pull-only sync model and is natively compatible with Cloud Run
- The message schema can be designed to natively express 1:1 and 1:many conversations, user permissions, artifact references, and sync state — none of which IRC supports natively
- No IRC bridge, transport adapter, or auth translation layer is required

### Messaging Model

- **Conversations** are either 1:1 (direct) or 1:many (named rooms/channels)
- **Messages** are ephemeral records — the unit of communication, not the unit of persistence
- **Artifacts** (markdown documents, files, links) generated in conversation are the persistent objects; they are managed by the document stash service, not the messaging service
- **History** is stored locally as flat files (`.md` for content, `.json` for metadata including sync state, participants, timestamps)
- **Sync is user-controlled** with four modes:
  - `sync` — history is synced to the server
  - `no-sync` — history remains local only
  - `selective-unsync` — user removes specific messages or threads from the server
  - `admin-silent-sync` — server-side sync initiated by an admin without user action (must be disclosed in documentation)
- **Notifications** are delivered as metadata in the next sync response — unread counts, new message indicators — not via push or persistent connection
- The CLI may display notification indicators (e.g., badge counts in the primary menu) after a sync, without requiring the user to be in a live message view

### Consequences

- Good, because the server is a simple stateless sync endpoint — no session management, no persistent connections, no fan-out broker
- Good, because Cloud Run's scale-to-zero model is fully compatible — each sync is a short-lived HTTP request
- Good, because the local flat-file history model is consistent with the document stash, reducing CLI complexity
- Good, because artifact persistence is delegated to the document stash service — the messaging service has a single responsibility
- Good, because sync modes give users meaningful control over their data
- Good, because `charmbracelet/charm`'s encrypted file store and user management can be evaluated as primitives without being committed to upfront
- Bad, because there is no real-time delivery — a message sent while both users have the CLI open is not delivered until the recipient syncs
- Bad, because admin silent sync must be carefully documented and disclosed to avoid trust issues
- Neutral, because designing a custom message schema requires upfront spec work, but this is offset by not needing an IRC compatibility layer

## Implementation Plan

*To be detailed in service-level specs. The following captures system-wide constraints.*

- **Affected paths**:
  - `rook-server/messaging/` — async message service (store, retrieve, sync endpoint)
  - `rook-cli/` — message views (conversation list, thread view, compose), sync logic, local flat-file store, notification indicators
- **Dependencies**:
  - Server transport: REST (`net/http`) or gRPC — coordinate with inter-service communication ADR
  - `charmbracelet/charm` — evaluate for user management and encrypted file store primitives before implementing custom equivalents
  - `charmbracelet/wish` — SSH auth layer (already established)
- **Message schema** (local flat files):
  - `.md` — message content (human-readable, renderable by `rook-cli` and any markdown viewer)
  - `.json` — metadata: conversation ID, participants, timestamp, sync state (`sync | no-sync | selective-unsync`), artifact references
- **Patterns to follow**:
  - Messaging service is stateless — no in-memory session state; all state is in the data store
  - CLI sync is always user- or event-initiated — never background polling
  - Artifact references in messages are pointers to document stash entries, not embedded content
  - Notification indicators are derived from sync response metadata, not a separate push channel
- **Patterns to avoid**:
  - Do not implement persistent connections (WebSocket, SSE, long-poll, IRC) in the messaging service
  - Do not store message content in the server's data layer beyond what is needed for sync — messages are ephemeral on the server side once delivered
  - Do not embed artifact content in message records — reference the document stash service instead
- **Configuration**:
  - Messaging service endpoint URL (env var)
  - Default sync mode per conversation (user-configurable, stored in local `.json` metadata)
  - Admin silent sync policy (server-side config, must be surfaced in `rook-docs` for transparency)

### Verification

- [ ] CLI sends a message to a 1:1 conversation and the recipient retrieves it on next sync
- [ ] CLI sends a message to a 1:many room and all participants retrieve it on next sync
- [ ] Local message history is stored as `.md` + `.json` flat files after a sync
- [ ] `no-sync` mode: message remains local only and is never sent to the server
- [ ] `selective-unsync`: a previously synced message is removed from the server; recipient's local copy is unaffected
- [ ] Sync response includes unread counts and new message indicators for all conversations
- [ ] Notification indicators appear in the CLI primary menu after a sync without entering a message view
- [ ] User A cannot retrieve User B's private conversation via the messaging service API
- [ ] Messaging service starts and operates independently of all other `rook-server` services
- [ ] Messaging service handles a sync request and returns in under 2 seconds (local dev environment)

## Pros and Cons of the Options

### IRC

- Good, because the protocol is battle-tested and extensively documented
- Good, because existing Go IRC libraries exist (e.g., `github.com/lrstanley/girc`)
- Neutral, because channel/topic semantics partially map to 1:many conversations
- Bad, because IRC requires persistent TCP connections — incompatible with Cloud Run and the pull-only CLI model
- Bad, because IRC's auth model (NICK/USER/PASS) conflicts with SSH-key identity — bridging is non-trivial
- Bad, because IRC has no native concept of offline message delivery, sync state, or artifact references
- Bad, because IRC's feature set is constrained by a 1988 protocol with no extensibility path

### Custom async messaging over HTTP

- Good, because the pull-only, request/response model maps exactly to the CLI's usage pattern
- Good, because stateless HTTP is natively compatible with Cloud Run's scale-to-zero model
- Good, because the message schema can natively express sync modes, artifact references, and notification metadata
- Good, because no auth bridging is required — identity flows directly from `charmbracelet/wish`
- Bad, because messages are not delivered in real-time — delivery depends on the recipient initiating a sync
- Bad, because a custom schema requires upfront design work before implementation can begin

## More Information

### Interaction with other ADRs

- **Supersedes (partially)**: [`2026-04-25-rook-reference-system-architecture.md`](2026-04-25-rook-reference-system-architecture.md) — the system architecture ADR provisionally named IRC as the primary protocol. Once this ADR is accepted, update the system architecture ADR to replace IRC references with "custom async messaging service."
- **Depends on**: Inter-service communication protocol ADR (REST vs. gRPC) — the messaging service's sync API must align with the chosen protocol.
- **Related**: Document stash design — artifact references in messages point into the document stash; the two services share the local flat-file format and sync conventions.

### Deferred Decisions

The following questions do not block acceptance of this ADR but must be resolved before the messaging service is implemented. Each should be captured in a follow-up ADR or appended here when resolved.

**1. Server-side message retention** ✅ Resolved
Messages are retained **indefinitely** in Firestore until explicitly removed by the user via `selective-unsync`. No TTL or automatic expiry. "Ephemeral" refers to the conversational nature of messages (as opposed to artifacts/documents), not to server-side lifetime. Storage cost is acceptable at PoC scale on Firestore's free tier.

**2. Server-side store technology** ✅ Resolved
`charmbracelet/charm` is **archived** (March 2025) and Charm Cloud was sunset November 2024. It is not suitable as a backend primitive — unmaintained, no security patches, and its SSH/HTTP port model is incompatible with Cloud Run's single-port HTTP constraint. Glow and Skate (the primary charm consumers) have already removed the integration.

**Chosen data layer: Google Cloud Firestore** for all `rook-server` services, including messaging. Rationale:
- Serverless and scales to zero alongside Cloud Run — no managed instance cost
- Document-oriented model maps naturally to conversations, messages, spaces, and permissions
- Free tier is sufficient for a PoC deployment
- Native GCP integration — no sidecar, no proxy, no extra infrastructure
- Go client: `cloud.google.com/go/firestore`
- Kubernetes portability is explicitly a non-goal for this deployment scale

Each service owns its own Firestore collection namespace. No shared collections across services.

**3. Admin silent sync disclosure** ✅ Resolved
Disclosure is **documentation-only**: the admin silent sync capability and its conditions of use must be clearly documented in `rook-docs` under the admin persona section. No in-app notice and no cryptographic audit log for the PoC. Server admins are expected to operate the server in good faith; the reference implementation documents the feature transparently so users understand the trust model when connecting to a rook-server.
