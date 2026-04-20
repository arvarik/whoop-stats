# Architecture

_This document is the definitive source of truth for system design, data models, API contracts, and technology boundaries. Every claim here has been verified against the actual source code. Update this document during the Design and Review phases._

## 0. Project Topology

**Topology:** `[frontend, backend]`

_Agents: Read the corresponding Gemstack topology profiles (`frontend.md` and `backend.md`) from `~/.gemini/antigravity/global_workflows/` before proceeding with any workflow step. These profiles enforce component state coverage, state management discipline, data integrity testing, and anti-mocking rules._

## 1. Tech Stack & Infrastructure

| Layer | Technology | Version / Notes |
|-------|-----------|-----------------|
| Backend Language | Go | 1.25.0 (`go.mod`) |
| Frontend Framework | Next.js (App Router) | 16.1.6 |
| Frontend Runtime | Node.js | 22+ (dev), 20 (CI + Docker) |
| HTTP Router | go-chi/chi | v5.2.5 |
| Database | PostgreSQL + TimescaleDB | PG 15 (`timescale/timescaledb:latest-pg15`) |
| DB Driver | jackc/pgx | v5.8.0 (`pgxpool` for connection pooling) |
| DB Code Gen | sqlc | v1.30.0 (generated header in `internal/db/`) |
| WHOOP SDK | arvarik/whoop-go | v1.1.0 |
| Config | spf13/viper | v1.21.0 (`WHOOP_STATS_` env prefix) |
| Auth (backend) | golang-jwt/jwt | v5.3.1 (HS256 JWT validation) |
| Auth (frontend) | jose | v6.2.1 (HS256 JWT generation, server-side only) |
| CSS | Tailwind CSS | v4 (`@tailwindcss/postcss`) |
| UI Components | shadcn v4, @base-ui/react v1.2 | Headless + styled primitives |
| Charts | Recharts | v2.15.4 |
| Animations | Framer Motion | v12.35.0 |
| API Client | openapi-fetch v0.17 + openapi-typescript v7.13 | Type-safe fetch from OpenAPI spec |
| Deployment | Docker Compose (self-hosted) | Dev + Prod configs |
| CI | GitHub Actions | `ci.yml` (test/lint) + `publish.yml` (Docker images → GHCR) |
| Testing | testify v1.11, testcontainers-go v0.40 | Unit + integration test framework |

## 2. System Boundaries & Data Flow

### Frontend Data Flow
Next.js pages in `web/src/app/` are **React Server Components** by default. On each page load:
1. The RSC calls `client.GET("/api/v1/...")` via `openapi-fetch` to **parallel-fetch** all required data (cycles, sleeps, workouts, recoveries, profile) using `Promise.all()`.
2. The `openapi-fetch` client (`web/src/lib/api/client.ts`) injects a JWT Bearer token generated **server-side** using `jose` (HS256, signed with `WHOOP_STATS_ENCRYPTION_KEY`). Tokens are cached in-memory for ~23 hours.
3. Server Actions (`web/src/app/actions.ts`) handle mutations — triggering a `POST /api/v1/sync` and calling `revalidatePath()` on all 5 dashboard routes to bust the RSC cache.
4. `error.tsx` provides a global error boundary with retry. `loading.tsx` provides skeleton UI matching the dashboard layout during RSC streaming.

### Backend Data Flow
```
HTTP Request
  → go-chi middleware stack (in order):
    → RequestID (chimiddleware)
    → RealIP (chimiddleware)
    → Recoverer (chimiddleware)
    → Structured slog Logger (internal/middleware) — skips /healthz
    → CORS (go-chi/cors, configurable origins)
    → IP Rate Limiter (internal/middleware, 20 req/s burst 50)
  → JWT Auth middleware (internal/middleware, HS256 only, on /api/v1/* routes)
  → Handler (internal/api)
    → storage layer (internal/storage) — maps whoop-go SDK types to sqlc params
    → sqlc DAL (internal/db) — auto-generated pgx/v5 queries
    → PostgreSQL 15 + TimescaleDB
```

**HTTP Server Timeouts** (from `cmd/server/main.go`):
- `ReadTimeout`: 15 seconds
- `WriteTimeout`: 30 seconds
- `IdleTimeout`: 60 seconds

### Dual-Ingestion Model
The application supports two operational modes via the `-mode` flag (`poll` or `webhook`):

**Polling Mode** (`-mode poll`, default):
- `internal/poller` spins up 4 independent goroutine polling loops: `cycles_recoveries`, `workouts`, `sleeps`, `profile`.
- Each loop runs on a configurable interval with **initial random jitter** (0-60s, cryptographically random via `crypto/rand`) to prevent thundering herd.
- A shared `rate.Limiter` (2 req/500ms) throttles all WHOOP API calls across goroutines.
- Sleep polling has **adaptive frequency**: normal interval during peak hours (6 AM–12 PM), extended `POLL_INTERVAL_SLEEP_OFFPEAK` otherwise. Off-peak polls are tracked via `lastOffpeakSleepPoll` to enforce the longer interval.
- Data is paginated through the WHOOP API using the `whoop-go` SDK's `NextPage()` iterator, terminating on `whoop.ErrNoNextPage`.

**Webhook Mode** (`-mode webhook`):
- Uses the **Inbox Pattern** for zero-data-loss guarantees.
- `internal/webhook/handler.go`: Validates HMAC signature via `whoop.ParseWebhook()`, marshals to JSON, inserts into `webhook_events` (status: `pending`), returns `200 OK` immediately.
- `internal/webhook/worker.go`: Background worker polls `webhook_events` every 5 seconds, fetches up to 50 pending events per batch, fetches the full object from the WHOOP API (with rate limiting at 2 req/500ms), upserts into hypertables, and batch-updates statuses to `processed` or `failed`.
- Handles event types: `recovery.updated`, `cycle.updated`, `workout.updated`, `sleep.updated`. Unknown types are logged and marked `processed`.

**Note:** In both modes, the Poller is instantiated — in webhook mode it's only used for ad-hoc `/sync` triggers from the UI.

### Concurrency Model
- **Sync Endpoint** (`POST /api/v1/sync`): Uses an **in-memory `sync.Mutex`** protecting two maps:
  - `activeSyncs map[string]bool` — prevents concurrent syncs per WHOOP user ID (returns `409 Conflict`).
  - `syncLimiters map[string]*rate.Limiter` — enforces 1 sync per 5 minutes per user (returns `429 Too Many Requests`).
  - The sync job runs in a background goroutine with a 10-minute context timeout.
  - **Lock ordering**: rate limit check → active sync check → release lock → spawn goroutine.
- **Poller**: Each poll loop is a goroutine managed by a `sync.WaitGroup`. The shared `rate.Limiter` prevents API abuse.
- **Auth Manager**: Uses `sync.Map` for client cache + per-user `sync.Mutex` for token refresh (double-checked locking pattern). Clients are cached for 55 minutes via `time.AfterFunc` eviction.
- **HTTP Server**: Graceful shutdown with `signal.Notify(quit, os.Interrupt, syscall.SIGTERM)`, 10-second shutdown deadline, and `sync.WaitGroup` drain for background workers.

## 3. Data Models & Database Schema

### Core Tables
| Table | Description |
|-------|-------------|
| `users` | User accounts with AES-256-GCM encrypted OAuth2 tokens (`BYTEA`). PK: `UUID` (auto-generated). Unique on `whoop_user_id`. |
| `user_profiles` | WHOOP profile data (email, first_name, last_name). FK to `users(id)` with `ON DELETE CASCADE`. |
| `body_measurements` | Height (meters), weight (kg), max HR. FK to `users(id)` with `ON DELETE CASCADE`. |
| `webhook_events` | Inbox for raw webhook payloads. Fields: `payload JSONB`, `status VARCHAR(50)`, `retry_count INT`, `processed_at TIMESTAMPTZ`. |

### TimescaleDB Hypertables
All time-series tables use **composite primary keys** `(id, start_time)` — required by TimescaleDB for chunk-level uniqueness.
| Hypertable | ID Type | Partition Column | Key Data |
|-----------|---------|-----------------|----------|
| `cycles` | `BIGINT` | `start_time` | Strain, kilojoule, avg/max HR, score_state |
| `recoveries` | `BIGINT` | `start_time` | Recovery score, RHR, HRV (rmssd_milli), SpO2, skin temp (celsius), user_calibrating |
| `sleeps` | `TEXT` | `start_time` | Performance, efficiency, respiratory rate, consistency, stage breakdowns (light/REM/deep/awake), sleep need/debt, cycle_id, disturbance_count |
| `workouts` | `TEXT` | `start_time` | Strain, HR zones (0-5 in milliseconds), distance, altitude gain/change, sport_id, sport_name, percent_recorded |

### Continuous Aggregates (5 materialized views)
Auto-refreshed by TimescaleDB policies (every 1 hour, looking back 3 days for daily, 1 month for weekly):
- `daily_strain` — avg/max strain per day from `cycles`
- `weekly_strain` — avg/max strain per week from `cycles`
- `daily_recovery` — avg recovery score per day from `recoveries`
- `weekly_recovery` — avg recovery score per week from `recoveries`
- `daily_sleep` — avg performance/efficiency per day from `sleeps` (naps excluded via `WHERE nap = false`)

### Performance Indexes
- `idx_cycles_user_start` — `(user_id, start_time DESC)` on `cycles`
- `idx_sleeps_user_start` — `(user_id, start_time DESC)` on `sleeps`
- `idx_workouts_user_start` — `(user_id, start_time DESC)` on `workouts`
- `idx_recoveries_user_start` — `(user_id, start_time DESC)` on `recoveries`
- `idx_webhook_status_created` — `(status, created_at ASC)` on `webhook_events`

### Schema Change Process
1. Write raw SQL migrations in `migrations/` (numbered: `000001_init_schema.up.sql`).
2. Run `sqlc generate` — generates type-safe Go code into `internal/db/` using pgx/v5.
3. **Never edit auto-generated files in `internal/db/`** — they are overwritten by sqlc.
4. **Exception**: `internal/db/batch.go` is a **hand-written** file that exposes sqlc-generated SQL strings as public constants (e.g., `UpsertCycleSQL`, `UpsertRecoverySQL`, `UpsertSleepSQL`, `UpsertWorkoutSQL`) for use with `pgx.Batch{}` in the storage layer.

## 4. API Contracts

### Authentication
All `/api/v1/*` endpoints require a JWT Bearer token (HS256) with a `whoop_user_id` string claim. On the frontend, JWTs are generated server-side by `web/src/lib/api/client.ts` using `jose`, signed with `WHOOP_STATS_ENCRYPTION_KEY`, with 24-hour expiration. Tokens are cached in a module-level variable for ~23 hours.

### Endpoints (from `internal/api/server.go`)
| Method | Path | Handler | Auth | Description |
|--------|------|---------|------|-------------|
| GET | `/healthz` | inline | No | DB ping health check (5s timeout) |
| GET | `/swagger/*` | http-swagger | No | Swagger UI |
| GET | `/api/v1/user/profile` | `GetProfile` | Yes | Fetch WHOOP profile via SDK (live API call) |
| GET | `/api/v1/cycles` | `GetCycles` | Yes | Cursor-paginated cycles from DB |
| GET | `/api/v1/sleeps` | `GetSleeps` | Yes | Cursor-paginated sleeps from DB |
| GET | `/api/v1/workouts` | `GetWorkouts` | Yes | Cursor-paginated workouts from DB |
| GET | `/api/v1/recoveries` | `GetRecoveries` | Yes | Cursor-paginated recoveries from DB |
| GET | `/api/v1/insights` | `GetInsights` | Yes | 30-day strain + recovery from continuous aggregates |
| POST | `/api/v1/sync` | `PostSync` | Yes | Trigger ad-hoc sync (202 Accepted) |
| POST | `/webhook` | webhook.Handler | No | Webhook inbox (webhook mode only, HMAC-validated) |

### Pagination
All list endpoints use **cursor-based pagination** via `RFC3339Nano` timestamp `cursor` query parameter + `limit` (default 50, max 200). When no cursor is provided, the current time (`time.Now()`) is used. **Never use OFFSET/LIMIT.**

### Error Response Format
```json
{ "error": { "code": "AUTH_ERROR", "message": "Invalid user" } }
```
Error codes: `AUTH_ERROR`, `DB_ERROR`, `API_ERROR`, `CONFLICT`, `RATE_LIMIT_EXCEEDED`, `INVALID_CURSOR`.

## 5. Security Architecture

### Token Lifecycle
1. **Bootstrap**: `cmd/auth/main.go` runs an OAuth2 Authorization Code flow via a local HTTP server (default port 8081). Saves tokens to `.whoop_token.json`. After successful token exchange, the CLI **auto-detects** the user's WHOOP User ID by calling `GET /developer/v1/user/profile/basic` and writes it to `.env` if the file exists. **Note:** this CLI uses `WHOOP_CLIENT_ID` and `WHOOP_CLIENT_SECRET` env vars (no `WHOOP_STATS_` prefix) and the standard `log` package.
2. **First Run**: `internal/auth/manager.go` reads `.whoop_token.json` as offline fallback when the user isn't found in the DB. It encrypts the tokens with AES-256-GCM and upserts into the `users` table.
3. **Steady State**: On each API call, the auth manager loads encrypted tokens from DB, decrypts in-memory, refreshes via WHOOP token endpoint (`https://api.prod.whoop.com/oauth/oauth2/token`), re-encrypts and persists the new tokens.
4. **Cache**: Authenticated `whoop.Client` instances are cached in a `sync.Map` with 55-minute TTL (via `time.AfterFunc`; tokens expire in 1 hour).
5. **JSON Sync**: After every refresh, `.whoop_token.json` is updated (best-effort, fails silently if mounted read-only) to prevent stale fallback after DB wipes. This is critical because WHOOP refresh tokens are **single-use** — once consumed, the old token is dead.

### OAuth Scopes (from `cmd/auth/main.go`)
`offline`, `read:recovery`, `read:cycles`, `read:workout`, `read:sleep`, `read:profile`, `read:body_measurement`

### Encryption
- `internal/crypto/aes.go`: AES-256-GCM with random nonce per encryption (12-byte nonce prepended to ciphertext). Key must be exactly 32 bytes.
- NEVER log, print, or commit access/refresh tokens or `ENCRYPTION_KEY`.

### JWT Authentication
- `internal/middleware/auth.go`: Validates HS256 JWT, explicitly rejects non-HMAC algorithms to prevent algorithm confusion attacks.
- Extracts `whoop_user_id` string claim and injects into request context via `WhoopUserIDKey`.
- Frontend generates JWTs server-side (never exposed to browser). The JWT secret is the same `WHOOP_STATS_ENCRYPTION_KEY` used for token encryption.

### Rate Limiting
- **IP-based rate limiter** (global middleware): Configurable rate + burst (default: 20 req/s, burst 50). Per-IP `rate.Limiter` instances stored in a `sync.Mutex`-protected map. Stale visitor entries (unseen for 30+ minutes) cleaned up every 10 minutes by a background goroutine.
- **Per-user sync rate limiter**: 1 sync per 5 minutes per WHOOP user ID.
- **WHOOP API rate limiter**: 2 req/500ms shared `rate.Limiter` in both poller and webhook worker.

### Docker Security
- Dockerfiles drop to non-root users (`appuser` in backend, `nextjs` UID 1001 in frontend).
- `tmpfs` mounts for `/tmp` and `/var/log` in both dev and prod compose to minimize SSD write amplification.
- `local` logging driver with 10MB max size on all containers.
- `.whoop_token.json` mounted read-only (`:ro`) in production compose.
- Frontend uses Next.js `output: "standalone"` for minimal Docker image size.

## 6. Directory Structure

```
whoop-stats/
├── cmd/
│   ├── server/main.go              # Main server entrypoint (-mode poll|webhook)
│   └── auth/main.go                # OAuth2 token bootstrap CLI (unprefixed env vars, auto-detects User ID)
├── internal/
│   ├── api/                        # HTTP handlers + chi router
│   │   ├── server.go               # Router setup, middleware wiring, route registration
│   │   ├── handlers.go             # All endpoint handlers + sync concurrency control
│   │   └── handlers_test.go        # API handler unit tests
│   ├── auth/                       # OAuth2 token lifecycle, client cache, offline fallback
│   │   └── manager.go              # GetClient, token refresh, JSON sync, cache eviction
│   ├── config/                     # Viper config loader (WHOOP_STATS_ prefix)
│   │   ├── config.go               # Config struct + LoadConfig() with validation
│   │   └── config_test.go          # Config validation tests
│   ├── crypto/                     # AES-256-GCM encrypt/decrypt
│   │   ├── aes.go                  # Encrypt() and Decrypt() with random nonce
│   │   └── aes_test.go             # Round-trip, invalid key, and tamper detection tests
│   ├── db/                         # ⚠️ AUTO-GENERATED by sqlc — never edit!
│   │   ├── db.go                   # Generated: DBTX interface + New()
│   │   ├── models.go               # Generated: Go structs for all tables/views
│   │   ├── querier.go              # Generated: Querier interface
│   │   ├── query.sql.go            # Generated: All query implementations
│   │   └── batch.go                # ✋ HAND-WRITTEN: exports SQL constants for pgx.Batch
│   ├── middleware/                  # HTTP middleware
│   │   ├── auth.go                 # JWT HS256 validation + context injection
│   │   ├── auth_test.go            # JWT validation, algorithm confusion prevention
│   │   ├── logging.go              # Structured slog request logger (skips /healthz)
│   │   ├── ratelimit.go            # IP-based rate limiter with stale cleanup
│   │   └── ratelimit_test.go       # Rate limiter behavior tests
│   ├── poller/                     # Periodic WHOOP API scraper (4 poll loops)
│   │   ├── manager.go              # Start(), pollLoop(), RunAdHocSync(), adaptive sleep
│   │   └── manager_test.go         # Off-peak adaptive polling logic tests
│   ├── storage/                    # Domain mapper: whoop-go types → sqlc params (upserts)
│   │   ├── storage.go              # UpsertX and batch UpsertXs for all data types
│   │   ├── storage_test.go         # Storage layer tests
│   │   ├── timezone.go             # ParseTimezoneOffset (WHOOP format → pgtype.Interval)
│   │   └── timezone_test.go        # Timezone offset parsing edge cases
│   ├── webhook/                    # Inbox handler + background worker
│   │   ├── handler.go              # ServeHTTP: HMAC validate → JSON marshal → insert → 200 OK
│   │   ├── worker.go               # Background poller: pending events → fetch full object → upsert
│   │   └── worker_bench_test.go    # Webhook worker benchmarks
│   └── worker/                     # Empty package (reserved)
├── migrations/
│   ├── 000001_init_schema.up.sql   # Full schema: tables, hypertables, indexes, aggregates
│   └── 000001_init_schema.down.sql # Teardown: drops all objects
├── queries/query.sql               # sqlc query definitions (all CRUD + aggregates)
├── sqlc.yaml                       # sqlc config (pgx/v5, output: internal/db)
├── docs/
│   ├── swagger.json                # Generated Swagger spec
│   ├── swagger.yaml                # Generated Swagger spec (YAML)
│   ├── openapi.json                # OpenAPI spec for frontend type generation
│   ├── docs.go                     # Generated Swagger docs package
│   ├── archive/                    # Historical documentation
│   ├── designs/                    # Design artifacts
│   ├── explorations/               # Feature explorations
│   └── plans/                      # Planning documents
├── web/                            # Next.js 16 frontend
│   ├── next.config.ts              # output: "standalone" for Docker
│   ├── src/app/                    # App Router pages + Server Actions
│   │   ├── page.tsx                # Dashboard (Overview) — RSC with parallel fetching
│   │   ├── recovery/page.tsx       # Recovery detail page
│   │   ├── sleep/page.tsx          # Sleep detail page
│   │   ├── strain/page.tsx         # Strain detail page
│   │   ├── workouts/page.tsx       # Workouts detail page
│   │   ├── actions.ts              # Server Actions (syncWhoopData)
│   │   ├── layout.tsx              # Root layout (Inter font, Sidebar, MobileNav, Toaster)
│   │   ├── globals.css             # Design tokens, glass-card utilities, shadcn compat
│   │   ├── error.tsx               # Global error boundary with retry button
│   │   └── loading.tsx             # Skeleton loading state matching dashboard layout
│   ├── src/components/             # UI components
│   │   ├── SyncButton.tsx          # Client component: triggers Server Action
│   │   ├── sidebar.tsx             # Desktop sidebar navigation
│   │   ├── mobile-nav.tsx          # Mobile bottom navigation bar
│   │   ├── metric-card.tsx         # Hero metric card (strain, recovery, sleep)
│   │   ├── strain-recovery-chart.tsx # 30-day strain vs recovery trend chart
│   │   ├── trend-chart.tsx         # Generic trend chart component
│   │   ├── recovery-gauge.tsx      # Recovery score gauge visualization
│   │   ├── recovery-panels.tsx     # Recovery detail panels
│   │   ├── sleep-panels.tsx        # Sleep detail panels
│   │   ├── sleep-stages-bar.tsx    # Sleep stage breakdown bar
│   │   ├── strain-panels.tsx       # Strain detail panels
│   │   ├── workout-card.tsx        # Individual workout card
│   │   ├── workout-detail.tsx      # Workout detail popup
│   │   ├── workout-feed.tsx        # Workout feed with filtering
│   │   ├── recent-workouts.tsx     # Recent workouts summary
│   │   ├── detail-popup.tsx        # Detail popup overlay
│   │   └── ui/                     # shadcn primitives
│   │       ├── button.tsx          # Button variants (cva)
│   │       ├── card.tsx            # Card components
│   │       ├── badge.tsx           # Badge component
│   │       ├── skeleton.tsx        # Skeleton loading component
│   │       ├── chart.tsx           # Chart wrapper (Recharts integration)
│   │       ├── chart.test.ts       # Chart component tests
│   │       └── sonner.tsx          # Sonner toast wrapper
│   └── src/lib/
│       ├── api/client.ts           # openapi-fetch client with JWT injection
│       ├── api/schema.d.ts         # Auto-generated TypeScript types from OpenAPI
│       ├── format.ts               # Display formatters (duration, dates, calories, recovery colors, HR zones)
│       ├── stats.ts                # Statistical helpers (avg, stddev, percentChange)
│       ├── stats.test.ts           # Stats utility tests
│       ├── types.ts                # Shared types (ApiRecord)
│       └── utils.ts                # Utility functions (cn via clsx + tailwind-merge)
├── .github/workflows/
│   ├── ci.yml                      # CI: Go build/vet/test + Next.js lint/build
│   └── publish.yml                 # Docker image publishing to GHCR (backend + frontend)
├── .agent/                         # AI agent context documentation
├── Dockerfile.backend              # Multi-stage Go build → alpine (non-root appuser)
├── web/Dockerfile                  # Multi-stage Next.js standalone build (non-root nextjs)
├── docker-compose.yml              # Dev compose (bind mounts, exposed ports)
├── docker-compose.prod.yml         # Prod compose (named volumes, networks, read-only mounts)
├── setup.sh                        # Interactive setup wizard (auto-generates secrets, OAuth, User ID)
├── GEMINI.md                       # AI system rules (root level)
├── README.md                       # Project README with architecture diagrams
├── .env.example                    # Documented environment variable template
├── .gitignore                      # Ignores .env, .whoop_token.json, web/.next, bin/
└── .swaggo                         # Swagger generation config
```

## 7. Environment Variables

All **backend** env vars use the `WHOOP_STATS_` prefix (via Viper). The `cmd/auth/` CLI uses **unprefixed** vars (`WHOOP_CLIENT_ID`, `WHOOP_CLIENT_SECRET`).

### Backend (WHOOP_STATS_ prefix)
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

### Docker / Frontend
| Variable | Context | Default | Description |
|----------|---------|---------|-------------|
| `POSTGRES_USER` | Docker | `whoop_user` | Postgres username |
| `POSTGRES_PASSWORD` | Docker | — | Postgres password |
| `WHOOP_USER_ID` | CLI flag `-user` | `12345` | Your WHOOP user ID |
| `BACKEND_PORT` | Docker | `8085` | Host-mapped backend port |
| `FRONTEND_PORT` | Docker | `3032` | Host-mapped frontend port |
| `NEXT_PUBLIC_API_URL` | Frontend | `http://localhost:8080` | Backend URL for API calls |
| `WHOOP_STATS_ENCRYPTION_KEY` | Frontend | — | Shared key for JWT signing |
| `WHOOP_STATS_WHOOP_USER_ID` | Frontend | — | WHOOP user ID for JWT generation |

### Auth CLI (cmd/auth/, NO prefix)
| Variable | Required | Description |
|----------|----------|-------------|
| `WHOOP_CLIENT_ID` | Yes | WHOOP OAuth client ID |
| `WHOOP_CLIENT_SECRET` | Yes | WHOOP OAuth client secret |
| `WHOOP_REDIRECT_URI` | No | Default: `http://localhost:8081/callback` |

## 8. Error Handling Patterns
- **Go Backend**: Explicit `if err != nil` with `fmt.Errorf("context: %w", err)` wrapping. HTTP errors use the `sendError()` helper with structured `ErrorResponse` JSON envelopes with codes like `AUTH_ERROR`, `DB_ERROR`, `API_ERROR`, `CONFLICT`, `RATE_LIMIT_EXCEEDED`.
- **Sync endpoint**: Returns `409 Conflict` for concurrent syncs, `429 Too Many Requests` for rate-limited syncs, `202 Accepted` on success.
- **Webhook worker**: Failed events are batch-marked `failed`. Processed events batch-marked `processed`. Unknown event types are logged and marked processed.
- **Frontend**: `error.tsx` provides a global error boundary with a glass-card error message and retry button. Server Actions throw errors surfaced via Sonner toast notifications. `loading.tsx` provides skeleton UI during streaming.

## 9. Key Dependencies

### Backend (Go)
| Package | Version | Role |
|---------|---------|------|
| `arvarik/whoop-go` | v1.1.0 | First-party WHOOP API SDK (cycles, sleeps, workouts, recoveries, user) |
| `go-chi/chi/v5` | v5.2.5 | HTTP router |
| `go-chi/cors` | v1.2.2 | CORS middleware |
| `jackc/pgx/v5` | v5.8.0 | PostgreSQL driver (connection pooling via `pgxpool`) |
| `spf13/viper` | v1.21.0 | Configuration management |
| `golang-jwt/jwt/v5` | v5.3.1 | JWT parsing/validation (backend) |
| `swaggo/swag` | v1.16.6 | Swagger spec generation from Go annotations |
| `swaggo/http-swagger/v2` | v2.0.2 | Swagger UI handler |
| `testcontainers-go` | v0.40.0 | Dockerized Postgres for integration tests |
| `stretchr/testify` | v1.11.1 | Test assertions and suites |
| `golang.org/x/time/rate` | v0.14.0 | Rate limiting (API + poller + webhook worker) |

### Frontend (Node.js)
| Package | Version | Role |
|---------|---------|------|
| `next` | 16.1.6 | React framework (App Router, RSC, Server Actions) |
| `react` / `react-dom` | 19.2.3 | UI runtime |
| `jose` | v6.2.1 | JWT generation (frontend, server-side only) |
| `openapi-fetch` | v0.17.0 | Type-safe API client |
| `openapi-typescript` | v7.13.0 | OpenAPI → TypeScript type generation (dev) |
| `recharts` | v2.15.4 | All charts (strain/recovery trends, HR zones) |
| `framer-motion` | v12.35.0 | Page transitions, micro-animations |
| `shadcn` | v4.0.0 | Component primitives (Button, Card, Badge, Skeleton, Sonner) |
| `@base-ui/react` | v1.2.0 | Headless UI primitives |
| `@tremor/react` | v3.18.7 | Data visualization components |
| `@number-flow/react` | v0.6.0 | Animated number transitions |
| `lucide-react` | v0.577.0 | Icon library |
| `date-fns` | v4.1.0 | Date formatting |
| `sonner` | v2.0.7 | Toast notifications |
| `next-themes` | v0.4.6 | Theme management (installed, dark mode hardcoded via `className="dark"`) |
| `class-variance-authority` | v0.7.1 | Component variant composition |
| `clsx` | v2.1.1 | Conditional class name utility |
| `tailwind-merge` | v3.5.0 | Tailwind class deduplication |
| `tw-animate-css` | v1.4.0 | CSS animations for shadcn |

## 10. CI/CD Pipelines

### CI Pipeline (`ci.yml`)
Runs on pushes and PRs to `main`. Two jobs:
- **Backend** (Go 1.25, ubuntu-latest): build → vet → unit tests (crypto, middleware, config, timezone, poller)
- **Frontend** (Node.js 20, ubuntu-latest): npm ci → lint (ESLint 9) → build (`NEXT_TELEMETRY_DISABLED=1`)

> **Note:** CI Go version is aligned with `go.mod` at 1.25.

### Publish Pipeline (`publish.yml`)
Runs on pushes to `main` and version tags (`v*`). Publishes Docker images to GHCR:
- **Backend**: `ghcr.io/${{ github.repository }}/backend` from `Dockerfile.backend`
- **Frontend**: `ghcr.io/${{ github.repository }}/frontend` from `web/Dockerfile`
- Tags: SHA, semver, `latest` (for default branch)
- Uses Docker Buildx with GitHub Actions cache (`type=gha`)
