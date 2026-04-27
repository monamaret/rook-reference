# Research: v0.2 Authentication Foundation

**Feature**: `002-auth-foundation`  
**Branch**: `002-auth-foundation`  
**Date**: 2026-04-26  
**Status**: Complete — no open NEEDS CLARIFICATION items

---

## Summary

All design questions for v0.2 are resolved by existing ADRs. No speculative research was required. This file consolidates findings from four ADRs into actionable implementation decisions.

---

## Finding 1: CLI dispatch model — cobra + fang

**Decision**: Replace `main.go` hand-rolled flag parsing with `cmd/root.go` + `fang.Execute(ctx, cmd.Root())`. The root command's `RunE` launches the local launcher TUI. `rook auth` is a cobra subcommand registered in `cmd/auth.go`.

**Rationale**: Resolved by [rook-cli Features and UX Architecture ADR](../../decisions/2026-04-25-rook-cli-features-and-ux-architecture.md). The Cobra + fang hybrid gives shell-invocable subcommands, styled errors, shell completions, and manpage generation via `fang`. The Charmbracelet team uses this pattern for their own tools (`gum`, `soft-serve`). `fang.Execute` handles `--version`, `SilenceErrors`, and styled output — eliminating significant boilerplate.

**Alternatives considered**: Pure Bubble Tea launcher (no subcommands) — rejected because it is not scriptable from CI, cannot generate manpages, and does not provide shell completions.

**New dependencies for `rook-cli/go.mod`**:
- `github.com/spf13/cobra` — command routing and flag parsing
- `charm.land/fang/v2` — styled output, `--version`, manpages, shell completions
- `charmbracelet/bubbletea` — TUI framework (needed for root command RunE launcher stub)
- `charmbracelet/bubbles` — list component for SSH key selection prompt
- `charmbracelet/lipgloss` — styling (pulled transitively by fang; explicit for clarity)
- `golang.org/x/crypto` — SSH key loading, signing, and signature verification

**Patterns**:
- `main.go` calls `fang.Execute(ctx, cmd.Root())` and returns; all logic in `cmd/`
- Do NOT call `cobra.Command.Execute()` directly — always use `fang.Execute`
- Do NOT set `cmd.SilenceErrors` or `cmd.SilenceUsage` — fang handles these
- Do NOT call `os.Exit` inside cobra `RunE` — return an error

---

## Finding 2: Auth protocol — HTTPS challenge-response with SSH key signing

**Decision**: Two-step HTTPS challenge-response:
1. `GET /auth/challenge` → returns a 32-byte hex nonce (TTL: 60 seconds, stored in Firestore)
2. CLI signs nonce with SSH private key using `golang.org/x/crypto/ssh`
3. `POST /auth/verify` → `{ public_key, signature, nonce }` → server verifies, returns session token

**Rationale**: Resolved by [SSH Auth Identity Chain and Cloud Run Topology ADR](../../decisions/2026-04-25-ssh-auth-identity-chain-and-cloud-run-topology.md). Cloud Run does not support raw TCP; `charmbracelet/wish` cannot run on Cloud Run. HTTP-only on Cloud Run with SSH request signing is the only option that keeps all services on Cloud Run with no additional managed infrastructure.

**Signing library**: `golang.org/x/crypto/ssh` — Go extended standard library; supports `rsa`, `ecdsa`, and `ed25519`; no new dependency class.

**Alternatives considered**:
- `charmbracelet/wish` on Compute Engine VM — rejected: persistent VM infrastructure, baseline cost, OS patching burden, leaves identity chain gap unresolved
- JWT-based session tokens — rejected: require a signing key to be provisioned and rotated across all services, complicate revocation (need a blocklist or accept a revocation lag)

---

## Finding 3: Session token — opaque 32-byte random, in-memory only

**Decision**: Session token is a 32-byte cryptographically random value, hex-encoded. Stored server-side in Firestore `sessions/{token}` with 1-hour TTL. Stored in-memory only by `rook-cli` — never written to disk. Discarded on process exit.

**Rationale**: Resolved by [v0.2 Auth Foundation Decisions ADR](../../decisions/2026-04-26-v0.2-auth-foundation-decisions.md). Opaque tokens are trivially revocable (delete the Firestore document), require no claims-decoding logic in services, and are generated with `crypto/rand` — no external dependency.

**Token generation**: `crypto/rand.Read` into a 32-byte slice → `hex.EncodeToString` → 64-character hex string.

**In-memory storage**: Package-level `auth.Session` struct or `context.Context`-carried value for the process lifetime. The value must never be passed to `config.Save()` or written to any file.

**Alternatives considered**: Signed JWT — rejected: requires a shared signing key provisioned across all services, complicates revocation, adds claims-decoding logic that must stay in sync with server-side schema.

---

## Finding 4: Nonce lifecycle — 60-second TTL, Firestore TTL policy

**Decision**: Nonces stored in Firestore `auth/nonces/{nonce}` with `expires_at` timestamp 60 seconds from issuance. Application code enforces the window by checking `expires_at` at read time. Firestore TTL policy on `expires_at` deletes expired documents within 72 hours. Nonces are marked used atomically on first successful verification (replay prevention independent of TTL).

**Rationale**: Resolved by [v0.2 Auth Foundation Decisions ADR](../../decisions/2026-04-26-v0.2-auth-foundation-decisions.md). Firestore TTL policies are zero-ops — no Cloud Scheduler job, no background goroutine, no on-read purge race conditions.

**Firestore TTL policy setup**: Must be configured before first deployment. Verify with:
```
gcloud firestore fields ttls describe --collection-group=nonces --field-path=expires_at
```
For the Firestore emulator: TTL policies are not enforced by the emulator — application-level `expires_at` check is the correctness mechanism; TTL is only collection hygiene.

**Alternatives considered**:
- Cloud Scheduler + Cloud Function purge — rejected: additional GCP resources, no correctness benefit
- On-read purge — rejected: write-on-read complicates handler logic, creates race conditions under concurrent auth

---

## Finding 5: SSH key selection — Bubble Tea prompt at auth time

**Decision**: At auth time, scan `~/.ssh/` for files matching `id_*` (excluding `.pub` extension and known non-key files: `known_hosts`, `authorized_keys`, `config`). Display filename, key type, and fingerprint in a Bubble Tea `list.Model`. Single-key fast path: skip prompt, proceed with an informational message. `--key <path>` flag bypasses discovery entirely.

**Rationale**: Resolved by [v0.2 Auth Foundation Decisions ADR](../../decisions/2026-04-26-v0.2-auth-foundation-decisions.md). Prompt-based discovery surfaces available keys explicitly — reduces auth failure caused by selecting an unregistered key. Consistent with the TUI-first design.

**Key discovery implementation**: `os.ReadDir("~/.ssh/")`, filter by `id_` prefix and no `.pub` extension. Parse key type from the `.pub` sibling file if present (first field of the wire-format line). Fingerprint: `ssh.FingerprintSHA256(pubKey)` from `golang.org/x/crypto/ssh`.

**Alternatives considered**:
- Config-file-only selection — rejected: poor first-run experience, user must know path before auth
- `ssh-agent` first — rejected: adds agent detection complexity, implicit selection hard to debug

---

## Finding 6: user-service structure — speculative skeleton

**Decision**: The `user-service` is scaffolded as a full Go binary under `rook-server/user-service/` with the directory layout prescribed by the gRPC ADR. The skeleton includes:
- `cmd/main.go` — HTTP server entry point (port 8080)
- `handlers/auth.go` — `GET /auth/challenge` and `POST /auth/verify`
- `handlers/session.go` — `ValidateSession` placeholder
- Internal Firestore client wiring (emulator-aware via `FIRESTORE_EMULATOR_HOST`)

All files are marked with `// SPECULATIVE: full user-service specification is owned by a future release.` The skeleton is buildable and passes CI — it is not a stub that exits immediately.

**Rationale**: The user-service skeleton must exist and compile for CI and Docker Compose to work. Marking it speculative in code comments ensures future developers understand it is not a finalized design.

**Proto schema** (scaffolded, not generated in v0.2):
- `rook-server/proto/user/v1/user.proto` — `UserService` with `ValidateSession` RPC (placeholder body)
- Generated stubs deferred to when `buf generate` is integrated (future release)

---

## Finding 7: Docker Compose local dev environment

**Decision**: Docker Compose config at `rook-server/docker-compose.yml` with two services:
1. `user-service` — built from `rook-server/user-service/`; listens on `localhost:8080`
2. `firestore-emulator` — `gcr.io/google.com/cloudsdktool/google-cloud-cli:emulators` image; listens on `localhost:8088`; `FIRESTORE_EMULATOR_HOST=firestore-emulator:8088` injected into `user-service`

A `docs/local-dev.md` usage guide covers: prerequisites (Docker, Go 1.23+, Mac/Linux only), `docker compose up`, public key seeding, CLI configuration, end-to-end auth verification.

**Rationale**: Required by spec FR-018 through FR-020. Allows all v0.2 development and testing without a cloud account. Mac/Linux only — no Windows support.

**Firestore emulator project ID**: `rook-local` (conventional, matches `GOOGLE_CLOUD_PROJECT=rook-local` env var).

---

## Finding 8: `rook-server-cli` — out of scope

**Decision**: `rook-server-cli` is unchanged in v0.2. No `user register` command is added. Public key registration for testing is performed by directly seeding the Firestore emulator (documented in the usage guide).

**Rationale**: PRD003 scoping interview — `rook-server-cli` admin commands deferred to a future release. Manual seeding is sufficient for v0.2 development.

---

## Patterns to Follow

From the UX Architecture ADR and SSH Auth ADR — mandatory for all v0.2 CLI code:

1. `main.go` → `fang.Execute(ctx, cmd.Root())` only; no other logic
2. TUI models launched via `tea.NewProgram(model, tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.OutOrStdout()))` inside cobra `RunE`
3. Session token: package-level `var currentSession *auth.Session` or equivalent — never written to disk
4. All nonce and token bytes from `crypto/rand.Read` — never `math/rand`
5. SSH signing: `signer.Sign(rand.Reader, []byte(nonce))` via `golang.org/x/crypto/ssh`
6. Do not call `os.Exit` inside cobra `RunE` — return error, let `fang.Execute` handle exit
7. Do not background-poll servers — all network activity is user-initiated
8. `user-service` reads all config from env vars at startup; never calls `os.Getenv` inside a handler

---

## Open Items

None. All NEEDS CLARIFICATION items are resolved. Ready for Phase 1 design.
