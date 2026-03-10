# WHOOP Stats

A premium, high-performance, open-source dashboard and ingestion engine for your WHOOP fitness data.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg)
![Next.js](https://img.shields.io/badge/Next.js-16+-black.svg)

---

## Architecture: Dual Ingestion Engines

This project is engineered to work in any environment by supporting two distinct data ingestion modes.

### 1. The Polling Engine (Standard for Homelabs/NAS)
**The "Why":** Most home users run servers behind NAT/Firewalls without a public domain or fixed IP. Polling is the **most secure and simplest** option for homelabs because it requires **zero open inbound ports** on your router. 
**The "How":** In `poll` mode, the backend makes outbound requests to WHOOP. Since your server initiates the connection, you don't need to configure port forwarding, Dynamic DNS, or reverse proxies (like ngrok or Cloudflare Tunnels). It uses cursor-based pagination to sync your entire history automatically.

### 2. The Webhook Inbox Pattern (Standard for Cloud)
**The "Why":** For users with public-facing cloud instances (DigitalOcean, AWS, etc.), webhooks provide near-instant synchronization when a WHOOP activity is completed.
**The "How":** We use an "Inbox Pattern" to ensure 100% data integrity. The server instantly acknowledges the webhook and saves it to a queue. A background worker then fetches the full data, protecting you from losing updates if the API is slow or throttled.

---

## Features

*   **Complete Data Coverage:** Tracks all available metrics including Cycles, Sleep Stages, Recoveries (SPO2, HRV, Skin Temp), Workouts (HR Zones), and User Profiles.
*   **SSD Wear Protection:** Optimized for 24/7 homelab usage with RAM-backed logs (`tmpfs`), compressed binary logging, and dynamic log levels to minimize write amplification.
*   **Continuous Aggregates:** Pre-computed TimescaleDB views for `O(1)` dashboard performance.
*   **Linear-Inspired UI:** Tactile Next.js 16 interface with Glassmorphism, Tailwind CSS v4, and interactive Recharts visualizations.
*   **Fully Typed:** 100% end-to-end type safety using `sqlc` and `openapi-typescript`.

---

## Getting Started

### 1. Prerequisites
* Docker and Docker Compose
* Go 1.22+ (one-time use to run the authentication script)
* A [WHOOP Developer account](https://developer.whoop.com/)

### 2. Environment Setup
Copy the example environment file and fill in your WHOOP API credentials and a random 32-character encryption key.
```bash
cp .env.example .env
```

### 3. First-Time Authentication
Since this app can run in environments without a public URL, we use a one-time script to generate your initial tokens:

1.  **Configure Redirect URI:** In your WHOOP Developer Dashboard, add `http://localhost:8081/callback` to your App's Redirect URIs.
2.  **Generate Token:** Run the following commands on your local machine (your PC, not necessarily the NAS):
    ```bash
    export WHOOP_CLIENT_ID=your_id
    export WHOOP_CLIENT_SECRET=your_secret
    go run cmd/auth/main.go
    ```
3.  **Authorize:** Follow the URL printed in your terminal, log in to WHOOP, and authorize the app.
4.  **Save JSON:** This will generate a file named `.whoop_token.json` in your current directory.
5.  **Deploy:** If you are deploying to a NAS or remote server, upload this `.whoop_token.json` to the root directory of the project on that server.

### 4. Deployment

#### Option A: Homelab / NAS (Polling Mode)
```bash
docker-compose up -d --build
```

#### Option B: Cloud Instance (Webhook Mode)
1. Ensure your server is accessible via HTTPS.
2. Configure your WHOOP Webhook URL to point to `https://your-domain.com/webhook`.
3. Start in webhook mode:
```bash
docker-compose -f docker-compose.prod.yml up -d --build
```

---

## Security

* **AES-256-GCM:** OAuth tokens are encrypted in memory and stored as ciphertext in the database.
* **Non-Root Containers:** All processes drop privileges to low-privileged users for enhanced security.
* **Thread-Safe Sync:** Advisory locks prevent race conditions during manual UI-triggered synchronizations.
