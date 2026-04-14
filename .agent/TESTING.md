# Testing Strategy & Results

_This file tracks test methods, scenarios, and results with concrete execution evidence. Bugs found here block the release of a feature. Agents must update this during the Test and Fix phases._

## 0. Local Development Setup
### Prerequisites
- Go 1.25.0+
- Node.js 22+, npm
- Docker
- golangci-lint

### Start the App
- Database: `docker compose up db` (PostgreSQL + TimescaleDB)
- Backend: `docker compose up backend` OR natively: `go run ./cmd/server`
- Frontend: `cd web && npm run dev`

### Code Generation
- Go DAL: `sqlc generate`
- API Spec: `swag init -g cmd/server/main.go`

## 1. Test Methods & Tools
### Backend / Go Tests
- **Run all tests**: `go test ./...`
- **Run with race detector**: `go test -race -count=1 ./...`
- **Linting**: `golangci-lint run ./...` (must produce 0 warnings)
- **Integration**: Database tests rely on `testcontainers-go` pulling PostgreSQL images.

### Frontend Tests
- **Linting**: `cd web && npm run lint` (`npx eslint`)
- **Type Checking**: `cd web && npx tsc --noEmit`

## 2. Execution Evidence Rules
_Never mark a test as PASS without evidence._
- For Go tests, paste the output of `go test -v ./...`.
- For linting/type-checking, paste the stdout `0 errors/warnings`.
- "PASS" with no evidence is treated as UNTESTED.

---

## Current Feature Scenarios: Bootstrapped

| Scenario | Status | Notes (Evidence) |
|----------|--------|------------------|
| Empty/null/missing inputs | UNTESTED | |

## Bugs Found (Fix Phase Queue)
- (None)

---

## Regression Scenarios (Persistent)
| Scenario | Last Verified | Notes |
|----------|---------------|-------|
| _Race detector passes_ | _YYYY-MM-DD_ | _Go: `go test -race ./...`_ |
