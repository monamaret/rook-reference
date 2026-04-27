# Data Model: v0.2 Authentication Foundation

**Feature**: `002-auth-foundation`  
**Date**: 2026-04-26  
**Status**: Complete

---

## Overview

v0.2 introduces three new persistent entities (all server-side in Firestore) and one transient in-memory entity (CLI-side). No new flat-file entities are introduced on disk in this release — the session token is in-memory only.

---

## Server-Side Entities (Firestore)

### 1. Nonce

**Collection**: `auth/nonces/{nonce}`  
**Purpose**: Short-lived challenge value issued during auth; prevents replay attacks  
**Owned by**: `user-service`

| Field | Type | Required | Notes |
|---|---|---|---|
| `nonce` | `string` | Yes | 32-byte value, hex-encoded (64 chars); also the document ID |
| `expires_at` | `timestamp` | Yes | 60 seconds from issuance; checked at read time by application code |
| `used` | `bool` | Yes | Set to `true` atomically on first successful verification; second use returns 401 |
| `identity_ref` | `string` | No | Public key fingerprint of the requesting identity (informational; not used for verification in v0.2) |

**Lifecycle**:
- Created by `GET /auth/challenge`
- Read + marked `used=true` by `POST /auth/verify` (atomic Firestore transaction)
- Application rejects nonces where `expires_at` < now OR `used == true`
- Deleted automatically by Firestore TTL policy on `expires_at` within 72 hours of expiry
- Firestore TTL policy: configured on `expires_at` field of the `nonces` collection (emulator: not enforced; production: required before first deployment)

**State transitions**:
```
issued (used=false, expires_at=future)
  → verified (used=true, expires_at=future)  [on successful POST /auth/verify]
  → expired (used=false, expires_at=past)    [after 60 seconds, not yet deleted]
  → deleted (by Firestore TTL, within 72h of expiry)
```

---

### 2. Session

**Collection**: `sessions/{token}`  
**Purpose**: Authenticated session credential issued after successful challenge-response  
**Owned by**: `user-service`

| Field | Type | Required | Notes |
|---|---|---|---|
| `token` | `string` | Yes | 32-byte value, hex-encoded (64 chars); also the document ID |
| `user_id` | `string` | Yes | Opaque user identifier; maps to the `users` collection |
| `expires_at` | `timestamp` | Yes | 1 hour from issuance |
| `created_at` | `timestamp` | Yes | Issuance timestamp |

**Lifecycle**:
- Created by `POST /auth/verify` on successful signature verification
- Read by `ValidateSession` on every authenticated request (future: cached in-process with 5-minute TTL)
- Revoked by deleting the Firestore document (server-side revocation; not implemented in v0.2 — no revocation endpoint)
- Application rejects sessions where `expires_at` < now

**Not stored in v0.2**:
- `space_id` — added when space-scoped sessions are implemented (PRD004)
- `group` — added when space membership is resolved (PRD004)

---

### 3. User (pre-seeded)

**Collection**: `users/{user_id}`  
**Purpose**: User identity record; holds the registered public key used for signature verification  
**Owned by**: `user-service`

| Field | Type | Required | Notes |
|---|---|---|---|
| `user_id` | `string` | Yes | Opaque identifier; also the document ID |
| `public_key` | `string` | Yes | OpenSSH wire format, base64-encoded; the key against which challenge signatures are verified |
| `display_name` | `string` | No | Human-readable label (informational only in v0.2) |
| `created_at` | `timestamp` | Yes | When the record was created |

**Registration in v0.2**: Manual Firestore emulator seed only. No user-facing registration endpoint. The usage guide provides the seed command.

**Full user model** (spaces, groups, ACL) is deferred to PRD004.

---

## CLI-Side Entities (In-Memory)

### 4. Session State

**Storage**: In-memory only — package-level value in `rook-cli/internal/auth/`  
**Purpose**: Holds the session token for the duration of the process; used to attach `Authorization: Bearer` headers  
**Never written to disk**

| Field | Type | Notes |
|---|---|---|
| `Token` | `string` | Hex-encoded session token received from `POST /auth/verify` |
| `ServerAddr` | `string` | The server address this token is valid for |
| `IssuedAt` | `time.Time` | Issuance timestamp (from local clock at receipt) |

**Lifecycle**:
- Created when `POST /auth/verify` returns success
- Read by any CLI code that needs to make an authenticated request
- Discarded when the process exits
- In v0.2: there is at most one active session per process

---

## HTTP API Contracts

### `GET /auth/challenge`

**Request**: No body, no auth required  
**Response** (200 OK):
```json
{
  "nonce": "<64-char hex string>"
}
```
**Error responses**:
- `500 Internal Server Error` — nonce generation or Firestore write failed

---

### `POST /auth/verify`

**Request body**:
```json
{
  "public_key": "<OpenSSH wire format, base64-encoded>",
  "nonce": "<64-char hex string>",
  "signature": "<base64-encoded SSH signature blob>"
}
```

**Response** (200 OK):
```json
{
  "token": "<64-char hex string>",
  "expires_at": "<RFC3339 timestamp>"
}
```

**Error responses**:
- `400 Bad Request` — malformed request body, missing fields
- `401 Unauthorized` — nonce not found, nonce expired, nonce already used, or signature verification failed
- `404 Not Found` — public key not registered (user not found in `users` collection)
- `500 Internal Server Error` — Firestore error

**Error body** (all error responses):
```json
{
  "error": "<human-readable message>"
}
```

---

## Firestore Collection Layout

```
firestore/
├── auth/
│   └── nonces/
│       └── {nonce}          ← Nonce documents (TTL on expires_at)
├── sessions/
│   └── {token}              ← Session documents (1-hour TTL)
└── users/
    └── {user_id}            ← User identity records (pre-seeded in v0.2)
```

**Namespace note**: All collections are at the root level in Firestore. No subcollection nesting is used in v0.2.

---

## Validation Rules

| Entity | Field | Rule |
|---|---|---|
| Nonce | `nonce` | Must be exactly 64 hex characters |
| Nonce | `expires_at` | Must be > now at read time |
| Nonce | `used` | Must be `false` at read time |
| Session | `token` | Must be exactly 64 hex characters |
| Session | `expires_at` | Must be > now at read time |
| User | `public_key` | Must be parseable by `ssh.ParsePublicKey` |
| POST /auth/verify | `public_key` | Must match a registered user's public key |
| POST /auth/verify | `signature` | Must verify against `public_key` for the given `nonce` |

---

## Not in v0.2

- Space-scoped sessions (`X-Rook-Space-ID` header processing and `space_id` on session)
- `SessionAuthMiddleware` shared package (scaffolded in `rook-server/internal/middleware/` but not wired to any handler)
- User self-registration endpoint
- Session revocation endpoint (`DELETE /sessions/{token}`)
- gRPC `ValidateSession` RPC (proto stub only; no generated code)
