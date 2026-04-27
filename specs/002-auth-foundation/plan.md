# Implementation Plan: v0.2 — Authentication Foundation

**Branch**: `002-auth-foundation` | **Date**: 2026-04-26 | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification from `specs/002-auth-foundation/spec.md`

## Summary

Replace the v0.1 hand-rolled `os.Args` dispatch with a cobra + fang command tree (`cmd/root.go`). Implement the `rook auth` command: SSH key discovery, Bubble Tea selection prompt, HTTPS challenge-response flow signing with `golang.org/x/crypto/ssh`, and in-memory session token. Scaffold the `user-service` Go binary in `rook-server/user-service/` with `GET /auth/challenge` and `POST /auth/verify` HTTP handlers backed by the Firestore emulator. Provide a Docker Compose local dev environment and end-to-end usage guide.

---

## Technical Context

**Language/Version**: Go 1.23 (pinned across all `go.mod` and `go.work`)  
**Primary Dependencies (rook-cli)**:
- `github.com/spf13/cobra` — command routing and flag parsing
- `charm.land/fang/v2` — styled output, `--version`, manpages, shell completions
- `charmbracelet/bubbletea` — TUI framework (root command launcher stub)
- `charmbracelet/bubbles` — `list.Model` for SSH key selection prompt
- `charmbracelet/lipgloss` — styling (transitive via fang; explicit for clarity)
- `golang.org/x/crypto` — SSH key loading, signing (`ssh` sub-package)

**Primary Dependencies (rook-server/user-service)**:
- `cloud.google.com/go/firestore` — Firestore client (emulator-aware via `FIRESTORE_EMULATOR_HOST`)
- `golang.org/x/crypto` — SSH signature verification (`ssh` sub-package)

**Storage**: Firestore (server-side) via emulator for local dev; in-memory only for CLI session token  
**Testing**: `go test ./...` with `-race -count=1`; stdlib `testing` package; `net/http/httptest` for handler tests  
**Target Platform**: Linux, macOS (no Windows)  
**Project Type**: CLI binary (`rook-cli`) + HTTP service binary (`user-service`)  
**Performance Goals**: `rook auth` full flow completes in under 30 seconds on a running local stack; `rook` (launcher) opens in under 2 seconds  
**Constraints**: Session token never written to disk; all nonces and tokens from `crypto/rand`; no `os.Exit` inside cobra `RunE`  
**Scale/Scope**: ~2 Go modules; ~10 new packages across both modules

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The project constitution (`.specify/memory/constitution.md`) is an unfilled template — no project-specific principles or gates are currently defined. **No gates apply.** This is a carry-forward gap from v0.1; a constitution should be authored before v0.3.

Post-Phase 1 re-check: No violations. The design follows established ADR patterns (cobra + fang dispatch, in-memory session token, flat-file local storage, XDG paths). No new persistence mechanisms are introduced beyond what existing ADRs prescribe.

---

## Project Structure

### Documentation (this feature)

```text
specs/002-auth-foundation/
├── plan.md              ← this file
├── research.md          ← Phase 0 output
├── data-model.md        ← Phase 1 output
├── quickstart.md        ← Phase 1 output
├── contracts/
│   └── cli-contract.md  ← Phase 1 output
└── tasks.md             ← Phase 2 output (/speckit.tasks — NOT created here)
```

### Source Code (repository root)

```text
# rook-cli module
rook-cli/
├── go.mod                                  # + cobra, fang, bubbletea, bubbles, lipgloss, x/crypto
├── go.sum
├── main.go                                 # REPLACE: fang.Execute(ctx, cmd.Root()) only
├── cmd/
│   ├── root.go                             # NEW: cobra root command; RunE launches launcher TUI stub
│   └── auth.go                             # NEW: rook auth, rook auth status, rook auth logout
├── internal/
│   ├── config/                             # UNCHANGED from v0.1
│   │   ├── config.go
│   │   ├── config_test.go
│   │   └── xdg.go
│   └── auth/
│       ├── auth.go                         # NEW: challenge-response flow, in-memory session state
│       ├── auth_test.go                    # NEW: unit tests with mock HTTP server
│       ├── keys.go                         # NEW: SSH key discovery and fingerprint display
│       └── keys_test.go                    # NEW: unit tests for key discovery
└── internal/
    └── launcher/
        └── launcher.go                     # NEW: minimal Bubble Tea launcher stub (local TUI root)

# rook-server module
rook-server/
├── go.mod                                  # module: github.com/rook-project/rook-reference/rook-server
├── go.sum
├── Makefile
├── docker-compose.yml                      # NEW: user-service + firestore emulator
├── docs/
│   └── local-dev.md                        # NEW: usage guide (mirrors quickstart.md)
├── cmd/
│   └── admin/
│       └── main.go                         # UNCHANGED: v0.1 rook-server-cli stub
├── proto/
│   └── user/
│       └── v1/
│           └── user.proto                  # NEW: UserService proto (speculative scaffold)
├── internal/
│   └── middleware/
│       └── session.go                      # NEW: SessionAuthMiddleware scaffold (speculative)
└── user-service/
    ├── go.mod                              # NEW: separate module or subfolder within rook-server
    ├── main.go                             # NEW: HTTP server entry point (port 8080)
    ├── handlers/
    │   ├── auth.go                         # NEW: GET /auth/challenge, POST /auth/verify
    │   ├── auth_test.go                    # NEW: handler tests with httptest + Firestore emulator
    │   └── session.go                      # NEW: ValidateSession placeholder
    └── store/
        └── firestore.go                    # NEW: Firestore client init, emulator-aware

# repo-root tooling
Makefile                                    # UPDATE: add user-service build target
.github/
└── workflows/
    └── ci-user-service.yml                 # NEW: path-filtered CI for rook-server/user-service/**
```

**Structure Decision**: `user-service` lives under `rook-server/user-service/` as a sub-directory within the `rook-server` Go module (same `go.mod`). This avoids introducing a third `go.work` module entry for a single v0.2 skeleton service. Future services (`stash-service`, `messaging-service`, etc.) follow the same pattern until a release requires independent module versioning.

---

## Complexity Tracking

> No constitution violations — section not applicable.

---

## Phase 0: Research

> **Output**: [research.md](research.md)

All NEEDS CLARIFICATION items are resolved by existing ADRs. See `research.md` for full findings. Summary:

| Finding | Decision | ADR Source |
|---|---|---|
| CLI dispatch model | cobra + fang; `cmd/root.go`; `fang.Execute` | rook-cli UX Architecture ADR |
| Auth protocol | HTTPS challenge-response; SSH signing via `x/crypto/ssh` | SSH Auth + Cloud Run Topology ADR |
| Session token format | Opaque 32-byte random, hex-encoded, in-memory only | v0.2 Auth Foundation Decisions ADR |
| Nonce lifecycle | 60-second TTL, `expires_at` check at read, Firestore TTL policy | v0.2 Auth Foundation Decisions ADR |
| SSH key selection | Bubble Tea list prompt; single-key fast path; `--key` flag | v0.2 Auth Foundation Decisions ADR |
| user-service structure | Speculative skeleton; buildable; marked with `// SPECULATIVE` comments | gRPC ADR + PRD003 scoping |
| Docker Compose | `user-service` + Firestore emulator; Mac/Linux only | PRD003 scoping interview |
| rook-server-cli | Unchanged in v0.2; manual Firestore seed for testing | PRD003 scoping interview |

---

## Phase 1: Design & Contracts

> **Outputs**: [data-model.md](data-model.md), [contracts/cli-contract.md](contracts/cli-contract.md), [quickstart.md](quickstart.md)

### Data Model Summary

Three new Firestore entities + one in-memory CLI entity. Full detail in [data-model.md](data-model.md).

| Entity | Storage | Owner | Purpose |
|---|---|---|---|
| `Nonce` | Firestore `auth/nonces/{nonce}` | `user-service` | Short-lived challenge; single-use; 60s TTL |
| `Session` | Firestore `sessions/{token}` | `user-service` | Auth credential; 1-hour TTL |
| `User` | Firestore `users/{user_id}` | `user-service` | Identity + registered public key; pre-seeded in v0.2 |
| `Session State` | In-memory (`internal/auth`) | `rook-cli` | Holds token for process lifetime; never persisted |

### HTTP API Surface

| Endpoint | Method | Auth | Purpose |
|---|---|---|---|
| `/auth/challenge` | `GET` | None | Issue a 32-byte hex nonce; store in Firestore with 60s TTL |
| `/auth/verify` | `POST` | None | Verify signed nonce; issue session token |

`ValidateSession` (gRPC or internal HTTP) is scaffolded as a placeholder — not wired to any handler in v0.2.

### Interface Contracts

Full detail in [contracts/cli-contract.md](contracts/cli-contract.md). Key v0.2 additions:

| Command | Description |
|---|---|
| `rook` | Opens local launcher TUI (no auth) |
| `rook auth` | Interactive SSH key auth flow |
| `rook auth --key <path>` | Non-interactive auth with explicit key |
| `rook auth status` | Stub: prints "no active session" |
| `rook auth logout` | Stub: prints "logged out" |
| `rook completion <shell>` | Shell completion (via cobra + fang) |
| `rook man` | Generate manpage (via fang + mango-cobra) |

### Agent Context Update

The `TABNINE.md` context pointer is updated after this plan is written to point to `specs/002-auth-foundation/plan.md`.

---

## Implementation Sequence

The following sequence is the recommended order for task generation (`/speckit.tasks`). Each group can be reviewed independently.

### Group A — CLI dispatch wiring (unblocks all other CLI work)

1. Add cobra, fang, bubbletea, bubbles, lipgloss, x/crypto to `rook-cli/go.mod`
2. Rewrite `rook-cli/main.go` to call `fang.Execute(ctx, cmd.Root())`
3. Create `rook-cli/cmd/root.go` — cobra root command, `RunE` launches launcher stub
4. Create `rook-cli/internal/launcher/launcher.go` — minimal Bubble Tea model (placeholder list)
5. Verify: `go build ./...` passes; `rook --version` and `rook --help` work; `rook` opens the stub TUI

### Group B — SSH key discovery (unblocks auth command)

6. Create `rook-cli/internal/auth/keys.go` — `DiscoverKeys()`, `ParseKeyType()`, `FingerprintKey()`
7. Create `rook-cli/internal/auth/keys_test.go` — unit tests using `t.TempDir()` to mock `~/.ssh/`

### Group C — Auth command (depends on A + B)

8. Create `rook-cli/cmd/auth.go` — `rook auth`, `rook auth status`, `rook auth logout`
9. Create `rook-cli/internal/auth/auth.go` — `ChallengeResponse()`, `SessionState`, `GetChallenge()`, `VerifyChallenge()`
10. Create `rook-cli/internal/auth/auth_test.go` — unit tests with `net/http/httptest` mock server

### Group D — user-service skeleton (can start in parallel with A–C)

11. Create `rook-server/user-service/` directory structure
12. Add `cloud.google.com/go/firestore` and `golang.org/x/crypto` to `rook-server/go.mod`
13. Create `rook-server/user-service/store/firestore.go` — Firestore client, emulator-aware init
14. Create `rook-server/user-service/handlers/auth.go` — `GET /auth/challenge`, `POST /auth/verify`
15. Create `rook-server/user-service/handlers/session.go` — `ValidateSession` placeholder
16. Create `rook-server/user-service/main.go` — HTTP server entry point
17. Create `rook-server/user-service/handlers/auth_test.go` — handler unit tests with `httptest`
18. Create `rook-server/proto/user/v1/user.proto` — `UserService` proto scaffold
19. Create `rook-server/internal/middleware/session.go` — `SessionAuthMiddleware` scaffold

### Group E — Docker Compose + local dev (depends on D)

20. Create `rook-server/docker-compose.yml` — `user-service` + Firestore emulator
21. Create `rook-server/docs/local-dev.md` — usage guide
22. Update root `Makefile` to include `user-service` in build and test targets
23. Create `.github/workflows/ci-user-service.yml` — path-filtered CI

### Group F — End-to-end validation

24. Run `make build` — all binaries compile
25. Run `make test` — all tests pass with `-race -count=1`
26. Run `make lint` — `golangci-lint` passes for both modules
27. Run Docker Compose stack + execute full auth flow per `quickstart.md`
28. Verify `rook auth status` and `rook auth logout` exit 0 with correct output
29. Verify all three error conditions produce distinct, non-empty error messages

---

## Key Patterns (mandatory)

From ADRs — enforced in all v0.2 code:

| Pattern | Source |
|---|---|
| `main.go` → `fang.Execute(ctx, cmd.Root())` only | rook-cli UX Architecture ADR |
| TUI: `tea.NewProgram(model, tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.OutOrStdout()))` | rook-cli UX Architecture ADR |
| Session token: in-memory only; never `config.Save()` or write to file | v0.2 Auth Foundation Decisions ADR |
| All random bytes: `crypto/rand.Read` — never `math/rand` | SSH Auth + Cloud Run Topology ADR |
| SSH signing: `signer.Sign(rand.Reader, []byte(nonce))` | SSH Auth + Cloud Run Topology ADR |
| No `os.Exit` inside cobra `RunE` — return error | rook-cli UX Architecture ADR |
| `user-service` reads config from env vars at startup; never `os.Getenv` inside handler | SSH Auth + Cloud Run Topology ADR |
| All speculative `user-service` code marked `// SPECULATIVE` | PRD003 scoping |

---

## Patterns to Avoid

| Anti-pattern | Reason |
|---|---|
| `cobra.Command.Execute()` directly | Use `fang.Execute` — handles styled errors, completions, manpages |
| Setting `cmd.SilenceErrors` or `cmd.SilenceUsage` | fang sets these; overriding breaks styled output |
| Writing session token to any file | In-memory only per ADR; file write is a security defect |
| `math/rand` for nonces or tokens | Must use `crypto/rand` |
| Background server polling | All network activity is user-initiated |
| `os.Getenv` inside a handler | Read env vars at startup, inject via constructor |
| Duplicating `SessionAuthMiddleware` logic in handlers | Middleware is the single auth enforcement point |
