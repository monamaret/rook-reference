# rook-cli

The local-first Go TUI client for [Rook](../README.md) — a private collaboration platform for teams that prioritise data ownership and offline-first workflows.

## Overview

`rook-cli` is the only client for `rook-server`. It works fully offline for document stash, message history, and search. Server connectivity is required only for sync operations.

**v0.1 status**: Project skeleton. The binary builds and prints version information. The TUI launcher and all user-facing features are implemented from v0.2 onwards.

## Prerequisites

| Tool | Version | Install |
|---|---|---|
| Go | 1.23+ | [go.dev/dl](https://go.dev/dl) or `brew install go` |
| `make` | Any | Pre-installed on Linux/macOS; WSL: `sudo apt install build-essential` |
| `golangci-lint` | 1.57+ | `brew install golangci-lint` or [golangci-lint.run](https://golangci-lint.run/welcome/install/) |

> **WSL users**: Linux XDG paths apply. No native Windows build is supported.

## Clone and Build

From the **repository root**:

```bash
git clone https://github.com/rook-project/rook-reference.git
cd rook-reference
make build
./dist/rook-cli --version
```

Or build only `rook-cli`:

```bash
cd rook-cli
make build
../dist/rook-cli --version
```

## Test

```bash
# From repo root:
make test

# From rook-cli/ only:
cd rook-cli && make test
```

## Lint

```bash
# From repo root:
make lint

# From rook-cli/ only:
cd rook-cli && make lint
```

## Config Paths

`rook-cli` reads its configuration from:

| Platform | Path |
|---|---|
| Linux | `$XDG_CONFIG_HOME/rook/config.json` → `~/.config/rook/config.json` |
| macOS | `$XDG_CONFIG_HOME/rook/config.json` → `~/.config/rook/config.json` |
| WSL | Same as Linux |

### Environment Variables

| Variable | Effect |
|---|---|
| `ROOK_CONFIG_PATH` | Overrides XDG resolution; config is read from this absolute path |
| `XDG_CONFIG_HOME` | Base directory for XDG config resolution |

### Branch Protection

Enable "Require status checks to pass" on `main` for:
- `ci-rook-cli / ci`
- `ci-rook-server-cli / ci`

## Per-Module Commands

```bash
cd rook-cli/
make build    # build rook-cli only
make test     # test rook-cli only
make lint     # lint rook-cli only
make clean    # remove dist/rook-cli
```

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `go: cannot find module…` | `go.work` out of sync | `go work sync` from repo root |
| `golangci-lint: command not found` | Not installed | See prerequisites |
| Config not found at startup | Expected in v0.1 | Binary exits cleanly; config setup is v0.2+ |
| `undefined: version` | Missing `-ldflags` | Use `make build`, not `go build` directly |
