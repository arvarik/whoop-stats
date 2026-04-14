# Style Guide & Code Conventions

_This document enforces the visual identity and coding patterns of the project. It prevents context drift as multiple agents work on the codebase. Agents MUST follow these rules strictly. Every rule here is verified against the actual source code._

## 1. Visual Design System

### Color Tokens (from `web/src/app/globals.css`)
The design system is defined via Tailwind CSS v4 `@theme inline` tokens. **Use these tokens exclusively — never use raw hex values or default Tailwind colors.**

#### Core Backgrounds
| Token | Value | Usage |
|-------|-------|-------|
| `bg-background` | `#09090B` | Page background |
| `bg-surface-0` | `#0F0F12` | Card backgrounds, sidebar |
| `bg-surface-1` | `#18181B` | Elevated surfaces |
| `bg-surface-2` | `#27272A` | Hover states, dividers |
| `bg-surface-3` | `#3F3F46` | Highest elevation surfaces |

#### Text
| Token | Value | Usage |
|-------|-------|-------|
| `text-text-primary` | `#FAFAFA` | Headings, primary content |
| `text-text-secondary` | `#A1A1AA` | Sidebar labels, secondary info |
| `text-text-tertiary` | `#71717A` | Timestamps, hints |
| `text-text-muted` | `#52525B` | Disabled states |

#### Borders
| Token | Value | Usage |
|-------|-------|-------|
| `border-border-subtle` | `rgba(255,255,255,0.06)` | Card borders |
| `border-border-default` | `rgba(255,255,255,0.1)` | Default borders |
| `border-border-hover` | `rgba(255,255,255,0.16)` | Hover borders |

#### Accent (Indigo)
| Token | Value | Usage |
|-------|-------|-------|
| `text-accent` / `bg-accent` | `#6366F1` | Primary accent |
| `text-accent-hover` / `bg-accent-hover` | `#818CF8` | Accent hover state |
| `bg-accent-muted` | `rgba(99, 102, 241, 0.15)` | Accent background tint |

#### Data Colors
| Token | Value | Usage |
|-------|-------|-------|
| `text-strain` / `bg-strain` | `#3B82F6` | Strain data |
| `bg-strain-muted` | `rgba(59, 130, 246, 0.15)` | Strain background tint |
| `text-sleep` / `bg-sleep` | `#8B5CF6` | Sleep data |
| `bg-sleep-muted` | `rgba(139, 92, 246, 0.15)` | Sleep background tint |

#### Recovery Spectrum
| Token | Value | Usage |
|-------|-------|-------|
| `--color-recovery-green` | `#10B981` | Recovery ≥ 66% |
| `--color-recovery-yellow` | `#F59E0B` | Recovery 34-65% |
| `--color-recovery-red` | `#EF4444` | Recovery < 34% |

#### HR Zone Colors (0-5)
| Token | Value | Zone |
|-------|-------|------|
| `--color-zone-0` | `#6B7280` | Below threshold |
| `--color-zone-1` | `#3B82F6` | Light effort |
| `--color-zone-2` | `#10B981` | Moderate |
| `--color-zone-3` | `#F59E0B` | Hard |
| `--color-zone-4` | `#F97316` | Very hard |
| `--color-zone-5` | `#EF4444` | Max effort |

#### Sidebar Tokens
| Token | Value | Usage |
|-------|-------|-------|
| `--color-sidebar-background` | `#0F0F12` | Sidebar background |
| `--color-sidebar-foreground` | `#A1A1AA` | Sidebar text |
| `--color-sidebar-primary` | `#FAFAFA` | Active item text |
| `--color-sidebar-primary-foreground` | `#09090B` | Active item background text |
| `--color-sidebar-accent` | `rgba(99, 102, 241, 0.15)` | Active item highlight |
| `--color-sidebar-accent-foreground` | `#FAFAFA` | Active item highlight text |
| `--color-sidebar-border` | `rgba(255, 255, 255, 0.06)` | Sidebar dividers |
| `--color-sidebar-ring` | `#6366F1` | Focus ring |

#### Radius Tokens
| Token | Value |
|-------|-------|
| `--radius-sm` | `0.375rem` |
| `--radius-md` | `0.5rem` |
| `--radius-lg` | `0.75rem` |
| `--radius-xl` | `1rem` |
| `--radius-2xl` | `1.25rem` |

### shadcn HSL Compatibility Layer
The `globals.css` file includes a `:root` block with HSL-based CSS custom properties for shadcn component compatibility. These are defined in the `@layer base` block and map to the design system:
```css
:root {
  --background: 0 0% 3.9%;   /* matches #09090B */
  --foreground: 0 0% 98%;    /* matches #FAFAFA */
  --card: 0 0% 5.5%;         /* matches surface-0 */
  --border: 240 3.7% 15.9%;  /* matches surface-2 */
  /* ... etc */
}
```
**Do not modify these HSL tokens directly** — they exist solely for shadcn component interop. Style using the `@theme inline` tokens above.

### Custom Utilities
```css
@utility glass-card {
  border-radius: 1rem;
  border: 1px solid rgba(255, 255, 255, 0.06);
  background-color: rgba(15, 15, 18, 0.8);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
}

@utility glass-card-hover {
  /* Same as glass-card + hover transition: border lightens, bg shifts */
  transition: all 300ms;
  &:hover {
    border-color: rgba(255, 255, 255, 0.16);
    background-color: rgba(24, 24, 27, 0.8);
  }
}
```
Use these instead of manually composing backdrop-blur classes.

### Typography
- **Font**: Inter (loaded via `next/font/google` with `--font-inter` CSS variable, applied as `font-sans` via `className` on `<body>`).
- **Headings**: `text-text-primary`, `font-semibold`, `tracking-tight`.
- **Body/Labels**: `text-text-secondary` or `text-text-tertiary`, `text-sm` or `text-xs`.
- **Uppercase labels**: `text-xs font-medium uppercase tracking-wider text-text-tertiary`.

### Icons
- `lucide-react` exclusively (v0.577). Common: `Activity`, `Moon`, `HeartPulse`, `Flame`, `TrendingUp`, `Dumbbell`.
- Standard size: `w-4 h-4` (inline with text) or `w-3 h-3` (in subtitles).

## 2. Frontend Conventions (Next.js 16)

### App Router Rules
- **ALL pages go in `web/src/app/`**. The `pages/` directory is forbidden.
- **ALL components go in `web/src/components/`**. Shadcn primitives go in `web/src/components/ui/`.
- **ALL utility/lib code goes in `web/src/lib/`**.
- Pages are **Server Components by default**. Only add `"use client"` when the component needs browser APIs, event handlers, or state.
- Use `export const dynamic = "force-dynamic"` on dashboard pages to ensure fresh data on every request.

### Special Pages
- `error.tsx`: Global error boundary (`"use client"`) with glass-card UI, error logging, and retry button.
- `loading.tsx`: Skeleton loading state (`"use client"`) matching the dashboard layout with `animate-pulse` shimmer.
- `layout.tsx`: Root layout — sets Inter font, wraps children in `<Sidebar>` + `<MobileNav>` + `<Toaster>`. Dark mode is hardcoded via `<html className="dark">`.

### Data Fetching
- **Initial loads**: Parallel `Promise.all()` via `openapi-fetch` in Server Components. Zero `useEffect` fetching.
- **Mutations**: Server Actions in `web/src/app/actions.ts`. Call `revalidatePath()` for all affected routes (`/`, `/recovery`, `/sleep`, `/strain`, `/workouts`).
- **Type safety**: All API responses are typed via `openapi-typescript` → `schema.d.ts` → `openapi-fetch`.
- **Lazy env validation**: `client.ts` validates env vars lazily at runtime (first API call), not at import time. This prevents CI build failures when env vars aren't set.

### Utility Libraries
| File | Purpose |
|------|---------|
| `web/src/lib/format.ts` | Display formatters: `formatDuration()`, `formatShortDate()`, `formatFullDate()`, `formatTime()`, `formatDistance()`, `formatCalories()`, `getRecoveryColor()`, `getRecoveryColorValue()`, `getRecoveryLabel()`, `HR_ZONE_COLORS`, `HR_ZONE_LABELS` |
| `web/src/lib/stats.ts` | Statistical helpers: `computeAvg()`, `computeStdDev()`, `percentChange()` |
| `web/src/lib/types.ts` | Shared types: `ApiRecord = Record<string, any>` (escape hatch for untyped API responses) |
| `web/src/lib/utils.ts` | `cn()` — conditional className merge via `clsx` + `tailwind-merge` |

### Component Toolkit
| Library | Version | Usage |
|---------|---------|-------|
| `shadcn` | v4 | Button, Card, Badge, Skeleton, Sonner (toast), Chart |
| `@base-ui/react` | v1.2 | Headless UI primitives |
| `@tremor/react` | v3.18 | Data visualization components |
| `recharts` | v2.15 | All charts (strain/recovery trends, HR zones) |
| `framer-motion` | v12 | Page transitions, micro-animations |
| `lucide-react` | v0.577 | Icon library |
| `date-fns` | v4 | Date formatting |
| `@number-flow/react` | v0.6 | Animated number transitions |
| `sonner` | v2 | Toast notifications |
| `next-themes` | v0.4 | Theme management (installed, dark mode hardcoded) |
| `class-variance-authority` | v0.7 | Component variant definitions (cva) |
| `clsx` | v2.1 | Conditional class names |
| `tailwind-merge` | v3.5 | Tailwind class deduplication |
| `tw-animate-css` | v1.4 | CSS animations for shadcn components |

### Dynamic Imports
Heavy client-side libraries that access `window` must be lazy-loaded:
```tsx
const ChartComponent = dynamic(() => import("@/components/chart"), { ssr: false });
```

## 3. Backend Conventions (Go)

### Package Layout
- `cmd/server/` — Server entrypoint. Flag parsing (`-mode`, `-user`), service wiring, graceful shutdown (SIGINT/SIGTERM, 10s deadline).
- `cmd/auth/` — OAuth2 bootstrap CLI. Standalone, no server dependency. Uses `log` (not `slog`), unprefixed env vars.
- `internal/api/` — HTTP handlers. Each handler method lives on the `Handler` struct. Generic `handleList[T]()` for paginated endpoints.
- `internal/auth/` — Token lifecycle (refresh, encrypt, cache, offline fallback). `sync.Map` + per-user `sync.Mutex`.
- `internal/config/` — Viper config with `WHOOP_STATS_` env prefix. Validates required fields and key length.
- `internal/crypto/` — AES-256-GCM encrypt/decrypt with random nonce.
- `internal/db/` — **Auto-generated by sqlc** (except `batch.go`). Never edit `db.go`, `models.go`, `querier.go`, or `query.sql.go`.
- `internal/middleware/` — HTTP middleware (auth, logging, rate limiting).
- `internal/poller/` — Polling engine (4 independent loops + ad-hoc sync via `RunAdHocSync()`).
- `internal/storage/` — Domain mapper: `whoop-go` SDK types → sqlc `UpsertXParams`. Batch operations via `pgx.Batch{}`.
- `internal/webhook/` — Webhook handler (inbox) + background worker.
- `internal/worker/` — Empty package (reserved for future use).

### Error Handling
- Always wrap errors with context: `fmt.Errorf("upserting cycle %d: %w", id, err)`.
- HTTP errors use the `sendError()` helper with structured `ErrorResponse` JSON.
- Use specific HTTP codes: `401 Unauthorized`, `409 Conflict`, `429 Too Many Requests`, `202 Accepted`.
- Auth manager uses `fmt.Errorf` chains that preserve the full error context through decrypt → refresh → encrypt → persist.

### Logging
- Use `log/slog` exclusively across all `internal/` packages. JSON handler by default.
- Logger is passed as dependency — never use the global `log` package (except `cmd/auth/` bootstrap).
- Respect `LOG_LEVEL` env var. Avoid `Debug` logs inside tight loops (SSD wear concern).
- Health check requests (`/healthz`) are excluded from request logging in `middleware.Logger()`.
- Webhook worker uses `logger.With()` to attach `trace_id` and `event_type` to all log entries for a given event.

### Database Operations
- Always go through `internal/db/` (sqlc-generated) for queries.
- For custom SQL: add to `queries/query.sql`, run `sqlc generate`.
- Use `pgx.Batch{}` for bulk operations in the storage layer (e.g., `storage.UpsertCycles()`).
- Batch SQL strings are exposed as public constants in `internal/db/batch.go` (e.g., `db.UpsertCycleSQL`).
- `internal/db/batch.go` is the **only hand-written file** in `internal/db/` — it exists to bridge sqlc's private SQL constants to the `pgx.Batch` API.

### Import Order (enforced by `goimports`)
```go
import (
    "standard/library"

    "github.com/third-party"

    "github.com/arvind/whoop-stats/internal/..."
)
```

## 4. Naming Conventions

| Context | Convention | Example |
|---------|-----------|---------|
| Go packages | Lowercase, single-word | `crypto`, `poller`, `storage` |
| Go files | `snake_case.go` | `manager.go`, `handlers_test.go` |
| Go test files | `*_test.go` in same package | `aes_test.go` |
| Database tables | Plural `snake_case` | `webhook_events`, `body_measurements` |
| Database columns | `snake_case` | `recovery_score`, `start_time` |
| Continuous aggregates | `snake_case` | `daily_strain`, `weekly_recovery` |
| TypeScript files | `kebab-case.tsx` | `metric-card.tsx`, `sleep-panels.tsx` |
| TypeScript exceptions | `PascalCase.tsx` for single-component files | `SyncButton.tsx` |
| React components | `PascalCase` | `SyncButton`, `MetricCard` |
| Server Actions | `camelCase` functions | `syncWhoopData()` |
| CSS tokens | `kebab-case` | `--color-text-primary` |
| Env vars | `SCREAMING_SNAKE_CASE` | `WHOOP_STATS_ENCRYPTION_KEY` |
| SQL queries (sqlc) | `PascalCase` | `UpsertCycle`, `GetCycles` |

## 5. Swagger / API Documentation
- Go handlers are annotated with `@Summary`, `@Description`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Router`, `@Security` swag directives.
- Generate Swagger spec: `swag init -g cmd/server/main.go`.
- Output: `docs/swagger.json`, `docs/swagger.yaml`, `docs/docs.go`.
- Frontend types generated from OpenAPI spec via `openapi-typescript` → `web/src/lib/api/schema.d.ts`.
- OpenAPI spec maintained at `docs/openapi.json`.

## 6. Anti-Patterns (FORBIDDEN)

❌ **NEVER edit auto-generated files in `internal/db/`** (`db.go`, `models.go`, `querier.go`, `query.sql.go`) — they are overwritten by sqlc. `batch.go` IS hand-written and CAN be edited.

❌ **NEVER use `database/sql`** — always use `jackc/pgx/v5` (`pgxpool`, `pgtype`).

❌ **NEVER use `useEffect` for data fetching on initial page load** — use RSC parallel fetching via `Promise.all()`.

❌ **NEVER create `pages/` directory routes** — App Router (`app/`) exclusively.

❌ **NEVER use `pages/api/` for mutations** — use Next.js Server Actions.

❌ **NEVER use `any` type in TypeScript** — all types must come from `schema.d.ts` or be explicitly defined. The only exception is `ApiRecord` in `types.ts` (escape hatch for untyped API responses).

❌ **NEVER use `interface{}` in Go** — use concrete types from `internal/db/` or `whoop-go`.

❌ **NEVER use `OFFSET/LIMIT` pagination** — use cursor-based pagination with `start_time < $cursor ORDER BY start_time DESC LIMIT $n`.

❌ **NEVER store tokens in plaintext** — `internal/crypto.Encrypt()` for storage, `crypto.Decrypt()` for use.

❌ **NEVER process webhooks synchronously** — insert into `webhook_events`, return `200 OK`, process in background worker.

❌ **NEVER trigger concurrent syncs** — the sync endpoint uses in-memory mutex + active sync map to prevent duplicates.

❌ **NEVER use raw Tailwind colors** (e.g., `bg-zinc-900`) — use the design tokens (`bg-surface-0`, `text-text-primary`, etc.).

❌ **NEVER create custom CSS classes with `@apply`** — use the `@utility` directive or inline Tailwind classes.

❌ **NEVER use the global `log` package in `internal/`** — pass `*slog.Logger` as a dependency. (`cmd/auth/` is the sole exception.)

❌ **NEVER validate env vars at module scope in frontend** — use lazy validation at runtime to prevent CI build failures (see `client.ts` pattern).
