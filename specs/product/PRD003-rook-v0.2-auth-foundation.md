# Rook v0.2 — Authentication Foundation

**ID:** PRD003  
**Version:** v0.2  
**Status:** Stub  
**Date:** 2026-04-25  
**Author:** Mona Maret  
**Parent:** [PRD001 — Rook Project Overview for PoC](PRD001-rook-overview-v1.0.md)  

---

## Overview

This release delivers the core SSH-key-based authentication flow between `rook-cli` and the `user-service`. It implements the challenge/verify handshake endpoints in `rook-server`, Firestore-backed nonce and session storage, and the `rook auth` CLI command that takes a user through login end-to-end. After this release, a user with a registered SSH key can authenticate and receive a session token used by all subsequent CLI commands.

---

## Feature Focus

_To be detailed in scoping. Proposed areas:_

- `user-service`: `GET /auth/challenge` endpoint — generate and store a short-lived nonce in Firestore keyed to the requesting identity
- `user-service`: `POST /auth/verify` endpoint — verify the signed challenge against the registered public key, issue a session token on success
- Firestore schema: nonce collection (TTL, identity ref) and session collection (token, expiry, identity ref)
- Session token format: signed JWT or opaque token; storage in `rook-cli` XDG state directory
- `rook auth` CLI command: interactive flow — SSH key selection, challenge fetch, sign, verify, confirm login success
- `rook auth status` subcommand: show current session validity and identity
- `rook auth logout` subcommand: revoke local session token
- Error handling: expired nonce, signature mismatch, unregistered key — clear user-facing messages

---

## Dependencies

- PRD002 v0.1 complete — config file, XDG paths, and module skeleton required

---

## Out of Scope for This Release

- Key registration — public keys must be pre-seeded in Firestore by an admin; no self-registration UI
- Space or group identity — auth is identity-only, no membership resolution yet
- Session revocation from the server side — logout is local-only in this release
- Any TUI beyond the `rook auth` command flow
- Multi-account or key-switching support

---

## Open Questions

_To be resolved during scoping._

- Should the session token be a signed JWT (with claims for identity and expiry) or an opaque random token looked up in Firestore on each request?
- How should `rook-cli` select which SSH key to use when multiple keys are present in `~/.ssh/` — prompt the user, use config, or check `ssh-agent`?
- What is the nonce TTL, and how are expired nonces cleaned up in Firestore (TTL field, Cloud Scheduler, or on-read purge)?

---

## Notes

_Space for any early design notes, constraints, or decisions relevant to this release._

- The challenge/verify pattern mirrors standard SSH certificate authentication; the nonce must be a cryptographically random value (≥ 32 bytes) to prevent replay.
- Session tokens stored on disk should have file permissions set to 0600 to limit exposure.
- The `user-service` should validate the SSH key format on receipt of `POST /auth/verify` and reject unsupported key types early.
- Consider whether `rook auth` should open a browser for a secondary confirmation step or remain fully terminal-native for the PoC.
- **Issue closure workflow (carry-forward from v0.1)**: GitHub Issues created by `/speckit.taskstoissues` are not automatically closed when code is pushed because no closing keywords (`Closes #NNN`) are present in commits or the PR description. For v0.2, the PR description opened from the feature branch → `main` must include a `Closes` line for every task issue. The T→# mapping is available in `tasks.md` immediately after `/speckit.taskstoissues` runs. This should be wired into the PR template or the after-implement workflow so issues close on merge without manual intervention.
