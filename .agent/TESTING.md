# Testing Strategy & Results

_This file tracks test methods, scenarios, and results with concrete execution evidence. Bugs found here block the release of a feature. Agents must update this during the Test and Fix phases._

## 0. Prerequisites & Setup

### Required Tools
- Go 1.25.0+ (Note: CI currently uses Go 1.22 — see Known Issues in STATUS.md)
- Node.js 20+ with npm
- Docker (required for `testcontainers-go` integration tests and TimescaleDB)
- `sqlc` v1.30.0+ (for code generation verification)
- `golangci-lint` (optional, not in CI)

### Start the Full Stack
```bash
# Database (TimescaleDB + auto-runs init migration via docker-entrypoint-initdb.d)
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
If `git diff internal/db/` shows unexpected changes in `db.go`, `models.go`, `querier.go`, or `query.sql.go`, the DAL is out of sync. Note: `internal/db/batch.go` is hand-written and should NOT be affected by `sqlc generate`.

After modifying Go handler annotations:
```bash
swag init -g cmd/server/main.go
```

## 1. Test Methods & Commands

### Backend (Go)

**Unit Tests** (no Docker required for most):
```bash
# Core unit tests (crypto, config, middleware)
go test -race -count=1 ./internal/crypto/...
go test -race -count=1 ./internal/middleware/...
go test -race -count=1 ./internal/config/...

# Timezone parsing edge cases
go test -race -count=1 ./internal/storage/ -run TestParseTimezoneOffset

# Adaptive off-peak polling logic
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

**Benchmarks**:
```bash
go test -bench=. -benchmem ./internal/webhook/...
```

### Frontend (Next.js)

**Lint** (ESLint 9 with `eslint-config-next` — flat config via `eslint.config.mjs`):
```bash
cd web && npm run lint
```

**Build** (catches TypeScript errors and SSR issues):
```bash
cd web && NEXT_TELEMETRY_DISABLED=1 npm run build
```
Note: The build uses `output: "standalone"` which requires all imports to be resolvable. Env vars are validated lazily at runtime, not build time.

**Unit Tests** (Node.js native test runner with experimental TypeScript strip):
```bash
cd web && npm test
# Equivalent to: node --experimental-strip-types --test "src/**/*.test.ts"
```

### Docker Image Verification
```bash
# Build backend image
docker build -f Dockerfile.backend -t whoop-stats-backend .

# Build frontend image
docker build -f web/Dockerfile -t whoop-stats-frontend ./web

# Verify non-root user in backend
docker run --rm whoop-stats-backend whoami  # should print "appuser"

# Verify non-root user in frontend
docker run --rm whoop-stats-frontend whoami  # should print "nextjs"
```

## 2. Test File Inventory

### Backend Test Files
| File | Tests | Docker Required |
|------|-------|-----------------|
| `internal/api/handlers_test.go` | API handler unit tests | No |
| `internal/config/config_test.go` | Config validation (missing required vars, invalid key length) | No |
| `internal/crypto/aes_test.go` | AES encrypt/decrypt round-trip, invalid key, tamper detection | No |
| `internal/middleware/auth_test.go` | JWT HS256 validation, algorithm confusion prevention, missing claims | No |
| `internal/middleware/ratelimit_test.go` | IP rate limiter enforcement, burst behavior | No |
| `internal/poller/manager_test.go` | Off-peak adaptive polling logic, interval parsing | No |
| `internal/storage/storage_test.go` | Storage layer mapping tests | Potentially |
| `internal/storage/timezone_test.go` | Timezone offset parsing (`+05:30`, `-0500`, `Z`, empty, malformed) | No |
| `internal/webhook/worker_bench_test.go` | Webhook worker benchmarks | No |

### Frontend Test Files
| File | Tests | Runner |
|------|-------|--------|
| `web/src/components/ui/chart.test.ts` | Chart component unit tests | `node --experimental-strip-types --test` |
| `web/src/lib/stats.test.ts` | Stats utility tests (computeAvg, computeStdDev, percentChange) | `node --experimental-strip-types --test` |

## 3. CI Pipeline

### CI Workflow (`.github/workflows/ci.yml`)
Runs on pushes and PRs to `main`. Two jobs:

**Backend Job** (Go 1.22, ubuntu-latest):
1. `go build -v ./cmd/... ./internal/...`
2. `go vet ./cmd/... ./internal/...`
3. Unit tests with race detector:
   - `go test -race -count=1 ./internal/crypto/... ./internal/middleware/... ./internal/config/...`
   - `go test -race -count=1 ./internal/storage/ -run TestParseTimezoneOffset`
   - `go test -race -count=1 ./internal/poller/ -run TestPoller_Offpeak`

**Frontend Job** (Node.js 20, ubuntu-latest):
1. `npm ci --legacy-peer-deps`
2. `npm run lint`
3. `npm run build` (with `NEXT_TELEMETRY_DISABLED=1`)

> **Known Issue:** CI uses Go 1.22 but `go.mod` specifies 1.25.0. These should be aligned.

### Publish Workflow (`.github/workflows/publish.yml`)
Runs on pushes to `main` and version tags (`v*`). Publishes Docker images:
- **Backend**: `ghcr.io/${{ github.repository }}/backend` from `Dockerfile.backend`
- **Frontend**: `ghcr.io/${{ github.repository }}/frontend` from `web/Dockerfile`
- Uses Docker Buildx with GitHub Actions cache (`type=gha`)
- Tags: git SHA, semver (`v1.2.3` → `1.2.3`, `1.2`), `latest` (default branch only)

## 4. Execution Evidence Rules

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
| Stats utility (avg, stddev, percentChange) | UNTESTED | `cd web && npm test` |
| Chart component tests | UNTESTED | `cd web && npm test` |
| Frontend ESLint clean | UNTESTED | `cd web && npm run lint` |
| Frontend TypeScript build | UNTESTED | `cd web && npm run build` |

---

## Backend Route Coverage Matrix

_Populated by the SDET during the Trap phase. One row per API endpoint. All cells must show PASS with execution evidence or FAIL with reproduction steps._

| Endpoint | Method | 200 OK | 400 Bad Req | 401/403 Auth | 404 Not Found | Idempotent | Edge Cases |
|----------|--------|--------|-------------|--------------|---------------|------------|------------|
| `/healthz` | GET | | | N/A | | | DB unreachable → 503 |
| `/swagger/*` | GET | | | N/A | | | |
| `/api/v1/user/profile` | GET | | | | | | SDK API failure |
| `/api/v1/cycles` | GET | | | | | | Invalid cursor, limit > 200, no data |
| `/api/v1/sleeps` | GET | | | | | | Invalid cursor, limit > 200, no data |
| `/api/v1/workouts` | GET | | | | | | Invalid cursor, limit > 200, no data |
| `/api/v1/recoveries` | GET | | | | | | Invalid cursor, limit > 200, no data |
| `/api/v1/insights` | GET | | | | | | No aggregate data |
| `/api/v1/sync` | POST | | | | | 409 concurrent, 429 rate limit | 10-min timeout |
| `/webhook` | POST | | | N/A (HMAC) | | | Invalid HMAC, unknown event type |

---

## Frontend Component State Matrix

_Populated by the SDET during the Trap phase. Every interactive component must be tested across all visual states._

| Component | Empty | Loading | Success | Error | Partial |
|-----------|-------|---------|---------|-------|---------|
| Overview Dashboard (`page.tsx`) | | | | | |
| Recovery Page (`recovery/page.tsx`) | | | | | |
| Sleep Page (`sleep/page.tsx`) | | | | | |
| Strain Page (`strain/page.tsx`) | | | | | |
| Workouts Page (`workouts/page.tsx`) | | | | | |
| SyncButton | | | | | |
| MetricCard | | | | | |
| StrainRecoveryChart | | | | | |
| TrendChart | | | | | |
| RecoveryGauge | | | | | |
| RecoveryPanels | | | | | |
| SleepPanels | | | | | |
| SleepStagesBar | | | | | |
| StrainPanels | | | | | |
| WorkoutCard | | | | | |
| WorkoutDetail | | | | | |
| WorkoutFeed | | | | | |
| RecentWorkouts | | | | | |
| DetailPopup | | | | | |
| Sidebar | | | | | |
| MobileNav | | | | | |
| Global Error Boundary (`error.tsx`) | | | | | |
| Skeleton Loading (`loading.tsx`) | | | | | |

---

## ML / AI Evaluation Thresholds

N/A — ML/AI topology is not active for this project.

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
| Docker backend builds | _YYYY-MM-DD_ | `docker build -f Dockerfile.backend .` |
| Docker frontend builds | _YYYY-MM-DD_ | `docker build -f web/Dockerfile ./web` |
