# Feature Specification: Authentication Foundation

**Feature Branch**: `002-auth-foundation`  
**Created**: 2026-04-26  
**Status**: Draft  
**Input**: User description: "v0.2 → Authentication Foundation — SSH-key-based auth flow, rook auth command, user-service skeleton, lazy auth model, Docker Compose local dev environment"

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Launch rook and access local content without authenticating (Priority: P1)

A developer opens `rook` on their machine. The application immediately displays their local stash files and a menu of available actions — no login prompt, no network access required. They can navigate local content freely without ever providing credentials.

**Why this priority**: The offline-first, no-barrier-to-entry launch is the foundational UX contract for every release. If a user cannot access local content instantly, the product fails its core promise.

**Independent Test**: Can be fully tested by running `rook` with no server reachable and verifying the stash file list and action menu appear without any authentication prompt or error.

**Acceptance Scenarios**:

1. **Given** a user has `rook` installed and configured, **When** they run `rook` with no network access, **Then** the local stash list and action menu appear immediately with no login prompt
2. **Given** a user has no stored session, **When** they run `rook`, **Then** the application opens to the local launcher without requesting credentials
3. **Given** a user selects a local-only action from the menu, **When** they confirm the action, **Then** it completes without triggering any authentication flow

---

### User Story 2 — Authenticate with a registered SSH key (Priority: P2)

A developer wants to sync their stash with the server. When they select a server-connected action, `rook` detects they have no active session and prompts them to authenticate. They see a list of their SSH keys, select one, and `rook` performs the challenge/verify handshake automatically. On success, a confirmation is shown and the server action proceeds.

**Why this priority**: The challenge/verify SSH auth flow is the primary deliverable of this release. Without it, no server-connected features in subsequent releases can be built or tested.

**Independent Test**: Can be fully tested by running `rook auth` directly against a local dev stack (Docker Compose), completing the interactive key selection and handshake, and verifying a success confirmation is displayed.

**Acceptance Scenarios**:

1. **Given** a user with a registered SSH public key selects a server-connected action, **When** the lazy auth prompt appears, **Then** `rook` presents their available SSH keys for selection
2. **Given** a user selects an SSH key from the list, **When** they confirm, **Then** `rook` fetches a challenge, signs it with the selected key, and submits the signed response to the server
3. **Given** the signature is valid, **When** the server responds with a session token, **Then** `rook` confirms authentication success and proceeds with the original action
4. **Given** the user has exactly one SSH key available, **When** auth is triggered, **Then** `rook` skips the selection prompt and proceeds directly with that key
5. **Given** the server returns a signature mismatch error, **When** the verify call fails, **Then** `rook` displays a clear, actionable error message and exits the auth flow without crashing

---

### User Story 3 — Check auth status and log out (Priority: P3)

A developer runs `rook auth status` to see whether they have an active session, and `rook auth logout` to end it. Both commands produce clear, informational responses.

**Why this priority**: These commands complete the `rook auth` subcommand tree so shell completions and help text are coherent from v0.2. Their full implementation is deferred; correct stubs are sufficient this release.

**Independent Test**: Can be fully tested by running each command and verifying the response text is present, non-empty, and non-error.

**Acceptance Scenarios**:

1. **Given** a user runs `rook auth status`, **When** no session is active (in-memory sessions are always lost at process exit), **Then** the command prints "no active session" and exits 0
2. **Given** a user runs `rook auth logout`, **When** no session token exists locally, **Then** the command prints "logged out" and exits 0

---

### User Story 4 — Run the full auth flow against a local dev stack (Priority: P2)

A developer (or CI) stands up the local Docker Compose environment, seeds a test public key into the Firestore emulator, points `rook-cli` at the local stack, and runs `rook auth`. The end-to-end challenge/verify flow completes successfully against locally-running services — no cloud account or production credentials required.

**Why this priority**: The local dev environment is required for all v0.2 development and testing. Without it, engineers cannot develop or verify any server-connected feature.

**Independent Test**: Can be fully tested by following the usage guide from a clean checkout with only Docker and Go installed, completing the auth flow, and verifying the session token is issued.

**Acceptance Scenarios**:

1. **Given** a developer has Docker and Go installed, **When** they follow the usage guide from a clean checkout, **Then** `docker compose up` starts `user-service` and the Firestore emulator within 2 minutes
2. **Given** the stack is running, **When** a developer follows the seed instructions to register a test public key, **Then** the key is queryable by `user-service` via the emulator
3. **Given** a registered test key is seeded, **When** the developer runs `rook auth --key <path>`, **Then** the challenge/verify handshake completes and a session token is returned
4. **Given** an unregistered key is used, **When** `rook auth` submits the verify request, **Then** the server rejects the request and `rook` displays an actionable error message

---

### Edge Cases

- What happens when `~/.ssh/` contains no private keys? → Auth fails gracefully with a message directing the user to generate or specify a key; the application does not crash.
- What happens when the nonce expires (server-side TTL) before the user selects a key? → The verify call fails; `rook` detects the expired-nonce error and offers to retry the auth flow.
- What happens when the Firestore emulator is unreachable? → `rook` surfaces a connection error with the configured server address and exits the auth flow; local features remain available.
- What happens when the selected SSH key requires a passphrase and `ssh-agent` is not running? → `rook` returns an actionable error; passphrase-protected keys without agent support are out of scope for v0.2.
- What happens when `docker compose up` is run on Windows? → Not supported; the usage guide explicitly states Mac/Linux only.

---

## Requirements *(mandatory)*

### Functional Requirements

**CLI command dispatch**

- **FR-001**: The CLI MUST use a structured command-dispatch model with `cmd/root.go` as the entry point, replacing any hand-rolled flag parsing from v0.1
- **FR-002**: The root command MUST launch the local TUI (stash file list and action menu) immediately on invocation, with no authentication check
- **FR-003**: Server-connected actions MUST trigger an inline authentication prompt if no active session exists — auth MUST NOT be demanded at launch

**rook auth command**

- **FR-004**: `rook auth` MUST scan `~/.ssh/` for private key files and present them in an interactive selection list
- **FR-005**: `rook auth` MUST skip the selection prompt when exactly one key is found and proceed with that key automatically
- **FR-006**: `rook auth` MUST accept a `--key <path>` flag to bypass key discovery for non-interactive use
- **FR-007**: `rook auth` MUST fetch a server-issued challenge, sign it using the selected private key, and submit the signed response to the server
- **FR-008**: On successful verification, `rook auth` MUST display a confirmation message and hold the session token in-memory for the process lifetime
- **FR-009**: On failure (expired nonce, signature mismatch, unregistered key), `rook auth` MUST display a clear, user-actionable error message and exit without crashing
- **FR-010**: `rook auth status` MUST print an informational "no active session" message and exit successfully (stub implementation)
- **FR-011**: `rook auth logout` MUST print an informational "logged out" message and exit successfully (stub implementation)

**user-service**

- **FR-012**: `user-service` MUST expose a challenge endpoint that generates a cryptographically random nonce (≥ 32 bytes) and stores it with a 60-second expiry
- **FR-013**: `user-service` MUST expose a verify endpoint that accepts a public key, nonce, and signature; validates the signature against a stored registered public key; and issues a session token on success
- **FR-014**: `user-service` MUST reject verify requests where the nonce has expired, returning an appropriate error response
- **FR-015**: `user-service` MUST reject verify requests where the public key is not registered, returning an appropriate error response
- **FR-016**: `user-service` MUST reject verify requests where the signature does not match the public key, returning an appropriate error response
- **FR-017**: Public key registration MUST be possible by directly seeding the data store in the local dev environment — no user-facing registration flow is required for this release

**Local dev environment**

- **FR-018**: A Docker Compose configuration MUST be provided that starts `user-service` and a Firestore emulator with a single command
- **FR-019**: A usage guide MUST document: prerequisites, stack startup, public key seeding, CLI configuration to target the local stack, and end-to-end auth verification steps
- **FR-020**: The Docker Compose environment MUST support Mac and Linux; Windows is explicitly out of scope

### Key Entities

- **SSH Key Pair**: A private/public key pair on the user's machine; identified by file path and fingerprint; the public key is registered server-side; the private key never leaves the user's machine
- **Nonce**: A short-lived cryptographically random value issued by `user-service` to a specific authentication attempt; expires after 60 seconds; single-use
- **Session Token**: An opaque random value issued by `user-service` on successful verification; held in-memory by `rook-cli` for the process lifetime; used to authorise server-connected requests within the session
- **Registered Public Key**: A public key stored server-side and associated with a user identity; the basis for verifying challenge signatures; must be pre-seeded for v0.2
- **User Identity**: A server-side record associating one or more registered public keys with a user account

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can run `rook` and reach the local stash launcher in under 2 seconds with no network access and no authentication prompt
- **SC-002**: A developer can complete the full `rook auth` interactive flow (key selection → challenge → sign → verify → confirmation) in under 30 seconds on a running local dev stack
- **SC-003**: The local Docker Compose environment starts and is ready to accept auth requests within 2 minutes of running `docker compose up` on a clean machine with Docker installed
- **SC-004**: All three `rook auth` error conditions (expired nonce, signature mismatch, unregistered key) produce distinct, non-empty, human-readable error messages — no raw stack traces or opaque codes presented to the user
- **SC-005**: The full end-to-end auth flow can be executed from a clean checkout using only the usage guide, without requiring access to any cloud account or production credentials
- **SC-006**: `rook auth status` and `rook auth logout` both exit with code 0 and produce non-empty output under all conditions reachable in this release

---

## Assumptions

- Users have an existing SSH key pair in `~/.ssh/` — key generation is out of scope for this release (deferred to PRD004 first-run setup flow)
- Passphrase-protected SSH keys require `ssh-agent` to be running and the key loaded; direct passphrase prompt is out of scope for v0.2
- Public key registration is performed manually by seeding the Firestore emulator directly — no self-registration or admin CLI command exists in this release
- Session tokens are ephemeral and in-memory only; the user authenticates on each new process invocation that requires a server-connected action
- The `user-service` skeleton is intentionally speculative — its directory layout and service wiring are established to support CI and Docker Compose, but the full service specification (spaces, groups, ACL) is owned by a future release
- `rook auth status` and `rook auth logout` are correct stubs that complete the subcommand tree; full implementation is deferred until session persistence is introduced
- The Firestore emulator is used exclusively for local development and testing; no cloud Firestore project or service account credentials are required for v0.2 development
- Mac and Linux are the only supported development platforms; Windows is explicitly unsupported
- Guides are server-connected only in v0.2–v0.4; offline guide sync is introduced in v0.5 (PRD006), at which point guides join the lazy-auth trigger set
