# Release Guides

Each release of Rook has a corresponding release guide in this directory. Release guides are living documents — they start as `draft` before the release begins and are updated to `ready` (pre-tag) and `released` (post-tag) as the release progresses.

## Conventions

- **Template:** Copy [`release-guide-template.md`](release-guide-template.md) to create a new guide.
- **File naming:** `vX.Y-release-guide.md` (e.g., `v0.2-release-guide.md`). Hotfixes: `vX.Y.Z-release-guide.md`.
- **Version naming:** `rook vMAJOR.MINOR[.PATCH]` — see the naming convention section in the template for full rules.
- **Git tag format:** `vX.Y` or `vX.Y.Z` — applied to `main` after CI passes and all checklist items are complete.
- **Status lifecycle:** `draft` → `ready` → `released`.

## Release Index

| Version | Codename | PRD | Status | Target Date |
|---------|----------|-----|--------|-------------|
| [v0.1](v0.1-release-guide.md) | Project Skeleton | [PRD002](../product/PRD002-rook-v0.1-project-skeleton.md) | `ready` | 2026-04-26 |
| v0.2 | Auth Foundation | [PRD003](../product/PRD003-rook-v0.2-auth-foundation.md) | — | — |
| v0.3 | Spaces and Identity | [PRD004](../product/PRD004-rook-v0.3-spaces-and-identity.md) | — | — |
| v0.4 | Stash Service | [PRD005](../product/PRD005-rook-v0.4-stash-service.md) | — | — |
| v0.5 | Stash Sync | [PRD006](../product/PRD006-rook-v0.5-stash-sync.md) | — | — |
| v0.6 | Messaging Service | [PRD007](../product/PRD007-rook-v0.6-messaging-service.md) | — | — |
| v0.7 | Messaging Sync | [PRD008](../product/PRD008-rook-v0.7-messaging-sync.md) | — | — |
| v0.8 | Guides Reader | [PRD009](../product/PRD009-rook-v0.8-guides-reader.md) | — | — |
| v0.9 | Guide Builder | [PRD010](../product/PRD010-rook-v0.9-guide-builder.md) | — | — |
| v1.0 | PoC Hardening + Admin CLI | [PRD011](../product/PRD011-rook-v1.0-poc-hardening-admin-cli.md) | — | — |

_Add a row and link the release guide here once a guide file is created for that version._

## Related

- [Project Guidelines — Speckit Workflow and Git Branching](../../.tabnine/guidelines/project-guidelines.md)
- [ADR Index](../decisions/README.md)
- [Product PRDs](../product/)
