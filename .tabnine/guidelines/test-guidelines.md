# Test Guidelines

## 1. Language and Framework

All production and test code is written in **Go**. The test assertion library is **[testify](https://github.com/stretchr/testify)**. No other test assertion or mocking framework may be introduced without an ADR.

Packages in use:

| Package | Purpose |
|---------|---------|
| `github.com/stretchr/testify/assert` | Non-fatal assertions (test continues on failure) |
| `github.com/stretchr/testify/require` | Fatal assertions (test stops on failure) — prefer for setup and preconditions |
| `github.com/stretchr/testify/mock` | Interface mocking |
| `github.com/stretchr/testify/suite` | Test suites for shared setup/teardown |

Use `require` for anything that would make subsequent assertions meaningless if it failed (e.g., error checks after function calls). Use `assert` for independent property checks.

---

## 2. Test-Driven Development (TDD)

TDD is **non-negotiable** for all feature work that produces code. The Red-Green-Refactor cycle must be followed strictly:

1. **Red** — Write a failing test that captures the acceptance criterion or contract being implemented. Confirm it fails before writing any production code.
2. **Green** — Write the minimum production code necessary to make the test pass. Do not over-engineer at this stage.
3. **Refactor** — Clean up both the production code and the test. Ensure all tests still pass.

In the Speckit workflow, test tasks in `tasks.md` carry the `[P]` parallelisable marker only when they cover completely independent files. Otherwise, each test task must be completed and committed before its corresponding implementation task begins.

**No implementation task may be marked `[X]` in `tasks.md` unless its tests are written, passing, and committed.**

---

## 3. Test Types and When to Write Each

### 3.1 Unit Tests

- Cover a single function, method, or type in isolation.
- External dependencies (Firestore, gRPC services, the filesystem) are replaced with mocks or fakes.
- Live alongside the code under test in the same package using the `_test.go` suffix.
- File naming: `<subject>_test.go` (e.g., `auth_service_test.go`).

### 3.2 Integration Tests

**Required** for:
- New service contract introductions (any new gRPC service or HTTPS endpoint).
- Changes to existing gRPC contracts (`UserService`, `AdminService`, inter-service calls).
- Inter-service communication paths (e.g., `messaging-service` → `user-service` `ValidateSession`).
- Shared Firestore schema changes (new collections, index changes, TTL policies).

Integration tests live in a dedicated `integration/` subdirectory within the service package and are gated by a build tag:

```go
//go:build integration
```

Run them explicitly:

```bash
go test -tags=integration ./...
```

### 3.3 Contract Tests

Each gRPC service interface must have a contract test that verifies the handler behaviour against the protobuf contract without a live Firestore instance. Use an in-process gRPC server with a mock storage backend. Contract tests live in `<service>/contract/` and use the `contract` build tag.

### 3.4 End-to-End Tests (PoC scope)

E2E tests are deferred until Milestone 1 Hardening. For the PoC, the manual success criteria in `PRD001 §7 PoC success criteria` serve as the E2E gate.

### 3.5 No Live Sandbox

**There is no live sandbox, staging, or shared test environment.** All automated tests at every level (unit, integration, contract) must be fully self-contained and run against mocks, fakes, or local emulators (e.g., the Firestore emulator). No test may depend on, or make calls to, a deployed `rook-server` instance. Tests that require external state must create and tear it down within the test itself using `t.TempDir()`, in-process gRPC servers (`bufconn`), or emulators started as test fixtures.

---

## 4. File and Package Conventions

```
<service>/
├── handler.go
├── handler_test.go          # unit tests — same package, _test suffix
├── integration/
│   └── handler_integration_test.go   # build tag: integration
└── contract/
    └── handler_contract_test.go      # build tag: contract
```

- Test files always use the `package <name>_test` external test package convention **except** when testing unexported symbols, in which case use `package <name>` (internal test).
- Do not mix internal and external test packages within the same `_test.go` file.

---

## 5. Test Naming

Follow Go's standard table-driven test pattern. Test function names must be descriptive:

```go
func TestAuthService_VerifyChallenge_RejectsReplayedNonce(t *testing.T) { ... }
func TestAuthService_VerifyChallenge_ReturnsTokenOnSuccess(t *testing.T) { ... }
```

Format: `Test<Type>_<Method>_<Condition>`.

For table-driven tests, sub-test names must be human-readable (they appear in CI output):

```go
t.Run("rejects replayed nonce", func(t *testing.T) { ... })
t.Run("returns session token on valid signature", func(t *testing.T) { ... })
```

---

## 6. Mocking

Generate mocks from interfaces using `mockery` or write them by hand using `testify/mock`. Mocks live in a `mocks/` subdirectory adjacent to the interface they implement:

```
<service>/
├── store.go          # defines Store interface
└── mocks/
    └── store.go      # mock implementation
```

- Never commit auto-generated mocks without verifying they compile and match the current interface.
- Mock method expectations must be asserted at the end of every test: `mockObj.AssertExpectations(t)`.

---

## 7. Coverage

- **Minimum line coverage:** 80% per package for `rook-server` services.
- **Minimum line coverage:** 70% per package for `rook-cli` (TUI code is harder to unit-test).
- Coverage is measured and enforced in CI. PRs that drop coverage below threshold are blocked from merging.
- Coverage reports are generated with:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Coverage for integration-tagged tests is measured separately and does not count toward the unit-test threshold.

---

## 8. CI/CD Integration

All tests are run automatically in the GitHub Actions pipeline. The pipeline is the authoritative gate — local test passage alone is not sufficient for merging.

### 8.1 Pipeline stages

| Stage | Trigger | What runs |
|-------|---------|-----------|
| **lint** | Every push, every PR | `golangci-lint run ./...` |
| **unit** | Every push, every PR | `go test -race -count=1 ./...` |
| **coverage** | Every PR | `go test -coverprofile=coverage.out ./...` + threshold check |
| **integration** | PR to `main` / feature branch merge | `go test -tags=integration -race ./...` |
| **contract** | PR to `main` / feature branch merge | `go test -tags=contract -race ./...` |
| **build** | Every push, every PR | `go build ./...` |

### 8.2 Race detector

The `-race` flag is **always** enabled in CI. Never merge code that introduces a data race. Run locally with:

```bash
go test -race ./...
```

### 8.3 PR merge rules

A PR may not be merged if any of the following are true:

- Any unit test fails.
- Any integration or contract test fails (on PRs targeting `main`).
- Coverage drops below the package threshold.
- The race detector reports a race condition.
- The lint stage reports errors.

### 8.4 Workflow file location

CI configuration lives in `.github/workflows/`. The primary test workflow is `ci.yml`. Each service may have its own workflow file for service-specific build matrix steps.

---

## 9. Project-Specific Testing Notes

### rook-server services

- All services call `user-service` `ValidateSession` on every request. Unit tests must mock this gRPC call — never call a live `user-service` in unit tests.
- Firestore interactions must be mocked or use the Firestore emulator (integration tests only).
- Auth nonce uniqueness (`POST /auth/verify` replay protection) must have a dedicated unit test verifying a second call with the same nonce returns `401`.
- Session tokens must never appear in test output, log lines, or error messages — assert on their presence/absence structurally, not on their value.

### rook-cli

- TUI components (Bubble Tea models) are tested using the `Update`/`View` cycle: construct a model, send a message, assert on the returned model state and rendered view string.
- Offline behaviour (stash reads, message history, search) must be testable without any network call — use the local flat-file store with a `t.TempDir()` root.
- The first-run setup flow must be tested by injecting a configurable XDG path so tests do not touch `$XDG_CONFIG_HOME`.

### gRPC contract tests

- Use `google.golang.org/grpc/test/bufconn` to create an in-process gRPC listener — no real port binding needed.
- Assert on both the happy path and all documented error codes (`codes.Unauthenticated`, `codes.NotFound`, `codes.PermissionDenied`).
