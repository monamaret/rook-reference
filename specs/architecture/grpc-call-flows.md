# rook-server: Architecture Diagrams and gRPC Call Flows

This document describes the inter-service architecture of `rook-server` and the gRPC call flows between services. It is the companion reference to [`../decisions/2026-04-25-grpc-inter-service-communication.md`](../decisions/2026-04-25-grpc-inter-service-communication.md).

For a higher-level view of how `rook-cli`, `rook-server`, and `rook-server-cli` relate to each other, see [`component-overview.md`](component-overview.md).

All inter-service calls are unary gRPC over HTTP/2. Cloud Run manages TLS. Each caller attaches a GCP OIDC bearer token. No raw HTTP calls cross service boundaries.

---

## System Overview

```
                          ┌─────────────────────────────────────────────────────┐
                          │                  rook-server (Cloud Run)            │
                          │                                                     │
  rook-cli ──SSH──────────┤──► user-service    ◄──── all services call this    │
            (wish/        │         │           ◄──── rook-server-cli (admin)  │
            wishlist)     │         │ gRPC (UserService / AdminService)        │
                          │         ▼                                           │
  rook-cli ──HTTP──────── ├──► messaging-service                               │
            (sync pull)   │         │                                           │
                          │         │ gRPC (UserService)                       │
                          │         ▼                                           │
  rook-cli ──HTTP──────── ├──► stash-service                                   │
            (sync pull)   │         │                                           │
                          │         │ gRPC (UserService)                       │
                          │         ▼                                           │
  rook-cli ──HTTP──────── └──► guides-service                                  │
            (guide fetch)                                                       │
                          └─────────────────────────────────────────────────────┘

  rook-server-cli ──gRPC (AdminService, bearer token)──► user-service
  (admin key registration, user/space management — never exposed to rook-cli)

  External interface: SSH (wish/wishlist) for auth; HTTP for data sync and guide fetch
  Admin interface: gRPC (AdminService) with pre-shared admin token — rook-server-cli only
  Internal interface: gRPC only — no HTTP between services
  Data layer: each service owns its own Firestore collection namespace
```

---

## Service Responsibilities

| Component | External interface | gRPC role | Firestore namespace |
|-----------|-------------------|-----------|---------------------|
| `user-service` | SSH (wish + wishlist) | Server: `UserService` + `AdminService` | `users/`, `spaces/`, `groups/` |
| `messaging-service` | HTTP (pull sync) | Client of `UserService` | `messages/` |
| `stash-service` | HTTP (pull sync) | Client of `UserService` | `stash/` |
| `guides-service` | HTTP (guide fetch) | Client of `UserService` | `guides/` |
| `rook-server-cli` | gRPC (`AdminService`, admin token) | Client of `AdminService` | — (writes via `user-service`) |

---

## UserService RPC Reference

All services except `user-service` are consumers of `UserService`. Defined in `rook-server/proto/user/v1/user.proto`.

```
UserService
├── GetUserByKey(fingerprint) → User
│     Resolves a user identity from their SSH public key fingerprint.
│     Called by: messaging-service, stash-service, guides-service
│
├── GetSpaceMembership(user_id, space_id) → SpaceMembership
│     Returns the user's group within a space, or NOT_FOUND if not a member.
│     Called by: messaging-service, stash-service, guides-service
│
└── CheckAppAccess(user_id, space_id, app_id) → AccessDecision
      Returns ALLOWED or DENIED based on the user's group ACL for the given app.
      Called by: guides-service (wishlist filtering), stash-service (shared doc access)
```

---

## AdminService RPC Reference

`AdminService` is implemented on `user-service` alongside `UserService`. It is consumed exclusively by `rook-server-cli`. All RPCs require a valid `ROOK_ADMIN_TOKEN` bearer token in gRPC metadata, validated by a unary server interceptor before any handler runs. Defined in `rook-server/proto/admin/v1/admin.proto`.

```
AdminService
├── RegisterUserKey(username, pubkey) → {user_id}
│     Creates a user record (if not exists) and stores the SSH public key fingerprint.
│     Called by: rook-server-cli user register-key
│
├── ListUsers() → []User
│     Returns all registered users with their IDs and usernames.
│     Called by: rook-server-cli user list
│
├── CreateSpace(name, description) → {space_id}
│     Creates a new space record in Firestore.
│     Called by: rook-server-cli space create
│
├── ListSpaces() → []Space
│     Returns all spaces.
│     Called by: rook-server-cli space list
│
├── AddUserToSpace(user_id, space_id, group) → {}
│     Adds a user to a space and assigns them to a group.
│     Called by: rook-server-cli user add-to-space
│
├── RemoveUserFromSpace(user_id, space_id) → {}
│     Removes a user's membership from a space.
│     Called by: rook-server-cli user remove-from-space
│
├── SetUserGroup(user_id, space_id, group) → {}
│     Updates a user's group assignment within a space without changing membership.
│     Called by: rook-server-cli user set-group
│
└── ListSpaceMembers(space_id) → []Member
      Returns all members of a space with their group assignments.
      Called by: rook-server-cli space members
```

---

## Call Flows

### 0. Admin: Key Registration

Before a user can authenticate, a server admin must register their SSH public key. The user provides their public key out-of-band (e.g. email or paste); the admin runs `rook-server-cli user register-key`.

```
rook-server-cli                        user-service
       │                                    │
       │ reads ROOK_ADMIN_TOKEN from env    │
       │ reads USER_SERVICE_ADDR from env   │
       │                                    │
       │──gRPC RegisterUserKey──────────────►│
       │  metadata: authorization: Bearer   │
       │  <admin_token>                     │ [interceptor] validate admin token
       │  body: {username, pubkey}          │ create user record in Firestore:
       │                                    │   users/{user_id}/profile
       │                                    │   users/{user_id}/keys/{fingerprint}
       │◄──{user_id: "u-abc"}───────────────│
       │                                    │
       │──gRPC AddUserToSpace───────────────►│  (optional, separate command)
       │  body: {user_id, space_id, group}  │ write to Firestore:
       │◄──{}───────────────────────────────│   users/{user_id}/spaces/{space_id}
       │                                    │   (group: "users")
```

The user can now authenticate:

```
rook-cli                               user-service (wish)
   │                                        │
   │──SSH connect (public key in handshake)─►│
   │                                        │ look up fingerprint in
   │                                        │ users/{id}/keys/ → match found
   │                                        │ resolve space membership + ACL
   │◄──wishlist (space-filtered apps)───────│
```

---

### 1. CLI Authentication and Wishlist

The user runs `rook ssh user@server`. This is an SSH connection handled by `charmbracelet/wish` and `charmbracelet/wishlist` — not a gRPC call. The wishlist response is filtered by the user's space and group ACL.

```
rook-cli                    user-service (wish/wishlist)
   │                               │
   │──SSH connect──────────────────►│
   │   (public key in handshake)    │
   │                               │ look up user by key fingerprint
   │                               │ (internal: Firestore users/ lookup)
   │                               │ resolve space membership
   │                               │ filter wishlist by group ACL
   │◄──wishlist (filtered apps)────│
   │
   │ (user selects a space if multi-space;
   │  wishlist is already scoped to that space)
```

No gRPC involved here — this is the SSH entry point. gRPC is used only for service-to-service calls triggered by subsequent CLI HTTP requests.

---

### 2. Messaging Sync (Pull)

The user triggers a sync from the messaging view in `rook-cli`. The CLI sends an HTTP request to `messaging-service`. Before returning messages, the messaging service verifies the caller's identity and space membership via gRPC to `user-service`.

```
rook-cli                messaging-service              user-service
   │                          │                             │
   │──HTTP GET /sync──────────►│                             │
   │  (Bearer: SSH key token)  │                             │
   │                          │──GetUserByKey(fingerprint)──►│
   │                          │◄──User{id, ...}─────────────│
   │                          │                             │
   │                          │──GetSpaceMembership──────────►│
   │                          │  (user_id, space_id)        │
   │                          │◄──SpaceMembership{group}────│
   │                          │                             │
   │                          │ fetch messages from Firestore
   │                          │ messages/ scoped to space_id + convo_id
   │                          │
   │◄──HTTP 200 {messages}────│
   │
   │ (CLI writes .md + .json flat files to
   │  <storage-dir>/messages/<space-id>/<convo-id>/)
```

Outbound messages (composed in `$EDITOR`, queued locally) are pushed in the same sync request body. The messaging service writes them to Firestore.

---

### 3. Document Stash Sync (Pull)

Same pattern as messaging sync. The CLI sends an HTTP pull request to `stash-service`; the service verifies identity and space membership via gRPC before returning documents.

```
rook-cli                 stash-service                 user-service
   │                          │                             │
   │──HTTP GET /sync──────────►│                             │
   │  (Bearer: SSH key token)  │                             │
   │                          │──GetUserByKey(fingerprint)──►│
   │                          │◄──User{id, ...}─────────────│
   │                          │                             │
   │                          │──GetSpaceMembership──────────►│
   │                          │◄──SpaceMembership{group}────│
   │                          │                             │
   │                          │ fetch documents from Firestore
   │                          │ stash/ scoped to space_id
   │                          │ filtered by: owned by user OR shared with user's group
   │                          │
   │◄──HTTP 200 {documents}───│
   │
   │ (CLI writes .md + .json flat files to
   │  <storage-dir>/stash/<space-id>/)
```

On conflict (same document modified locally and on server), last-write-wins — the server timestamp is authoritative. No concurrent editing is expected at PoC scale.

---

### 4. Guide Fetch (Read-only)

When the user selects a guide from the wishlist, the CLI fetches its assets from `guides-service`. The service verifies access via `CheckAppAccess` before returning the guide bundle.

```
rook-cli                 guides-service                user-service
   │                          │                             │
   │──HTTP GET /guide/{id}────►│                             │
   │  (Bearer: SSH key token)  │                             │
   │                          │──GetUserByKey(fingerprint)──►│
   │                          │◄──User{id, ...}─────────────│
   │                          │                             │
   │                          │──CheckAppAccess──────────────►│
   │                          │  (user_id, space_id, guide_id)│
   │                          │◄──AccessDecision{ALLOWED}───│
   │                          │                             │
   │                          │ fetch guide assets from Firestore
   │                          │ guides/ — .md content, lipgloss .yml, YAML config
   │                          │
   │◄──HTTP 200 {guide bundle}│
   │
   │ (CLI renders guide full-screen via charmbracelet/glamour + lipgloss)
```

If `CheckAppAccess` returns `DENIED`, `guides-service` returns HTTP 403. The guide is not surfaced in the wishlist for that user in the first place (filtered at auth time by `user-service`), so this is a defence-in-depth check.

---

### 5. Guide Publish (Builder → guides-service)

When the user publishes a guide from the guide builder TUI, the CLI uploads the validated guide bundle to `guides-service`. The service verifies the caller is an authenticated space member before accepting the upload.

```
rook-cli (guide builder)  guides-service               user-service
   │                          │                             │
   │──HTTP POST /guide────────►│                             │
   │  body: {.md, .yml,        │                             │
   │   yaml-config, meta}      │                             │
   │  (Bearer: SSH key token)  │                             │
   │                          │──GetUserByKey(fingerprint)──►│
   │                          │◄──User{id, ...}─────────────│
   │                          │                             │
   │                          │──GetSpaceMembership──────────►│
   │                          │◄──SpaceMembership{group}────│
   │                          │                             │
   │                          │ write guide assets to Firestore
   │                          │ set creator as guide owner
   │                          │ set initial ACL (creator's group, or as specified)
   │                          │
   │◄──HTTP 201 {guide_id}────│
   │
   │ (guide now visible in wishlist for permitted groups)
```

---

## Error Handling Conventions

| gRPC status | Meaning | HTTP equivalent returned to CLI |
|-------------|---------|--------------------------------|
| `codes.NotFound` | User key not registered, or space/guide does not exist | 404 |
| `codes.PermissionDenied` | User is not a member of the space, or ACL denies access | 403 |
| `codes.Unauthenticated` | Missing or invalid OIDC token on inter-service call | 500 (internal — caller bug) |
| `codes.Unavailable` | Downstream service unreachable | 503 |
| `codes.Internal` | Unexpected error in callee | 500 |

Services must not leak internal gRPC error details to CLI HTTP responses. Map gRPC errors to appropriate HTTP status codes at the service boundary.

---

## Firestore Collection Layout

Each service owns its collections. No cross-service collection access.

```
Firestore root
├── users/
│   └── {user_id}/                  # user-service
│       ├── profile (doc)
│       ├── keys/ (subcollection)   # SSH key fingerprints → user_id index
│       └── spaces/ (subcollection) # space memberships + group assignments
│
├── spaces/
│   └── {space_id}/                 # user-service
│       ├── config (doc)
│       └── groups/ (subcollection) # group definitions + app ACLs
│
├── messages/
│   └── {space_id}/                 # messaging-service
│       └── {convo_id}/
│           └── {message_id} (doc)  # content + metadata
│
├── stash/
│   └── {space_id}/                 # stash-service
│       └── {doc_id} (doc)          # content + metadata + permissions
│
└── guides/
    └── {space_id}/                 # guides-service
        └── {guide_id}/
            ├── meta (doc)          # title, description, owner, ACL
            └── assets/ (subcollection) # .md content, lipgloss .yml, YAML config
```

---

## Key Constraints (Do Not Violate)

- All inter-service calls are gRPC — no raw HTTP between services
- `user-service` is the only gRPC server; all other services are clients only (of `UserService`)
- gRPC endpoints are never exposed to `rook-cli` directly — the CLI always talks HTTP or SSH
- Services never access another service's Firestore collections — only their own namespace
- All service addresses are environment-variable-driven — see `USER_SERVICE_ADDR` and equivalents in the gRPC ADR
- OIDC bearer tokens are required on all inter-service gRPC calls in deployed (Cloud Run) environments
