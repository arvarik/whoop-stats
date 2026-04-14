# Style Guide & Code Conventions

_This document enforces the visual identity and coding patterns of the project. It prevents context drift as multiple agents work on the codebase. Agents MUST follow these rules strictly._

## 1. Visual Language & Tokens
### Colors
- **Deep Dark Mode**: Strictly designed using `zinc-950`.
- **Borders**: Harsh borders are strictly avoided. Use 1px `white/10` borders, backdrop-blurs, and radial glowing gradients.

### Typography
- Minimalist, premium consumer application vibe.

### Spacing & Layout
- **Interactions**: Tactile feedback (e.g., cards lifting on hover, progress bars animating via Framer Motion) to create a polished consumer aesthetic over a raw admin panel.

## 2. Component Patterns
- **Frontend Tools**: Tailwind CSS v4, Shadcn UI, Recharts, Framer Motion.
- **Dynamic Hydration**: Wrap heavy browser `window` dependent components with `next/dynamic` and `ssr: false`.

## 3. Code Conventions
### Architecture Patterns
- **Next.js Server-First**: Use Server Components (RSC) for initial data fetching. Zero client-side fetching for the dashboard initial load.
- **Go Standard Layout**: `cmd/` for entrypoints, `internal/` for handlers/services, `queries/` for DAL.
- **Idempotent Upserts**: Use `INSERT ... ON CONFLICT DO UPDATE` via sqlc in Go, preventing duplication.
- **Webhooks**: Use Inbox pattern—store webhook payloads immediately, process asynchronously via background worker.

### State Management
- Next.js data fetched via RSC. Use Server Actions for mutations. Client caches invalidated via `revalidatePath("/")`.

### Strict Typing
- **Go**: End-to-end type safety from sqlc-generated database types to Go handlers to Swagger schema, down to Next.js API client `openapi-fetch`.
- **TypeScript**: Rely on `openapi-typescript` definitions from `schema.d.ts`. No `any` types.

## 4. Naming Conventions
- **Go Packages**: Lowercase, single-word. No underscores.
- **Files**: `snake_case.go` for Go, kebab-case/camelCase per Next.js conventions in `web/`.
- **Database Tables**: Plural `snake_case` (e.g., `webhook_events`, `users`).
- **Next.js**: Use Server Components defaults unless explicitly marked `'use client'`.

## 5. Import Ordering
- **Go** (enforced by `goimports`):
  1. Standard library
  2. Third-party (`github.com/...`)
  3. Internal packages

## 6. Documentation Standards
- **Swagger**: Go API endpoints annotated with swag declarations to dynamically build Swagger JSON used by Next.js.
- Focus on clean, robust API definitions to ensure type-safe contracts with the frontend.

## 7. Anti-Patterns (FORBIDDEN)
- ❌ NEVER use `any` / `interface{}` in Go or TypeScript unless explicitly acting on a generic constraint.
- ❌ NEVER write raw SQL in Go handler or service code — ALWAYS go through `sqlc` generated queries.
- ❌ NEVER use heavy client-side API requests (`useEffect` data fetching) for dashboard initial load. MUST use Next.js RSC parallel fetching to eliminate layout shifts.
- ❌ NEVER use `OFFSET/LIMIT` for query pagination on timeseries data. MUST use keyset/cursor pagination over composite indexes.
- ❌ NEVER store WHOOP OAuth tokens in plaintext. MUST be AES-256-GCM encrypted `BYTEA`.
- ❌ NEVER block the WHOOP webhook request. MUST insert into `webhook_events` (Inbox pattern) and return `200 OK` instantly to prevent timeouts.
- ❌ NEVER process `Sync` manually without an advisory DB lock (`pg_advisory_xact_lock`) on the user id to avoid duplicate ingestions from spam-clicks.
