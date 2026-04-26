# Architecture Decision Records (ADR)

An Architecture Decision Record (ADR) captures an important architecture decision along with its context and consequences.

## Conventions

- Directory: `specs/decisions/`
- Naming: Date-prefixed files — `YYYY-MM-DD-short-title.md`
- Status values: `proposed`, `accepted`, `rejected`, `deprecated`, `superseded`

## Workflow

- Create a new ADR as `proposed`.
- Discuss and iterate.
- When the team commits: mark it `accepted` (or `rejected`).
- If replaced later: create a new ADR and mark the old one `superseded` with a link.

## ADRs

| Date | Title | Status |
|------|-------|--------|
| 2026-04-25 | [Adopt Architecture Decision Records](2026-04-25-adopt-adrs.md) | accepted |
| 2026-04-25 | [Rook Reference System Architecture](2026-04-25-rook-reference-system-architecture.md) | proposed |
| 2026-04-25 | [Real-Time Messaging Protocol: IRC vs. Custom](2026-04-25-real-time-messaging-protocol.md) | proposed |
| 2026-04-25 | [rook-cli Features and UX Architecture](2026-04-25-rook-cli-features-and-ux-architecture.md) | proposed |
| 2026-04-25 | [Use gRPC for rook-server Inter-Service Communication](2026-04-25-grpc-inter-service-communication.md) | accepted |
| 2026-04-25 | [SSH Auth Identity Chain and Cloud Run Deployment Topology](2026-04-25-ssh-auth-identity-chain-and-cloud-run-topology.md) | proposed |
| 2026-04-25 | [Guides Service Architecture](2026-04-25-guides-service-architecture.md) | proposed |
| 2026-04-25 | [Guides Offline Store](2026-04-25-guides-offline-store.md) | proposed |
| 2026-04-25 | [rook-server Admin CLI](2026-04-25-rook-server-admin-cli.md) | proposed |
| 2026-04-26 | [Adopt GitHub Actions CI/CD Pipeline](2026-04-26-github-actions-ci-cd-pipeline.md) | accepted |
