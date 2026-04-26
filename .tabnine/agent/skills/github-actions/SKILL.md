# GitHub Actions Skill

## Purpose

This skill governs authoring, scaffolding, and validating GitHub Actions workflows for the Rook Reference monorepo. It encodes the conventions established in ADR [`2026-04-26-github-actions-ci-cd-pipeline.md`](../../../../specs/decisions/2026-04-26-github-actions-ci-cd-pipeline.md) so that coding agents can create correct, convention-compliant workflows without asking follow-up questions.

Use this skill when:
- Adding a new rook-server microservice that needs a CI workflow
- Adding a new top-level component (e.g., a new Go tool, a second docs site)
- Validating existing workflow files for convention compliance
- Debugging a failing GitHub Actions pipeline
- Reviewing a PR that modifies `.github/workflows/`

## Workflow Structure (Canonical)

```
.github/
  workflows/
    _go-ci.yml              # Reusable — called by all Go component workflows
    ci-auth-service.yml
    ci-user-service.yml
    ci-stash-service.yml
    ci-messaging-service.yml
    ci-guides-service.yml
    ci-rook-cli.yml
    ci-rook-docs.yml
    release.yml             # Tag-triggered release + Cloud Run deploy
```

## Operations

### Scaffold: New Go Service CI Workflow

When the user asks to add CI for a new Go service:

1. Determine the service directory (e.g., `rook-server/my-new-service`)
2. Copy `assets/templates/go-ci-component.yml`
3. Replace all `<service-name>` placeholders with the actual service directory name
4. Replace all `<display-name>` placeholders with a human-readable name
5. Write the file to `.github/workflows/ci-<service-name>.yml`
6. Confirm that `_go-ci.yml` exists; if not, scaffold it from `assets/templates/go-ci-reusable.yml`
7. Run the Validate operation (below) against the new file

### Scaffold: New Hugo Site CI Workflow

1. Copy `assets/templates/hugo-ci.yml`
2. Replace `<site-dir>` with the actual directory (e.g., `rook-docs`)
3. Write to `.github/workflows/ci-<site-dir>.yml`
4. Run the Validate operation against the new file

### Validate: Existing Workflow Files

For each `.github/workflows/*.yml` file (excluding files starting with `_`):

Check each rule in `references/workflow-conventions.md`. Report:
- ✅ PASS — convention met
- ❌ FAIL — convention violated (state the rule and the offending line)
- ⚠️ WARN — convention partially met or ambiguous

A workflow passes validation if it has zero FAILs. WARNs should be discussed with the user.

### Debug: Failing Pipeline

1. Read the failing workflow file
2. Read `references/workflow-conventions.md` for context
3. Identify the failing step from the error message
4. Check: correct `working-directory`? Pinned action version? Correct `go.mod` path?
5. Propose a targeted fix — do not rewrite the whole file unless necessary

## Key Constraints (Never Violate)

- Action versions MUST be pinned to a major version tag (e.g., `@v4`) — never `@latest`, never a SHA
- GCP credentials MUST use Workload Identity Federation — never a service account JSON key secret
- Deploy steps MUST NOT appear in CI (PR) workflows — only in `release.yml`
- Per-component CI workflows MUST call `_go-ci.yml` via `uses:` — do not inline Go CI steps
- Path filters MUST include the workflow files themselves (changes to `_go-ci.yml` trigger all consumers)

## Resources

- `references/workflow-conventions.md` — full rule set for validation
- `references/wif-setup.md` — Workload Identity Federation provisioning guide
- `assets/templates/go-ci-component.yml` — per-component Go CI template
- `assets/templates/go-ci-reusable.yml` — `_go-ci.yml` reusable workflow template
- `assets/templates/release.yml` — tag-triggered release workflow template
- `assets/templates/hugo-ci.yml` — Hugo site CI template
