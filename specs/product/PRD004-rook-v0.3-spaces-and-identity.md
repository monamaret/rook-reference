# Rook v0.3 — Spaces and Identity

**ID:** PRD004  
**Version:** v0.3  
**Status:** Stub  
**Date:** 2026-04-25  
**Author:** Mona Maret  
**Parent:** [PRD001 — Rook Project Overview for PoC](PRD001-rook-overview-v1.0.md)  

---

## Overview

This release introduces the Spaces data model — the organizational unit that groups users, apps, and content in Rook. The `user-service` gains endpoints for resolving space membership and listing apps available to a space. The `rook-cli` launcher home screen is implemented as a static TUI shell: it renders the authenticated user's spaces and their app tiles but does not yet connect to live service data for stash, messaging, or guides.

---

## Feature Focus

_To be detailed in scoping. Proposed areas:_

- `user-service` Firestore schema: spaces, groups, and membership collections with identity references
- `user-service`: `GET /spaces` — list spaces the authenticated user is a member of
- `user-service`: `GET /spaces/{id}/apps` — list apps enabled for a given space
- Admin endpoint: `POST /admin/spaces/{id}/keys` — register a public SSH key to an identity in a space (admin-only)
- `rook-cli` launcher home screen TUI: space selector, app tile grid, status bar with identity and session state
- Authenticated HTTP client in `rook-cli`: attach session token to all outbound requests, detect 401 and prompt re-auth
- `rook-cli` identity resolution: on startup, fetch and cache space/group membership for the session
- Graceful degradation: launcher renders with cached data when server is unreachable

---

## Dependencies

- PRD003 v0.2 complete — session token and authenticated identity required

---

## Out of Scope for This Release

- Live app data within the launcher — stash, messaging, and guides tiles are present but non-functional
- Group-level permission enforcement beyond simple membership checks
- Space creation or self-service membership management — spaces are admin-provisioned
- Push notifications or real-time membership updates
- Multi-space switching within a single session beyond initial selection

---

## Open Questions

_To be resolved during scoping._

- What is the data model relationship between groups and spaces — are groups always scoped to a single space, or can a group span multiple spaces?
- How should the launcher handle a user who is a member of zero spaces — show an empty state, or redirect to a setup flow?
- Should app tile availability be driven by a static list in the space config or dynamically composed from which services are reachable?

---

## Notes

_Space for any early design notes, constraints, or decisions relevant to this release._

- The launcher home screen TUI is the primary UX surface for Rook; its layout decisions (bubbletea model structure, lipgloss styles) will set the pattern for all subsequent TUI screens.
- Space and group documents should include a `displayName` and optional `description` field for the launcher to render without additional lookups.
- The admin key-registration endpoint must be protected by a separate admin credential (not a regular session token) to prevent privilege escalation.
- Membership cache should be written to the XDG cache directory and invalidated on session change or explicit refresh.
