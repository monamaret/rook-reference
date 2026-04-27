# Implementation Plan: Rook v0.1 — Project Skeleton

**Branch**: `001-v0.1-project-skeleton` | **Date**: 2026-04-26 | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification from `specs/001-v0.1-project-skeleton/spec.md`

## Summary

Establish the foundational monorepo structure for Rook: two independent Go modules (`rook-cli`, `rook-server`) under a single `go.work` workspace, a JSON config read/write layer with XDG path resolution, a stub `rook-server-cli` binary, Makefile-driven build tooling, path-filtered GitHub Actions CI per component, and developer documentation. No user-facing features are delivered; the goal is a buildable, testable skeleton that satisfies the CI/CD pipeline wired in the GitHub Actions ADR.

---

## Technical Context

**Language/Version**: Go 1.23 (pinned consistently across all `go.mod` files)  
**Primary Dependencies**:
- Standard library only for config (`encoding/json`, `os`, `path/filepath`)
- `golangci-lint` for static analysis (CI only — not a runtime dependency)
- `github.com/google/go-cmp` for test assertions (lightweight, no reflect magic)

**Storage**: Flat files — `$XDG_CONFIG_HOME/rook/config.json` (JSON, atomic write via temp+rename)  
**Testing**: `go test ./...` with `-race -count=1`; table-driven unit tests for config utilities  
**Target Platform**: Linux, macOS, WSL (no native Windows build)  
**Project Type**: CLI binary (`rook-cli`) + admin CLI binary (`rook-server-cli`)  
**Performance Goals**: Launcher home screen renders within 500ms (deferred to later releases; v0.1 has no TUI yet)  
**Constraints**: Zero external runtime dependencies for v0.1; no network calls; no Firestore  
**Scale/Scope**: Single developer iteration; 2 Go modules; ~5 packages total across both modules

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The project constitution (`/.specify/memory/constitution.md`) is an unfilled template — no project-specific principles or gates are currently defined. **No gates apply.** This will be flagged as a gap; a constitution should be authored before v0.2 work begins.

Post-Phase 1 re-check: N/A (no gates to re-evaluate).

---

## Project Structure

### Documentation (this feature)

```text
specs/001-v0.1-project-skeleton/
├── plan.md              ← this file
├── research.md          ← Phase 0 output
├── data-model.md        ← Phase 1 output
├── quickstart.md        ← Phase 1 output
├── contracts/           ← Phase 1 output
│   └── cli-contract.md
└── tasks.md             ← Phase 2 output (/speckit.tasks — NOT created here)
```

### Source Code (repository root)

```text
# go.work workspace root
go.work
go.work.sum

# rook-cli module
rook-cli/
├── go.mod                           # module: github.com/rook-project/rook-reference/rook-cli
├── go.sum
├── Makefile                         # build, lint, test targets for rook-cli
├── main.go                          # entrypoint: prints version/help, exits cleanly
├── cmd/
│   └── root.go                      # cobra/flag root command wiring
├── internal/
│   └── config/
│       ├── config.go                # Config struct, Load(), Save() with atomic write
│       ├── config_test.go           # table-driven unit tests for Load/Save/XDG resolution
│       └── xdg.go                   # XDG_CONFIG_HOME resolution with macOS/WSL fallbacks
└── README.md                        # dev setup, prerequisites, contributing guide

# rook-server module
rook-server/
├── go.mod                           # module: github.com/rook-project/rook-reference/rook-server
├── go.sum
├── Makefile                         # build, lint, test targets for rook-server
└── cmd/
    └── admin/
        └── main.go                  # stub: prints "rook-server-cli version <version>", exits 0

# repo-root tooling
.golangci.yml                        # lint rules: gofmt, govet, errcheck, staticcheck
Makefile                             # root-level: delegates to rook-cli/ and rook-server/ Makefiles

# CI workflows
.github/
└── workflows/
    ├── _go-ci.yml                   # reusable: checkout, setup-go, golangci-lint, go test, go build
    ├── ci-rook-cli.yml              # path-filtered: rook-cli/**, triggers _go-ci.yml
    └── ci-rook-server-cli.yml       # path-filtered: rook-server/cmd/admin/**, triggers _go-ci.yml
```

**Structure Decision**: Single-project layout per component, Go workspace at repo root. Two independent CI workflow files for v0.1 (one per deployable binary). Additional per-service CI files (`ci-auth-service.yml`, etc.) will be added in later releases as services are scaffolded.

---

## Complexity Tracking

> No constitution violations — section not applicable.

---

## Phase 0: Research

> **Output**: [research.md](research.md)

All NEEDS CLARIFICATION items from Technical Context are resolved by existing ADRs. No open research tasks remain. Research findings are consolidated below and in `research.md`.

### Finding 1: Config format and path — JSON at XDG path

**Decision**: JSON (`encoding/json`), path `$XDG_CONFIG_HOME/rook/config.json`, fallback `~/.config/rook/config.json`.  
**Rationale**: Closed in the [v0.1 Foundational Decisions ADR](../../decisions/2026-04-26-v0.1-foundational-decisions.md) — standard library only, consistent with PRD001's explicit `config.json` reference, machine-written so comment support is irrelevant.  
**Alternatives considered**: TOML (adds dependency), YAML (parsing hazards). Both rejected.

### Finding 2: Go workspace strategy — `go.work` with per-module `go.mod`

**Decision**: Single `go.work` at repo root listing `rook-cli/` and `rook-server/`. Each module has its own `go.mod`. Both `go.work` and `go.work.sum` are committed.  
**Rationale**: Closed in the [v0.1 Foundational Decisions ADR](../../decisions/2026-04-26-v0.1-foundational-decisions.md) — idiomatic Go monorepo pattern; clean per-module dependency graphs; eliminates `replace` directives.  
**Alternatives considered**: Single root `go.mod`. Rejected — couples unrelated binaries into one version graph.

### Finding 3: CI pipeline — path-filtered per-component GitHub Actions

**Decision**: Per-component workflow files calling a shared `_go-ci.yml` reusable workflow. Path filters on `push` and `pull_request` to `main`.  
**Rationale**: Closed in the [GitHub Actions CI/CD ADR](../../decisions/2026-04-26-github-actions-ci-cd-pipeline.md) — proportional feedback, reusable workflow prevents drift, tag-triggered release is a separate job.  
**Alternatives considered**: Single unified workflow (scales poorly), external CI (unnecessary cost). Both rejected.

### Finding 4: Atomic config writes — write-to-temp + rename

**Decision**: `Save()` writes to a temp file in the same directory, then calls `os.Rename()` to replace the target atomically. On POSIX systems `rename(2)` is atomic; on Windows (WSL counts as Linux) same applies.  
**Rationale**: `os.Rename` on Linux is `rename(2)` — atomic across a power failure after the kernel flushes. Writing directly to the target can leave a partial file. PRD002 notes explicitly call this out.  
**Alternatives considered**: `fsync` before rename (belt-and-suspenders; acceptable if desired but not required for v0.1).

### Finding 5: `rook-server-cli` stub — `rook-server/cmd/admin/main.go`

**Decision**: `main.go` prints `rook-server-cli version <version>` and exits 0. Version injected at build time via `-ldflags "-X main.version=<ver>"`.  
**Rationale**: Closed in the [v0.1 Foundational Decisions ADR](../../decisions/2026-04-26-v0.1-foundational-decisions.md) — the GitHub Actions `release.yml` lists this binary as a v0.1 artifact; the entrypoint must exist for the pipeline to compile.  
**Alternatives considered**: Omit stub, skip from pipeline. Rejected — leaves an untested gap.

### Finding 6: Lint toolchain — `golangci-lint` with `.golangci.yml`

**Decision**: `golangci-lint` run in CI via `golangci/golangci-lint-action@v6`. Config in `.golangci.yml` at repo root. Minimum linters: `gofmt`, `govet`, `errcheck`, `staticcheck`.  
**Rationale**: Prescribed by the [GitHub Actions CI/CD ADR](../../decisions/2026-04-26-github-actions-ci-cd-pipeline.md) and PRD002 notes. Single config file applies to both modules through workspace; linters catch classes of bugs that `go vet` misses.  
**Alternatives considered**: `go vet` alone (insufficient), `staticcheck` standalone (fewer ecosystem integrations).

---

## Phase 1: Design & Contracts

> **Outputs**: [data-model.md](data-model.md), [contracts/cli-contract.md](contracts/cli-contract.md), [quickstart.md](quickstart.md)

### Data Model

> Full detail in [data-model.md](data-model.md).

**Config** (the only persistent entity in v0.1):

| Field | Type | Required | Notes |
|---|---|---|---|
| `servers` | `[]ServerEntry` | No | List of configured `rook-server` addresses (empty at first launch) |
| `active_space` | `string` | No | Space ID last active (empty until auth) |
| `storage_dir` | `string` | No | Absolute path to flat-file store root; defaults to `$XDG_DATA_HOME/rook/storage/` |
| `feature_flags` | `map[string]bool` | No | Reserved for future use; empty map at v0.1 |

**ServerEntry** (nested in Config):

| Field | Type | Required | Notes |
|---|---|---|---|
| `address` | `string` | Yes | Base URL of the server (e.g., `https://rook.example.com`) |
| `alias` | `string` | No | Human-readable label for display in the launcher |

State transitions: Config is loaded on startup → validated → used read-only. On first-run or explicit save, the new value is written atomically. No in-process mutation after load in v0.1.

Validation rules:
- `address` must be a non-empty, valid URL if `servers` is non-empty.
- `storage_dir`, if set, must be an absolute path.
- Unknown JSON fields are silently ignored (forward-compatibility for future releases).

### Interface Contracts

> Full detail in [contracts/cli-contract.md](contracts/cli-contract.md).

`rook-cli` and `rook-server-cli` are CLI tools. Their contracts are the command-line interface each binary exposes.

**`rook-cli` (v0.1)**:

```
rook [--version] [--help]

Flags:
  --version    Print version string and exit 0
  --help       Print usage and exit 0

Exit codes:
  0   success
  1   unrecoverable error (printed to stderr)

Config:
  Read from: $XDG_CONFIG_HOME/rook/config.json
             fallback: ~/.config/rook/config.json
  Env override: ROOK_CONFIG_PATH overrides the resolved path entirely
```

**`rook-server-cli` (v0.1 stub)**:

```
rook-server-cli [--version] [--help]

Flags:
  --version    Print "rook-server-cli version <version>" and exit 0
  --help       Print usage and exit 0

Exit codes:
  0   success
```

Version is injected at build time: `go build -ldflags "-X main.version=<ver>"`. The zero value (`""`) produces `rook-server-cli version dev`.

### Quickstart

> Full detail in [quickstart.md](quickstart.md).

Summary of the developer bootstrap sequence:

1. **Prerequisites**: Go 1.23+, `make`, `git`
2. **Clone**: `git clone https://github.com/rook-project/rook-reference && cd rook-reference`
3. **Bootstrap**: `make build` — runs `go work sync` then builds both binaries to `dist/`
4. **Test**: `make test` — runs `go test ./...` with `-race` in both modules
5. **Lint**: `make lint` — runs `golangci-lint run` against both modules
6. **Verify**: `./dist/rook-cli --version` and `./dist/rook-server-cli --version`

XDG fallback paths:

| Platform | Config path |
|---|---|
| Linux | `$XDG_CONFIG_HOME/rook/config.json` → `~/.config/rook/config.json` |
| macOS | `$XDG_CONFIG_HOME/rook/config.json` → `~/.config/rook/config.json` |
| WSL | Same as Linux |

---

## Agent Context

The `TABNINE.md` context pointer has been updated (see below) to point to this plan file so subsequent commands have direct access to the implementation plan.
