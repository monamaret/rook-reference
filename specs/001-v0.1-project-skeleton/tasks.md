# Tasks: Rook v0.1 — Project Skeleton

**Input**: Design documents from `specs/001-v0.1-project-skeleton/`  
**Prerequisites**: plan.md ✅ spec.md ✅ research.md ✅ data-model.md ✅ contracts/ ✅ quickstart.md ✅

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Tests are not explicitly requested; unit tests for the config package are included because they are contractually required by FR-009 and the contract doc.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Repository scaffolding — Go workspace, module files, Makefile, and CI directory structure. No application logic.

- [ ] T001 Create `go.work` and `go.work.sum` at repo root listing `rook-cli/` and `rook-server/` modules
- [ ] T002 [P] Create `rook-cli/go.mod` with module path `github.com/rook-project/rook-reference/rook-cli` and `go 1.23` directive
- [ ] T003 [P] Create `rook-server/go.mod` with module path `github.com/rook-project/rook-reference/rook-server` and `go 1.23` directive
- [ ] T004 [P] Create root `.golangci.yml` enabling linters: `gofmt`, `govet`, `errcheck`, `staticcheck`
- [ ] T005 [P] Create `.github/workflows/` directory structure (empty placeholder files for workflow files added in T020–T022)
- [ ] T006 Create root `Makefile` with targets: `build` (delegates to both modules), `test`, `lint`, `clean`; version injected via `git describe --tags --always` falling back to `dev`
- [ ] T007 [P] Create `rook-cli/Makefile` with targets: `build` (output to `../../dist/rook-cli`), `test`, `lint`
- [ ] T008 [P] Create `rook-server/Makefile` with targets: `build` (output to `../../dist/rook-server-cli`), `test`, `lint`

**Checkpoint**: Repository has a valid Go workspace; `go work sync` succeeds; Makefile targets are wired.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure shared by all user stories — directory layout, XDG path resolution, and the Config entity. These must complete before any user story implementation.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T009 Create `rook-cli/internal/config/xdg.go` — `ConfigPath()` function: returns `ROOK_CONFIG_PATH` if set, else `$XDG_CONFIG_HOME/rook/config.json`, else `~/.config/rook/config.json`
- [ ] T010 Create `rook-cli/internal/config/config.go` — `Config` struct (fields: `Servers []ServerEntry`, `ActiveSpace string`, `StorageDir string`, `FeatureFlags map[string]bool`), `ServerEntry` struct (fields: `Address string`, `Alias string`), sentinel `var ErrNotFound`, `Load(path string) (Config, error)` using `json.Decoder` (unknown fields permitted), `Save(path string, cfg Config) error` using atomic write-to-temp + `os.Rename`
- [ ] T011 Create `rook-cli/internal/config/config_test.go` — table-driven tests: `Load` returns `ErrNotFound` for missing file; `Load` round-trips a valid JSON config; `Save` writes indented JSON and file is readable by `Load`; `Save` is atomic (file is unchanged if marshal fails); `XDG_CONFIG_HOME` override; `ROOK_CONFIG_PATH` override
- [ ] T012 [P] Create `rook-cli/go.sum` (populated by running `go mod tidy` inside `rook-cli/` after any dependencies are added; at v0.1 stdlib-only so sum file will be minimal)
- [ ] T013 [P] Create `rook-server/go.sum` (same — `go mod tidy` inside `rook-server/`)

**Checkpoint**: `go test ./...` in `rook-cli/` passes for the config package. Foundation ready.

---

## Phase 3: User Story 1 — Developer Clones and Builds the Project (Priority: P1) 🎯 MVP

**Goal**: A contributor with Go 1.23+ can clone the repo, run `make build`, and produce working `rook-cli` and `rook-server-cli` binaries. Each binary prints a version string and exits 0.

**Independent Test**: Run `make build` from repo root → `./dist/rook-cli --version` prints `rook version dev` and exits 0 → `./dist/rook-server-cli --version` prints `rook-server-cli version dev` and exits 0.

### Implementation for User Story 1

- [ ] T014 [US1] Create `rook-cli/main.go` — parses `--version` / `-v` (prints `rook version <version>`, exits 0) and `--help` / `-h` (prints usage including config path, exits 0); calls `config.Load(config.ConfigPath())`; on `ErrNotFound` prints a single-line notice and exits 0; on other errors prints to stderr and exits 1
- [ ] T015 [P] [US1] Create `rook-server/cmd/admin/main.go` — stub: `var version string`; prints `rook-server-cli version <v>` (zero value → `dev`) and exits 0; accepts `--version` / `-v` and `--help` / `-h`
- [ ] T016 [US1] Run `go work sync` and verify `go build ./...` succeeds in both modules (update `go.work.sum` as needed)
- [ ] T017 [US1] Verify `make build` from repo root produces `dist/rook-cli` and `dist/rook-server-cli`; add `dist/` to `.gitignore`

**Checkpoint**: `make build && ./dist/rook-cli --version && ./dist/rook-server-cli --version` all succeed. User Story 1 is independently testable and complete.

---

## Phase 4: User Story 2 — CI Validates Every Push Automatically (Priority: P2)

**Goal**: Path-filtered GitHub Actions workflows trigger lint, build, and test per component on every push/PR to `main`. A failing check blocks merge.

**Independent Test**: Push a change to `rook-cli/` → only `ci-rook-cli.yml` triggers → all steps (lint, test, build) pass. Introduce a lint error → CI reports failure.

### Implementation for User Story 2

- [ ] T018 [US2] Create `.github/workflows/_go-ci.yml` — reusable workflow (`workflow_call`) with `working-directory` input; steps: `actions/checkout@v4`, `actions/setup-go@v5` (version from `go.mod`), `golangci/golangci-lint-action@v6`, `go test ./... -race -count=1`, `go build ./...`
- [ ] T019 [P] [US2] Create `.github/workflows/ci-rook-cli.yml` — triggers on `push`/`pull_request` to `main` with `paths: ['rook-cli/**', '.github/workflows/ci-rook-cli.yml', '.github/workflows/_go-ci.yml']`; calls `_go-ci.yml` with `working-directory: rook-cli`
- [ ] T020 [P] [US2] Create `.github/workflows/ci-rook-server-cli.yml` — same pattern; `paths: ['rook-server/cmd/admin/**', '.github/workflows/ci-rook-server-cli.yml', '.github/workflows/_go-ci.yml']`; `working-directory: rook-server`
- [ ] T021 [US2] Document branch protection setup in `rook-cli/README.md`: enable "Require status checks to pass" on `main` for `ci-rook-cli / ci` and `ci-rook-server-cli / ci`

**Checkpoint**: CI workflow files are syntactically valid YAML; path filters are correct; reusable workflow is referenced correctly with no hardcoded credentials or `@latest` action refs.

---

## Phase 5: User Story 3 — Developer Reads and Writes Config (Priority: P3)

**Goal**: Config is loaded from the correct XDG-resolved path; missing config is handled gracefully; writes are atomic. This user story is implemented primarily via the foundational config package (Phase 2) and the main entrypoint wiring (Phase 3). This phase covers integration verification and documentation.

**Independent Test**: Set `ROOK_CONFIG_PATH=/tmp/test.json` → run `rook-cli` → no crash; write a valid config to that path → run `rook-cli` → config loaded correctly; interrupt a `Save()` mid-write (simulated in unit tests via T011) → config file unchanged.

### Implementation for User Story 3

- [ ] T022 [US3] Add `config.Validate(cfg Config) error` to `rook-cli/internal/config/config.go` — validates `ServerEntry.Address` is non-empty absolute URL when servers list is non-empty; validates `StorageDir` is absolute path if non-empty; returns nil for empty/zero-value Config
- [ ] T023 [US3] Update `rook-cli/main.go` to call `config.Validate()` after `config.Load()`; on validation error, print actionable message to stderr and exit 1
- [ ] T024 [US3] Add test cases to `rook-cli/internal/config/config_test.go` for `Validate`: valid config passes; empty Address in ServerEntry fails; relative StorageDir fails; unknown FeatureFlags keys round-trip without loss
- [ ] T025 [P] [US3] Document config path resolution and env-var overrides in `rook-cli/README.md` config section (XDG table, `ROOK_CONFIG_PATH`, `XDG_CONFIG_HOME`)

**Checkpoint**: All config unit tests pass with `-race`; `rook-cli` handles missing, invalid, and valid configs correctly in all tested scenarios.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, developer tooling, and repo hygiene that spans all user stories.

- [ ] T026 [P] Create `rook-cli/README.md` — sections: Overview, Prerequisites, Clone & Build, Test, Lint, Config Paths (XDG table), Per-Module Commands, Troubleshooting (from quickstart.md)
- [ ] T027 [P] Create root `README.md` — project overview, monorepo layout, quick-start (`make build`), link to `rook-cli/README.md` and `specs/`
- [ ] T028 [P] Create `CONTRIBUTING.md` at repo root — Go version pinning note, `go work sync` requirement, Makefile targets reference, PR process, branch naming (`NNN-feature-name`)
- [ ] T029 Add `.gitignore` entries: `dist/`, `*.test`, `*.out`, coverage files
- [ ] T030 [P] Run `make lint && make test && make build` from repo root and confirm all pass; fix any issues discovered

**Checkpoint**: `make lint && make test && make build` succeeds cleanly from a fresh clone with Go 1.23+. All three user stories verified. Repository is contributor-ready.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 completion — **BLOCKS all user stories**
- **User Story 1 (Phase 3)**: Depends on Phase 2 — no dependency on US2 or US3
- **User Story 2 (Phase 4)**: Depends on Phase 2 — no dependency on US1 or US3
- **User Story 3 (Phase 5)**: Depends on Phase 2 **and** Phase 3 (T014 must exist for `main.go` wiring in T023)
- **Polish (Phase 6)**: Depends on all user story phases complete

### User Story Dependencies

- **US1 (P1)**: Independent after Foundational — no story dependencies
- **US2 (P2)**: Independent after Foundational — no story dependencies; can run in parallel with US1
- **US3 (P3)**: Requires US1 complete (T014 `main.go` exists before T023 can update it)

### Within Each User Story

- Config package (T009–T011) before entrypoint (T014, T015)
- Entrypoint (T014) before integration verification (T016, T017)
- Reusable workflow (T018) before per-component workflows (T019, T020)

### Parallel Opportunities

- T002, T003, T004, T005 — all Phase 1 setup, different files
- T007, T008 — per-module Makefiles, different directories
- T009, T010, T011 — different files in the same package (T011 can be written alongside T009/T010)
- T014, T015 — different binaries, different directories
- T019, T020 — different workflow files
- T026, T027, T028 — different documentation files

---

## Parallel Example: User Story 1

```bash
# Once Foundational (Phase 2) is complete, these can run in parallel:
Task T014: "Create rook-cli/main.go"
Task T015: "Create rook-server/cmd/admin/main.go"
# Then sequentially:
Task T016: "go work sync + go build ./... verification"
Task T017: "make build verification + .gitignore"
```

## Parallel Example: User Story 2

```bash
# T018 must complete first, then T019 and T020 can run in parallel:
Task T019: "Create .github/workflows/ci-rook-cli.yml"
Task T020: "Create .github/workflows/ci-rook-server-cli.yml"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (config package + tests)
3. Complete Phase 3: User Story 1 (binary entrypoints + `make build` verification)
4. **STOP and VALIDATE**: `make build && ./dist/rook-cli --version && ./dist/rook-server-cli --version`
5. This is a shippable skeleton — both binaries build and run

### Incremental Delivery

1. Setup + Foundational → workspace builds, config tests pass
2. Add US1 → both binaries runnable → **MVP deliverable**
3. Add US2 → CI workflows live → push gates enforced
4. Add US3 → config validation complete → edge cases covered
5. Polish → documentation and repo hygiene complete → v0.1 ready to tag

### Parallel Team Strategy (2 developers)

1. Both complete Phase 1 + Phase 2 together
2. Developer A: Phase 3 (US1 — binary entrypoints)
3. Developer B: Phase 4 (US2 — CI workflows) — independent of US1
4. Developer A: Phase 5 (US3 — config validation) after Phase 3
5. Both: Phase 6 (Polish) in parallel

---

## Notes

- `[P]` tasks target different files with no shared in-progress dependencies — safe to execute concurrently
- `[Story]` label maps each task to its user story for traceability to spec.md acceptance scenarios
- No TDD/test-first workflow was requested; unit tests for config (T011, T024) are included because atomic-write correctness (FR-009) and config validation (FR-008) are verifiable requirements that benefit from automated tests
- The `go.sum` files (T012, T013) are populated by `go mod tidy` — at v0.1 with stdlib-only dependencies they will be minimal or empty; the tasks serve as an explicit reminder to commit them
- Version injection (`-ldflags`) is handled by the Makefile, not the source code — source only reads the injected `var version string`
- Commit after each phase checkpoint or after each `[P]`-group completes
