# Workflow Conventions

These are the canonical rules for all GitHub Actions workflow files in the Rook Reference monorepo. They are enforced by the github-actions skill's Validate operation and by code review.

## Rule Index

| ID | Rule | Severity |
|---|---|---|
| WF-01 | Action versions pinned to major version tag | FAIL |
| WF-02 | No `@latest` or branch-pinned references | FAIL |
| WF-03 | No hardcoded secrets or credentials | FAIL |
| WF-04 | Deploy steps only in `release.yml` | FAIL |
| WF-05 | Per-component Go workflows call `_go-ci.yml` via `uses:` | FAIL |
| WF-06 | Path filters include the workflow file itself and `_go-ci.yml` | FAIL |
| WF-07 | `working-directory` set correctly to the component root | FAIL |
| WF-08 | Release workflow uses Workload Identity Federation | FAIL |
| WF-09 | Workflow has a `name:` field | WARN |
| WF-10 | Each job has a `name:` field | WARN |
| WF-11 | `release.yml` tags images with both `${{ github.ref_name }}` and `latest` | WARN |

## Rule Details

### WF-01 — Action versions pinned to major version tag

All `uses:` references must end with `@v<N>` where N is a number.

✅ `uses: actions/checkout@v4`
❌ `uses: actions/checkout@main`
❌ `uses: actions/checkout@abc1234def` (SHA pinning not used in this project)
❌ `uses: actions/checkout@latest`

### WF-02 — No `@latest` or branch-pinned references

No `uses:` line may reference `@latest` or a branch name.

### WF-03 — No hardcoded secrets or credentials

No workflow file may contain literal values that look like tokens, keys, passwords, or project IDs. GCP project IDs and region names may appear as variables in `env:` blocks but must be sourced from `${{ secrets.* }}` or `${{ vars.* }}`.

### WF-04 — Deploy steps only in `release.yml`

Steps using `google-github-actions/deploy-cloudrun`, `gcloud run deploy`, or any equivalent must not appear in CI (PR) workflows. They belong exclusively in `release.yml`.

### WF-05 — Per-component Go workflows call `_go-ci.yml`

Every `ci-*.yml` file for a Go component must have exactly one job that calls `./.github/workflows/_go-ci.yml` via:

```yaml
jobs:
  ci:
    uses: ./.github/workflows/_go-ci.yml
```

Go CI steps (checkout, setup-go, lint, test, build) must not be inlined in per-component files.

### WF-06 — Path filters include workflow files

The `paths:` trigger in a per-component CI workflow must include:

```yaml
paths:
  - '<component-dir>/**'
  - '.github/workflows/ci-<component>.yml'
  - '.github/workflows/_go-ci.yml'
```

This ensures changes to the reusable workflow re-trigger all consumers.

### WF-07 — `working-directory` set correctly

The `with: working-directory:` passed to `_go-ci.yml` must match the directory containing the component's `go.mod` file.

### WF-08 — Release workflow uses Workload Identity Federation

`release.yml` must authenticate to GCP using `google-github-actions/auth@v2` with `workload_identity_provider` and `service_account` inputs sourced from secrets. It must NOT use `credentials_json`.

### WF-09 — Workflow has a `name:` field (WARN)

Every workflow file should have a top-level `name:` for readability in the GitHub Actions UI.

### WF-10 — Each job has a `name:` field (WARN)

Every job block should have a `name:` field.

### WF-11 — Release images tagged with version and latest (WARN)

The `docker/build-push-action` step in `release.yml` should produce two tags:
- `<registry>/<image>:${{ github.ref_name }}`
- `<registry>/<image>:latest`
