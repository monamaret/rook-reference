# Quickstart: Rook v0.1 — Developer Setup

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)  
**Date**: 2026-04-26  
**Audience**: Contributors and developers setting up Rook for the first time.

---

## Prerequisites

| Tool | Version | Install |
|---|---|---|
| Go | 1.23+ | [go.dev/dl](https://go.dev/dl) or `brew install go` / `asdf install golang 1.23.x` |
| `make` | Any | Pre-installed on Linux/macOS; WSL: `sudo apt install build-essential` |
| `git` | 2.x+ | Pre-installed or `brew install git` |
| `golangci-lint` | 1.57+ | `brew install golangci-lint` or see [golangci-lint.run](https://golangci-lint.run/welcome/install/) |

> **WSL users**: Use Linux XDG paths. No native Windows build is supported; WSL is the recommended Windows environment.

---

## Clone and Bootstrap

```bash
git clone https://github.com/rook-project/rook-reference.git
cd rook-reference
```

Verify the Go workspace is initialised:

```bash
go work sync
```

---

## Build

Build both binaries to `dist/`:

```bash
make build
```

This runs `go work sync` then builds:
- `dist/rook-cli` from `rook-cli/`
- `dist/rook-server-cli` from `rook-server/cmd/admin/`

Version is injected from `git describe --tags` (falls back to `dev` if no tag exists).

Verify:

```bash
./dist/rook-cli --version
# rook version dev

./dist/rook-server-cli --version
# rook-server-cli version dev
```

---

## Test

```bash
make test
```

Runs `go test ./... -race -count=1` in both `rook-cli/` and `rook-server/`. The race detector is always enabled.

---

## Lint

```bash
make lint
```

Runs `golangci-lint run` against both modules using `.golangci.yml` at the repo root. The same linter configuration runs in CI — local and CI lint results should be identical.

---

## Per-Module Commands

Work inside a single module without the workspace:

```bash
cd rook-cli/
make build    # build rook-cli only
make test     # test rook-cli only
make lint     # lint rook-cli only

cd ../rook-server/
make build    # build rook-server-cli only
make test     # test rook-server only
make lint     # lint rook-server only
```

---

## Config Paths

`rook-cli` reads its config from:

| Platform | Path |
|---|---|
| Linux | `$XDG_CONFIG_HOME/rook/config.json` → `~/.config/rook/config.json` |
| macOS | `$XDG_CONFIG_HOME/rook/config.json` → `~/.config/rook/config.json` |
| WSL | Same as Linux |

Override the path for testing:

```bash
ROOK_CONFIG_PATH=/tmp/test-config.json ./dist/rook-cli
```

In v0.1 there is no first-run setup flow — the CLI exits cleanly with a version/help message if no config exists. Config handling is fully implemented in v0.2.

---

## CI Workflow (for context)

Every push and pull request targeting `main` triggers per-component CI via GitHub Actions:

- Pushes to `rook-cli/**` trigger `ci-rook-cli.yml` only
- Pushes to `rook-server/cmd/admin/**` trigger `ci-rook-server-cli.yml` only
- Both workflows call the shared `_go-ci.yml` reusable workflow (lint → test → build)

Branch protection on `main` requires all relevant CI jobs to pass before a PR can merge.

To reproduce CI locally:

```bash
make lint && make test && make build
```

---

## Clean Up

```bash
make clean
# Removes dist/
```

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `go: cannot find module providing package ...` | `go.work` out of sync | Run `go work sync` |
| `golangci-lint: command not found` | Not installed | See prerequisites above |
| Config file not found at startup | Expected in v0.1 | CLI proceeds with defaults; config is written on first-run in v0.2 |
| Build fails: `undefined: version` | `-ldflags` not passed | Use `make build` not `go build` directly |
