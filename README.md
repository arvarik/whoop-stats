# WHOOP Stats

A premium, high-performance, open-source dashboard and ingestion engine for your WHOOP fitness data. Built for homelabs, NAS devices, and cloud deployments.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg)
![Next.js](https://img.shields.io/badge/Next.js-16+-black.svg)
![TimescaleDB](https://img.shields.io/badge/TimescaleDB-15+-FDB515.svg)

<img width="2433" height="1878" alt="image" src="https://github.com/user-attachments/assets/11e453dc-4059-495f-a84c-0690675f4609" />


---

## Architecture

This project supports **two mutually exclusive data ingestion modes**, letting it run in any environment.

### Polling Engine (Homelabs / NAS)
For users behind NAT/firewalls without a public domain. The backend makes outbound requests to WHOOP on configurable intervals — **zero open inbound ports required**. Uses cursor-based pagination to sync your entire history automatically.

### Webhook Inbox Pattern (Cloud)
For public-facing cloud instances. WHOOP pushes events in real-time. We use a store-then-process inbox pattern: acknowledge instantly, persist to a queue, and process asynchronously — ensuring 100% data integrity even when the API is slow.

> See [design.md](design.md) for the full architecture diagram and detailed technical justifications.

---

## Features

- **Complete Data Coverage** — Cycles, sleep stages, recoveries (SpO2, HRV, skin temp), workouts (HR zones), user profiles, and body measurements
- **Continuous Aggregates** — Pre-computed TimescaleDB views for daily/weekly strain, recovery, and sleep metrics. Auto-refreshed hourly.
- **Linear-Inspired Dashboard** — Dark glassmorphism UI with interactive Recharts visualizations, skeleton loading states, and error recovery
- **End-to-End Type Safety** — `sqlc` (Go ↔ SQL) + `openapi-typescript` (Go ↔ TypeScript)
- **Security Hardened** — AES-256-GCM token encryption, JWT with HS256 enforcement, per-IP rate limiting, non-root containers
- **SSD Wear Protection** — RAM-backed logs (`tmpfs`), dynamic log levels, compressed binary logging
- **Tested** — 33+ unit tests covering crypto, auth, rate limiting, config validation, and timezone parsing. Integration tests with testcontainers for database upserts.

---

## Getting Started

### 1. Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| Docker + Docker Compose | v2+ | Required for all deployments |
| Go | 1.22+ | One-time use for the auth script |
| WHOOP Developer Account | — | [Register here](https://developer.whoop.com) |

### 2. Clone and Configure

```bash
git clone https://github.com/arvarik/whoop-stats.git
cd whoop-stats
cp .env.example .env
```

Edit `.env` and fill in the **required** values:

```env
# [REQUIRED] Database password (change from default)
POSTGRES_PASSWORD=your_secure_password

# [REQUIRED] AES-256 encryption key (exactly 32 chars)
# Generate: openssl rand -hex 16
ENCRYPTION_KEY=

# [REQUIRED] From https://developer.whoop.com
WHOOP_CLIENT_ID=
WHOOP_CLIENT_SECRET=

# [REQUIRED] Your WHOOP user ID
WHOOP_USER_ID=
```

### 3. First-Time Authentication

Since this app can run without a public URL, we use a one-time local script to generate your OAuth tokens:

1. **Configure Redirect URI:** In your [WHOOP Developer Dashboard](https://developer.whoop.com), add `http://localhost:8081/callback` to your App's Redirect URIs.

2. **Generate Token:**
   ```bash
   export WHOOP_CLIENT_ID=your_id
   export WHOOP_CLIENT_SECRET=your_secret
   go run cmd/auth/main.go
   ```

3. **Authorize:** Open the URL printed in your terminal, log in to WHOOP, and authorize the app.

4. **Save:** The script generates `.whoop_token.json` with restricted permissions (`0600`).

5. **Deploy:** If deploying to a NAS or remote server, upload `.whoop_token.json` to the project root on that server.

### 4. Deploy

#### Option A: Homelab / NAS (Polling Mode)

Database migrations run automatically on first startup.

```bash
docker compose up -d --build
```

The dashboard will be available at `http://your-server:3032` and the API at `http://your-server:8082`.

#### Option B: Cloud (Webhook Mode)

1. Ensure your server is accessible via HTTPS.
2. Configure your WHOOP Webhook URL: `https://your-domain.com/webhook`.
3. Set `WHOOP_WEBHOOK_SECRET` in `.env`.
4. Start:
   ```bash
   docker-compose -f docker-compose.prod.yml up -d --build
   ```

---

## User Guide

### Dashboard Overview

The dashboard has **five main sections**, accessible from the sidebar (desktop) or bottom nav (mobile):

| Tab | What it shows |
|-----|---------------|
| **Overview** | Today's strain, recovery score, sleep performance, 7-day recovery strip, 30-day strain/recovery trend chart, and recent workouts |
| **Recovery** | Recovery gauge, HRV, resting heart rate, SpO2, skin temperature trends, recovery distribution, and 14-day history |
| **Sleep** | Sleep performance, efficiency, duration, stage breakdowns (light/REM/deep/awake), sleep debt, respiratory rate, and consistency |
| **Strain** | Daily strain score, calorie burn (converted from kJ), HR zones, sport breakdown, peak strain days, and workout statistics |
| **Workouts** | Full workout feed with sport type, duration, strain, calories, average/max HR, distance, and HR zone breakdowns |

### Syncing Data

- **Automatic (Polling Mode):** Data syncs on configurable intervals — defaults: cycles every 4h, workouts every 30m, sleep every 1h, profile daily.
- **Manual:** Click the **Sync** button on the Overview page. This triggers an ad-hoc API call and refreshes all dashboard routes.

### Configuring Poll Intervals

Adjust these in `.env` using [Go duration format](https://pkg.go.dev/time#ParseDuration):

```env
POLL_INTERVAL_CYCLE=4h          # How often to fetch physiological cycles
POLL_INTERVAL_WORKOUT=30m       # How often to fetch workouts
POLL_INTERVAL_SLEEP=1h          # How often to fetch sleep data
POLL_INTERVAL_SLEEP_OFFPEAK=4h  # Sleep polling outside 6 AM–12 PM
POLL_INTERVAL_PROFILE=24h       # User profile data (rarely changes)
```

### Changing Ports

Default ports are `8082` (backend) and `3032` (frontend). Change them in `.env`:

```env
BACKEND_PORT=9090
FRONTEND_PORT=3000
```

Then update `NEXT_PUBLIC_API_URL` in `docker-compose.yml` if you changed the backend port.

### Understanding Metrics

| Metric | Description | Source |
|--------|-------------|--------|
| **Strain** | 0–21 scale of daily cardiovascular load | WHOOP cycles |
| **Recovery** | 0–100% readiness score (green ≥66%, yellow ≥34%, red <34%) | WHOOP recovery |
| **HRV (RMSSD)** | Heart rate variability in ms — higher is better | WHOOP recovery |
| **Sleep Performance** | Percentage of sleep need achieved | WHOOP sleep |
| **Sleep Efficiency** | Time asleep / time in bed (%) | WHOOP sleep |
| **Calories** | Total energy expenditure, converted from kilojoules (kJ × 0.239) | WHOOP cycles |

### Running Tests

```bash
# Unit tests (no Docker required)
go test ./internal/crypto/... ./internal/middleware/... ./internal/config/... -v

# Timezone parser tests
go test ./internal/storage/ -run TestParseTimezoneOffset -v

# Integration tests (requires Docker for testcontainers)
go test ./internal/storage/ -v

# All tests
go test ./...
```

---

## Security

- **AES-256-GCM Encryption** — OAuth tokens encrypted at rest with a user-provided key. Database dumps cannot compromise API access.
- **JWT HS256 Enforcement** — Auth middleware rejects tokens using any algorithm other than HS256, preventing algorithm confusion attacks.
- **Per-IP Rate Limiting** — 20 req/s with burst of 50. Stale entries cleaned up automatically to prevent memory leaks.
- **Non-Root Containers** — Both backend and frontend processes run as unprivileged users.
- **Token File Permissions** — `.whoop_token.json` created with `0600` (owner read/write only).
- **No Hardcoded Secrets** — All secrets fail fast at startup if missing, with clear error messages.

---

## Troubleshooting

### Backend won't start: "ENCRYPTION_KEY is required"
You must set `WHOOP_STATS_ENCRYPTION_KEY` (exactly 32 characters) in `.env`. Generate one:
```bash
openssl rand -hex 16
```

### Backend won't start: "WHOOP_CLIENT_ID is required"
Register at [developer.whoop.com](https://developer.whoop.com) and set `WHOOP_CLIENT_ID` and `WHOOP_CLIENT_SECRET` in `.env`.

### Frontend shows "Something went wrong"
This means the frontend can't reach the backend API. Check:
1. Is the backend container running? `docker compose ps`
2. Is `NEXT_PUBLIC_API_URL` pointing to the correct backend address/port?
3. Check backend logs: `docker compose logs backend`

### OAuth flow fails: "invalid_client"
Your `WHOOP_CLIENT_ID` and `WHOOP_CLIENT_SECRET` may be incorrect. Double-check them in your [WHOOP Developer Dashboard](https://developer.whoop.com). Make sure `http://localhost:8081/callback` is listed as a redirect URI.

### Data not appearing on dashboard
1. **First deploy?** It takes a few minutes for the initial poll to complete. Check backend logs: `docker compose logs -f backend`
2. **Check polling logs** for API errors: `docker compose logs backend | grep ERROR`
3. **Try manual sync:** Click the "Sync" button on the Overview page.
4. **Token expired?** Re-run `go run cmd/auth/main.go` to generate a fresh `.whoop_token.json`.

### Database connection errors
1. Ensure the TimescaleDB container is healthy: `docker compose ps`
2. Check if `POSTGRES_PASSWORD` in `.env` matches what the database was initialized with. If you change the password after first run, you'll need to delete the volume: `docker compose down -v && docker compose up -d`
3. Verify the database URL in the backend logs.

### Port conflicts
If you see `bind: address already in use`, another service is using the same port. Change `BACKEND_PORT` or `FRONTEND_PORT` in `.env`:
```env
BACKEND_PORT=9090
FRONTEND_PORT=4000
```

### Dashboard shows stale/old data
Continuous aggregates refresh hourly with a 3-day lookback. For immediate results:
1. Click "Sync" on the Overview page
2. Check that your backend is running in poll mode and the intervals are reasonable

### Docker build is slow
Add a `.dockerignore` file (included) to exclude `node_modules` and `.next` from the build context. For the backend, `.dockerignore` at the project root excludes `web/` and `.git/`.

### Running on ARM (Raspberry Pi / Apple Silicon)
Both the Go backend and Next.js frontend build on ARM64 via Docker's multi-platform support. The TimescaleDB image (`timescale/timescaledb:latest-pg15`) supports ARM64 natively.

---

## Tech Stack

| Layer | Technology | Role |
|-------|-----------|------|
| **Database** | PostgreSQL 15 + TimescaleDB | Time-series storage with hypertables and continuous aggregates |
| **Backend** | Go 1.22+, go-chi, sqlc | REST API, dual-mode ingestion, type-safe DB queries |
| **Frontend** | Next.js 16, React 19, Tailwind CSS v4 | Server-rendered dashboard with glassmorphism UI |
| **Charts** | Recharts, Framer Motion | Interactive data visualization and animations |
| **Auth** | JWT (HS256), AES-256-GCM | API authentication and token encryption |
| **DevOps** | Docker, Docker Compose | Container orchestration with SSD-optimized logging |

---

## Project Structure

```
whoop-stats/
├── cmd/
│   ├── auth/          # One-time OAuth token generator
│   └── server/        # Main backend entrypoint (poll + webhook modes)
├── internal/
│   ├── api/           # HTTP handlers and router setup
│   ├── auth/          # OAuth2 token management
│   ├── config/        # Environment configuration (viper)
│   ├── crypto/        # AES-256-GCM encryption
│   ├── db/            # Generated sqlc code (DO NOT EDIT)
│   ├── middleware/     # Auth, rate limiting, logging
│   ├── poller/        # WHOOP API polling engine
│   ├── storage/       # Database abstraction layer
│   └── webhook/       # Webhook handler and background worker
├── migrations/        # SQL schema migrations
├── queries/           # sqlc query definitions
├── web/               # Next.js frontend
│   └── src/
│       ├── app/       # Pages (overview, recovery, sleep, strain, workouts)
│       ├── components/# UI components
│       └── lib/       # API client, utilities, formatting helpers
├── docker-compose.yml # Development / homelab deployment
├── design.md          # Architecture and design decisions
└── .env.example       # Configuration template
```

---

## License

MIT
