# whoop-stats Status
Last updated: 2026-04-14

_This file tracks the current state of development. It is the single source of truth for "where am I?" Agents must update this file after completing tasks or making progress._

## Current Objective
Hardening `.agent/` documentation suite for accuracy and comprehensiveness.

## Completed Work

### Core Application (Fully Built)
- [x] PostgreSQL + TimescaleDB schema with 4 hypertables and 5 continuous aggregates
- [x] sqlc code generation pipeline (pgx/v5 → `internal/db/`)
- [x] Dual-mode CLI (`-mode poll` and `-mode webhook`)
- [x] Polling engine with 4 independent loops + adaptive sleep scheduling
- [x] Webhook inbox handler + background worker
- [x] AES-256-GCM token encryption (`internal/crypto/`)
- [x] OAuth2 token lifecycle with offline JSON fallback (`internal/auth/`)
- [x] JWT auth middleware (HS256, algorithm confusion protection)
- [x] IP-based rate limiting middleware
- [x] Structured slog request logging (health checks excluded)
- [x] CORS middleware with configurable origins
- [x] Swagger API documentation (`swag init`)
- [x] OAuth2 bootstrap CLI (`cmd/auth/`)
- [x] `openapi-fetch` + `jose` JWT client for type-safe frontend API calls
- [x] Cursor-based pagination on all list endpoints

### Dashboard (Fully Built)
- [x] Overview page (strain, recovery, sleep cards + 7-day recovery strip + 30-day trend chart)
- [x] Recovery detail page
- [x] Sleep detail page
- [x] Strain detail page
- [x] Workouts detail page
- [x] Sync button with Server Action integration
- [x] Sidebar navigation + mobile bottom nav
- [x] Glass-card design system
- [x] Sonner toast notifications

### Infrastructure
- [x] Docker Compose (dev + prod)
- [x] Dockerfile.backend (multi-stage Go build)
- [x] web/Dockerfile (Next.js standalone build)
- [x] GitHub Actions CI (Go build/vet/test + Next.js lint/build)
- [x] SSD-friendly tmpfs mounts and local logging driver
- [x] `.env.example` with documentation

### Documentation
- [x] README.md with architecture diagrams (Mermaid)
- [x] `.agent/` documentation suite (ARCHITECTURE, PHILOSOPHY, STYLE, TESTING, STATUS)
- [x] GEMINI.md system rules

## Known Issues
- `docker-compose.prod.yml` references `Dockerfile.frontend` but the actual file is `web/Dockerfile` — potential build failure in prod compose.
- `docker-compose.prod.yml` backend env vars missing `WHOOP_STATS_` prefix for some variables (e.g., `ENCRYPTION_KEY` instead of `WHOOP_STATS_ENCRYPTION_KEY`).
- CI uses Go 1.22 but `go.mod` specifies Go 1.25.0 — should be aligned.
- `.whoop_token.json` is tracked in git — should be in `.gitignore`.

## What's Next
- Fix prod Docker Compose env var prefix inconsistencies
- Add `.whoop_token.json` to `.gitignore`
- Align CI Go version with `go.mod`
- Consider adding `testcontainers-go` integration tests to CI

## Dashboard Routes
| Route | Page |
|-------|------|
| `/` | Overview dashboard |
| `/recovery` | Recovery detail |
| `/sleep` | Sleep detail |
| `/strain` | Strain detail |
| `/workouts` | Workouts detail |

## Relevant Context
- `.agent/ARCHITECTURE.md` — System design, data flow, API contracts
- `.agent/STYLE.md` — Code conventions, design tokens, anti-patterns
- `.agent/TESTING.md` — Test commands, CI pipeline, evidence rules
- `.agent/PHILOSOPHY.md` — Product beliefs, target user, UX principles
- `GEMINI.md` — AI system rules (root level)
