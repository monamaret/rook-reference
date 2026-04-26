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
  │  │  └── guides/drafts/          (.md + .yml + yaml-config)      │               │
  │  └──────────────────────────────────────────────────────────────┘               │
  │       │                │                  │                                     │
  │       │ SSH            │ HTTP             │ HTTP                                │
  │       │ (wish/         │ (pull sync:      │ (guide fetch /                      │
  │       │  wishlist)     │  stash, messages)│  guide publish)                     │
  └───────┼────────────────┼──────────────────┼─────────────────────────────────────┘
          │                │                  │
          ▼                ▼                  ▼
  ┌───────────────────────────────────────────────────────────────────────────────────┐
  │  rook-server  (Go microservices — Google Cloud Run)                               │
  │                                                                                   │
  │  ┌─────────────────────────────────────────────────────────────────────────────┐  │
  │  │  user-service                                                               │  │
  │  │  • SSH handshake + key validation  (charmbracelet/wish + wishlist)          │  │
  │  │  • Space-filtered, ACL-gated wishlist                                       │  │
  │  │  • UserService gRPC  ◄── messaging-service, stash-service, guides-service   │  │
  │  │  • AdminService gRPC ◄── rook-server-cli only (admin token required)        │  │
  │  │  • Firestore: users/, spaces/, groups/                                      │  │
  │  └────────────────────────────┬────────────────────────────────────────────────┘  │
  │                               │ gRPC (UserService)                                │
  │              ┌────────────────┼────────────────┐                                  │
  │              ▼                ▼                ▼                                  │
  │  ┌──────────────────┐ ┌──────────────┐ ┌──────────────────┐                      │
  │  │ messaging-service│ │ stash-service│ │  guides-service  │                      │
  │  │ HTTP pull sync   │ │ HTTP pull    │ │  HTTP guide      │                      │
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
| `rook-cli` | `user-service` | SSH (wish + wishlist) | SSH public key |
| `rook-cli` | `messaging-service` | HTTP (pull sync) | SSH key token (session) |
| `rook-cli` | `stash-service` | HTTP (pull sync) | SSH key token (session) |
| `rook-cli` | `guides-service` | HTTP (guide fetch / publish) | SSH key token (session) |
| `messaging-service` | `user-service` | gRPC (UserService) | GCP OIDC bearer token |
| `stash-service` | `user-service` | gRPC (UserService) | GCP OIDC bearer token |
| `guides-service` | `user-service` | gRPC (UserService) | GCP OIDC bearer token |
| `rook-server-cli` | `user-service` | gRPC (AdminService) | Pre-shared `ROOK_ADMIN_TOKEN` |

---

## Key Boundaries

- **`rook-cli` is always local-first.** All stash, message, and guide draft operations work offline against flat files. Server contact is user-initiated.
- **`rook-server` services never share data.** Each service owns its own Firestore namespace; no cross-service collection access.
- **`AdminService` is never reachable from `rook-cli`.** Admin operations require `ROOK_ADMIN_TOKEN`, which is only known to `user-service` and `rook-server-cli`.
- **Key lifecycle is split across components.** Key *generation* is local (`rook-cli` first-run). Key *registration* is an admin operation (`rook-server-cli user register-key`). Key *validation* at connect time is handled by `charmbracelet/wish` on `user-service`, which looks up the fingerprint in Firestore.
- **Space is the universal segregation boundary.** All server-side data is scoped to a space; all local flat-file paths include `<space-id>/`; the wishlist a user sees is filtered to their space and group ACL.
