# Rook v0.2 — Authentication Foundation

**ID:** PRD003  
**Version:** v0.2  
**Status:** Refined  
**Date:** 2026-04-25  
**Updated:** 2026-04-26  
**Author:** Mona Maret  
**Parent:** [PRD001 — Rook Project Overview for PoC](PRD001-rook-overview-v1.0.md)  

---

## Overview

This release delivers the core SSH-key-based authentication flow between `rook-cli` and `user-service`. It implements the challenge/verify handshake endpoints in `rook-server`, Firestore-backed nonce and session storage, and the `rook auth` CLI command. It also introduces the cobra + fang command dispatch layer that all subsequent releases build on, and a Docker Compose local dev environment for containerised testing.

After this release, a developer can run `rook` to immediately access local TUI functionality, and can authenticate against a locally-running `user-service` to test server-connected flows. Auth is lazy — demanded only at the point a server-connected action is attempted, not at launch.

---

## Feature Focus

### `rook-cli` — cobra + fang wiring (new in v0.2)

- Replace hand-rolled `main.go` flag parsing with `cmd/root.go` + `fang.Execute(ctx, cmd.Root())`
- Root command `RunE` launches the local TUI (stash file list + menu) immediately — no auth required
- Auth is demanded lazily when a server-connected action is selected; the CLI prompts inline
- Session token is held **in-memory only** for the process lifetime; discarded on exit
- No session persistence to disk in this release — the user authenticates on each server-connected session

### `rook-cli` — `rook auth` command

- `rook auth` — interactive SSH key selection (Bubble Tea list), challenge fetch, sign, verify, confirm success
  - Key discovery: scan `~/.ssh/` for `id_*` files; display filename, type, and fingerprint
  - Single-key fast path: skip prompt if only one key found
  - `--key <path>` flag: bypass discovery for scripted/non-interactive use
- `rook auth status` — stub: print "no active session" (session persistence deferred)
- `rook auth logout` — stub: print "logged out" (no local token to revoke in this release)
- Error messages: expired nonce, signature mismatch, unregistered key — all user-facing and actionable

### `rook-server` — `user-service` skeleton

- Full Go service skeleton under `rook-server/` — marked **speculative**: directory layout and package structure are established now but the full service specification is deferred to the release that owns `user-service` fully
- `GET /auth/challenge` — generate a 32-byte cryptographically random nonce; store in Firestore `auth/nonces/{nonce}` with 60-second TTL; return nonce to caller
- `POST /auth/verify` — accept `{ public_key, signature, nonce }`; fetch nonce from Firestore; verify signature against registered public key; on success issue an opaque 32-byte session token stored in Firestore `sessions/{token}`; return token
- Firestore schema: nonce collection (`nonce`, `expires_at`, `identity_ref`); session collection (`token`, `user_id`, `expires_at`)
- Public key registration: **manual Firestore seed only** — no admin CLI command in this release; a developer seeds the `users` collection directly to register a public key for testing
- Session token format and nonce lifecycle per [v0.2 Auth Foundation Decisions ADR](../decisions/2026-04-26-v0.2-auth-foundation-decisions.md)

### Local dev environment — Docker Compose

- Docker Compose config at `rook-server/docker-compose.yml` (or repo root) launching:
  - `user-service` container (built from `rook-server/`)
  - Firestore emulator container
- Usage guide documenting:
  - Prerequisites: Docker, Docker Compose, Go 1.23+ (Mac/Linux only; no Windows support)
  - How to start the stack: `docker compose up`
  - How to seed a public key into the emulator for testing
  - How to point `rook-cli` at the local stack (config or env var override)
  - How to run the full auth flow end-to-end against the local stack

---

## Dependencies

- PRD002 v0.1 complete — `internal/config`, XDG paths, Go workspace, CI skeleton required
- [v0.2 Auth Foundation Decisions ADR](../decisions/2026-04-26-v0.2-auth-foundation-decisions.md) — session token format, nonce lifecycle, SSH key selection
- [SSH Auth Identity Chain ADR](../decisions/2026-04-25-ssh-auth-identity-chain-and-cloud-run-topology.md) — challenge/verify protocol, deployment topology
- [rook-cli Features and UX Architecture ADR](../decisions/2026-04-25-rook-cli-features-and-ux-architecture.md) — cobra + fang dispatch model, lazy auth pattern

---

## Out of Scope for This Release

- First-run setup flow (SSH key generation, server address prompt) — deferred to **PRD004 v0.3**
- `rook-server-cli user register` admin command for public key registration — deferred to a future release; manual Firestore seed used instead
- Session token persistence to XDG state directory — deferred until a release requires cross-invocation auth
- Space or group identity — auth is identity-only; no membership resolution
- Server-side session revocation — logout is a stub; no server call made
- Multi-account or key-switching support
- Cloud Run deployment — local Docker Compose only for this release
- Any TUI screens beyond the root launcher stub and `rook auth` interactive flow

---

## Open Questions

All open questions from the stub are resolved:

- **Session token format** → opaque 32-byte random token; resolved by [v0.2 Auth Foundation Decisions ADR](../decisions/2026-04-26-v0.2-auth-foundation-decisions.md)
- **SSH key selection** → prompt at auth time with Bubble Tea list; `--key` flag for non-interactive use; resolved by [v0.2 Auth Foundation Decisions ADR](../decisions/2026-04-26-v0.2-auth-foundation-decisions.md)
- **Nonce TTL and cleanup** → 60-second TTL enforced at read time; Firestore TTL policy for collection hygiene; resolved by [v0.2 Auth Foundation Decisions ADR](../decisions/2026-04-26-v0.2-auth-foundation-decisions.md)

---

## Notes

- `rook` at launch opens the local TUI (stash file list + action menu) immediately without auth. Auth is demanded lazily only when a server-connected action is selected (sync, messaging). This is the canonical UX model for all releases — local features are always available offline.
- Guides join the lazy-auth trigger set in **v0.5** when offline sync is introduced (PRD006); they are server-connected only in v0.2–v0.4.
- The `user-service` skeleton is intentionally speculative — its package layout and gRPC/HTTP wiring are established now so CI and the Docker Compose stack work, but the full service spec (spaces, group membership, ACL) is not owned by this release. Mark speculative sections clearly in code comments.
- `rook auth status` and `rook auth logout` are stubbed with informational responses. They are correct stubs — not silent no-ops — so the cobra subcommand tree is complete and shell completions work correctly from v0.2.
- Session tokens must never be written to disk by `rook-cli` — in-memory only. File-permission note from the stub (0600) is moot until a future release introduces persistence.
- The `rook auth` flow is fully terminal-native — no browser step.
- The challenge/verify nonce must be cryptographically random (≥ 32 bytes) to prevent replay; use `crypto/rand`.
- **Issue closure workflow (carry-forward from v0.1)**: PR descriptions must include a `Closes #NNN` line for every task issue. The T→# mapping is available in `tasks.md` after `/speckit.taskstoissues` runs.
