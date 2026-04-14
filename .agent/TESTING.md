# Testing Strategy & Results

_This file tracks test methods, scenarios, and results with concrete execution evidence. Bugs found here block the release of a feature. Agents must update this during the Test and Fix phases._

## 0. Prerequisites & Setup

### Required Tools
- Go 1.25.0+
- Node.js 22+ with npm
- Docker (required for `testcontainers-go` integration tests and TimescaleDB)
- `golangci-lint` (optional, not in CI)

### Start the Full Stack
```bash
# Database (TimescaleDB + auto-runs init migration)
docker compose up -d timescaledb

# Backend (poll mode, default)
go run cmd/server/main.go -mode poll -user YOUR_WHOOP_USER_ID

# Frontend
cd web && npm install --legacy-peer-deps && npm run dev
```

### Code Generation Verification
After modifying `queries/query.sql` or `migrations/*.sql`:
```bash
sqlc generate
```
If `git diff internal/db/` shows unexpected changes, the DAL is out of sync.

After modifying Go handler annotations:
```bash
swag init -g cmd/server/main.go
```

## 1. Test Methods & Commands

### Backend (Go)

**Unit Tests** (no Docker required for most):
```bash
# Core unit tests (crypto, config, middleware, timezone parsing, poller logic)
go test -race -count=1 ./internal/crypto/... ./internal/middleware/... ./internal/config/...
go test -race -count=1 ./internal/storage/ -run TestParseTimezoneOffset
go test -race -count=1 ./internal/poller/ -run TestPoller_Offpeak
```

**All Tests** (may require Docker for testcontainers):
```bash
go test -v -race -count=1 ./...
```

**Build Verification**:
```bash
go build -v ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
```

### Frontend (Next.js)

**Lint** (ESLint 9 with eslint-config-next):
```bash
cd web && npm run lint
```

**Build** (catches TypeScript errors and SSR issues):
```bash
cd web && NEXT_TELEMETRY_DISABLED=1 npm run build
```

**Unit Tests** (Node.js native test runner with TypeScript strip):
```bash
cd web && npm test
# Equivalent to: node --experimental-strip-types --test "src/**/*.test.ts"
```

### Test File Inventory
| File | Tests |
|------|-------|
| `internal/api/handlers_test.go` | API handler unit tests |
| `internal/config/config_test.go` | Config validation tests |
| `internal/crypto/aes_test.go` | AES encrypt/decrypt round-trip |
| `internal/middleware/auth_test.go` | JWT validation, algorithm confusion prevention |
| `internal/middleware/ratelimit_test.go` | IP rate limiter behavior |
| `internal/poller/manager_test.go` | Off-peak adaptive polling logic |
| `internal/storage/storage_test.go` | Storage layer tests |
| `internal/storage/timezone_test.go` | Timezone offset parsing |
| `internal/webhook/worker_bench_test.go` | Webhook worker benchmarks |
| `web/src/components/ui/chart.test.ts` | Chart component tests |
| `web/src/lib/stats.test.ts` | Stats utility tests |

## 2. CI Pipeline (`.github/workflows/ci.yml`)

The CI runs on pushes and PRs to `main`. Two jobs:

**Backend Job** (Go 1.22, ubuntu-latest):
1. `go build -v ./cmd/... ./internal/...`
2. `go vet ./cmd/... ./internal/...`
3. Unit tests with race detector (specific packages, no testcontainers in CI)

**Frontend Job** (Node.js 20, ubuntu-latest):
1. `npm ci --legacy-peer-deps`
2. `npm run lint`
3. `npm run build` (with `NEXT_TELEMETRY_DISABLED=1`)

## 3. Execution Evidence Rules

**Never mark a test as PASS without pasting the actual stdout/stderr output.**

- For Go tests: paste the `go test -v` output in a fenced code block.
- For frontend tests: paste the `npm run lint` and `npm run build` output.
- For visual changes: capture a screenshot via the browser tool.
- "PASS" with no evidence is treated as **UNTESTED**.

---

## Current Feature Scenarios

| Scenario | Status | Evidence |
|----------|--------|----------|
| AES-256-GCM encrypt/decrypt round-trip | UNTESTED | `go test -v ./internal/crypto/...` |
| JWT HS256 validation + algorithm rejection | UNTESTED | `go test -v ./internal/middleware/...` |
| Config validation (missing required vars) | UNTESTED | `go test -v ./internal/config/...` |
| IP rate limiter enforcement | UNTESTED | `go test -v ./internal/middleware/...` |
| Timezone offset parsing | UNTESTED | `go test -v ./internal/storage/ -run TestParse` |
| Adaptive sleep polling (off-peak skip) | UNTESTED | `go test -v ./internal/poller/ -run TestPoller_Offpeak` |
| Frontend ESLint clean | UNTESTED | `cd web && npm run lint` |
| Frontend TypeScript build | UNTESTED | `cd web && npm run build` |

## Bugs Found (Fix Phase Queue)
- (None)

---

## Regression Scenarios (Persistent)

These must pass before any release:

| Scenario | Last Verified | Command |
|----------|---------------|---------|
| Race detector passes (all Go) | _YYYY-MM-DD_ | `go test -race -count=1 ./internal/...` |
| Idempotent upsert (no duplicates) | _YYYY-MM-DD_ | `go test -v ./internal/storage/...` |
| Webhook HMAC validation | _YYYY-MM-DD_ | `go test -v ./internal/webhook/...` |
| Frontend builds without errors | _YYYY-MM-DD_ | `cd web && npm run build` |
| Frontend lints without warnings | _YYYY-MM-DD_ | `cd web && npm run lint` |
| sqlc generation is clean | _YYYY-MM-DD_ | `sqlc generate && git diff --exit-code internal/db/` |
