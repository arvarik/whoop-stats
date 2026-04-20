# WHOOP Stats

A premium, high-performance, open-source dashboard and ingestion engine for your WHOOP fitness data. Built for homelabs, NAS devices, and cloud deployments.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8.svg)
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

### Quick Start (Recommended)

The setup wizard handles everything — secret generation, WHOOP API credentials, OAuth tokens, and user ID detection:

```bash
git clone https://github.com/arvarik/whoop-stats.git
cd whoop-stats
./setup.sh
```

The wizard will:
1. ✅ Create `.env` from the template
2. ✅ Auto-generate `ENCRYPTION_KEY` and `POSTGRES_PASSWORD`
3. ✅ Ask for your WHOOP Client ID and Secret (from [developer.whoop.com](https://developer.whoop.com))
4. ✅ Run the OAuth flow and detect your WHOOP User ID automatically
5. ✅ Validate everything is ready

Then deploy:

```bash
# Homelab / NAS (recommended)
docker compose up -d --build

# Production (named volumes, networks)
docker compose -f docker-compose.prod.yml up -d --build
```

Dashboard: `http://your-server:3032` · API: `http://your-server:8085`

### Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| Docker + Docker Compose | v2+ | Required for all deployments |
| Go | 1.25+ | One-time use for the OAuth token generation |
| WHOOP Developer Account | — | [Register here](https://developer.whoop.com) |

> **Deploying to a remote server?** Run `./setup.sh` locally (needs Go + browser), then copy `.env` and `.whoop_token.json` to your server.

<details>
<summary><strong>Manual Setup (without setup.sh)</strong></summary>

#### 1. Configure Environment

```bash
cp .env.example .env
```

Fill in `.env`:

```env
# Generate: openssl rand -hex 16
ENCRYPTION_KEY=your_32_char_key_here

# From https://developer.whoop.com
WHOOP_CLIENT_ID=your_client_id
WHOOP_CLIENT_SECRET=your_client_secret

# Database password
POSTGRES_PASSWORD=your_secure_password
```

#### 2. First-Time Authentication

1. Add `http://localhost:8081/callback` to your WHOOP App's Redirect URIs in the [Developer Dashboard](https://developer.whoop.com).
2. Generate tokens:
   ```bash
   export WHOOP_CLIENT_ID=your_id
   export WHOOP_CLIENT_SECRET=your_secret
   go run cmd/auth/main.go
   ```
3. Complete the authorization in your browser. Your WHOOP User ID will be auto-detected and saved to `.env`.
4. If deploying remotely, copy `.whoop_token.json` to the server.

#### 3. Deploy

```bash
docker compose up -d --build
```

</details>

---

### Webhook Mode (Cloud)

1. Ensure your server is accessible via HTTPS.
2. Configure your WHOOP Webhook URL: `https://your-domain.com/webhook`.
3. Set `WHOOP_WEBHOOK_SECRET` in `.env`.
4. Start:
   ```bash
   docker compose -f docker-compose.prod.yml up -d --build
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

Default ports are `8085` (backend) and `3032` (frontend). Change them in `.env`:

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
Set `ENCRYPTION_KEY` (exactly 32 hex characters) in `.env`. Generate one:
```bash
openssl rand -hex 16
```

### Backend won't start: "WHOOP_CLIENT_ID is required"
Register at [developer.whoop.com](https://developer.whoop.com) and set `WHOOP_CLIENT_ID` and `WHOOP_CLIENT_SECRET` in `.env`.

### Frontend shows "Something went wrong"
The frontend can't reach the backend API:
1. Is the backend container running? `docker compose ps`
2. Is `NEXT_PUBLIC_API_URL` pointing to the correct backend address/port?
3. Check backend logs: `docker compose logs backend`

### OAuth flow fails: "invalid_client"
Double-check `WHOOP_CLIENT_ID` and `WHOOP_CLIENT_SECRET` in your [WHOOP Developer Dashboard](https://developer.whoop.com). Ensure `http://localhost:8081/callback` is added as a Redirect URI in your WHOOP app settings.

### Token refresh fails: "invalid_request" / HTTP 400
WHOOP refresh tokens are **single-use** — each refresh returns a new token and invalidates the old one. This error usually means the refresh token in `.whoop_token.json` has already been consumed.

**Fix:** Regenerate the token locally and copy to your server:
```bash
# On your local machine (needs Go + browser)
go run cmd/auth/main.go

# Copy to NAS/server
scp .whoop_token.json user@your-server:/path/to/whoop-stats/
```

Then on the server:
```bash
docker compose exec timescaledb psql -U whoop_user -d whoop_stats -c "DELETE FROM users;"
docker compose restart backend
```

> **Note:** As of v0.0.1, the backend automatically writes refreshed tokens back to `.whoop_token.json` after each successful refresh. This means DB wipes no longer invalidate your token — you only need to re-run the OAuth flow if the token has been consumed without being persisted.

### Data not appearing on dashboard
1. **First deploy?** The initial sync takes 1–2 minutes. Watch: `docker compose logs -f backend`
2. **Check for errors:** `docker compose logs backend | grep ERROR`
3. **Try manual sync:** Click the "Sync" button on the Overview page.
4. **Token issue?** Look for `refreshing token` errors in logs — see "Token refresh fails" above.

### Database connection errors
1. Ensure TimescaleDB is healthy: `docker compose ps`
2. If you changed `POSTGRES_PASSWORD` after first run, wipe the volume:
   ```bash
   docker compose down
   sudo rm -rf ./data/timescaledb
   docker compose up -d
   ```

### Port conflicts: "address already in use"
Another service is using the same port. Change `BACKEND_PORT` or `FRONTEND_PORT` in `.env`:
```env
BACKEND_PORT=9090
FRONTEND_PORT=4000
```

### Dashboard shows stale/old data
Continuous aggregates refresh hourly with a 3-day lookback. For immediate results:
1. Click "Sync" on the Overview page
2. Verify poll intervals in `.env` are reasonable

### Docker build uses cached layers
If code changes don't seem to take effect, Docker may be using cached layers:
```bash
docker compose build --no-cache backend
docker compose up -d backend
```

### Running on ARM (Raspberry Pi / Apple Silicon)
Both images build on ARM64 natively. The TimescaleDB image (`timescale/timescaledb:latest-pg15`) supports ARM64.

### Using Watchtower for auto-updates
Watchtower works seamlessly — it only replaces container images, not volumes. Your `.whoop_token.json` (bind mount) and database (`./data/timescaledb`) persist across updates. Add to your `docker-compose.yml`:
```yaml
  watchtower:
    image: containrrr/watchtower
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    command: --interval 86400 whoop-stats-backend whoop-stats-frontend
```

---

## Tech Stack

| Layer | Technology | Role |
|-------|-----------|------|
| **Database** | PostgreSQL 15 + TimescaleDB | Time-series storage with hypertables and continuous aggregates |
| **Backend** | Go 1.25+, go-chi, sqlc | REST API, dual-mode ingestion, type-safe DB queries |
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
├── docker-compose.yml      # Development / homelab deployment
├── docker-compose.prod.yml # Production deployment (named volumes)
├── setup.sh               # Interactive setup wizard
└── .env.example           # Configuration template
```

---

## License

MIT
