# whoop-stats Status
Last updated: 2026-04-14

_This file tracks the current state of development. It is the single source of truth for "where am I?" Agents must update this file after completing tasks or making progress._

## Current Objective
Hardening `.agent/` documentation suite for accuracy and comprehensiveness.

## Completed Work

### Core Application (Fully Built)
- [x] PostgreSQL + TimescaleDB schema with 4 hypertables and 5 continuous aggregates
- [x] sqlc code generation pipeline (pgx/v5 → `internal/db/`, v1.30.0)
- [x] Hand-written `internal/db/batch.go` for pgx.Batch SQL constant exports
- [x] Dual-mode CLI (`-mode poll` and `-mode webhook`)
- [x] Polling engine with 4 independent loops + adaptive sleep scheduling (off-peak 6 AM–12 PM)
- [x] Webhook inbox handler + background worker (5s poll, 50 events/batch)
- [x] AES-256-GCM token encryption (`internal/crypto/`)
- [x] OAuth2 token lifecycle with offline JSON fallback (`internal/auth/`)
- [x] OAuth2 bootstrap CLI with token refresh support (`cmd/auth/`)
- [x] JWT auth middleware (HS256, algorithm confusion protection)
- [x] IP-based rate limiting middleware (20 req/s burst 50, 30m stale cleanup)
- [x] Per-user sync rate limiting (1 sync/5 min, 409 Conflict for concurrent)
- [x] Structured slog request logging (health checks excluded)
- [x] CORS middleware with configurable origins
- [x] Swagger API documentation (`swag init`)
- [x] `openapi-fetch` + `jose` JWT client for type-safe frontend API calls
- [x] Cursor-based pagination on all list endpoints (RFC3339Nano, default 50, max 200)
- [x] HTTP server with configurable timeouts (15s read, 30s write, 60s idle)
- [x] Graceful shutdown (SIGINT/SIGTERM, 10s deadline, WaitGroup drain)
- [x] ParseTimezoneOffset utility (WHOOP format → pgtype.Interval)

### Dashboard (Fully Built)
- [x] Overview page (strain, recovery, sleep cards + 7-day recovery strip + 30-day trend chart)
- [x] Recovery detail page
- [x] Sleep detail page
- [x] Strain detail page
- [x] Workouts detail page
- [x] Sync button with Server Action integration (revalidates all 5 routes)
- [x] Sidebar navigation + mobile bottom nav
- [x] Glass-card design system with full token palette
- [x] Sonner toast notifications
- [x] Global error boundary (`error.tsx`) with glass-card retry UI
- [x] Skeleton loading state (`loading.tsx`) matching dashboard layout
- [x] Display formatters (`format.ts`): duration, dates, calories, recovery colors, HR zones
- [x] Statistical helpers (`stats.ts`): avg, stddev, percentChange
- [x] Lazy env var validation in API client (prevents CI build failures)

### Infrastructure
- [x] Docker Compose dev (`docker-compose.yml` — bind mounts, exposed ports)
- [x] Docker Compose prod (`docker-compose.prod.yml` — named volumes, networks)
- [x] Dockerfile.backend (multi-stage Go build → alpine, non-root `appuser`)
- [x] web/Dockerfile (multi-stage Next.js standalone build, non-root `nextjs`)
- [x] Next.js `output: "standalone"` for minimal Docker image
- [x] GitHub Actions CI (`ci.yml` — Go build/vet/test + Next.js lint/build)
- [x] GitHub Actions Publish (`publish.yml` — Docker images to GHCR)
- [x] SSD-friendly tmpfs mounts and local logging driver (10MB max)
- [x] `.env.example` with documented variable template
- [x] `.gitignore` with `.env`, `.whoop_token.json`, `web/.next`, `bin/`

### Documentation
- [x] README.md with architecture diagrams (Mermaid)
- [x] `.agent/` documentation suite (ARCHITECTURE, PHILOSOPHY, STYLE, TESTING, STATUS)
- [x] GEMINI.md system rules

## Known Issues
- `docker-compose.prod.yml` references `Dockerfile.frontend` (line 63) but the actual file is `web/Dockerfile` — **will cause prod build failure**.
- `docker-compose.prod.yml` backend env vars missing `WHOOP_STATS_` prefix for several variables (`ENCRYPTION_KEY`, `WHOOP_CLIENT_ID`, `WHOOP_CLIENT_SECRET`, `WHOOP_WEBHOOK_SECRET`, `CORS_ALLOWED_ORIGINS`, `LOG_LEVEL`, `POLL_INTERVAL_*`). These won't be picked up by Viper.
- CI uses Go 1.22 but `go.mod` specifies Go 1.25.0 — should be aligned.
- `docker-compose.prod.yml` frontend uses `WHOOP_STATS_TOKEN` env var which doesn't exist in code — should be `WHOOP_STATS_ENCRYPTION_KEY` + `WHOOP_STATS_WHOOP_USER_ID`.
- `docker-compose.prod.yml` prod compose doesn't mount the init migration into TimescaleDB (unlike dev compose).
- `docker-compose.prod.yml` frontend `NEXT_PUBLIC_API_URL` points to `http://localhost:8082` which won't work for server-side fetching inside the Docker network — should be `http://backend:8080`.

## What's Next
- Fix prod Docker Compose (`docker-compose.prod.yml`):
  - Correct Dockerfile reference to `Dockerfile` (not `Dockerfile.frontend`)
  - Add `WHOOP_STATS_` prefix to all backend env vars
  - Fix frontend env vars (`WHOOP_STATS_ENCRYPTION_KEY`, `WHOOP_STATS_WHOOP_USER_ID`)
  - Mount init migration for TimescaleDB
  - Fix `NEXT_PUBLIC_API_URL` for Docker networking
- Align CI Go version with `go.mod` (1.22 → 1.25)
- Consider adding `testcontainers-go` integration tests to CI
- Implement retry logic with exponential backoff for failed webhook events

## Dashboard Routes
| Route | Page |
|-------|------|
| `/` | Overview dashboard |
| `/recovery` | Recovery detail |
| `/sleep` | Sleep detail |
| `/strain` | Strain detail |
| `/workouts` | Workouts detail |

## Relevant Context
- `.agent/ARCHITECTURE.md` — System design, data flow, API contracts, full directory tree
- `.agent/STYLE.md` — Code conventions, design tokens, anti-patterns, component toolkit
- `.agent/TESTING.md` — Test commands, CI pipeline, evidence rules
- `.agent/PHILOSOPHY.md` — Product beliefs, target user, UX principles, SSD protection
- `GEMINI.md` — AI system rules (root level)

---

## Stub Audit Tracker

_Track mock/stub status across the frontend. Populated during Build phase, cleared during Ship._

| Stub Location | Type | Real API Endpoint | Status |
|---------------|------|-------------------|--------|

_No active stubs detected. Populate during the next Build phase._

---

## Prompt Versioning Changelog

N/A — No LLM prompts in this project.
