# Feature Specification: Rook v0.1 — Project Skeleton

**Feature Branch**: `001-v0.1-project-skeleton`  
**Created**: 2026-04-26  
**Status**: Draft  
**Input**: User description: "review specs/ and spec v0.1"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Developer Clones and Builds the Project (Priority: P1)

A new contributor clones the monorepo, runs the bootstrap command, and successfully builds both `rook-cli` and `rook-server-cli` binaries on their local machine. They do not need any external setup instructions beyond what the repo provides.

**Why this priority**: This is the foundational deliverable of v0.1. If a contributor cannot clone-and-build, every subsequent release is blocked. It de-risks the entire project setup for all future contributors.

**Independent Test**: A contributor with Go installed can clone the repo, run `make dev` (or equivalent), and produce working binaries — confirming a viable, standalone developer experience before any other feature is built.

**Acceptance Scenarios**:

1. **Given** a clean machine with the required Go toolchain, **When** a developer clones the repo and runs the documented bootstrap command, **Then** both `rook-cli` and `rook-server-cli` binaries are built without errors.
2. **Given** the built `rook-cli` binary, **When** the developer runs it with no config present, **Then** it prints version information and exits cleanly (no panic, no missing config error).
3. **Given** the built `rook-server-cli` binary, **When** the developer runs it, **Then** it prints `rook-server-cli version <version>` and exits 0.

---

### User Story 2 — CI Validates Every Push Automatically (Priority: P2)

When a contributor opens a pull request or pushes to a branch, the CI pipeline automatically lints, builds, and tests the affected component(s). The contributor sees pass/fail status without any manual intervention.

**Why this priority**: CI is the quality gate for all future work. Establishing it in v0.1 prevents lint and build debt from accumulating and gives contributors immediate feedback.

**Independent Test**: A push to the repository triggers CI and reports a pass/fail result — validating that the automated quality gate is functional independently of any feature code.

**Acceptance Scenarios**:

1. **Given** a push to `rook-cli/` code, **When** CI runs, **Then** only the `rook-cli` workflow triggers (not `rook-server`); lint, build, and test all pass.
2. **Given** a push to `rook-server/` code, **When** CI runs, **Then** only the `rook-server` workflow triggers; lint, build, and test all pass.
3. **Given** a PR with a lint violation (e.g., unused import), **When** CI runs, **Then** the pipeline reports failure and the contributor sees which check failed.

---

### User Story 3 — Developer Reads and Writes Config (Priority: P3)

A developer using `rook-cli` can confirm that the config file is loaded from the correct XDG path, that missing config is handled gracefully, and that any write operation is atomic (no corrupt config after an interrupted write).

**Why this priority**: Config read/write utilities are the lowest-level dependency for every subsequent release. Correctness here prevents hard-to-debug failures later, but delivering this after clone-and-build and CI is the right sequencing.

**Independent Test**: A developer can observe that `rook-cli` reads from `$XDG_CONFIG_HOME/rook/config.json` and that a simulated interrupted write does not corrupt the file — demonstrating correctness of the config layer independently of any connected feature.

**Acceptance Scenarios**:

1. **Given** `$XDG_CONFIG_HOME` is set, **When** `rook-cli` starts, **Then** it loads config from `$XDG_CONFIG_HOME/rook/config.json`; if absent, it falls back to `~/.config/rook/config.json`.
2. **Given** no config file exists, **When** `rook-cli` starts, **Then** it handles the absence gracefully (exits with a clear message or enters a setup flow) without panicking.
3. **Given** a config write in progress, **When** the process is interrupted mid-write, **Then** the config file is either fully written or unchanged (write-to-temp + rename ensures atomicity).
4. **Given** an environment variable override is set, **When** `rook-cli` reads config, **Then** the env-var value takes precedence over the file value.

---

### Edge Cases

- What happens when `$XDG_CONFIG_HOME` is not set and `~/.config` does not exist? (The CLI must create the directory or report a clear error.)
- What happens when the `go.work` file lists a module that does not exist on disk? (CI should fail with a clear error rather than silently skipping the module.)
- What happens when two contributors have different Go minor versions installed? (The toolchain version pinned in `go.mod`/`go.work` must be documented and enforced.)
- What happens when a CI path filter is misconfigured and a push triggers no workflow? (Contributor sees no CI feedback — must be caught during v0.1 CI setup.)

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The repository MUST contain `rook-cli/` and `rook-server/` as distinct Go modules under a single `go.work` workspace root.
- **FR-002**: A `go.work` file MUST be committed at the repository root listing all Go modules; `go.work.sum` MUST also be committed.
- **FR-003**: Each Go module MUST have its own `go.mod` with a pinned Go version consistent across all modules.
- **FR-004**: The monorepo MUST include a Makefile (or equivalent) with a documented bootstrap command that builds all binaries.
- **FR-005**: `rook-cli` MUST build to a runnable binary; when run with no config, it MUST exit cleanly with a version/help message rather than panicking.
- **FR-006**: `rook-server/cmd/admin/main.go` MUST exist as a stub binary that prints `rook-server-cli version <version>` (version injected via build-time `-ldflags`) and exits 0.
- **FR-007**: `rook-cli` MUST read its configuration from `$XDG_CONFIG_HOME/rook/config.json`, falling back to `~/.config/rook/config.json` on Linux/macOS when `$XDG_CONFIG_HOME` is unset.
- **FR-008**: The configuration file MUST use JSON format; the read/write utilities MUST support environment-variable overrides for config values.
- **FR-009**: Config file writes MUST be atomic: write to a temporary file then rename, to prevent corruption on interrupted writes.
- **FR-010**: The CI pipeline MUST include path-filtered GitHub Actions workflows that trigger lint, build, and test independently per component (`rook-cli` and `rook-server`).
- **FR-011**: CI MUST gate on `go vet`, a static analysis tool (e.g., `staticcheck` or `golangci-lint`), and `go test` before merging.
- **FR-012**: The repository MUST include a README with documented prerequisites, local dev setup steps, and contributor guidelines.
- **FR-013**: XDG path resolution MUST degrade gracefully on macOS; fallback paths MUST be documented for both Linux/macOS and WSL.

### Key Entities *(include if feature involves data)*

- **Config file**: JSON document at the XDG-resolved path. At v0.1, contains: server endpoint(s), a placeholder for identity info, and feature flags. Read on CLI startup; written atomically on first-run setup or explicit save.
- **Go workspace**: The `go.work` root that links `rook-cli` and `rook-server` modules. Controls local module resolution during development and CI.
- **CI workflow**: Per-component GitHub Actions workflow files. Triggered by path filters; produce pass/fail signals per component.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer with no prior Rook experience can clone the repository and produce working binaries in under 10 minutes by following only the documented setup steps.
- **SC-002**: Every push to a component directory triggers the corresponding CI workflow; a lint or build failure is surfaced as a failed check within 5 minutes of the push.
- **SC-003**: An interrupted config write leaves the config file either fully intact or unchanged — data corruption rate is 0%.
- **SC-004**: The CI pipeline catches 100% of `go vet` and lint violations on every pull request before merge.
- **SC-005**: `rook-server-cli` binary is produced as a release artifact from the first tagged release, with no manual pipeline intervention required.

---

## Assumptions

- Contributors use Linux, macOS, or WSL; no native Windows build is required or supported at this stage.
- The required Go toolchain version is available via standard package managers (Homebrew, `apt`, or `asdf`); contributors are responsible for installing it.
- GitHub Actions is the CI platform; no alternative CI system is in scope for v0.1.
- The config file is machine-written by `rook-cli`; human comment support in the config format is not a requirement.
- A single `go.work` file at the repository root is sufficient; no nested workspace files are needed.
- `rook-docs/` may be added to `go.work` later if it becomes a Go module; it is not included in v0.1 if it contains no Go code.
- Version is injected at binary build time via `-ldflags`; no version file is read from disk at runtime.
- The Makefile is the canonical build interface; contributors do not need to know the raw `go build` invocations.
