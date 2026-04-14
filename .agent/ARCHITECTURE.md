# Architecture

_This document acts as the definitive anchor for understanding system design, data models, API contracts, and technology boundaries. Update this document during the Design and Review phases._

## 1. Tech Stack & Infrastructure
- **Language / Runtime**: Go 1.25.0 / Node.js
- **Frontend**: Next.js 16 (App Router)
- **Backend / API**: Go (go-chi HTTP router)
- **Database**: PostgreSQL 15 + TimescaleDB via sqlc
- **Deployment**: Docker Compose (Self-hosted)
- **Package Management**: npm / Go modules (`go.mod`)

## 2. System Boundaries & Data Flow
### Request / Data Flow
- **Frontend flow**: Next.js Dashboard performs Server-Side Data Fetching (RSC) to parallel fetch profiles, cycles, and sleeps from the API via `openapi-fetch`. The client renders these using UI tools (Shadcn UI, Recharts, Framer Motion). Ad-hoc syncs triggered by Server Actions interact with the Go backend to `POST /sync`, followed by `revalidatePath("/")`.
- **Backend flow**: HTTP request → go-chi middleware (Auth, Rate Limiting, Request IDs) → handlers → internal/service layer → sqlc DAL → PostgreSQL 15 (TimescaleDB).
- **Webhook Ingestion flow (Inbox Pattern)**: External WHOOP webhook payload → dumped to Postgres immediately (Inbox) → background worker polls and processes with exponential backoff → upserts data.
- **Polling Ingestion flow**: Configurable concurrent scraping worker runs locally, bypassing external NATs to poll WHOOP directly and upserts data.

### Concurrency / Threading Model
- **Go Backend**: Extensive use of Goroutines for concurrent polling/webhook workers. Mutex/Advisory DB Lock (`pg_advisory_xact_lock`) prevents concurrent syncs per user ID.
- **Node.js Frontend**: Single Node.js process with parallel fetching via Next.js RSCs.

## 3. Data Models & Database Schema
- **`users`**: Main entity tracking configured users.
- **`webhook_events`**: Stores raw payload data immediately upon webhook receipt for async processing (Inbox Pattern).
- **Time-Series (TimescaleDB)**:
  - **Hypertables**: Used for `cycles`, `sleeps`, `workouts`, `recoveries` (automatically partitioned across time intervals).
  - **Continuous Aggregates**: Materialized views (e.g., `daily_strain`, `daily_recovery`) incrementally pre-calculate data for instantaneous O(1) queries.

### Schema Change Process
- Write raw SQL migrations in `migrations/`. Run `sqlc generate` via `sqlc.yaml` to regenerate type-safe Go DAL in `queries/`.

## 4. API Contracts
- Frontend API Client (`openapi-fetch`) is strongly typed against the generated `schema.d.ts` from Go backend's Swagger spec (`docs/swagger.json`).
- Core Endpoints:
  - `POST /sync`: Called by Server Actions to initiate an ad-hoc sync job with WHOOP API.

## 5. External Integrations / AI
- **WHOOP Developer API**: Used for ingesting sleep, strain, cycle, and recovery data either via webhook or direct polling.
- **Auth & Crypto Manager**: Application-level AES-256-GCM encryption manager. In-memory decryption of OAuth Access and Refresh tokens using `ENCRYPTION_KEY`. Tokens rest in Postgres as `BYTEA` ciphertext.

### Caching Strategy
- TimescaleDB continuous aggregates (materialized views) function as the primary database-level cache.
- Next.js RSC caches data, invalidated specifically via Server Actions using `revalidatePath("/")`.

## 6. Invariants & Safety Rules
- NEVER store plaintext OAuth tokens in the database. MUST be AES-256-GCM encrypted at rest as `BYTEA`.
- NEVER process webhooks synchronously. MUST use the Inbox pattern to drop raw payloads to the database, ensuring 200 OK responses to WHOOP API without timing out.
- ALWAYS enforce idempotency via `PRIMARY KEY (id, start_time)` and composite indexes (`user_id, start_time DESC`). Use keyset pagination, NEVER `OFFSET/LIMIT`. Use `INSERT ... ON CONFLICT DO UPDATE` to prevent duplicate WHOOP cycle records.
- ALWAYS engage a Mutex/Advisory Lock when user triggers a manual Sync to prevent data race conditions on `user_id`.
- NEVER use heavy client-side fetching in Next.js initial loads. Dashboard TTFB MUST be optimized by parallel pre-fetching via RSC. Hydrate heavy client libraries (`react-simple-maps`) via `next/dynamic` with `ssr: false`.

## 7. Error Handling Patterns
- Go Backend: Uses explicit `if err != nil` error handling with custom error types and structured HTTP status codes. `409 Conflict` or `429 Too Many Requests` handled gracefully during concurrent sync attempts.
- Next.js Server Actions pass errors back to the client interface for surfacing via Toast notifications or Error boundaries.

## 8. Directory Structure
- `web/src/` — Next.js UI routing, pages, server actions.
- `cmd/` — Go application entrypoints.
- `internal/` — Go private packages containing handlers, service logic, webhook processors.
- `migrations/` — Raw SQL schema migrations.
- `queries/` — sqlc-generated type-safe Go DAL.
- `docs/` — Swagger docs auto-generated from Go code, plus historical archives.
- `.agent/` — Current directory for AI rules.

## 9. Local Development
- **Database Setup**: `docker compose up db` for PostgreSQL + TimescaleDB. `docker-compose.yml` orchestrates backend+db.
- **Frontend Start**: `cd web && npm run dev`.
- **Code Generation**: `sqlc generate` for Go types. `swag init` for Swagger docs.
- **Required Environment**:
  - `DATABASE_URL`
  - `ENCRYPTION_KEY`
  - (Check `.env.example` for details)

## 10. Environment Variables
| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | Postgres connection string |
| `ENCRYPTION_KEY` | Yes | 32-byte key for AES-256-GCM token encryption |
