---
status: proposed
date: 2026-04-25
decision-makers: Mona Maret
consulted: Mona Maret
informed: Mona Maret
---

# rook-server Admin CLI

## Context and Problem Statement

`rook-server` exposes no web UI and has no built-in admin console. Server-side operations that require direct authority over user records, space configuration, and key registration must be performed by a server admin. The `rook-cli` first-run setup flow generates an SSH key pair locally and displays the public key for the user to provide to their admin — but there is no defined mechanism for the admin to register that key with `user-service`.

Without a defined admin interface, key registration and space management are entirely undocumented and unimplementable. What is the admin tooling model for `rook-server`?

## Decision Drivers

- Key registration must be an explicit, implementable operation — not hand-waving
- Admin operations must not be accessible to regular users via `rook-cli`
- Tooling must be usable by a solo server admin without a web browser or GUI
- Consistent with the Go + gRPC stack already established for `rook-server`
- PoC scope: simple, not production-hardened; no RBAC, no audit log, no web console

## Considered Options

- **`rook-server-cli`** — a standalone Go binary that talks gRPC to `user-service` via a protected `AdminService`; authenticated with a pre-shared admin token
- **Direct Firestore writes** — admin uses the GCP console or `gcloud` to write records directly to Firestore collections
- **Admin HTTP endpoint on `user-service`** — a protected HTTP REST endpoint on `user-service` for admin operations

## Decision Outcome

Chosen option: **`rook-server-cli`** — a dedicated Go binary for server admin operations — because it:
- Makes admin operations explicit and testable (typed gRPC contract, not freeform Firestore writes)
- Keeps admin auth separate from user auth — no risk of a misconfigured admin endpoint leaking to regular users
- Is consistent with the Go toolchain already used for `rook-server`
- Gives admins a scriptable, inspectable interface without requiring a browser or GCP console access for day-to-day tasks

Direct Firestore writes are retained as a break-glass fallback for PoC use but are not the primary admin path.

### Consequences

- Good, because key registration is now a first-class, documented, implementable operation
- Good, because admin auth is completely separate from user auth — different credential type, different service boundary
- Good, because `AdminService` RPCs are typed and versioned via Protobuf — the same compile-time safety as all other inter-service communication
- Good, because the CLI is scriptable — suitable for automated onboarding scripts
- Bad, because it adds another binary to build and deploy alongside the services
- Bad, because the admin token model is simple but not suitable for production (no rotation, no audit trail)

---

## Feature Specifications

### Admin Authentication

Admin operations are authenticated with a **pre-shared admin token** — a secret string set as an environment variable on both `user-service` and `rook-server-cli` at deploy/run time:

- `user-service` reads `ROOK_ADMIN_TOKEN` at startup; all `AdminService` RPCs require a matching token in gRPC metadata (`authorization: Bearer <token>`)
- `rook-server-cli` reads `ROOK_ADMIN_TOKEN` from its environment and attaches it as gRPC metadata on every call
- Token is never logged, never included in error responses, and never written to disk by either binary
- For local development, set `ROOK_ADMIN_TOKEN` in a `.env` file excluded from source control

This is PoC-grade auth. A production system would use a proper admin RBAC model with short-lived tokens.

### AdminService Proto

Defined in `rook-server/proto/admin/v1/admin.proto`. This is a distinct service from `UserService` — admin RPCs are never mixed into the user-facing service.

```protobuf
syntax = "proto3";

package rook.admin.v1;

option go_package = "github.com/rook-project/rook-reference/rook-server/gen/go/admin/v1;adminv1";

service AdminService {
  // Register a user's SSH public key. Creates the user record if it does not exist.
  rpc RegisterUserKey(RegisterUserKeyRequest) returns (RegisterUserKeyResponse);

  // List all registered users.
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);

  // Create a new space.
  rpc CreateSpace(CreateSpaceRequest) returns (CreateSpaceResponse);

  // List all spaces.
  rpc ListSpaces(ListSpacesRequest) returns (ListSpacesResponse);

  // Add a user to a space and assign them to a group.
  rpc AddUserToSpace(AddUserToSpaceRequest) returns (AddUserToSpaceResponse);

  // Remove a user from a space.
  rpc RemoveUserFromSpace(RemoveUserFromSpaceRequest) returns (RemoveUserFromSpaceResponse);

  // Set (or change) a user's group within a space.
  rpc SetUserGroup(SetUserGroupRequest) returns (SetUserGroupResponse);

  // List all members of a space.
  rpc ListSpaceMembers(ListSpaceMembersRequest) returns (ListSpaceMembersResponse);
}
```

`AdminService` is implemented on `user-service` alongside `UserService`. It is the only binary that exposes admin RPCs — no other service implements admin operations.

### Command Set

`rook-server-cli` is structured as a multi-level CLI using Go's `flag` package or `cobra` (evaluate at implementation time):

```
rook-server-cli user register-key --username <name> --pubkey <path-or-string>
rook-server-cli user list
rook-server-cli user add-to-space --user <id> --space <id> --group <group-name>
rook-server-cli user remove-from-space --user <id> --space <id>
rook-server-cli user set-group --user <id> --space <id> --group <group-name>

rook-server-cli space create --name <name> --description <text>
rook-server-cli space list
rook-server-cli space members --space <id>
```

All commands:
- Read `USER_SERVICE_ADDR` from the environment for the gRPC target address
- Read `ROOK_ADMIN_TOKEN` from the environment for auth
- Print structured output to stdout (human-readable table for interactive use; `--json` flag for scripting)
- Return non-zero exit codes on error

### Key Registration Flow

This is the primary use case that motivated this ADR. The full flow:

```
User (rook-cli first-run)              Admin (rook-server-cli)         user-service
        │                                      │                            │
        │ generates SSH key pair locally        │                            │
        │ displays public key to user           │                            │
        │                                      │                            │
        │ user sends public key to admin        │                            │
        │ (out of band — e.g. email, paste)     │                            │
        │                                      │                            │
        │                      rook-server-cli user register-key            │
        │                          --username alice                         │
        │                          --pubkey "ssh-ed25519 AAAA..."           │
        │                                      │                            │
        │                                      │──RegisterUserKey(key)──────►│
        │                                      │◄──{user_id: "u-abc"}───────│
        │                                      │                            │ writes to Firestore:
        │                                      │                            │ users/{u-abc}/profile
        │                                      │                            │ users/{u-abc}/keys/{fingerprint}
        │                                      │                            │
        │                      admin optionally adds user to a space:       │
        │                      rook-server-cli user add-to-space            │
        │                          --user u-abc --space s-xyz --group users │
        │                                      │                            │
        │                                      │──AddUserToSpace(...)───────►│
        │                                      │◄──AddUserToSpaceResponse───│
        │                                      │                            │ writes to Firestore:
        │                                      │                            │ users/{u-abc}/spaces/{s-xyz}
        │                                      │                            │ (group: "users")
        │                                      │                            │
        │ user runs: rook auth alice@server     │                            │
        │──HTTPS GET /auth/challenge──────────────────────────────────────►│
        │◄──nonce──────────────────────────────────────────────────────────│
        │ signs nonce with SSH private key      │                            │
        │──HTTPS POST /auth/verify { pubkey, nonce, signature }───────────►│
        │                                       │                            │ verifies signature
        │                                       │                            │ against registered key
        │◄──session token──────────────────────────────────────────────────│
        │──HTTPS GET /spaces (Bearer: token)──────────────────────────────►│
        │◄──space list + ACL-filtered app list────────────────────────────│
```

The out-of-band key exchange (user → admin) is an explicit PoC constraint. A production system would implement a self-registration flow with admin approval, but that is out of scope.

---

## Implementation Plan

- **Affected paths**:
  - `rook-server/proto/admin/v1/admin.proto` — `AdminService` definition
  - `rook-server/gen/go/admin/v1/` — generated Go stubs (committed, not gitignored)
  - `rook-server/user-service/` — implements both `UserService` and `AdminService`; admin token middleware
  - `rook-server/admin-cli/` — `rook-server-cli` binary; one `go.mod`; cobra or flag-based command tree
- **Dependencies**:
  - `rook-server/admin-cli`: `google.golang.org/grpc`, `google.golang.org/protobuf`, generated stubs from `rook-server/gen/go/admin/v1/`
  - Optional: `github.com/spf13/cobra` (evaluate vs. stdlib `flag` at implementation time — prefer stdlib if command set stays small)
- **Patterns to follow**:
  - `AdminService` is implemented in `user-service` — it is the single source of truth for all user and space data
  - Admin token is always read from `ROOK_ADMIN_TOKEN` environment variable — never hardcoded, never logged
  - `rook-server-cli` attaches the token as gRPC metadata key `authorization` with value `Bearer <token>` on every call
  - `user-service` validates the token in a unary server interceptor before any `AdminService` RPC handler runs
  - All Firestore writes for admin operations go through `user-service` — `rook-server-cli` never writes to Firestore directly
  - `--json` flag on all commands emits machine-readable JSON to stdout for scripting
  - Non-zero exit code on any gRPC error; error message to stderr
- **Patterns to avoid**:
  - Do not expose `AdminService` RPCs to `rook-cli` — admin operations are never available to end users
  - Do not share the admin token with any other service — it is only known to `user-service` and `rook-server-cli`
  - Do not implement admin operations as HTTP endpoints on `user-service` — keep all inter-binary communication on gRPC
  - Do not write to Firestore from `rook-server-cli` directly — always go through `user-service`
  - Do not log or print the admin token in any output, error, or debug message
- **Configuration**:
  - `USER_SERVICE_ADDR` — gRPC address of `user-service` (e.g. `user-service-xyz-uc.a.run.app:443`)
  - `ROOK_ADMIN_TOKEN` — pre-shared admin secret; required by both `user-service` and `rook-server-cli`
- **Migration steps**: N/A — greenfield

### Verification

- [ ] `rook-server-cli` builds from source with `go build` in `rook-server/admin-cli/`
- [ ] `rook-server-cli user register-key` registers a public key and creates a user record in Firestore via `user-service`
- [ ] After `register-key`, the user can authenticate via `rook auth user@server`; `POST /auth/verify` succeeds with the registered key and returns a session token
- [ ] `rook-server-cli user list` returns all registered users in human-readable and `--json` formats
- [ ] `rook-server-cli user add-to-space` assigns the user to a space and group; subsequent `GetSpaceMembership` gRPC call returns the correct group
- [ ] `rook-server-cli user set-group` updates a user's group within a space without removing their membership
- [ ] `rook-server-cli space create` creates a space record in Firestore
- [ ] `rook-server-cli space members` lists all members of a space with their groups
- [ ] All commands fail with a non-zero exit code and a clear error message to stderr when `USER_SERVICE_ADDR` or `ROOK_ADMIN_TOKEN` is unset
- [ ] All commands fail with HTTP 401 / gRPC `UNAUTHENTICATED` when an incorrect admin token is provided
- [ ] A regular `rook-cli` user cannot call `AdminService` RPCs — the token interceptor on `user-service` rejects calls without a valid admin token
- [ ] `ROOK_ADMIN_TOKEN` does not appear in any log output from `user-service` or `rook-server-cli`
- [ ] `buf lint` passes on `rook-server/proto/admin/v1/admin.proto` with zero errors

## More Information

### Interaction with other ADRs

- **Refines**: [`2026-04-25-rook-reference-system-architecture.md`](2026-04-25-rook-reference-system-architecture.md) — adds `rook-server-cli` as an explicit system component; resolves the SSH key registration gap
- **Depends on**: [`2026-04-25-grpc-inter-service-communication.md`](2026-04-25-grpc-inter-service-communication.md) — `rook-server-cli` uses the same gRPC + Protobuf + `buf` toolchain and generated stub conventions
- **Closes gap in**: [`2026-04-25-rook-cli-features-and-ux-architecture.md`](2026-04-25-rook-cli-features-and-ux-architecture.md) — the first-run setup flow displays the public key "for the user to register with a server admin"; this ADR defines the admin side of that handshake

### Deferred Decisions

**1. CLI framework** — `cobra` vs. stdlib `flag`. Evaluate at implementation time. Prefer stdlib if the command set stays at ≤ 8 commands; adopt `cobra` if subcommand depth or flag complexity justifies it.

**2. Self-registration** — Whether to add a user-initiated key submission flow (user submits key to `user-service`; admin approves via `rook-server-cli user approve-key`) is deferred. The out-of-band exchange is sufficient for PoC and avoids an unauthenticated write surface on `user-service`.
