# Rook v0.1 — Project Skeleton

**ID:** PRD002  
**Version:** v0.1  
**Status:** Stub  
**Date:** 2026-04-25  
**Author:** Mona Maret  
**Parent:** [PRD001 — Rook Project Overview for PoC](PRD001-rook-overview-v1.0.md)  

---

## Overview

This release establishes the foundational structure of the Rook project: the `rook-cli` and `rook-server` repository and module layouts, CI pipelines, and local development tooling. It also defines the configuration file schema and XDG-compliant path conventions that all subsequent releases depend on. No user-facing features are delivered; the goal is a working skeleton that any contributor can clone, build, and run locally.

---

## Feature Focus

_To be detailed in scoping. Proposed areas:_

- `rook-cli` Go module structure: package layout, entrypoint, build targets, and Makefile
- `rook-server` Go module structure: service directory layout, shared packages, and Makefile
- CI pipeline: GitHub Actions workflows for lint, build, and test on push/PR for both repos
- Local dev setup: documented prerequisites, `make dev` bootstrap, and toolchain version pinning
- Configuration file schema: TOML or YAML format, field definitions for server endpoint, identity, and feature flags
- XDG Base Directory compliance: config, data, cache, and state paths for `rook-cli` with OS fallbacks
- Config read/write utilities: load-from-disk, write-to-disk, and env-var override support
- Developer documentation: README, local dev guide, and contributing guidelines

---

## Dependencies

- None

---

## Out of Scope for This Release

- Authentication or identity — no user sessions, no keypair generation
- Any running services — no gRPC or HTTP servers started
- TUI or interactive CLI screens — CLI exits after printing version/help
- Firestore or cloud infrastructure — no GCP resources provisioned
- Docker or Cloud Run deployment configuration

---

## Open Questions

_To be resolved during scoping._

- Should `rook-cli` and `rook-server` live in the same monorepo or separate repos, and how does the CI strategy differ between the two?
- What is the canonical config file name and location (e.g. `~/.config/rook/config.toml`), and what fields are required vs. optional at this stage?
- Which Go version and toolchain constraints should be pinned in `go.work` or `go.mod` to ensure reproducible builds across contributors?

---

## Notes

_Space for any early design notes, constraints, or decisions relevant to this release._

- XDG paths should degrade gracefully on macOS and Windows; document the fallback locations explicitly.
- CI should gate on `go vet`, `staticcheck`, and `golangci-lint` in addition to `go test` from day one to avoid accruing lint debt.
- Config file writes should be atomic (write-to-temp, rename) to prevent corruption on interrupted writes.
