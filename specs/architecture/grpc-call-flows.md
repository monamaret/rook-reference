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
  rook-cli ──HTTPS────────┤──► user-service    ◄──── all services call this    │
            (auth +       │         │           ◄──── rook-server-cli (admin)  │
            space/app     │         │ gRPC (UserService / AdminService)        │
            discovery)    │         ▼                                           │
                          │                                                     │
  rook-cli ──HTTPS──────── ├──► messaging-service                               │
            (sync pull)   │         │                                           │
                          │         │ gRPC (UserService)                       │
                          │         ▼                                           │
  rook-cli ──HTTPS──────── ├──► stash-service                                   │
            (sync pull)   │         │                                           │
                          │         │ gRPC (UserService)                       │
                          │         ▼                                           │
  rook-cli ──HTTPS──────── └──► guides-service                                  │
            (guide fetch)                                                       │
                          └─────────────────────────────────────────────────────┘

  rook-server-cli ──gRPC (AdminService, bearer token)──► user-service
  (admin key registration, user/space management — never exposed to rook-cli)

  External interface: HTTPS for all rook-cli communication (auth, data sync, guide fetch)
  Admin interface: gRPC (AdminService) with pre-shared admin token — rook-server-cli only
  Internal interface: gRPC only — no HTTP between services
  Data layer: each service owns its own Firestore collection namespace
```

---

## Service Responsibilities

| Component | External interface | gRPC role | Firestore namespace |
|-----------|-------------------|-----------|---------------------|
| `user-service` | HTTPS (auth challenge-response, space/app discovery) | Server: `UserService` + `AdminService` | `users/`, `spaces/`, `groups/`, `sessions/`, `auth/nonces/` |
| `messaging-service` | HTTPS (pull sync) | Client of `UserService` | `messages/` |
| `stash-service` | HTTPS (pull sync) | Client of `UserService` | `stash/` |
| `guides-service` | HTTPS (guide fetch + publish) | Client of `UserService` | `guides/` |
| `rook-server-cli` | gRPC (`AdminService`, admin token) | Client of `AdminService` | — (writes via `user-service`) |

---

## UserService RPC Reference

All services except `user-service` are consumers of `UserService`. Defined in `rook-server/proto/user/v1/user.proto`.

```
UserService
├── ValidateSession(token, space_id) → {user_id, space_membership}
│     Validates an opaque session token and resolves both user identity and space
│     membership in one call. Called by the SessionAuthMiddleware in every service
│     on every authenticated request. The X-Rook-Space-ID request header provides
│     the space_id. Returns user_id + group on success; UNAUTHENTICATED if token
│     is missing/expired/unknown; PERMISSION_DENIED if user is not a member of space_id.
│     Called by: messaging-service, stash-service, guides-service (via shared middleware)
│
├── GetUserByKey(fingerprint) → User
│     Resolves a user identity from their SSH public key fingerprint.
│     Internal to user-service only — called during POST /auth/verify to look up
│     the registered public key before verifying the challenge signature.
│     NOT called by downstream services directly.
│
├── GetSpaceMembership(user_id, space_id) → SpaceMembership
│     Returns the user's group within a space, or NOT_FOUND if not a member.
│     Retained for use cases where space membership must be checked independently
│     of session validation (e.g. admin operations, internal user-service logic).
│     NOT called by downstream services in the normal request path — ValidateSession
│     subsumes this call.
│
└── CheckAppAccess(user_id, space_id, app_id) → AccessDecision
      Returns ALLOWED or DENIED based on the user's group ACL for the given app.
      Called by: guides-service (defence-in-depth access check on guide fetch)
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

The user can now authenticate using `rook auth`:

```
rook-cli                               user-service
   │                                        │
   │──HTTPS GET /auth/challenge─────────────►│
   │                                        │ generate nonce (32 bytes, hex)
   │                                        │ store in Firestore auth/nonces/{nonce}
   │                                        │ with 60s TTL
   │◄──{nonce}──────────────────────────────│
   │                                        │
   │ signs nonce with SSH private key        │
   │ (golang.org/x/crypto/ssh)              │
   │                                        │
   │──HTTPS POST /auth/verify───────────────►│
   │  body: {public_key, nonce, signature}  │ fetch registered key from
   │                                        │ users/{id}/keys/{fingerprint}
   │                                        │ verify signature (pubKey.Verify)
   │                                        │ mark nonce used (atomic write)
   │                                        │ create session token (32 bytes, hex)
   │                                        │ store in Firestore sessions/{token}
   │                                        │ with 1h TTL
   │◄──{session_token}──────────────────────│
   │                                        │
   │ cache token in-memory (session-scoped) │
   │                                        │
   │──HTTPS GET /spaces─────────────────────►│
   │  Authorization: Bearer <session_token> │ validate token, return space list
   │◄──{spaces: [...], acl: {...}}──────────│
```

---

### 1. CLI Authentication (HTTPS Challenge-Response)

The user runs `rook auth user@server`. This is an HTTPS exchange with `user-service` — not an SSH connection.

```
rook-cli                    user-service
   │                               │
   │──HTTPS GET /auth/challenge────►│
   │                               │ generate nonce (random 32 bytes, hex-encoded)
   │                               │ store in Firestore auth/nonces/{nonce} (TTL: 60s)
   │◄──{nonce}─────────────────────│
   │                               │
   │ load SSH private key from     │
   │ config (key path)             │
   │ sign nonce with               │
   │ golang.org/x/crypto/ssh       │
   │                               │
   │──HTTPS POST /auth/verify──────►│
   │  {public_key, nonce,          │ look up fingerprint in users/{id}/keys/
   │   signature}                  │ call GetUserByKey(fingerprint) [internal]
   │                               │ verify signature: pubKey.Verify(nonce, sig)
   │                               │ confirm nonce not expired, not yet used
   │                               │ mark nonce used in Firestore (atomic)
   │                               │ create session token (random 32 bytes, hex)
   │                               │ write to Firestore sessions/{token}:
   │                               │   {user_id, expires_at: now+1h}
   │◄──{session_token}─────────────│
   │                               │
   │ hold token in-memory          │
   │ (discarded on process exit)   │
   │                               │
   │──HTTPS GET /spaces────────────►│
   │  Authorization: Bearer <token>│ ValidateSession(token, space_id="")
   │  (no space yet)               │ return all spaces user belongs to
   │◄──{spaces: [...]}─────────────│
   │                               │
   │ (user selects active space    │
   │  if multi-space; space        │
   │  selector shown in launcher)  │
   │                               │
   │──HTTPS GET /spaces/{id}/apps──►│
   │  Authorization: Bearer <token>│ ACL-filter apps by user's group
   │  X-Rook-Space-ID: {space_id}  │
   │◄──{apps: [...]}───────────────│
   │                               │
   │ cache to                      │
   │ <storage-dir>/cache/spaces.json
```

No gRPC involved in the CLI auth flow — this is entirely HTTPS between `rook-cli` and `user-service`.

---

### 2. Messaging Sync (Pull)

The user triggers a sync from the messaging view. All requests carry the session token and `X-Rook-Space-ID`. The `SessionAuthMiddleware` resolves identity and space membership in a single `ValidateSession` gRPC call before the handler runs.

```
rook-cli                messaging-service              user-service
   │                          │                             │
   │──HTTPS GET /sync──────────►│                             │
   │  Authorization: Bearer    │                             │
   │  X-Rook-Space-ID: {id}    │                             │
   │                          │──ValidateSession(token,──────►│
   │                          │   space_id)                  │ look up sessions/{token}
   │                          │                             │ verify not expired
   │                          │                             │ look up space membership
   │                          │◄──{user_id, group}──────────│
   │                          │                             │
   │                          │ [middleware injects user_id + group into context]
   │                          │                             │
   │                          │ fetch messages from Firestore
   │                          │ messages/ scoped to space_id + convo_id
   │                          │ filtered to conversations user participates in
   │                          │
   │◄──HTTPS 200 {messages}────│
   │
   │ (CLI writes .md + .json flat files to
   │  <storage-dir>/messages/<space-id>/<convo-id>/)
```

Outbound messages (composed in `$EDITOR`, queued locally) are pushed in the same sync request body.

---

### 3. Document Stash Sync (Pull)

Same middleware pattern as messaging sync. One `ValidateSession` call per request; no additional gRPC round-trips in the handler.

```
rook-cli                 stash-service                 user-service
   │                          │                             │
   │──HTTPS GET /sync──────────►│                             │
   │  Authorization: Bearer    │                             │
   │  X-Rook-Space-ID: {id}    │                             │
   │                          │──ValidateSession(token,──────►│
   │                          │   space_id)                  │ verify token + membership
   │                          │◄──{user_id, group}──────────│
   │                          │                             │
   │                          │ fetch documents from Firestore
   │                          │ stash/ scoped to space_id
   │                          │ filtered by: owned by user OR shared with user's group
   │                          │
   │◄──HTTPS 200 {documents}───│
   │
   │ (CLI writes .md + .json flat files to
   │  <storage-dir>/stash/<space-id>/)
```

On conflict (same document modified locally and on server), last-write-wins — the server timestamp is authoritative.

---

### 4. Guide Fetch (Read-only)

When the user selects a guide, the CLI fetches its assets from `guides-service`. The service validates the session and then performs a defence-in-depth `CheckAppAccess` call before returning the bundle.

```
rook-cli                 guides-service                user-service
   │                          │                             │
   │──HTTPS GET /guide/{id}────►│                             │
   │  Authorization: Bearer    │                             │
   │  X-Rook-Space-ID: {id}    │                             │
   │                          │──ValidateSession(token,──────►│
   │                          │   space_id)                  │ verify token + membership
   │                          │◄──{user_id, group}──────────│
   │                          │                             │
   │                          │──CheckAppAccess──────────────►│
   │                          │  (user_id, space_id,        │ check group ACL
   │                          │   guide_id)                 │
   │                          │◄──AccessDecision{ALLOWED}───│
   │                          │                             │
   │                          │ fetch guide assets from Firestore
   │                          │ guides/ — .md content, lipgloss .yml, YAML config
   │                          │
   │◄──HTTPS 200 {guide bundle}│
   │
   │ (CLI renders guide full-screen via charmbracelet/glamour + lipgloss)
```

If `CheckAppAccess` returns `DENIED`, `guides-service` returns HTTP 403. The guide is not surfaced in the app list for that user in the first place (filtered at auth time via `GET /spaces/{id}/apps`), so this is a defence-in-depth check.

---

### 5. Guide Publish (Builder → guides-service)

When the user publishes a guide from the builder TUI, the CLI uploads the validated bundle to `guides-service`.

```
rook-cli (guide builder)  guides-service               user-service
   │                          │                             │
   │──HTTPS POST /guide────────►│                             │
   │  body: {.md, .yml,        │                             │
   │   yaml-config, meta}      │                             │
   │  Authorization: Bearer    │                             │
   │  X-Rook-Space-ID: {id}    │                             │
   │                          │──ValidateSession(token,──────►│
   │                          │   space_id)                  │ verify token + membership
   │                          │◄──{user_id, group}──────────│
   │                          │                             │
   │                          │ write guide assets to Firestore
   │                          │ set creator as guide owner
   │                          │ set initial ACL (creator's group, or as specified)
   │                          │
   │◄──HTTPS 201 {guide_id}────│
   │
   │ (guide now visible in app list for permitted groups)
```

---

## Error Handling Conventions

| gRPC status | Meaning | HTTP equivalent returned to CLI |
|-------------|---------|--------------------------------|
| `codes.NotFound` | Session token unknown, or space/guide does not exist | 404 |
| `codes.PermissionDenied` | User is not a member of the space, or ACL denies access | 403 |
| `codes.Unauthenticated` | Missing/expired session token, or invalid OIDC token on inter-service call | 401 (user token) / 500 (inter-service — caller bug) |
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
├── sessions/
│   └── {token} (doc)               # user-service — session token → user_id + expiry
│
├── auth/
│   └── nonces/
│       └── {nonce} (doc)           # user-service — {expires_at, used}; TTL policy on expires_at auto-deletes within 72h
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
- gRPC endpoints are never exposed to `rook-cli` directly — the CLI always talks HTTPS
- Services never access another service's Firestore collections — only their own namespace
- All service addresses are environment-variable-driven — see `USER_SERVICE_ADDR` and equivalents in the gRPC ADR
- OIDC bearer tokens are required on all inter-service gRPC calls in deployed (Cloud Run) environments
- `SessionAuthMiddleware` is applied to all HTTP handlers in all services — session validation is never done inside a handler
- All CLI requests must include `X-Rook-Space-ID`; `ValidateSession` resolves both user identity and space membership in one gRPC call — no second `GetSpaceMembership` call per request
- `GetUserByKey` is internal to `user-service` only — downstream services must never call it directly
