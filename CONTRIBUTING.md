# Contributing to Rook Reference

## Prerequisites

| Tool | Version | Notes |
|---|---|---|
| Go | 1.23.x | Pin to minor version; see `go.mod` in each module for the exact directive |
| `make` | Any | Build interface for all modules |
| `golangci-lint` | 1.57+ | Required for local lint; same version used in CI |
| `git` | 2.x+ | — |

> Install Go via [go.dev/dl](https://go.dev/dl), `brew install go`, or `asdf install golang 1.23.x`.

## Repository Structure

This is a Go monorepo managed with a `go.work` workspace:

```
go.work            # workspace root — lists rook-cli/ and rook-server/
rook-cli/          # module: github.com/rook-project/rook-reference/rook-cli
rook-server/       # module: github.com/rook-project/rook-reference/rook-server
```

## Setup

```bash
git clone https://github.com/rook-project/rook-reference.git
cd rook-reference
go work sync       # sync workspace after clone or after adding new modules
make build         # build all binaries to dist/
make test          # run all tests
make lint          # run all linters
```

## Development Workflow

1. Create a feature branch: `git checkout -b NNN-short-description` (e.g. `002-auth-foundation`)
2. Make changes in the relevant module directory
3. Run `make lint && make test && make build` before committing
4. Commit with a clear, concise message (see style below)
5. Open a pull request against `main`

## Commit Message Style

Follow conventional commit format where practical:

```
type(scope): short description

Longer explanation if needed. Reference issues with #NNN.
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `ci`

Examples:
- `feat(config): add XDG path resolution with env override`
- `ci: add path-filtered CI workflow for rook-cli`
- `docs: update CONTRIBUTING with go work sync step`

## Go Workspace

- `go.work` and `go.work.sum` are committed to the repository
- Run `go work sync` after cloning or after adding a new Go module
- Do **not** add `replace` directives to `go.mod` files — use the workspace instead
- Each module maintains its own `go.mod` and `go.sum`

## Pull Request Guidelines

- PRs must pass all CI checks (`lint`, `test`, `build`) before merge
- Branch protection is enabled on `main` — direct pushes are not permitted
- Keep PRs focused: one logical change per PR
- Reference the related issue number in the PR description

## Makefile Targets

| Target | Description |
|---|---|
| `make build` | Build all binaries (root) or current module's binary (per-module) |
| `make test` | Run tests with `-race -count=1` |
| `make lint` | Run `golangci-lint` with project config |
| `make clean` | Remove `dist/` |

## Adding a New Go Module

1. Create the module directory and `go.mod`
2. Add the module to `go.work` under the `use` block
3. Run `go work sync`
4. Add a per-module CI workflow in `.github/workflows/` following the existing pattern
5. Update the root `Makefile` `build`, `test`, and `lint` targets
