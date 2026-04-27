# Rook Reference

A private, self-hosted collaboration platform for small teams that prioritise local-first workflows and data ownership.

Rook gives teams a shared space for async messaging and a document stash — both usable entirely offline — that sync on demand with a central server.

## Repository Layout

```
rook-reference/
├── rook-cli/          # Local-first Go TUI client
├── rook-server/       # Go microservices (Cloud Run)
│   └── cmd/admin/     # rook-server-cli admin binary
├── rook-docs/         # Documentation site (Hugo)
├── specs/             # Speckit specifications, plans, and ADRs
│   ├── architecture/  # Component overview and gRPC call flows
│   ├── decisions/     # Architecture Decision Records
│   └── product/       # Product requirement documents
├── go.work            # Go workspace (links rook-cli and rook-server modules)
├── Makefile           # Root build, test, lint, and clean targets
└── .golangci.yml      # Shared golangci-lint configuration
```

## Quick Start

**Prerequisites**: Go 1.23+, `make`, `git`, `golangci-lint`

```bash
git clone https://github.com/rook-project/rook-reference.git
cd rook-reference
make build
./dist/rook-cli --version
./dist/rook-server-cli --version
```

## Available Make Targets

| Target | Description |
|---|---|
| `make build` | Build both binaries to `dist/` |
| `make test` | Run all tests with race detector |
| `make lint` | Lint both modules with golangci-lint |
| `make clean` | Remove `dist/` |

## Documentation

- [rook-cli README](rook-cli/README.md) — CLI setup, config paths, and usage
- [Component Overview](specs/architecture/component-overview.md) — system architecture diagram
- [gRPC Call Flows](specs/architecture/grpc-call-flows.md) — inter-service communication
- [Product Overview](specs/product/PRD001-rook-overview-v1.0.md) — full feature roadmap
- [Architecture Decisions](specs/decisions/) — ADR index

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development workflow, branch naming, and PR guidelines.
