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
- **Idempotent upserts**: Every database write is an `INSERT ... ON CONFLICT DO UPDATE`. Running the same sync twice produces identical results with zero duplicates.
- **Offline token fallback**: The auth manager reads `.whoop_token.json` as a bootstrap fallback, and keeps it synced after every token refresh, so a database wipe doesn't permanently lose authentication.

### Privacy First
- OAuth tokens are **never stored in plaintext** — AES-256-GCM encrypted `BYTEA` columns in Postgres.
- JWTs are generated **server-side only** (via `jose` in Next.js RSC) — never exposed to the browser.
- Self-hosting means biometric queries never leave the user's network.
- Docker containers run as non-root users.

### Enterprise-Grade Performance at Homelab Scale
- TimescaleDB **continuous aggregates** pre-compute daily/weekly trends — dashboard queries are O(1) lookups, not full table scans on years of minute-level data.
- The Go backend uses **`pgxpool` connection pooling** with concurrent goroutine polling.
- The dashboard renders instantly via **Next.js RSC** with parallel `Promise.all()` data fetching — zero client-side loading spinners on initial page load.
- Poller rate limiting (2 req/500ms shared limiter) prevents WHOOP API abuse.

## 4. Design & UX Principles

### Premium Consumer Aesthetic
This should look and feel like a **$100M VC-funded health app**, not an open-source admin panel. Specific mandates:
- **Deep dark mode only**: Background is `#09090B` (zinc-950). No light mode. No toggles.
- **Glassmorphism surfaces**: Cards use `glass-card` utility — `backdrop-blur(24px)`, `rgba(15, 15, 18, 0.8)` background, `rgba(255, 255, 255, 0.06)` borders.
- **No harsh borders**: Borders are always low-opacity white (`white/6` to `white/16`), never solid colored lines.
- **Inter font**: Google Font, loaded via `next/font` for zero-FOUT.
- **Color-coded health data**: Recovery uses a green/yellow/red spectrum. Strain is blue. Sleep is violet. HR zones have a distinct 6-color palette.
- **Fluid animations**: Framer Motion for page transitions and state changes.

### Instantaneous Feedback
- Dashboard TTFB is optimized by parallel RSC fetching — the user sees data, not skeletons.
- Sync button provides immediate toast feedback via Sonner.
- `force-dynamic` export ensures every page load fetches fresh data.

### Tactile Interactions
- Cards lift on hover (`glass-card-hover` with border color transition).
- Recovery strip uses opacity-scaled color bars that convey score at a glance.
- Charts use macOS-style tooltips with smooth cursor tracking.

## 5. What This Is NOT

- **Not a SaaS product**: No user accounts, no billing, no multi-tenancy. One user, one instance.
- **Not a WHOOP replacement**: It doesn't replace the WHOOP app or strap — it's a data archive and analytics layer.
- **Not a generic health aggregator**: Purpose-built exclusively for WHOOP data via the `whoop-go` SDK.
- **Not cloud-native**: No Kubernetes, no Lambda, no managed databases. Docker Compose on a single machine.
- **Not visually "open source"**: It should feel premium, polished, and intentional — not like a Bootstrap template.
