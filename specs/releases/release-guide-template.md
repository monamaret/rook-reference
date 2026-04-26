# Release Guide — `rook vX.Y` — <Release Name>

**Release ID:** REL-XXX
**Version:** vX.Y
**Status:** `draft` | `ready` | `released`
**PRD:** [PRDXXX — <PRD Title>](../product/PRDXXX-rook-vX.Y-<slug>.md)
**Release Date (target):** YYYY-MM-DD
**Release Date (actual):** —
**Author:** <name>

---

## Release Naming Convention

Rook releases follow the pattern:

```
rook vMAJOR.MINOR[.PATCH]
```

| Segment | Rule |
|---------|------|
| **MAJOR** | Incremented for architectural breaks or scope resets (post-PoC only). |
| **MINOR** | Incremented for each planned product release (v0.1, v0.2, …). Matches the milestone PRD number. |
| **PATCH** | Reserved for hotfixes on a released minor. Created from a `hotfix/vX.Y.Z-<slug>` branch. |

**Release naming examples:**

| Version | PRD | Codename |
|---------|-----|----------|
| v0.1 | PRD002 | Project Skeleton |
| v0.2 | PRD003 | Auth Foundation |
| v0.3 | PRD004 | Spaces and Identity |
| v0.4 | PRD005 | Stash Service |
| v0.5 | PRD006 | Stash Sync |
| v0.6 | PRD007 | Messaging Service |
| v0.7 | PRD008 | Messaging Sync |
| v0.8 | PRD009 | Guides Reader |
| v0.9 | PRD010 | Guide Builder |
| v1.0 | PRD011 | PoC Hardening + Admin CLI |

The **codename** is the human-readable short title taken directly from the corresponding PRD title. It is used in branch names, release tags, and this guide's heading. Do not invent a separate codename that diverges from the PRD title.

**Git tag format:** `v0.3`, `v1.0.1` (no `release/` prefix — tags are the canonical version marker).

---

## 1. Overview

<!--
One paragraph: what does this release accomplish and why does it matter?
Refer to the parent PRD overview for context. Do not repeat the PRD verbatim —
summarise the release outcome in terms of user and operator value.
-->

---

## 2. Planned Features

<!--
List every feature included in this release. Each row must link to its
governing PRD section and, where a formal architectural decision exists,
to the relevant ADR.

Add rows from the PRD's "Feature Focus" section. Use the acceptance criteria
in the PRD as the definition of done.
-->

| # | Feature | PRD Section | ADR | Notes |
|---|---------|------------|-----|-------|
| 1 | <feature name> | [§X.X PRD title](../product/PRDXXX-rook-vX.Y-<slug>.md#xx-section-anchor) | [ADR title](../decisions/YYYY-MM-DD-<slug>.md) | — |
| 2 | | | — | |

**Definition of done:** All acceptance criteria in the linked PRD sections are met, all tasks in `tasks.md` are marked `[X]`, and all linked GitHub Issues are closed.

---

## 3. Architecture and Design Decisions

<!--
List only the ADRs that are *first introduced* or *updated* in this release.
ADRs that remain unchanged from a prior release do not need to be listed here.
-->

| ADR | Status | Summary |
|-----|--------|---------|
| [<title>](../decisions/YYYY-MM-DD-<slug>.md) | `proposed` / `accepted` | One-line rationale. |

---

## 4. Speckit Workflow Checklist

> The project uses the **Speckit Full SDD Cycle** for all feature development.
> See [Project Guidelines §1](../../.tabnine/guidelines/project-guidelines.md#1-speckit-workflow) for the full stage definitions and review gate rules.

| Step | Command | Artefact | Done |
|------|---------|----------|------|
| 1. Specify | `/speckit.specify` | `spec.md` | ☐ |
| — | **Review gate** — approve spec | — | ☐ |
| 2. Plan | `/speckit.plan` | `plan.md`, `research.md`, `data-model.md`, `contracts/` | ☐ |
| — | **Review gate** — approve plan | — | ☐ |
| 3. Tasks | `/speckit.tasks` | `tasks.md` | ☐ |
| 4. Issues | `/speckit.taskstoissues` | GitHub Issues | ☐ |
| — | **Review gate** — review issues | — | ☐ |
| 5. Implement | `/speckit.implement` | code, tests, PRs | ☐ |

---

## 5. Git and Branch Strategy

> See [Project Guidelines §2](../../.tabnine/guidelines/project-guidelines.md#2-git-branching) for full branching rules and commit conventions.

### 5.1 Feature branch

Feature branches for this release are created by the `before_specify` hook or manually via:

```bash
.specify/extensions/git/scripts/bash/create-new-feature.sh 
  --json 
  --short-name "<action-noun-slug>" 
  "<feature description>"
```

Branch names follow the `sequential` mode: `NNN-<action-noun-slug>` (e.g., `007-add-auth-foundation`).

### 5.2 Task branches (if applicable)

When the release contains many tasks, create short-lived task branches off the feature branch:

```
main
 └─ NNN-<feature-slug>             ← feature branch
     ├─ NNN-<feature-slug>/T001-<task-slug>
     ├─ NNN-<feature-slug>/T005-<task-slug>
     └─ NNN-<feature-slug>/T012-<task-slug>
```

Each task branch is PR'd back into the feature branch. The feature branch is PR'd to `main` once all tasks are complete.

### 5.3 Release tag

After the feature branch is merged to `main` and CI passes, apply the version tag:

```bash
git tag -a vX.Y -m "rook vX.Y — <Release Name>"
git push origin vX.Y
```

---

## 6. CI / CD Gates

> See the [GitHub Actions CI/CD ADR](../decisions/2026-04-26-github-actions-ci-cd-pipeline.md) for pipeline architecture.

Before the release tag is applied, confirm:

- [ ] All required CI checks pass on `main` (lint, build, test).
- [ ] No open PRs targeting this release's feature branch.
- [ ] All tasks in `tasks.md` are marked `[X]`.
- [ ] All linked GitHub Issues are closed.

---

## 7. Release Readiness Checklist

### Code quality
- [ ] All acceptance criteria from the PRD are met (verified by tests or manual verification as noted).
- [ ] No `TODO`/`FIXME` comments left in code added this release (or tracked as follow-up issues).
- [ ] No secrets or credentials committed.
- [ ] All new code follows project conventions and constitution.

### Documentation
- [ ] Relevant ADRs updated to `accepted` status.
- [ ] This release guide updated: status → `ready`, actual release date filled in.
- [ ] `specs/releases/README.md` index updated with this release entry.
- [ ] PRD status updated if applicable.

### Verification
- [ ] Release has been smoke-tested end-to-end (describe scenario below).

**Smoke test scenario:**

<!-- Describe the minimal end-to-end scenario that validates this release is working. -->

---

## 8. Scope Boundaries

<!--
Explicitly state what is NOT in this release to prevent scope creep.
Copy or reference the "Out of Scope" section from the PRD if applicable.
-->

The following items are explicitly deferred from this release:

- <item>

See [PRD001 §6 — Out of Scope](../product/PRD001-rook-overview-v1.0.md#6-out-of-scope) for the global exclusion list.

---

## 9. Known Issues and Risks

| # | Issue / Risk | Severity | Mitigation / Owner |
|---|-------------|----------|--------------------|
| 1 | | | |

---

## 10. References

| Document | Path |
|----------|------|
| Project PRD (overview) | [`specs/product/PRD001-rook-overview-v1.0.md`](../product/PRD001-rook-overview-v1.0.md) |
| Release PRD | [`specs/product/PRDXXX-rook-vX.Y-<slug>.md`](../product/PRDXXX-rook-vX.Y-<slug>.md) |
| ADR index | [`specs/decisions/README.md`](../decisions/README.md) |
| Project Guidelines (Speckit + Git) | [`.tabnine/guidelines/project-guidelines.md`](../../.tabnine/guidelines/project-guidelines.md) |
| Component Overview | [`specs/architecture/component-overview.md`](../architecture/component-overview.md) |
| gRPC Call Flows | [`specs/architecture/grpc-call-flows.md`](../architecture/grpc-call-flows.md) |
