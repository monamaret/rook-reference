# Workload Identity Federation Setup Guide

This guide covers provisioning GCP Workload Identity Federation (WIF) so that the `release.yml` GitHub Actions workflow can deploy to Cloud Run without storing a long-lived service account JSON key.

## Why WIF

Long-lived service account keys are a credential leak risk. WIF uses short-lived OIDC tokens issued by GitHub Actions, exchanged for a GCP access token scoped to specific permissions. No key file is stored anywhere.

## Prerequisites

- GCP project with billing enabled
- `gcloud` CLI authenticated as a project owner or IAM admin
- The GitHub repository URL (e.g., `https://github.com/rook-project/rook-reference`)

## Provisioning Steps

### 1. Enable required APIs

```bash
gcloud services enable 
  iamcredentials.googleapis.com 
  sts.googleapis.com 
  run.googleapis.com 
  artifactregistry.googleapis.com 
  --project=<GCP_PROJECT_ID>
```

### 2. Create a Workload Identity Pool

```bash
gcloud iam workload-identity-pools create "github-actions-pool" 
  --project=<GCP_PROJECT_ID> 
  --location="global" 
  --display-name="GitHub Actions Pool"
```

### 3. Create a Workload Identity Provider

```bash
gcloud iam workload-identity-pools providers create-oidc "github-provider" 
  --project=<GCP_PROJECT_ID> 
  --location="global" 
  --workload-identity-pool="github-actions-pool" 
  --display-name="GitHub OIDC Provider" 
  --issuer-uri="https://token.actions.githubusercontent.com" 
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.ref=assertion.ref" 
  --attribute-condition="assertion.repository=='rook-project/rook-reference'"
```

### 4. Create a Service Account for Deployments

```bash
gcloud iam service-accounts create "github-actions-deployer" 
  --project=<GCP_PROJECT_ID> 
  --display-name="GitHub Actions Deployer"
```

Grant it the minimum required permissions:

```bash
# Push images to Artifact Registry
gcloud projects add-iam-policy-binding <GCP_PROJECT_ID> 
  --member="serviceAccount:github-actions-deployer@<GCP_PROJECT_ID>.iam.gserviceaccount.com" 
  --role="roles/artifactregistry.writer"

# Deploy to Cloud Run
gcloud projects add-iam-policy-binding <GCP_PROJECT_ID> 
  --member="serviceAccount:github-actions-deployer@<GCP_PROJECT_ID>.iam.gserviceaccount.com" 
  --role="roles/run.developer"

# Act as the Cloud Run runtime service account (if different)
gcloud iam service-accounts add-iam-policy-binding <CLOUD_RUN_SA>@<GCP_PROJECT_ID>.iam.gserviceaccount.com 
  --member="serviceAccount:github-actions-deployer@<GCP_PROJECT_ID>.iam.gserviceaccount.com" 
  --role="roles/iam.serviceAccountUser"
```

### 5. Allow the Workload Identity Pool to Impersonate the Service Account

```bash
gcloud iam service-accounts add-iam-policy-binding 
  "github-actions-deployer@<GCP_PROJECT_ID>.iam.gserviceaccount.com" 
  --project=<GCP_PROJECT_ID> 
  --role="roles/iam.workloadIdentityUser" 
  --member="principalSet://iam.googleapis.com/projects/<PROJECT_NUMBER>/locations/global/workloadIdentityPools/github-actions-pool/attribute.repository/rook-project/rook-reference"
```

Get your project number: `gcloud projects describe <GCP_PROJECT_ID> --format='value(projectNumber)'`

### 6. Store Values as GitHub Repository Secrets

In the GitHub repository → Settings → Secrets and variables → Actions:

| Secret Name | Value |
|---|---|
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | `projects/<PROJECT_NUMBER>/locations/global/workloadIdentityPools/github-actions-pool/providers/github-provider` |
| `GCP_SERVICE_ACCOUNT` | `github-actions-deployer@<GCP_PROJECT_ID>.iam.gserviceaccount.com` |

### 7. Create Artifact Registry Repository

```bash
gcloud artifacts repositories create rook-images 
  --repository-format=docker 
  --location=<REGION> 
  --project=<GCP_PROJECT_ID> 
  --description="Rook Reference container images"
```

Image path pattern: `<REGION>-docker.pkg.dev/<GCP_PROJECT_ID>/rook-images/<service-name>`

## Verification

```bash
# Confirm pool exists
gcloud iam workload-identity-pools describe github-actions-pool 
  --location=global --project=<GCP_PROJECT_ID>

# Confirm provider exists
gcloud iam workload-identity-pools providers describe github-provider 
  --workload-identity-pool=github-actions-pool 
  --location=global --project=<GCP_PROJECT_ID>
```

Push a `v0.0.1-test` tag to the repository and confirm that `release.yml` runs to completion without authentication errors.

## Troubleshooting

| Error | Likely Cause | Fix |
|---|---|---|
| `Error: Unable to generate tokens` | Wrong `workload_identity_provider` value | Re-check the full resource name including project number |
| `Permission denied on Artifact Registry` | Missing `artifactregistry.writer` role | Re-run step 4 IAM binding |
| `PERMISSION_DENIED: Permission 'run.services.update' denied` | Missing `run.developer` role | Re-run step 4 IAM binding |
| `iam.serviceAccounts.actAs denied` | Missing `serviceAccountUser` binding | Re-run the `actAs` grant in step 4 |
