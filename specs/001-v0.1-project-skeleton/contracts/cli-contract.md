# CLI Contract: Rook v0.1

**Feature**: [spec.md](../spec.md) | **Plan**: [plan.md](../plan.md)  
**Date**: 2026-04-26

Both `rook-cli` and `rook-server-cli` are command-line tools. This document defines the interface contract each binary exposes in v0.1: accepted flags, exit codes, output format, and environment-variable conventions. Downstream tooling (CI scripts, release pipeline, future shell completions) must rely only on the behaviour documented here.

---

## `rook-cli`

### Invocation

```
rook [flags]
```

### Flags

| Flag | Short | Description | Exit on use |
|---|---|---|---|
| `--version` | `-v` | Print `rook version <version>` to stdout and exit 0 | Yes |
| `--help` | `-h` | Print usage summary to stdout and exit 0 | Yes |

No positional arguments or subcommands in v0.1. All future subcommands (`auth`, `stash`, `messages`, etc.) are out of scope for this release.

### Output

```
# --version
rook version 0.1.0

# --help
Usage: rook [flags]

Flags:
  -v, --version   Print version and exit
  -h, --help      Print this help and exit

Config is read from $XDG_CONFIG_HOME/rook/config.json
(fallback: ~/.config/rook/config.json)
```

### Exit Codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Unrecoverable error (message printed to stderr) |

### Environment Variables

| Variable | Effect |
|---|---|
| `XDG_CONFIG_HOME` | Base directory for config resolution. If unset, falls back to `~/.config`. |
| `ROOK_CONFIG_PATH` | If set, overrides XDG resolution entirely; config is read from this absolute path. |

### Config Resolution Order

1. `ROOK_CONFIG_PATH` (if set and non-empty)
2. `$XDG_CONFIG_HOME/rook/config.json`
3. `~/.config/rook/config.json`

If the resolved path does not exist, `rook-cli` proceeds with a zero-value `Config{}` (no error, no crash). The caller is responsible for interpreting `ErrNotFound` from `config.Load()`.

### Stability Guarantees

- `--version` output format is stable: `rook version <semver>`. Parsers may rely on the third whitespace-delimited token being the version string.
- `--help` output is informational only; its format may change between releases.
- Exit codes `0` and `1` are stable. Additional exit codes may be added in future releases with distinct semantics.

---

## `rook-server-cli`

### Invocation (v0.1 stub)

```
rook-server-cli [flags]
```

### Flags

| Flag | Short | Description | Exit on use |
|---|---|---|---|
| `--version` | `-v` | Print `rook-server-cli version <version>` to stdout and exit 0 | Yes |
| `--help` | `-h` | Print usage summary to stdout and exit 0 | Yes |

No subcommands in v0.1. The admin subcommand tree (`user`, `space`) is implemented in v0.3+.

### Output

```
# --version
rook-server-cli version 0.1.0

# default (no flags)
rook-server-cli version dev
```

The default invocation (no flags) prints the version string and exits 0 in v0.1. This behaviour will change in v0.3 when subcommands are added (no flags will then print help).

### Exit Codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Unrecoverable error (future releases; stub always exits 0) |

### Environment Variables

None in v0.1. In v0.3+ the binary will consume `USER_SERVICE_ADDR` and `ROOK_ADMIN_TOKEN`.

### Version Injection

Both binaries receive their version string at build time:

```
go build -ldflags "-X main.version=0.1.0" ./...
```

The zero value (empty string) produces the suffix `dev` in output:

```go
// In main.go:
var version string  // set by -ldflags

func main() {
    v := version
    if v == "" {
        v = "dev"
    }
    fmt.Printf("rook-server-cli version %s
", v)
}
```

The Makefile injects the version from a `VERSION` file or `git describe --tags` at build time.

---

## Build Contract (Makefile Targets)

The root and per-module Makefiles expose these targets as the stable build interface:

| Target | Scope | Description |
|---|---|---|
| `make build` | Root | Runs `go work sync` then builds both binaries to `dist/` |
| `make test` | Root | Runs `go test ./... -race -count=1` in both modules |
| `make lint` | Root | Runs `golangci-lint run` against both modules |
| `make clean` | Root | Removes `dist/` |
| `make build` | Per-module | Builds that module's binary only |
| `make test` | Per-module | Tests that module only |
| `make lint` | Per-module | Lints that module only |

Binary output paths:

```
dist/
├── rook-cli              (from rook-cli/main.go)
└── rook-server-cli       (from rook-server/cmd/admin/main.go)
```

Cross-compiled release binaries follow the naming convention `<binary>_<os>_<arch>` (e.g., `rook-cli_linux_amd64`), but cross-compilation is handled by `release.yml` in CI — not by the Makefile.
