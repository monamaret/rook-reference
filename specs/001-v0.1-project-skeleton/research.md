# Research: Rook v0.1 — Project Skeleton

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)  
**Date**: 2026-04-26  
**Status**: Complete — all decisions resolved via existing ADRs; no open clarifications.

---

## Summary

All technical decisions for v0.1 are closed. This document records the findings that inform `plan.md`, linking each to its authoritative source so implementers can trace decisions to their rationale without re-reading the full ADR corpus.

---

## Finding 1: Config format and path — JSON at XDG path

**Decision**: JSON (`encoding/json`), at `$XDG_CONFIG_HOME/rook/config.json`; fallback `~/.config/rook/config.json`.

**Rationale**:
- Standard library only — zero external dependencies for config parsing.
- PRD001 explicitly references `config.json` by name.
- Config is machine-written (not hand-edited), so TOML's comment support offers no benefit.
- JSON is unambiguous — no TOML multiline edge cases, no YAML indentation hazards.

**Alternatives considered**:
- TOML — adds a third-party dependency (`BurntSushi/toml`); comment benefit irrelevant for machine-written config.
- YAML — complex spec, implicit type coercions, indentation sensitivity. Not warranted for a small config schema.

**Source**: [v0.1 Foundational Decisions ADR §2](../../decisions/2026-04-26-v0.1-foundational-decisions.md)

---

## Finding 2: Go workspace strategy — `go.work` with per-component `go.mod`

**Decision**: Single `go.work` at repo root listing `rook-cli/` and `rook-server/`. Separate `go.mod` per deployable component. Both `go.work` and `go.work.sum` committed.

**Rationale**:
- Idiomatic solution for Go monorepos with multiple modules.
- Eliminates `replace` directives; keeps each module's dependency graph clean.
- Each component can independently pin its Go version and dependency set.
- CI can run `go test ./...` scoped to a single module directory.

**Alternatives considered**:
- Single root `go.mod` — couples `rook-cli` and `rook-server` into one version graph; prevents independent binary scoping.

**Source**: [v0.1 Foundational Decisions ADR §3](../../decisions/2026-04-26-v0.1-foundational-decisions.md)

---

## Finding 3: CI pipeline — path-filtered per-component GitHub Actions

**Decision**: Per-component workflow files (`ci-rook-cli.yml`, `ci-rook-server-cli.yml`) call a shared `_go-ci.yml` reusable workflow. Path filters on `push` and `pull_request` to `main`. Reusable workflow steps: `actions/checkout@v4`, `actions/setup-go@v5`, `golangci/golangci-lint-action@v6`, `go test ./... -race -count=1`, `go build ./...`.

**Rationale**:
- GitHub Actions available at zero additional cost.
- Path filtering ensures only affected components trigger on a given PR — keeps feedback fast as service count grows.
- Reusable workflow is the single source of truth for Go CI logic — prevents drift across per-component files.

**Alternatives considered**:
- Single unified workflow — triggers all components on every push; does not scale; an unrelated failure blocks unrelated PRs.
- External CI (CircleCI, Buildkite) — unnecessary cost and configuration surface at PoC scale.

**Source**: [GitHub Actions CI/CD ADR](../../decisions/2026-04-26-github-actions-ci-cd-pipeline.md)

---

## Finding 4: Atomic config writes — write-to-temp + `os.Rename`

**Decision**: `Save()` marshals JSON to a temporary file in the same directory (same filesystem), then calls `os.Rename()` to replace the target path. This is atomic on POSIX systems (`rename(2)`).

**Rationale**:
- `os.Rename` on Linux maps to `rename(2)`, which is atomic — the target either refers to the old file or the new file; a partial write is never visible.
- Writing directly to the target leaves a window where the file is empty or partial on interruption.
- PRD002 notes explicitly require atomic writes.

**Implementation note**: Temp file must be on the same filesystem as the target (same directory) for `rename(2)` to be atomic; `os.CreateTemp(dir, "config-*.json")` in the config directory satisfies this.

**Alternatives considered**:
- `fsync` before rename — additional durability guarantee; acceptable if desired but not strictly required for v0.1.
- Direct write — ruled out; risks corruption on interrupted writes.

---

## Finding 5: `rook-server-cli` stub

**Decision**: `rook-server/cmd/admin/main.go` exists from v0.1 as a no-op stub. Prints `rook-server-cli version <version>` and exits 0. Version injected via `-ldflags "-X main.version=<ver>"`; zero value produces `dev`.

**Rationale**:
- The GitHub Actions `release.yml` lists `rook-server-cli` (linux/amd64, darwin/arm64) as a v0.1 release binary. Without the entrypoint the pipeline fails from the first tag.
- A stub satisfies the pipeline requirement with minimal code; admin functionality is deferred to v0.3+.

**Source**: [v0.1 Foundational Decisions ADR §4](../../decisions/2026-04-26-v0.1-foundational-decisions.md)

---

## Finding 6: Lint toolchain — `golangci-lint`

**Decision**: `golangci-lint` run in CI via `golangci/golangci-lint-action@v6`. Config at `.golangci.yml` (repo root). Minimum enabled linters: `gofmt`, `govet`, `errcheck`, `staticcheck`.

**Rationale**:
- `golangci-lint` is the de-facto standard aggregator for Go linters; single binary, parallelised, widely cached in CI.
- `errcheck` and `staticcheck` catch error-handling bugs that `go vet` does not.
- `.golangci.yml` at repo root applies to both modules through the Go workspace.
- PRD002 notes name `go vet`, `staticcheck`, and `golangci-lint` explicitly.

**Alternatives considered**:
- `go vet` alone — insufficient; misses unchecked errors and many static analysis classes.
- `staticcheck` standalone — fewer ecosystem integrations and slower CI caching story than `golangci-lint`.

**Source**: [GitHub Actions CI/CD ADR §Configuration](../../decisions/2026-04-26-github-actions-ci-cd-pipeline.md)

---

## Finding 7: Monorepo structure confirmed

**Decision**: Single repository containing `rook-cli/`, `rook-server/`, and `rook-docs/` under one root.

**Rationale**:
- All Speckit artefacts, ADRs, and PRDs already coexist in this repo.
- GitHub Actions CI/CD ADR is designed around monorepo path-filtered workflows.
- No benefit to splitting at PoC scale; future split possible but requires no architectural commitment now.

**Source**: [v0.1 Foundational Decisions ADR §1](../../decisions/2026-04-26-v0.1-foundational-decisions.md)

---

## Open Questions / Risks

| # | Item | Risk | Mitigation |
|---|---|---|---|
| 1 | Constitution is an unfilled template | No architectural gates enforced for v0.1 | Author constitution before v0.2 begins |
| 2 | Go version not pinned to a specific patch (e.g. `1.23.0` vs `1.23`) | Minor divergence between contributors | Use `go 1.23` directive in `go.mod` and document exact toolchain in README/CONTRIBUTING |
| 3 | `rook-docs/` Go module status unclear | May need to be added to `go.work` later | Omit from `go.work` until `rook-docs` contains Go code |
| 4 | Workload Identity Federation for release pipeline | GCP WIF must be provisioned before first tag-triggered release | Document prerequisite in release guide; not a v0.1 blocker (no deploy, only binary build) |
