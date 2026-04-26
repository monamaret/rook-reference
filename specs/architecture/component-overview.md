# Rook System: Component Overview

This diagram shows the high-level relationships between the three Rook binaries — `rook-cli`, `rook-server`, and `rook-server-cli` — their personas, protocols, and data boundaries.

For per-service detail and gRPC call flows, see [`grpc-call-flows.md`](grpc-call-flows.md).

---

```
  ┌─────────────────────────────────────────────────────────────────────────────────┐
  │  USER                                                                           │
  │                                                                                 │
  │  ┌──────────────────────────────────────────────────────────────┐               │
  │  │  rook-cli  (Go TUI — runs on user's machine)                 │               │
  │  │                                                              │               │
  │  │  Launcher ──► Stash    (offline: create / browse / view)     │               │
  │  │           ──► Search   (offline: local full-text)            │               │
  │  │           ──► Messages (offline: compose / read)             │               │
  │  │           ──► Guides   (online:  read from server)           │               │
  │  │           ──► Builder  (online:  author + publish guides)     │               │
  │  │                                                              │               │
  │  │  Local flat-file store:                                      │               │
  │  │  $XDG_CONFIG_HOME/rook/storage/                              │               │
  │  │  ├── stash/<space-id>/       (.md + .json)                   │               │
  │  │  ├── messages/<space-id>/    (.md + .json)                   │               │
  │  │  ├── guides/drafts/          (.md + .yml + yaml-config)      │               │
  │  │  └── cache/                  (spaces.json — app ACL cache)   │               │
  │  └──────────────────────────────────────────────────────────────┘               │
  │       │                │                  │                                     │
  │       │ HTTPS          │ HTTPS            │ HTTPS                               │
  │       │ (auth +        │ (pull sync:      │ (guide fetch /                      │
  │       │  space/app     │  stash, messages)│  guide publish)                     │
  │       │  discovery)    │                  │                                     │
  └───────┼────────────────┼──────────────────┼─────────────────────────────────────┘
          │                │                  │
          ▼                ▼                  ▼
  ┌───────────────────────────────────────────────────────────────────────────────────┐
  │  rook-server  (Go microservices — Google Cloud Run, all HTTPS)                    │
  │                                                                                   │
  │  ┌─────────────────────────────────────────────────────────────────────────────┐  │
  │  │  user-service                                                               │  │
  │  │  • HTTPS challenge-response auth (GET /auth/challenge, POST /auth/verify)   │  │
  │  │  • Session token issuance + validation (Firestore sessions/)                │  │
  │  │  • Space/app discovery (GET /spaces, GET /spaces/{id}/apps)                 │  │
  │  │  • UserService gRPC  ◄── messaging-service, stash-service, guides-service   │  │
  │  │  • AdminService gRPC ◄── rook-server-cli only (admin token required)        │  │
  │  │  • Firestore: users/, spaces/, groups/, sessions/, auth/nonces/             │  │
  │  └────────────────────────┬────────────────────────────────────────────────────┘  │
  │                           │ gRPC (UserService — ValidateSession + space membership)│
  │              ┌────────────┴────────────────┐                                      │
  │              ▼                             ▼                        ▼             │
  │  ┌──────────────────┐ ┌──────────────┐ ┌──────────────────┐                      │
  │  │ messaging-service│ │ stash-service│ │  guides-service  │                      │
  │  │ HTTPS pull sync  │ │ HTTPS pull   │ │  HTTPS guide     │                      │
  │  │ Firestore:       │ │ sync         │ │  fetch + publish │                      │
  │  │ messages/        │ │ Firestore:   │ │  Firestore:      │                      │
  │  │                  │ │ stash/       │ │  guides/         │                      │
  │  └──────────────────┘ └──────────────┘ └──────────────────┘                      │
  └───────────────────────────────────────────────────────────────────────────────────┘
          ▲
          │ gRPC (AdminService)
          │ ROOK_ADMIN_TOKEN bearer
          │ (user-service only — never rook-cli)
          │
  ┌───────────────────────────────────────────────────────┐
  │  ADMIN                                                │
  │                                                       │
  │  ┌───────────────────────────────────────────────┐    │
  │  │  rook-server-cli  (Go CLI — admin's machine)  │    │
  │  │                                               │    │
  │  │  user register-key   ──► AdminService         │    │
  │  │  user add-to-space   ──► AdminService         │    │
  │  │  user set-group      ──► AdminService         │    │
  │  │  user list           ──► AdminService         │    │
  │  │  space create        ──► AdminService         │    │
  │  │  space list          ──► AdminService         │    │
  │  │  space members       ──► AdminService         │    │
  │  │                                               │    │
  │  │  Env: USER_SERVICE_ADDR, ROOK_ADMIN_TOKEN     │    │
  │  └───────────────────────────────────────────────┘    │
  └───────────────────────────────────────────────────────┘
```

---

## Protocol Summary

| From | To | Protocol | Auth |
|------|----|----------|------|
| `rook-cli` | `user-service` | HTTPS (challenge-response: `/auth/challenge`, `/auth/verify`) | SSH private key signature |
| `rook-cli` | `user-service` | HTTPS (`/spaces`, `/spaces/{id}/apps`) | Bearer session token |
| `rook-cli` | `messaging-service` | HTTPS (pull sync) | Bearer session token |
| `rook-cli` | `stash-service` | HTTPS (pull sync) | Bearer session token |
| `rook-cli` | `guides-service` | HTTPS (guide fetch / publish) | Bearer session token |
| `messaging-service` | `user-service` | gRPC (`ValidateSession` + `GetSpaceMembership`) | GCP OIDC bearer token |
| `stash-service` | `user-service` | gRPC (`ValidateSession` + `GetSpaceMembership`) | GCP OIDC bearer token |
| `guides-service` | `user-service` | gRPC (`ValidateSession` + `CheckAppAccess`) | GCP OIDC bearer token |
| `rook-server-cli` | `user-service` | gRPC (AdminService) | Pre-shared `ROOK_ADMIN_TOKEN` |

---

## Key Boundaries

- **`rook-cli` is always local-first.** All stash, message, and guide draft operations work offline against flat files. Server contact is user-initiated.
- **`rook-server` services never share data.** Each service owns its own Firestore namespace; no cross-service collection access.
- **`AdminService` is never reachable from `rook-cli`.** Admin operations require `ROOK_ADMIN_TOKEN`, which is only known to `user-service` and `rook-server-cli`.
- **All client-server communication is HTTPS.** There is no SSH transport on the server. `rook-cli` uses the user's SSH private key only to sign the auth challenge locally — the key never leaves the CLI.
- **Key lifecycle is split across components.** Key *generation* is local (`rook-cli` first-run). Key *registration* is an admin operation (`rook-server-cli user register-key`). Key *validation* happens in `user-service` during `POST /auth/verify` — it looks up the registered public key from Firestore and verifies the signature.
- **Session tokens are ephemeral on the CLI.** The session token returned by `POST /auth/verify` is held in-memory only; it is never written to disk and is discarded on process exit.
- **Space is the universal segregation boundary.** All server-side data is scoped to a space; all local flat-file paths include `<space-id>/`; the app list a user sees is filtered to their space and group ACL via `GET /spaces/{id}/apps`.
- **`ValidateSession` resolves both identity and space membership in one call.** All requests include `X-Rook-Space-ID`; the session middleware calls `user-service.ValidateSession` once and injects both user ID and space membership into the request context — no second gRPC round-trip per handler.
