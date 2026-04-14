# Product Philosophy

_This is the soul of the product. It explains why the app exists and what its core beliefs are. Product Visionaries and UI/UX Designers use this to make feature and design decisions. Engineers use it to resolve ambiguity._

## 1. Why This Exists
WHOOP data is rich and deeply personal, but managed entirely in an external B2B ecosystem. This tool exists to allow users to pull, store, and analyze their lifetime of WHOOP data safely and permanently on their own hardware. It is a high-performance, self-hostable, zero-data-loss platform.

## 2. Target User
Homelabbers, sysadmins, and power users who prefer to own their data. They value high performance, privacy, and full ownership over their infrastructure. They are comfortable running Docker Compose stacks and maintaining local databases.

## 3. Core Beliefs
- **Zero Data Loss**: Every piece of WHOOP biometric data matters. Ingestion paths (whether Webhooks or Polling) must be rock solid, utilizing Inbox patterns and idempotent upserts.
- **Privacy First**: Sensitive API keys and OAuth tokens are never left in plaintext; AES-256-GCM encryption is mandatory. Self-hosting ensures data never leaves the user's network.
- **Enterprise-Grade Performance at Homelab Scale**: Using TimescaleDB hypertables and Go goroutines guarantees that even years of granular time-series data load instantly.

## 4. Design & UX Principles
- **Premium Aesthetics**: Eschew the "raw admin panel" look. Incorporate glassmorphism, deep dark mode (zinc-950), backdrop-blurs, glowing gradients, and fluid Framer Motion animations to mimic high-end consumer apps.
- **Instantaneous Feedback**: Dashboards render instantly using Next.js Server Components. Client hydration happens lazily, ensuring the fastest Time To First Byte (TTFB).
- **Tactile Interactions**: Elements respond physically to user actions (hover states, progress bar animations).

## 5. What This Is NOT
- Not a managed B2B SaaS.
- Not a public cloud service where data is commoditized.
- Not a generic health aggregator; it is specifically built for deep WHOOP data analysis.
