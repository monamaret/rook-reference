# Rook v1.0 — PoC Hardening and Admin CLI

**ID:** PRD011  
**Version:** v1.0  
**Status:** Stub  
**Date:** 2026-04-25  
**Author:** Mona Maret  
**Parent:** [PRD001 — Rook Project Overview for PoC](PRD001-rook-overview-v1.0.md)  

---

## Overview

This release hardens the full Rook PoC for demonstration readiness. It delivers the complete admin CLI for key registration, space and group management, and session management; closes the graceful re-auth loop so expired sessions are handled seamlessly; adds a polished first-run setup flow; and validates the end-to-end system running on Cloud Run. After this release, the Rook PoC is feature-complete and operationally verified as a demonstrable proof of concept.

---

## Feature Focus

_To be detailed in scoping. Proposed areas:_

- Graceful re-auth flow: detect 401 responses anywhere in the CLI, prompt for re-authentication inline, and replay the original request on success without losing user context
- Session revocation: `RevokeSession` gRPC RPC on `AdminService` in user-service; `rook-server-cli session revoke --all` to revoke all sessions for an identity
- Admin CLI complete: `rook-server-cli user`, `rook-server-cli space`, `rook-server-cli session` — full CRUD for all admin-provisioned resources
- First-run setup flow polish: detect unconfigured state, guide user through server endpoint config, `rook auth`, and space selection in a single onboarding sequence
- End-to-end PoC validation: scripted walkthrough covering auth, stash sync, messaging, and guide read/publish across two identities in a shared space
- Cloud Run deployment verification: all services (`user-service`, `stash-service`, `messaging-service`, `guides-service`) deployed and reachable on Cloud Run with the CLI configured against the live endpoints
- Observability baseline: structured logging in all services, request IDs propagated through the stack, and error response bodies consistent across all endpoints
- Documentation: operator runbook for deploying and administering a Rook instance; user guide for the complete CLI workflow

---

## Dependencies

- PRD010 v0.9 complete — all features (auth, spaces, stash, messaging, guides) must be implemented and individually verified
- All prior PRDs (PRD002–PRD010) complete

---

## Out of Scope for This Release

- Production hardening beyond PoC scale — no load testing, rate limiting, or multi-region deployment
- User self-service registration — key and space provisioning remains admin-only
- Billing, quotas, or multi-tenancy isolation at the infrastructure level
- Mobile or web clients — terminal CLI only
- Automated end-to-end test suite in CI — manual PoC walkthrough script only

---

## Open Questions

_To be resolved during scoping._

- What is the minimum set of admin CLI operations required to run a PoC demo without manual Firestore access — is there a clear checklist?
- How should the re-auth flow handle commands that modify state (e.g. a `stash push` interrupted by a 401) — retry the full operation, resume from failure point, or surface the error and let the user retry?
- What does "Cloud Run deployment verified" mean operationally — a single shared instance, per-developer instances, or a dedicated PoC environment with a stable URL?

---

## Notes

_Space for any early design notes, constraints, or decisions relevant to this release._

- The first-run setup flow should be idempotent — running it again on an already-configured system should detect existing config and offer to update only the fields the user selects.
- Structured logs should include at minimum: `timestamp`, `service`, `requestId`, `method`, `path`, `statusCode`, `latencyMs`, and `identityId` (when authenticated). Avoid logging any session token values.
- The admin CLI should require a separate admin token (distinct from a regular session token) configured via the config file or an environment variable; never derive admin access from a regular user session.
- The PoC walkthrough script should be committed to the repository as a runnable shell script so it can serve as a regression check for future releases.
