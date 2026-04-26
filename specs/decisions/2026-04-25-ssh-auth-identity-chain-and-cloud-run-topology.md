---
status: proposed
date: 2026-04-25
decision-makers: Mona Maret
consulted: Mona Maret
informed: Mona Maret
---

# SSH Auth Identity Chain and Cloud Run Deployment Topology

## Context and Problem Statement

The system architecture ADR ([`2026-04-25-rook-reference-system-architecture.md`](2026-04-25-rook-reference-system-architecture.md)) establishes that:

- Users authenticate via SSH key through `charmbracelet/wish`
- `rook-server` services are deployed to Google Cloud Run
- The messaging and sync services expose HTTP endpoints

Two gaps left unresolved in that ADR block auth and deployment implementation:

**Gap 1 — Identity chain**: After an SSH-key-based auth event, what credential does `rook-cli` attach to subsequent HTTP sync calls to Cloud Run services? Without an explicit identity-carrying credential, services have no way to verify who is making a sync call.

**Gap 2 — Deployment topology**: Cloud Run serves HTTP/1.1 and HTTP/2 on a single configurable port (default 8080). It does not support raw TCP — the transport required by SSH. `charmbracelet/wish` is an SSH server and cannot run directly on Cloud Run. This creates a hard incompatibility between the system architecture ADR's use of `charmbracelet/wish` and the chosen deployment target.

What is the correct deployment topology and auth identity chain for a system where all services must run on Cloud Run?

## Decision Drivers

- Cloud Run does not support raw TCP — SSH requires raw TCP (port 22); `charmbracelet/wish` cannot run on Cloud Run
- All Cloud Run services are HTTPS automatically on `*.run.app` — no cert management required
- `rook-cli` is the only client; there is no browser or third-party consumer of the auth layer
- The CLI already holds the user's SSH private key — it is the natural anchor of the identity chain and must remain so
- Auth state must be stateless on the server side — no server-side session store
- Session credential storage on the CLI must be ephemeral (session-scoped)
- Operational simplicity is preferred over production hardening at PoC scope — no additional managed infrastructure beyond Cloud Run
- The system architecture ADR requires portability from Cloud Run to Kubernetes — the topology must not foreclose that path
- `rook-server-cli` (admin CLI) connects directly to `user-service` via gRPC with `ROOK_ADMIN_TOKEN` — its auth path is already resolved and is unaffected by this decision

## Considered Options

- **Option A — Compute Engine VM for `wish` + Cloud Run for HTTP services**: Run `charmbracelet/wish` on a Compute Engine VM exposed via a TCP Load Balancer on port 22; all HTTP sync services remain on Cloud Run; identity chain gap remains unresolved
- **Option B — HTTP-only on Cloud Run, SSH private key request signing**: Remove `charmbracelet/wish` from the server boundary entirely; all services run on Cloud Run; the CLI signs each HTTP request with its SSH private key using a standard signing scheme; services validate signatures against the registered public key fetched from `user-service`
- **Option C — Compute Engine VM for `wish` issues a short-lived JWT; CLI uses JWT for HTTP sync calls**: `wish` runs on a VM; after the SSH handshake the VM issues a signed JWT bound to the SSH key fingerprint; the CLI caches the JWT for the session and attaches it as `Authorization: Bearer` on all HTTP calls

## Decision Outcome

Chosen option: **Option B — HTTP-only on Cloud Run, SSH private key request signing**, because:

- It is the only option that keeps every service on Cloud Run with no additional managed infrastructure — no VM to provision, patch, or monitor
- The CLI already holds the user's SSH private key; signing HTTP requests with it is a direct, coherent use of the existing credential with no new secret material
- Signature validation is stateless on the server — `user-service` looks up the registered public key in Firestore and verifies; no session store, no shared secret to rotate
- HTTPS on Cloud Run is automatic on `*.run.app` — no certificate management required
- The topology (all services on Cloud Run) maps directly to Kubernetes deployments — portability is preserved and simplified
- `charmbracelet/wish` and `charmbracelet/wishlist` are removed from the server boundary; their responsibilities (identity verification and app discovery) are replaced by authenticated HTTP endpoints on `user-service`, which is simpler to deploy and test on Cloud Run

Option A was rejected because it introduces a Compute Engine VM as persistent infrastructure with a baseline cost, OS patching burden, and uptime responsibility — all of which contradict the operational simplicity driver — and it still leaves the identity chain gap unresolved.

Option C was rejected for the same infrastructure reasons as Option A, with the added complexity of a shared JWT secret that must be provisioned and rotated manually across the VM and all Cloud Run services.

### Deployment Topology

All `rook-server` services, including `user-service`, run on Cloud Run. There is no separate SSH entry point. The CLI interacts with the server exclusively over HTTPS.

```
┌──────────────────────────────────────────────────────────────────┐
│  rook-cli (local)                                                │
│                                                                  │
│  1. First connection:                                            │
│     GET /auth/challenge  ──HTTPS──►  user-service (Cloud Run)   │
│     ◄── nonce (random, short-lived)                              │
│                                                                  │
│  2. CLI signs nonce with SSH private key                         │
│     POST /auth/verify { public_key, signature, nonce }          │
│     ──HTTPS──►  user-service                                     │
│       - fetches registered public key from Firestore             │
│       - verifies signature against nonce                         │
│     ◄── session token (short-lived, opaque, stored in Firestore) │
│                                                                  │
│  3. All subsequent requests:                                     │
│     Authorization: Bearer <session-token>                        │
│     ──HTTPS──►  any Cloud Run service                            │
│       - middleware calls user-service.ValidateSession(token)     │
│       - returns user ID on success                               │
│                                                                  │
│  4. App/space discovery:                                         │
│     GET /spaces  ──HTTPS──►  user-service                        │
│     ◄── space list + per-space app ACL (replaces wishlist)       │
└──────────────────────────────────────────────────────────────────┘
```

**HTTPS on Cloud Run**: All Cloud Run services are automatically served over HTTPS on their `*.run.app` domain with a Google-managed TLS certificate. TLS terminates at the Google Front End; services receive plain HTTP internally on port 8080. No certificate management, load balancer, or sidecar required. Custom domain HTTPS requires a domain mapping (`gcloud run domain-mappings create`) and a DNS CNAME — not needed for the PoC.

### Request Signing Scheme

The CLI authenticates its identity using a **challenge-response protocol** over HTTPS, then uses the resulting session token for all subsequent requests within a session. This replaces the SSH handshake that `charmbracelet/wish` previously provided.

**Challenge-response flow:**

1. `rook-cli` calls `GET /auth/challenge` on `user-service`, receiving a random nonce (32 bytes, hex-encoded, TTL: 60 seconds)
2. The CLI signs the nonce with its SSH private key using `ssh.Sign()` from `golang.org/x/crypto/ssh`
3. The CLI calls `POST /auth/verify` with the public key (PEM or OpenSSH wire format), the nonce, and the signature
4. `user-service` fetches the registered public key for the claimed identity from Firestore, verifies the signature using `ssh.ParsePublicKey` + `pubKey.Verify()`, and confirms the nonce has not expired or been used
5. On success, `user-service` creates a session token (random, 32-byte, hex-encoded), stores it in Firestore under `sessions/{token}` with the user ID and a 1-hour TTL, and returns it to the CLI
6. The CLI caches the session token in-memory for the duration of the process and attaches it as `Authorization: Bearer <token>` on all subsequent HTTPS requests

**Nonce storage**: Nonces are stored in Firestore under `auth/nonces/{nonce}` with a 60-second TTL and marked used on first verification — preventing replay attacks.

**Session token properties:**

| Property | Value |
|----------|-------|
| Format | Random 32-byte value, hex-encoded |
| Storage (server) | Firestore `sessions/{token}` — user ID + expiry |
| Storage (CLI) | In-memory only; discarded on process exit |
| TTL | 1 hour |
| Revocation | Delete the Firestore document; next request returns `401` |

**Signing library**: `golang.org/x/crypto/ssh` — part of the Go extended standard library, already a transitive dependency of `charmbracelet/wish` in the ecosystem; no new external dependency class is introduced. Supports `rsa`, `ecdsa`, and `ed25519` keys — all key types the CLI's first-run setup may generate.

### App and Space Discovery (Replacing `charmbracelet/wishlist`)

`charmbracelet/wishlist` performed space-scoped, ACL-filtered app discovery over the SSH connection. With Option B, this is replaced by an authenticated HTTP endpoint on `user-service`:

- `GET /spaces` — returns the list of spaces the authenticated user belongs to, with per-space metadata
- `GET /spaces/{space-id}/apps` — returns the ACL-filtered list of apps available to the user's group in that space

The CLI calls these endpoints after a successful `/auth/verify` exchange, caches the result locally in `.json` (same cache location as currently specced: `<storage-dir>/cache/`), and refreshes on each new session. The data model is identical to what `wishlist` would have returned — only the transport changes from SSH to HTTPS.

### Session Validation Middleware

Every Cloud Run service that handles CLI requests wraps its HTTP handlers with a shared validation middleware:

1. Extract `Authorization: Bearer <token>` header; reject with `401` if absent
2. Extract `X-Rook-Space-ID` header; reject with `400` if absent (all CLI requests must include a space ID)
3. Call `user-service.ValidateSession(token, space_id)` via gRPC (internal Cloud Run service-to-service call, OIDC-authenticated per the gRPC ADR)
4. On success, inject both the **user ID** and **space membership** (group) into the request `context.Context` via typed keys — handlers receive both without a second gRPC call
5. On failure (`token` not found or expired), return `401 Unauthorized`
6. On space non-membership (`PERMISSION_DENIED` from `ValidateSession`), return `403 Forbidden`

`ValidateSession` subsumes the previous `GetSpaceMembership` pattern — handlers never make a separate space membership call. This eliminates one gRPC round-trip per request across all services.

The middleware is implemented once in `rook-server/internal/middleware/` and imported by all Cloud Run services. Per-handler validation is prohibited.

### Consequences

- Good, because all services run on Cloud Run — no VM, no TCP load balancer, no persistent infrastructure to operate
- Good, because Cloud Run scales to zero, keeping PoC costs minimal across all services including auth
- Good, because the CLI's SSH private key remains the single credential anchor — no new secret material is introduced
- Good, because signature validation and session lookup are stateless operations from the service's perspective — session state lives in Firestore, consistent with the data layer decision
- Good, because HTTPS on Cloud Run requires zero additional infrastructure for the PoC
- Good, because the all-Cloud-Run topology maps cleanly to all-Kubernetes-Deployment for the portability requirement
- Good, because removing `charmbracelet/wish` and `charmbracelet/wishlist` from the server eliminates a dependency on SSH transport entirely, simplifying local development and debugging
- Bad, because `charmbracelet/wish` is removed from the server boundary, which supersedes the system architecture ADR's explicit choice of the Charmbracelet SSH ecosystem for auth — that ADR must be updated
- Bad, because the challenge-response auth flow requires two HTTP round-trips before the first authenticated request, versus one SSH handshake
- Bad, because nonce storage in Firestore adds a write on every new session initiation — negligible at PoC scale
- Neutral, because `ValidateSession` adds a gRPC round-trip on every authenticated request — acceptable at PoC scale; an in-process cache keyed on the token (TTL: 5 minutes) can be added if latency is a concern

## Implementation Plan

### Affected paths

- `rook-server/user-service/handlers/auth.go` — new HTTP handlers: `GET /auth/challenge`, `POST /auth/verify`; nonce generation, storage, and expiry; signature verification via `golang.org/x/crypto/ssh`; session token creation and Firestore storage
- `rook-server/user-service/handlers/spaces.go` — new HTTP handlers: `GET /spaces`, `GET /spaces/{space-id}/apps`; returns space membership and ACL-filtered app list for the authenticated user (replaces `charmbracelet/wishlist` app discovery)
- `rook-server/user-service/handlers/session.go` — `ValidateSession` gRPC RPC (or internal HTTP handler); used by the shared middleware in other services to validate a session token
- `rook-server/proto/user/v1/user.proto` — add `ValidateSession(ValidateSessionRequest) returns (ValidateSessionResponse)` RPC to `UserService`; `ValidateSessionRequest` includes `token` and `space_id`; `ValidateSessionResponse` returns `user_id` and `group` (space membership); re-run `buf generate` after
- `rook-server/internal/middleware/` — new shared Go package; exports `SessionAuthMiddleware(userServiceClient UserServiceClient) func(http.Handler) http.Handler`; applied to all HTTP handlers in all Cloud Run services
- `rook-server/internal/middleware/session.go` — extracts `Authorization: Bearer` and `X-Rook-Space-ID` headers; calls `user-service.ValidateSession(token, space_id)`; injects both user ID and group (space membership) into `context.Context` via typed keys; rejects with `401` on missing/expired token, `400` on missing space ID, `403` on space non-membership
- `rook-server/messaging-service/main.go` — apply `SessionAuthMiddleware`
- `rook-server/stash-service/main.go` — apply `SessionAuthMiddleware`
- `rook-server/guides-service/main.go` — apply `SessionAuthMiddleware`
- `rook-cli/auth/` — implement challenge-response auth flow: `GET /auth/challenge`, sign nonce with SSH private key, `POST /auth/verify`, cache session token in-memory
- `rook-cli/auth/signer.go` — SSH private key loading and signing using `golang.org/x/crypto/ssh`
- `rook-cli/spaces/` — replace `charmbracelet/wishlist` negotiation with `GET /spaces` and `GET /spaces/{space-id}/apps` HTTP calls; cache result to `<storage-dir>/cache/spaces.json`

### Dependencies

Add to `rook-server/user-service/go.mod` and `rook-cli/go.mod`:

```
golang.org/x/crypto    # ssh package — key parsing, signing, and signature verification
```

No new dependency class: `golang.org/x/crypto` is part of the Go extended standard library and is already a transitive dependency throughout the Charmbracelet ecosystem.

Add to `rook-server/internal/middleware/` (shared, imported by each service):

```
google.golang.org/grpc    # already required per the gRPC ADR
```

**Remove** from `rook-server/` service(s) and `rook-cli/`:

```
charmbracelet/wish        # no longer used on the server boundary
charmbracelet/wishlist    # replaced by authenticated HTTP endpoints on user-service
```

### Patterns to follow

- `SessionAuthMiddleware` is the single source of truth for request authentication across all Cloud Run services — never duplicate session validation logic in handlers
- All env vars are read at startup and injected via constructor parameters or a config struct — never call `os.Getenv` inside a request handler or middleware
- Both the user ID and group (space membership) extracted by the middleware are passed to handlers via `context.Context` using typed unexported keys — handlers must not re-validate, re-parse the token, or make a separate `GetSpaceMembership` call
- All CLI requests to Cloud Run services must include `X-Rook-Space-ID`; the middleware rejects requests missing this header with `400 Bad Request`
- Nonces are single-use — mark as used in Firestore atomically on first successful verification; a second use of the same nonce returns `401`
- Session tokens are random and opaque — no embedded claims; all session data lives in Firestore
- Cloud Run service-to-service calls (`SessionAuthMiddleware` → `user-service`) use OIDC tokens as specified in [`2026-04-25-grpc-inter-service-communication.md`](2026-04-25-grpc-inter-service-communication.md) — this is the machine identity layer, separate from the user session token

### Patterns to avoid

- Do not validate session tokens inside individual HTTP handler functions — always use `SessionAuthMiddleware`
- Do not store the session token on disk in `rook-cli` — in-memory only; discard on process exit
- Do not embed user claims in the session token — it is opaque; all user data is resolved server-side via `ValidateSession`
- Do not reuse nonces — once verified, a nonce is permanently invalidated regardless of TTL
- Do not use `charmbracelet/wish` or `charmbracelet/wishlist` on the server — they are no longer part of the server boundary
- Do not use `grpc.WithInsecure()` for any service-to-service call in the deployed environment

### Configuration

| Component | Variable | Purpose |
|-----------|----------|---------|
| `user-service` | `USER_SERVICE_ADDR` | Self-referential gRPC address (used by middleware in other services) |
| All Cloud Run services | `USER_SERVICE_ADDR` | gRPC address of `user-service` for `ValidateSession` calls |
| All Cloud Run services | (none new) | HTTPS is automatic on Cloud Run — no TLS config required |

No shared secrets are required by this option. The only credential is the user's SSH private key, which never leaves the CLI.

### Migration steps

N/A — greenfield. No existing service is being replaced; `charmbracelet/wish` and `charmbracelet/wishlist` were planned dependencies that are now removed before any implementation begins.

**Required update**: The system architecture ADR ([`2026-04-25-rook-reference-system-architecture.md`](2026-04-25-rook-reference-system-architecture.md)) must be updated to:
- Remove `charmbracelet/wish` and `charmbracelet/wishlist` from the server-side dependency list and auth description
- Replace the auth description with: "SSH key-based challenge-response over HTTPS; session tokens stored in Firestore"
- Replace the wishlist description with: "authenticated HTTP endpoints on `user-service` return space membership and ACL-filtered app lists"
- Note this ADR as the resolving decision for the auth topology gap

### Verification

- [ ] `GET /auth/challenge` returns a unique 32-byte hex nonce; a second call returns a different nonce
- [ ] `POST /auth/verify` with a valid signature returns a session token; subsequent use of that token on any service succeeds with `200`
- [ ] `POST /auth/verify` with an invalid signature returns `401 Unauthorized`
- [ ] `POST /auth/verify` with an expired nonce (>60 seconds old) returns `401 Unauthorized`
- [ ] `POST /auth/verify` with a replayed (already-used) nonce returns `401 Unauthorized`
- [ ] A Cloud Run service rejects a request with no `Authorization` header with `401 Unauthorized`
- [ ] A Cloud Run service rejects a request with an expired session token with `401 Unauthorized`
- [ ] A Cloud Run service rejects a request with an unknown session token with `401 Unauthorized`
- [ ] A valid session token and matching `X-Rook-Space-ID` allows a request to proceed; the handler receives both the authenticated user ID and group (space membership) via `context.Context` — no additional gRPC call required in the handler
- [ ] A request missing `X-Rook-Space-ID` is rejected with `400 Bad Request`
- [ ] A request with a valid token but a space the user does not belong to is rejected with `403 Forbidden`
- [ ] Session token is held in-memory by `rook-cli` for the session duration; it is not written to disk
- [ ] `GET /spaces` returns only spaces the authenticated user belongs to
- [ ] `GET /spaces/{space-id}/apps` returns only apps the user's group ACL permits in that space
- [ ] All Cloud Run services are reachable over HTTPS on their `*.run.app` domain with no additional TLS configuration
- [ ] `charmbracelet/wish` and `charmbracelet/wishlist` are not imported anywhere in `rook-server/` — verified by `grep -r "charmbracelet/wish" rook-server/ --include="*.go"` returning zero results
- [ ] No SSH private key material or session token is written to any file on disk by `rook-cli`
- [ ] `rook-cli` treats a `401` response as a session expiry signal and re-runs the challenge-response flow before retrying

## Pros and Cons of the Options

### Option A — Compute Engine VM for `wish` + Cloud Run for HTTP (identity chain unresolved)

- Good, because `charmbracelet/wish` runs on infrastructure designed for raw TCP
- Good, because Cloud Run services handle HTTP natively with automatic HTTPS
- Bad, because a Compute Engine VM is persistent infrastructure with baseline cost, OS patching burden, and uptime responsibility
- Bad, because this option leaves the identity chain gap unresolved — the SSH session authenticates the user to the VM, but HTTP services still have no credential to validate
- Bad, because introducing a VM breaks the scale-to-zero property of the full system

### Option B — HTTP-only on Cloud Run, SSH private key request signing (chosen)

- Good, because all services run on Cloud Run — no VM, no persistent infrastructure
- Good, because the full system scales to zero; PoC cost is minimal
- Good, because the CLI's SSH private key is the only credential — no new secret material
- Good, because validation is stateless on the service side — session state is in Firestore
- Good, because the topology maps directly to Kubernetes deployments
- Good, because local development and debugging use standard HTTP tooling (`curl`, `httpie`)
- Bad, because `charmbracelet/wish` and `charmbracelet/wishlist` are removed from the server boundary — the system architecture ADR must be updated to reflect this
- Bad, because the challenge-response flow requires two HTTP round-trips before the first authenticated request
- Neutral, because `ValidateSession` adds a gRPC call per request (mitigable with an in-process cache)

### Option C — Compute Engine VM (`wish`) + short-lived JWT

- Good, because SSH auth model and Charmbracelet ecosystem are preserved at the server boundary
- Good, because JWT gives services a stateless, self-contained credential per request
- Bad, because it inherits all the VM infrastructure drawbacks of Option A
- Bad, because a shared `ROOK_JWT_SECRET` must be provisioned and rotated across the VM and all Cloud Run services
- Bad, because adding a VM solely to preserve `charmbracelet/wish` is disproportionate for a PoC where `wish` provides no end-user-visible benefit over HTTPS

## More Information

- **Supersedes (partially)**: [`2026-04-25-rook-reference-system-architecture.md`](2026-04-25-rook-reference-system-architecture.md) — the server-side auth description (`charmbracelet/wish`, `charmbracelet/wishlist`) and the dependency list must be updated to reflect Option B. The SSH key identity model (key generation in CLI, key storage in Firestore, key registration via `rook-server-cli`) is unchanged.
- **Depends on**: [`2026-04-25-grpc-inter-service-communication.md`](2026-04-25-grpc-inter-service-communication.md) — `SessionAuthMiddleware` calls `user-service.ValidateSession` via gRPC with OIDC tokens; this is the established inter-service communication pattern
- **SSH signing library**: `golang.org/x/crypto/ssh` — Go extended standard library; supports `rsa`, `ecdsa`, and `ed25519` SSH keys; `pubKey.Verify(data, sig)` is the server-side validation call
- **Cloud Run HTTPS**: automatic on `*.run.app`; see https://cloud.google.com/run/docs/mapping-custom-domains for custom domain setup (not needed for the PoC)
- **Cloud Run TCP limitation**: https://cloud.google.com/run/docs/container-contract — Cloud Run containers must listen on a single HTTP/HTTPS port; raw TCP is not supported; this is the root constraint that drives Option B
- **Revisit if**: (1) a web UI or third-party client is added — the challenge-response scheme may need to be replaced with OAuth 2.0 / OIDC; (2) Kubernetes migration occurs — the all-Cloud-Run topology maps cleanly to all-Deployment with no topology changes required
