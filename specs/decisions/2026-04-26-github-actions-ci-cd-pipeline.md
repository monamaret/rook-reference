---
status: accepted
date: 2026-04-26
decision-makers: rook-project team
consulted: ''
informed: ''
---

# Adopt GitHub Actions CI/CD Pipeline for Rook Reference

## Context and Problem Statement

Rook Reference is a multi-component Go monorepo containing:

- `rook-server/` — multiple Go microservices deployed to Cloud Run (auth, user, stash, messaging, guides services)
- `rook-server/cmd/admin` — `rook-server-cli`, an admin/ops binary for server operators (ships before `rook-cli`)
- `rook-cli/` — Go TUI client for end users (standalone value, no server required to be useful)
- `rook-docs/` — Hugo static site, deployed to GitHub Pages on release

Each release produces: Docker images per microservice (deployed to Cloud Run), two cross-compiled binaries (`rook-server-cli` and `rook-cli`) as GitHub Release assets, and an updated documentation site on GitHub Pages.

Development of the v0.1 skeleton is beginning. Without an automated pipeline, there are no gates to prevent broken code from reaching `main`, no reproducible build process for container images, and no automated path from a release tag to a running Cloud Run service. This decision formalises the pipeline before the first line of application code is merged.

**Decision question**: How should the project implement CI (PR validation) and CD (build, image publishing, and Cloud Run deployment) using GitHub Actions, and should a new Tabnine CLI skill be created to support ongoing workflow authoring and validation?

## Decision Drivers

- Gates must block merges to `main` when lint, tests, or build fails
- The monorepo has multiple independently deployable components — pipeline noise should be proportional to what actually changed
- Cloud Run is the deployment target; images must be pushed to Google Artifact Registry
- Release cadence is tag-driven (`v*` semver tags), not continuous deploy on every merge
- The pipeline must be self-documenting and consistent enough that a coding agent can scaffold new service workflows without human guidance
- `rook-docs/` (Hugo) has a different build toolchain and must be handled separately from Go services

## Considered Options

1. **Path-filtered per-component GitHub Actions workflows** — each component (or service) has its own workflow file triggered only when its paths change; a shared reusable workflow handles common steps
2. **Single unified workflow** — one workflow file runs lint, test, and build for all components on every push
3. **External CI platform** (CircleCI, Buildkite, etc.) — offload pipeline to a dedicated tool with richer primitives

## Decision Outcome

Chosen option: **Option 1 — Path-filtered per-component GitHub Actions workflows**, because:

- GitHub Actions is already available with zero additional tooling cost
- Path filtering (`paths:` trigger) ensures only affected components run on a given PR, keeping feedback fast and noise low as the service count grows
- Reusable workflows (`workflow_call`) allow a single canonical Go CI definition consumed by every service, preventing drift
- Tag-triggered release jobs are natively supported and compose cleanly with the per-component structure

### Consequences

- Good, because PRs touching only `rook-cli/` do not trigger rook-server service builds
- Good, because a single reusable Go workflow enforces consistent lint/test/build standards across all services
- Good, because tag-triggered deploys give the team explicit control over what reaches Cloud Run
- Bad, because the initial setup requires more workflow files than a unified approach (one per component + shared reusables)
- Bad, because path filter maintenance is required when new services are added — mitigated by the github-actions skill
- Neutral, because GitHub Actions secrets management must be configured for GCP credentials (Workload Identity Federation preferred over long-lived keys)

## Implementation Plan

### Directory Structure

```
.github/
  workflows/
    # Reusable workflow — called by all Go component workflows
    _go-ci.yml

    # Per-component CI workflows (path-filtered)
    ci-auth-service.yml
    ci-user-service.yml
    ci-stash-service.yml
    ci-messaging-service.yml
    ci-guides-service.yml
    ci-rook-server-cli.yml   # admin binary — rook-server/cmd/admin
    ci-rook-cli.yml
    ci-rook-docs.yml

    # Release workflow — tag-triggered
    # Produces: service images → Cloud Run, two binaries → GitHub Release, docs → GitHub Pages
    release.yml
```

### Reusable Go CI Workflow (`_go-ci.yml`)

Called by all Go component workflows via `workflow_call`. Steps:

1. `actions/checkout@v4`
2. `actions/setup-go@v5` — version pinned to match each component's `go.mod` `go` directive
3. `golangci-lint-action` — lint with project `.golangci.yml` config
4. `go test ./... -race -count=1` — run tests with race detector
5. `go build ./...` — verify compilation

### Per-Component CI Workflow Pattern

Each `ci-<component>.yml` follows this structure:

```yaml
on:
  push:
    branches: [main]
    paths:
      - 'rook-server/<service-name>/**'
      - '.github/workflows/ci-<service-name>.yml'
      - '.github/workflows/_go-ci.yml'
  pull_request:
    branches: [main]
    paths:
      - 'rook-server/<service-name>/**'
      - '.github/workflows/ci-<service-name>.yml'
      - '.github/workflows/_go-ci.yml'

jobs:
  ci:
    uses: ./.github/workflows/_go-ci.yml
    with:
      working-directory: rook-server/<service-name>
```

### `ci-rook-docs.yml`

Hugo-specific. Steps:

1. `actions/checkout@v4` with submodules
2. `peaceiris/actions-hugo@v3` — pin Hugo version
3. `hugo --minify` — build the site
4. No deploy step in CI (docs deploy is out of scope for this ADR)

### Release Workflow (`release.yml`)

Triggered on `push: tags: ['v*.*.*']`. Produces three categories of artifact:

**1. Service images → Cloud Run** (one job per service)

For each of `auth-service`, `user-service`, `stash-service`, `messaging-service`, `guides-service`:

1. `actions/checkout@v4`
2. `google-github-actions/auth@v2` — Workload Identity Federation (no stored JSON keys)
3. `docker/login-action@v3` — authenticate to Artifact Registry
4. `docker/build-push-action@v6` — build and push image tagged with the git tag and `latest`
5. `google-github-actions/deploy-cloudrun@v2` — deploy to Cloud Run

**2. Binaries → GitHub Release** (one job, two binaries)

Builds cross-compiled binaries and attaches them to the GitHub Release created by the tag:

- `rook-server-cli` — from `rook-server/cmd/admin`; **ships from v0.1**
  - Targets: `linux/amd64`, `darwin/arm64`
- `rook-cli` — from `rook-cli/`; ships when the client reaches a releasable state
  - Targets: `linux/amd64`, `linux/arm64`, `darwin/arm64`, `windows/amd64`

Steps:
1. `actions/checkout@v4`
2. `actions/setup-go@v5`
3. `go build` for each binary × each platform → `dist/`
4. `softprops/action-gh-release@v2` — create/update GitHub Release and upload `dist/*`

**3. Documentation → GitHub Pages**

1. `actions/checkout@v4` (with submodules, full history)
2. `peaceiris/actions-hugo@v3` — pinned Hugo version
3. `hugo --minify` — build to `public/`
4. `peaceiris/actions-gh-pages@v4` — push `public/` to the `gh-pages` branch

GCP authentication (for service jobs) uses `google-github-actions/auth@v2` with Workload Identity Federation — no long-lived service account keys stored as secrets. Binary and docs jobs do not require GCP credentials.

### New Tabnine CLI Skill: `github-actions`

A new skill at `.tabnine/agent/skills/github-actions/` covering:

- **Scaffold**: generate a new per-component workflow file from a template (Go service or Hugo), updating path filters and job names automatically
- **Validate**: check existing workflow files against project conventions (path filters present, pinned action versions, reusable workflow referenced correctly, no hardcoded credentials)

Skill structure:
```
.tabnine/agent/skills/github-actions/
  SKILL.md
  references/
    workflow-conventions.md   # canonical rules for this project's workflows
    wif-setup.md              # Workload Identity Federation setup guide
  assets/
    templates/
      go-ci-component.yml     # per-component CI template
      go-ci-reusable.yml      # _go-ci.yml template
      release.yml             # release workflow template
      hugo-ci.yml             # rook-docs CI template
```

### Affected Paths

- `.github/workflows/` — create entire directory and all workflow files
- `rook-server/<each-service>/` — each service needs a `Dockerfile` for the release workflow (Dockerfile authoring is a separate concern but must exist)
- `rook-server/cmd/admin/` — `rook-server-cli` entry point; must exist for the binary release job
- `.golangci.yml` — create at repo root with agreed lint rules
- `.tabnine/agent/skills/github-actions/` — new skill directory

### Dependencies / Actions Used

| Action | Version | Purpose |
|---|---|---|
| `actions/checkout` | `v4` | Checkout code |
| `actions/setup-go` | `v5` | Go toolchain |
| `golangci/golangci-lint-action` | `v6` | Linting |
| `google-github-actions/auth` | `v2` | GCP Workload Identity |
| `google-github-actions/deploy-cloudrun` | `v2` | Cloud Run deploy |
| `docker/login-action` | `v3` | Artifact Registry auth |
| `docker/build-push-action` | `v6` | Image build + push |
| `peaceiris/actions-hugo` | `v3` | Hugo build |
| `peaceiris/actions-gh-pages` | `v4` | Deploy to GitHub Pages |
| `softprops/action-gh-release` | `v2` | Create GitHub Release + upload binary assets |

### Configuration

- **GCP Workload Identity Provider** — must be provisioned in GCP before the release workflow can run; stored as `GCP_WORKLOAD_IDENTITY_PROVIDER` and `GCP_SERVICE_ACCOUNT` repository secrets
- **Artifact Registry repository** — must exist at `<region>-docker.pkg.dev/<project>/<repo>` before first release
- **`.golangci.yml`** — place at repo root; minimum rules: `gofmt`, `govet`, `errcheck`, `staticcheck`
- **Branch protection** — enable "Require status checks to pass" on `main` for each component's CI job

### Patterns to Follow

- All action versions must be pinned to a specific major version tag (e.g., `@v4`), not `@latest` or a SHA
- Secrets are never hardcoded; use `${{ secrets.NAME }}` or Workload Identity Federation
- Reusable workflow (`_go-ci.yml`) is the single source of truth for Go CI steps — do not duplicate steps in per-component files
- New services follow the per-component pattern exactly; use the github-actions skill to scaffold

### Patterns to Avoid

- Do not use a single monolithic workflow that runs all components unconditionally — it does not scale
- Do not store GCP service account JSON keys as GitHub secrets — use Workload Identity Federation
- Do not use `@latest` or branch-pinned action references — they are not reproducible
- Do not add deploy steps to CI (PR) workflows — only the tag-triggered `release.yml` deploys

## Verification

- [ ] Each per-component CI workflow is triggered on a PR touching only that component's path and not others
- [ ] A PR with a failing `go test` is blocked from merging to `main` by branch protection
- [ ] A PR with a `golangci-lint` violation is blocked from merging to `main`
- [ ] `ci-rook-server-cli.yml` triggers on changes to `rook-server/cmd/admin/**` and not on other `rook-server/**` paths
- [ ] Pushing a `v*.*.*` tag triggers `release.yml` and produces a tagged image in Artifact Registry for each service
- [ ] `release.yml` deploys service images to Cloud Run without requiring manual GCP credentials input (WIF)
- [ ] `release.yml` produces `rook-server-cli` binaries (linux/amd64, darwin/arm64) attached to the GitHub Release
- [ ] `release.yml` produces `rook-cli` binaries (linux/amd64, linux/arm64, darwin/arm64, windows/amd64) attached to the GitHub Release
- [ ] `release.yml` deploys the Hugo-built site to the `gh-pages` branch and the GitHub Pages site reflects the new release
- [ ] `rook-docs` CI workflow builds successfully with `hugo --minify` and fails if Hugo exits non-zero
- [ ] The github-actions skill can scaffold a new Go service workflow file that passes the validate check without manual edits
- [ ] No workflow file references `@latest` or stores credentials outside of `${{ secrets.* }}` or WIF
- [ ] `act` (local GitHub Actions runner) can execute the `_go-ci.yml` reusable workflow against a stub Go module

## Pros and Cons of the Options

### Option 1: Path-filtered per-component workflows (chosen)

- Good, because scales linearly with service count — adding a service adds one workflow file
- Good, because reusable workflow prevents CI logic drift across services
- Good, because feedback is fast — only the affected component's jobs run on a PR
- Bad, because initial file count is higher than a unified approach
- Neutral, because path filter lists must be kept in sync with directory renames

### Option 2: Single unified workflow

- Good, because simpler to set up initially (one file)
- Bad, because every PR triggers builds for all components regardless of what changed
- Bad, because as the service count grows, CI times become unacceptable
- Bad, because a failure in an unrelated service blocks unrelated PRs

### Option 3: External CI platform

- Good, because richer native primitives (e.g., Buildkite's dynamic pipelines)
- Bad, because introduces a new paid dependency with no clear advantage over GitHub Actions for this project's scale
- Bad, because splits CI configuration away from the repository, reducing discoverability
- Neutral, because migration is always possible later if GitHub Actions becomes a bottleneck

## More Information

- Workload Identity Federation setup: see `.tabnine/agent/skills/github-actions/references/wif-setup.md` (to be created with the skill)
- Dockerfile authoring for each rook-server service is a prerequisite for the release workflow but is outside the scope of this ADR
- Revisit this decision if: the number of services exceeds 10 (at which point a matrix strategy or dynamic pipeline generation may be warranted), or if Cloud Run deploy complexity grows (e.g., traffic splitting, multi-region)
- Revisit per-service tagging: the current `release.yml` deploys all services on every `v*.*.*` tag. When services reach independent release cadences (e.g., stash-service ships a patch while auth-service is frozen), consider adopting a per-service tag convention (e.g., `stash/v1.2.3`) with a separate release job per service. This is out of scope for v0.1 but should be evaluated before v1.0.
- Related ADRs:
  - [`2026-04-25-rook-reference-system-architecture.md`](2026-04-25-rook-reference-system-architecture.md) — defines the components this pipeline builds
  - [`2026-04-25-ssh-auth-identity-chain-and-cloud-run-topology.md`](2026-04-25-ssh-auth-identity-chain-and-cloud-run-topology.md) — Cloud Run topology this pipeline deploys to
