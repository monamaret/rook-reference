# Quickstart: v0.2 Authentication Foundation

**Feature**: `002-auth-foundation`  
**Date**: 2026-04-26

---

## Prerequisites

- Go 1.23+
- Docker + Docker Compose v2 (`docker compose` — not the legacy `docker-compose`)
- `make`
- `git`
- Mac or Linux only — Windows is not supported

---

## 1. Clone and build

```bash
git clone https://github.com/rook-project/rook-reference
cd rook-reference
make build
# Outputs: dist/rook-cli and dist/rook-server-cli
```

---

## 2. Start the local dev stack

```bash
cd rook-server
docker compose up
```

This starts:
- `user-service` on `http://localhost:8080`
- Firestore emulator on `http://localhost:8088`

Wait until both services are healthy (logs show `user-service: listening on :8080` and `Firestore emulator running`).

---

## 3. Seed a test user and public key

The Firestore emulator exposes a REST API. Use `curl` to seed your public key:

```bash
# Export your SSH public key in OpenSSH wire format (base64 only, no type prefix)
PUB_KEY=$(ssh-keygen -e -f ~/.ssh/id_ed25519.pub -m pkcs8 | base64 | tr -d '
')
# Or for a simple approach, use the raw content of the .pub file
PUB_KEY_RAW=$(cat ~/.ssh/id_ed25519.pub)

# Create a user document in the Firestore emulator
curl -s -X PATCH 
  "http://localhost:8088/v1/projects/rook-local/databases/(default)/documents/users/test-user-001" 
  -H "Content-Type: application/json" 
  -d '{
    "fields": {
      "user_id":      {"stringValue": "test-user-001"},
      "public_key":   {"stringValue": "'"$PUB_KEY_RAW"'"},
      "display_name": {"stringValue": "Test User"},
      "created_at":   {"timestampValue": "2026-04-26T00:00:00Z"}
    }
  }'
```

Verify the seed:
```bash
curl -s "http://localhost:8088/v1/projects/rook-local/databases/(default)/documents/users/test-user-001" | jq .
```

---

## 4. Configure rook-cli to point at the local stack

```bash
# Create the config directory
mkdir -p ~/.config/rook

# Write a minimal config pointing at the local user-service
cat > ~/.config/rook/config.json <<EOF
{
  "servers": [{"address": "http://localhost:8080"}],
  "active_space": "",
  "storage_dir": "",
  "feature_flags": {}
}
EOF
```

---

## 5. Run the auth flow

```bash
# Interactive (prompts for SSH key selection if multiple keys found)
./dist/rook-cli auth

# Non-interactive (specify key directly)
./dist/rook-cli auth --key ~/.ssh/id_ed25519
```

Expected output on success:
```
✓ Authenticated as test-user-001
  Session token valid for 1 hour
```

---

## 6. Test error conditions

**Unregistered key**:
```bash
# Generate a temporary key not seeded in Firestore
ssh-keygen -t ed25519 -f /tmp/test-unregistered -N ""
./dist/rook-cli auth --key /tmp/test-unregistered
# Expected: error "public key not registered"
```

**Expired nonce** (requires waiting or manual TTL manipulation — integration test only):
See `rook-cli/internal/auth/auth_test.go` for the mock-based test.

---

## 7. Check status and logout (stubs)

```bash
./dist/rook-cli auth status
# Output: no active session

./dist/rook-cli auth logout
# Output: logged out
```

---

## 8. Run tests

```bash
# All tests
make test

# CLI only
cd rook-cli && go test ./... -race -count=1

# Server only (unit tests; no Docker required)
cd rook-server && go test ./... -race -count=1
```

---

## 9. Tear down the local stack

```bash
cd rook-server
docker compose down
```

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `connection refused localhost:8080` | user-service not started | `docker compose up` in `rook-server/` |
| `nonce expired` on verify | More than 60 seconds elapsed between challenge and verify | Run `rook auth` again |
| `public key not registered` | Key not seeded, or wrong key file | Re-run step 3 with the correct key |
| `no keys found in ~/.ssh/` | No `id_*` files in `~/.ssh/` | Generate a key with `ssh-keygen -t ed25519` |
| Docker Compose v1 (`docker-compose`) | Legacy Compose not supported | Install Docker Desktop 4.x+ or `docker compose` plugin |
