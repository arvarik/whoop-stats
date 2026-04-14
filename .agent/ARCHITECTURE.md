# Architecture

_This document is the definitive source of truth for system design, data models, API contracts, and technology boundaries. Every claim here has been verified against the actual source code. Update this document during the Design and Review phases._

## 1. Tech Stack & Infrastructure

| Layer | Technology | Version |
|-------|-----------|---------|
| Backend Language | Go | 1.25.0 |
| Frontend Framework | Next.js (App Router) | 16.1.6 |
| Frontend Runtime | Node.js | 22+ |
| HTTP Router | go-chi/chi | v5 |
| Database | PostgreSQL + TimescaleDB | PG 15 |
| DB Driver | jackc/pgx | v5 |
| DB Code Gen | sqlc | v2 |
| WHOOP SDK | arvarik/whoop-go | v1.1.0 |
| Config | spf13/viper | (WHOOP_STATS_ prefix) |
| Auth | golang-jwt/jwt + jose (frontend) | v5 / v6 |
| CSS | Tailwind CSS | v4 |
| UI Components | shadcn v4, @base-ui/react | |
| Charts | Recharts | v2 |
| Animations | Framer Motion | v12 |
| API Client | openapi-fetch + openapi-typescript | |
| Deployment | Docker Compose (self-hosted) | |
| CI | GitHub Actions | |

## 2. System Boundaries & Data Flow

### Frontend Data Flow
Next.js pages in `web/src/app/` are **React Server Components** by default. On each page load:
1. The RSC calls `client.GET("/api/v1/...")` via `openapi-fetch` to **parallel-fetch** all required data (cycles, sleeps, workouts, recoveries, profile) using `Promise.all()`.
2. The `openapi-fetch` client (`web/src/lib/api/client.ts`) injects a JWT Bearer token generated **server-side** using `jose` (HS256, signed with `WHOOP_STATS_ENCRYPTION_KEY`).
3. Server Actions (`web/src/app/actions.ts`) handle mutations — triggering a `POST /api/v1/sync` and calling `revalidatePath()` on all dashboard routes to bust the RSC cache.

### Backend Data Flow
```
HTTP Request
  → go-chi middleware stack:
    → RequestID (chimiddleware)
    → RealIP (chimiddleware)
    → Recoverer (chimiddleware)
    → Structured slog Logger (internal/middleware)
    → CORS (go-chi/cors)
    → IP Rate Limiter (internal/middleware, 20 req/s burst 50)
  → JWT Auth middleware (internal/middleware, HS256 only)
  → Handler (internal/api)
    → storage layer (internal/storage) — maps whoop-go SDK types to sqlc params
    → sqlc DAL (internal/db) — auto-generated pgx/v5 queries
    → PostgreSQL 15 + TimescaleDB
```

### Dual-Ingestion Model
The application supports two operational modes via the `-mode` flag (`poll` or `webhook`):

**Polling Mode** (`-mode poll`, default):
- `internal/poller` spins up 4 independent goroutine polling loops: `cycles_recoveries`, `workouts`, `sleeps`, `profile`.
- Each loop runs on a configurable interval with **initial random jitter** (0-60s) to prevent thundering herd.
- A shared `rate.Limiter` (2 req/500ms) throttles all WHOOP API calls across goroutines.
- Sleep polling has **adaptive frequency**: normal interval during peak hours (6 AM–12 PM), extended `POLL_INTERVAL_SLEEP_OFFPEAK` otherwise.
- Data is paginated through the WHOOP API using the `whoop-go` SDK's `NextPage()` iterator.

**Webhook Mode** (`-mode webhook`):
- Uses the **Inbox Pattern** for zero-data-loss guarantees.
- `internal/webhook/handler.go`: Validates HMAC signature via `whoop.ParseWebhook()`, marshals to JSON, inserts into `webhook_events` (status: `pending`), returns `200 OK` immediately.
- `internal/webhook/worker.go`: Background worker polls `webhook_events` every 5 seconds, fetches up to 50 pending events per batch, fetches the full object from the WHOOP API, upserts into hypertables, and batch-updates statuses to `processed` or `failed`.
- Handles event types: `recovery.updated`, `cycle.updated`, `workout.updated`, `sleep.updated`.

**Note:** In both modes, the Poller is instantiated — in webhook mode it's only used for ad-hoc `/sync` triggers from the UI.

### Concurrency Model
- **Sync Endpoint** (`POST /api/v1/sync`): Uses an **in-memory `sync.Mutex`** protecting two maps:
  - `activeSyncs map[string]bool` — prevents concurrent syncs per WHOOP user ID (returns `409 Conflict`).
  - `syncLimiters map[string]*rate.Limiter` — enforces 1 sync per 5 minutes per user (returns `429 Too Many Requests`).
  - The sync job runs in a background goroutine with a 10-minute context timeout.
- **Poller**: Each poll loop is a goroutine managed by a `sync.WaitGroup`. The shared `rate.Limiter` prevents API abuse.
- **Auth Manager**: Uses `sync.Map` for client cache + per-user `sync.Mutex` for token refresh (double-checked locking pattern).

## 3. Data Models & Database Schema

### Core Tables
| Table | Description |
|-------|-------------|
| `users` | User accounts with AES-256-GCM encrypted OAuth2 tokens (`BYTEA`). PK: `UUID`. |
| `user_profiles` | WHOOP profile data (name, email). FK to `users(id)`. |
| `body_measurements` | Height, weight, max HR. FK to `users(id)`. |
| `webhook_events` | Inbox for raw webhook payloads. Fields: `payload JSONB`, `status`, `retry_count`. |

### TimescaleDB Hypertables
All time-series tables use **composite primary keys** `(id, start_time)` — required by TimescaleDB for chunk-level uniqueness.
| Hypertable | Partition Column | Key Data |
|-----------|-----------------|----------|
| `cycles` | `start_time` | Strain, kilojoule, avg/max HR |
| `recoveries` | `start_time` | Recovery score, RHR, HRV, SpO2, skin temp |
| `sleeps` | `start_time` | Performance, efficiency, stage breakdowns (light/REM/deep/awake), sleep need/debt |
| `workouts` | `start_time` | Strain, HR zones (0-5), distance, altitude, sport classification |

### Continuous Aggregates (5 materialized views)
Auto-refreshed by TimescaleDB policies (every 1 hour, looking back 3 days):
- `daily_strain` — avg/max strain per day from `cycles`
- `weekly_strain` — avg/max strain per week from `cycles`
- `daily_recovery` — avg recovery score per day from `recoveries`
- `weekly_recovery` — avg recovery score per week from `recoveries`
- `daily_sleep` — avg performance/efficiency per day from `sleeps` (naps excluded)

### Performance Indexes
- `idx_cycles_user_start` — `(user_id, start_time DESC)` on all hypertables
- `idx_webhook_status_created` — `(status, created_at ASC)` on `webhook_events`

### Schema Change Process
1. Write raw SQL migrations in `migrations/` (numbered: `000001_init_schema.up.sql`).
2. Run `sqlc generate` — generates type-safe Go code into `internal/db/` using pgx/v5.
3. **Never edit files in `internal/db/` directly** — they are overwritten by sqlc.

## 4. API Contracts

### Authentication
All `/api/v1/*` endpoints require a JWT Bearer token (HS256) with a `whoop_user_id` string claim. On the frontend, JWTs are generated server-side by `web/src/lib/api/client.ts` using `jose`, signed with `WHOOP_STATS_ENCRYPTION_KEY`, cached for ~23 hours.

### Endpoints (from `internal/api/server.go`)
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/healthz` | inline | DB ping health check |
| GET | `/swagger/*` | http-swagger | Swagger UI |
| GET | `/api/v1/user/profile` | `GetProfile` | Fetch WHOOP profile via SDK |
| GET | `/api/v1/cycles` | `GetCycles` | Cursor-paginated cycles |
| GET | `/api/v1/sleeps` | `GetSleeps` | Cursor-paginated sleeps |
| GET | `/api/v1/workouts` | `GetWorkouts` | Cursor-paginated workouts |
| GET | `/api/v1/recoveries` | `GetRecoveries` | Cursor-paginated recoveries |
| GET | `/api/v1/insights` | `GetInsights` | 30-day strain + recovery from continuous aggregates |
| POST | `/api/v1/sync` | `PostSync` | Trigger ad-hoc sync (202 Accepted) |
| POST | `/webhook` | webhook.Handler | Webhook inbox (webhook mode only) |

### Pagination
All list endpoints use **cursor-based pagination** via RFC3339 timestamp `cursor` query parameter + `limit` (default 50, max 200). **Never use OFFSET/LIMIT**.

### Error Response Format
```json
{ "error": { "code": "AUTH_ERROR", "message": "Invalid user" } }
```

## 5. Security Architecture

### Token Lifecycle
1. **Bootstrap**: `cmd/auth/main.go` runs an OAuth2 Authorization Code flow, saves tokens to `.whoop_token.json`.
2. **First Run**: `internal/auth/manager.go` reads `.whoop_token.json` as offline fallback, encrypts tokens with AES-256-GCM, stores as `BYTEA` in `users` table.
3. **Steady State**: On each API call, the auth manager loads encrypted tokens from DB, decrypts in-memory, refreshes via WHOOP token endpoint, re-encrypts and persists the new tokens.
4. **Cache**: Authenticated `whoop.Client` instances are cached in a `sync.Map` with 55-minute TTL (tokens expire in 1 hour).
5. **JSON Sync**: After every refresh, `.whoop_token.json` is updated (best-effort) to prevent stale fallback after DB wipes.

### Encryption
- `internal/crypto/aes.go`: AES-256-GCM with random nonce per encryption. Key must be exactly 32 bytes.
- NEVER log, print, or commit access/refresh tokens or `ENCRYPTION_KEY`.

### JWT Authentication
- `internal/middleware/auth.go`: Validates HS256 JWT, explicitly rejects non-HMAC algorithms to prevent algorithm confusion attacks.
- Frontend generates JWTs server-side (never exposed to browser).

### Rate Limiting
- IP-based rate limiter with configurable rate + burst (default: 20 req/s, burst 50).
- Stale visitor entries cleaned up every 10 minutes.
- Per-user sync rate limiter: 1 sync per 5 minutes.

### Docker Security
- Dockerfiles drop to non-root users (`appuser`, `nextjs`).
- `tmpfs` mounts for `/tmp` and `/var/log` to minimize SSD write amplification.
- `local` logging driver with 10MB max size.
- `.whoop_token.json` mounted read-only in production.

## 6. Directory Structure

```
whoop-stats/
├── cmd/
│   ├── server/main.go          # Main server entrypoint (-mode poll|webhook)
│   └── auth/main.go            # OAuth2 token bootstrap CLI
├── internal/
│   ├── api/                    # HTTP handlers + chi router (server.go, handlers.go)
│   ├── auth/                   # OAuth2 token lifecycle, client cache, offline fallback
│   ├── config/                 # Viper config loader (WHOOP_STATS_ prefix)
│   ├── crypto/                 # AES-256-GCM encrypt/decrypt
│   ├── db/                     # ⚠️ AUTO-GENERATED by sqlc — never edit!
│   ├── middleware/             # JWT auth, slog logger, IP rate limiter
│   ├── poller/                 # Periodic WHOOP API scraper (4 poll loops)
│   ├── storage/                # Domain mapper: whoop-go types → sqlc params (upserts)
│   └── webhook/                # Inbox handler + background worker
├── migrations/                 # SQL schema migrations
├── queries/query.sql           # sqlc query definitions
├── sqlc.yaml                   # sqlc config (pgx/v5, output: internal/db)
├── docs/                       # Swagger JSON/YAML, OpenAPI spec, archives
├── web/                        # Next.js 16 frontend
│   ├── src/app/                # App Router pages + Server Actions
│   │   ├── page.tsx            # Dashboard (Overview)
│   │   ├── recovery/page.tsx   # Recovery detail page
│   │   ├── sleep/page.tsx      # Sleep detail page
│   │   ├── strain/page.tsx     # Strain detail page
│   │   ├── workouts/page.tsx   # Workouts detail page
│   │   ├── actions.ts          # Server Actions (syncWhoopData)
│   │   ├── layout.tsx          # Root layout (Inter font, Sidebar, MobileNav, Toaster)
│   │   └── globals.css         # Design tokens, glass-card utilities
│   ├── src/components/         # UI components (metric cards, charts, panels)
│   │   └── ui/                 # shadcn primitives (button, card, badge, etc.)
│   └── src/lib/
│       ├── api/client.ts       # openapi-fetch client with JWT injection
│       ├── api/schema.d.ts     # Auto-generated TypeScript types from OpenAPI
│       ├── format.ts           # Display formatters
│       ├── types.ts            # Shared frontend types
│       └── utils.ts            # Utility functions (cn)
├── .github/workflows/          # CI pipeline (Go build/vet/test + Next.js lint/build)
├── .agent/                     # AI agent context documentation
├── docker-compose.yml          # Dev compose (exposes ports, bind mounts)
├── docker-compose.prod.yml     # Prod compose (named volumes, networks)
├── Dockerfile.backend          # Multi-stage Go build
└── web/Dockerfile              # Multi-stage Next.js standalone build
```

## 7. Environment Variables

All backend env vars use the `WHOOP_STATS_` prefix (via Viper). The full list from `.env.example` and `internal/config/config.go`:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `WHOOP_STATS_DATABASE_URL` | Yes | — | Postgres connection string |
| `WHOOP_STATS_ENCRYPTION_KEY` | Yes | — | Exactly 32 bytes for AES-256-GCM |
| `WHOOP_STATS_WHOOP_CLIENT_ID` | Yes | — | From developer.whoop.com |
| `WHOOP_STATS_WHOOP_CLIENT_SECRET` | Yes | — | From developer.whoop.com |
| `WHOOP_STATS_WHOOP_WEBHOOK_SECRET` | Webhook mode | — | HMAC secret for webhook validation |
| `WHOOP_STATS_SERVER_PORT` | No | `8080` | Internal server port |
| `WHOOP_STATS_LOG_LEVEL` | No | `info` | debug, info, warn, error |
| `WHOOP_STATS_CORS_ALLOWED_ORIGINS` | No | `http://localhost:3032` | Comma-separated origins |
| `WHOOP_STATS_POLL_INTERVAL_CYCLE` | No | `4h` | Cycle polling interval |
| `WHOOP_STATS_POLL_INTERVAL_WORKOUT` | No | `30m` | Workout polling interval |
| `WHOOP_STATS_POLL_INTERVAL_SLEEP` | No | `1h` | Sleep polling interval (peak hours) |
| `WHOOP_STATS_POLL_INTERVAL_SLEEP_OFFPEAK` | No | `4h` | Sleep polling interval (off-peak) |
| `WHOOP_STATS_POLL_INTERVAL_PROFILE` | No | `24h` | Profile/body measurement interval |
| `POSTGRES_USER` | Docker | `whoop_user` | Postgres username |
| `POSTGRES_PASSWORD` | Docker | — | Postgres password |
| `WHOOP_USER_ID` | Yes | — | Your WHOOP user ID |
| `BACKEND_PORT` | No | `8085` | Host-mapped backend port |
| `FRONTEND_PORT` | No | `3032` | Host-mapped frontend port |
| `NEXT_PUBLIC_API_URL` | Frontend | `http://localhost:8080` | Backend URL for API calls |
| `WHOOP_STATS_WHOOP_USER_ID` | Frontend | — | WHOOP user ID for JWT generation |

## 8. Error Handling Patterns
- **Go Backend**: Explicit `if err != nil` with `fmt.Errorf("context: %w", err)` wrapping. HTTP errors use structured `ErrorResponse` JSON envelopes with codes like `AUTH_ERROR`, `DB_ERROR`, `CONFLICT`, `RATE_LIMIT_EXCEEDED`.
- **Sync endpoint**: Returns `409 Conflict` for concurrent syncs, `429 Too Many Requests` for rate-limited syncs, `202 Accepted` on success.
- **Webhook worker**: Failed events are marked `failed` in batch. Processed events marked `processed`.
- **Frontend**: Server Actions throw errors surfaced via Sonner toast notifications.

## 9. Key Dependencies

| Package | Role |
|---------|------|
| `arvarik/whoop-go` | First-party WHOOP API SDK (cycles, sleeps, workouts, recoveries, user) |
| `go-chi/chi/v5` | HTTP router |
| `jackc/pgx/v5` | PostgreSQL driver (connection pooling via `pgxpool`) |
| `spf13/viper` | Configuration management |
| `golang-jwt/jwt/v5` | JWT parsing/validation (backend) |
| `jose` (npm) | JWT generation (frontend, server-side only) |
| `openapi-fetch` | Type-safe API client (frontend) |
| `testcontainers-go` | Dockerized Postgres for integration tests |
| `golang.org/x/time/rate` | Rate limiting (API + poller) |
| `swaggo/swag` | Swagger spec generation from Go annotations |
