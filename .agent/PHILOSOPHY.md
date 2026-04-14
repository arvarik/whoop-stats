# Product Philosophy

_This is the soul of the product. It explains why the app exists and what its core beliefs are. Product Visionaries and UI/UX Designers use this to make feature and design decisions. Engineers use it to resolve ambiguity._

## 1. Why This Exists

WHOOP generates some of the richest biometric data available — granular HR zones, sleep staging, HRV, SpO2, skin temperature — but it lives entirely in a proprietary SaaS ecosystem. If WHOOP ever changes their API, raises prices, or sunsets a feature, that lifetime of personal health data is at risk.

This tool exists to give users **permanent, sovereign ownership** of their WHOOP data on their own hardware. It is a high-performance ingestion engine and analytics dashboard that runs forever on a Raspberry Pi, NAS, or any Docker-capable machine.

## 2. Target User

The primary user is a **technical homelab enthusiast** who:
- Runs Docker Compose stacks on Synology NAS, Proxmox, or bare-metal servers.
- Values data sovereignty over convenience — they want their biometric data on their LAN, not in someone else's cloud.
- Is comfortable with `.env` files, port mappings, and reading logs.
- Uses WHOOP daily and wants deeper analysis than the mobile app provides.
- May not have a public IP or domain — their server sits behind NAT on a residential connection.

The **secondary user** is a developer or data scientist who wants to build custom analyses on top of a clean, well-typed PostgreSQL/TimescaleDB schema of their WHOOP data.

## 3. Core Beliefs

### Zero Data Loss
Every WHOOP data point — every heartbeat zone, every sleep stage transition — matters across a lifetime. The system is designed around **never losing data**:
- **Dual ingestion engines**: Webhook mode uses the Inbox Pattern (store raw payload immediately, process asynchronously) to guarantee data capture even during WHOOP API rate limiting. Polling mode actively re-fetches data on configurable intervals.
- **Idempotent upserts**: Every database write is an `INSERT ... ON CONFLICT DO UPDATE`. Running the same sync twice produces identical results with zero duplicates. All hypertables use composite PKs `(id, start_time)` for TimescaleDB chunk-level uniqueness.
- **Offline token fallback**: The auth manager reads `.whoop_token.json` as a bootstrap fallback, and keeps it synced after every token refresh, so a database wipe doesn't permanently lose authentication. This is critical because WHOOP refresh tokens are **single-use** — once consumed, the old token is dead.
- **Batch error isolation**: When batch-upserting records, individual failures don't prevent other records in the batch from being processed. Webhook events that fail processing are marked `failed` without affecting the rest of the batch.

### Privacy First
- OAuth tokens are **never stored in plaintext** — AES-256-GCM encrypted `BYTEA` columns in Postgres. Each encryption uses a unique random nonce.
- JWTs are generated **server-side only** (via `jose` in Next.js RSC) — never exposed to the browser. Environment variables are validated lazily at runtime (not import time) to prevent accidental exposure during CI builds.
- Self-hosting means biometric queries never leave the user's network.
- Docker containers run as non-root users (`appuser` in backend, `nextjs` UID 1001 in frontend).
- `.whoop_token.json` is mounted read-only in production Docker compose.
- `.whoop_token.json` and `.env` are in `.gitignore` to prevent credential commits.

### Enterprise-Grade Performance at Homelab Scale
- TimescaleDB **continuous aggregates** pre-compute daily/weekly trends — dashboard queries are O(1) lookups, not full table scans on years of minute-level data. Refresh policies run hourly with 3-day lookback.
- The Go backend uses **`pgxpool` connection pooling** with concurrent goroutine polling, protected by a shared `rate.Limiter` (2 req/500ms).
- The dashboard renders instantly via **Next.js RSC** with parallel `Promise.all()` data fetching — zero client-side loading spinners on initial page load. `force-dynamic` ensures fresh data on every request.
- Next.js `output: "standalone"` produces a minimal Docker image by tree-shaking unused dependencies, reducing image size and startup time.
- Poller rate limiting (2 req/500ms shared limiter) prevents WHOOP API abuse while maximizing data freshness.
- HTTP server timeouts (15s read, 30s write, 60s idle) prevent slow-client resource exhaustion.

### Resilient by Design
- **Graceful shutdown**: SIGINT/SIGTERM triggers context cancellation for all goroutines, HTTP server drains with a 10-second deadline, `sync.WaitGroup` ensures all background workers finish.
- **Error boundaries**: `error.tsx` catches any page-level rendering or API failure and presents a user-friendly retry screen. `loading.tsx` provides skeleton UI during RSC streaming.
- **Adaptive polling**: Sleep data polling automatically reduces frequency during off-peak hours (outside 6 AM–12 PM) to minimize unnecessary API calls and SSD writes.
- **Stale connection cleanup**: IP rate limiter evicts visitor entries unseen for 30+ minutes every 10 minutes, preventing unbounded memory growth.

## 4. Design & UX Principles

### Premium Consumer Aesthetic
This should look and feel like a **$100M VC-funded health app**, not an open-source admin panel. Specific mandates:
- **Deep dark mode only**: Background is `#09090B` (zinc-950). No light mode. No toggles. Hardcoded via `<html className="dark">`.
- **Glassmorphism surfaces**: Cards use `glass-card` utility — `backdrop-blur(24px)`, `rgba(15, 15, 18, 0.8)` background, `rgba(255, 255, 255, 0.06)` borders.
- **No harsh borders**: Borders are always low-opacity white (`white/6` to `white/16`), never solid colored lines.
- **Inter font**: Google Font, loaded via `next/font` with `--font-inter` CSS variable for zero-FOUT.
- **Color-coded health data**: Recovery uses a green/yellow/red spectrum (thresholds: 66/34). Strain is blue. Sleep is violet. HR zones have a distinct 6-color palette (gray → blue → green → yellow → orange → red).
- **Fluid animations**: Framer Motion for page transitions and state changes. `@number-flow/react` for animated number counters.
- **Design token driven**: All colors, borders, and radii come from the `@theme inline` system in `globals.css`. No raw hex values in component code.

### Instantaneous Feedback
- Dashboard TTFB is optimized by parallel RSC fetching — the user sees data, not skeletons.
- `loading.tsx` provides a polished skeleton matching the actual layout for the rare cases where streaming is slow.
- Sync button provides immediate toast feedback via Sonner.
- `force-dynamic` export ensures every page load fetches fresh data.

### Tactile Interactions
- Cards lift on hover (`glass-card-hover` with border color transition and background shift, 300ms).
- Recovery strip uses opacity-scaled color bars that convey score at a glance (0.3 + score/100 * 0.7 opacity).
- Charts use macOS-style tooltips with smooth cursor tracking via Recharts.
- Number transitions animate smoothly via `@number-flow/react`.

## 5. What This Is NOT

- **Not a SaaS product**: No user accounts, no billing, no multi-tenancy. One user, one instance.
- **Not a WHOOP replacement**: It doesn't replace the WHOOP app or strap — it's a data archive and analytics layer.
- **Not a generic health aggregator**: Purpose-built exclusively for WHOOP data via the `whoop-go` SDK.
- **Not cloud-native**: No Kubernetes, no Lambda, no managed databases. Docker Compose on a single machine.
- **Not visually "open source"**: It should feel premium, polished, and intentional — not like a Bootstrap template.
- **Not a real-time dashboard**: Data is fetched on page load (RSC) and on-demand via sync. No WebSocket streams, no live heart rate.

## 6. SSD Protection Philosophy

This application is designed to run 24/7 on consumer hardware (NAS, mini-PCs) with consumer SSDs that have limited write endurance:
- **tmpfs mounts**: `/tmp` and `/var/log` are mounted as `tmpfs` in Docker to eliminate filesystem writes for temporary data and container logs.
- **Local logging driver**: Docker uses the `local` logging driver with 10MB max size, minimizing log rotation writes.
- **Minimal write amplification**: Poller intervals are tuned (30m–24h) to write only when new data exists. Adaptive sleep polling further reduces off-peak writes.
- **Structured slog logging**: `LOG_LEVEL` is respected throughout — setting to `warn` in production eliminates routine request logging writes. Health check requests are never logged regardless of level.
