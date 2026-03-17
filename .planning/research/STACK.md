# Stack Research

**Domain:** Personal finance dashboard -- next-step feature additions (v1.2)
**Researched:** 2026-03-17
**Confidence:** HIGH (existing stack verified from codebase, additions verified via npm/pkg.go.dev)

## Existing Stack (DO NOT change)

Already validated and working in v1.0/v1.1 -- listed for reference only:

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.25 | Backend, cron scheduler, API |
| chi/v5 | 5.2.5 | HTTP router |
| modernc.org/sqlite | 1.46.1 | Pure-Go SQLite driver (FTS5 compiled in) |
| golang-migrate/migrate/v4 | 4.19.1 | Database migrations |
| shopspring/decimal | 1.4.0 | Decimal arithmetic for balances |
| expr-lang/expr | 1.17.8 | Alert expression evaluation |
| wneessen/go-mail | 0.7.2 | SMTP email for alerts |
| React | 19.2.4 | Frontend framework |
| TypeScript | 5.9.3 | Type safety |
| Vite | 7.0.0 | Build tooling |
| Recharts | 3.8.0 | Charting (line, area, composed) |
| Tailwind CSS | 4.2.1 | Styling |
| react-router-dom | 7.13.1 | Client-side routing |
| @dnd-kit | 0.3.2 | Drag and drop (group management) |
| Vitest | 3.2.1 | Testing |

## Recommended Stack Additions

### Backend Libraries (Go)

| Library | Version | Purpose | Why Recommended |
|---------|---------|---------|-----------------|
| `encoding/csv` (stdlib) | go1.25 | CSV export of balance history, transactions | Standard library -- zero dependencies, streaming Writer with Flush, handles RFC 4180 correctly. No reason to add a third-party CSV library for export-only use. |
| `bojanz/currency` | latest | Currency metadata (symbols, formatting, digits) | Only needed IF multi-currency support is added. Provides CLDR v48 data (~40KB), locale-aware formatting, and ISO 4217 codes. Pairs well with existing shopspring/decimal for arithmetic. |

**NOT recommended for backend:**

| Library | Why Not |
|---------|---------|
| `govalues/decimal` or `govalues/money` | Faster than shopspring/decimal in benchmarks, but project already uses shopspring throughout. Migration cost for no meaningful benefit at single-user scale. Stick with shopspring/decimal. |
| `go-pdf/fpdf` | PDF export is over-engineering for a single-user dashboard. CSV covers 95% of export needs. If PDF is ever wanted, generate it client-side from the browser print dialog or use the backend to serve CSV and let the user open in a spreadsheet. The fpdf library was also archived on GitHub in March 2025, migrated to Codeberg. |
| Any exchange rate API client | Premature. Multi-currency is a large feature. If needed later, use a simple HTTP client to call a free API (e.g., exchangerate-api.com) rather than adding a Go SDK dependency. |

### Frontend Libraries (React/TypeScript)

| Library | Version | Purpose | Why Recommended |
|---------|---------|---------|-----------------|
| `@tanstack/react-table` | ^8.21.3 | Transaction list with sort/filter/pagination | Headless, 5-14KB, works with Tailwind. Only needed if transaction viewing is built. Provides sorting, filtering, pagination, column visibility out of the box without imposing UI opinions. |
| `date-fns` | ^4.1.0 | Date formatting, parsing, relative time | Current codebase uses native Date with a custom `timeAgo()` utility. Adding transactions/budgeting means significantly more date manipulation (month boundaries, week ranges, period comparisons). date-fns is tree-shakeable (~18KB gzipped for typical usage), functional style. Only import what you need. |
| `sonner` | ^2.0.7 | Toast notifications for user feedback | The app currently has no toast/notification system. Operations like "export complete", "sync triggered", "settings saved" need feedback. Sonner is 5KB, requires zero hooks setup, works with React 19, and persists across route changes. |
| `clsx` | ^2.1.1 | Conditional CSS class composition | 239 bytes. Currently the codebase uses string interpolation for conditional classes. As components grow more complex (transaction rows with status colors, category badges), clsx keeps class logic clean. Trivial to adopt incrementally. |

**NOT recommended for frontend:**

| Library | Why Not |
|---------|---------|
| `zustand` / `jotai` / Redux | The app uses React Context + prop drilling and it works fine for a single-user dashboard. No complex shared state, no performance issues from Context re-renders at this scale. Adding a state manager is premature complexity. |
| `luxon` | Overkill timezone handling for a single-user app where everything is in local time. date-fns covers the use cases better with smaller bundle. |
| `dayjs` | Moment.js-style mutable API is less ergonomic than date-fns functional approach. Plugin system adds friction. |
| `react-number-format` | Use native `Intl.NumberFormat` for currency display. It is built into browsers, zero bundle cost, and handles locale-aware formatting. The app already displays dollar amounts -- just standardize on a utility function wrapping `Intl.NumberFormat`. |
| `papaparse` / `react-papaparse` | For CSV *export*, the backend generates CSV via `encoding/csv` and sends it as a download. No client-side CSV generation needed. PapaParse is for *parsing* uploaded CSVs, which is not a planned feature. |
| `file-saver` | Modern browsers handle `Content-Disposition: attachment` headers natively. The backend sets the header, browser downloads the file. No client-side blob saving needed for export. |
| Material React Table / AG Grid | Commercial or heavy. TanStack Table + Tailwind gives full control without framework lock-in. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| SQLite FTS5 | Full-text search on transaction descriptions | Already compiled into modernc.org/sqlite. No new dependency needed -- just create FTS5 virtual tables in migrations. Useful for searching transaction descriptions. |

## Installation

### Backend (only if building specific features)

```bash
# CSV export -- no installation needed, encoding/csv is stdlib

# Multi-currency (only if building that feature):
go get github.com/bojanz/currency
```

### Frontend

```bash
# Core additions for transaction/budgeting features:
npm install @tanstack/react-table date-fns sonner clsx

# That is it. All 4 combined add ~45KB gzipped to the bundle.
```

## Stack Patterns by Feature

**If building transaction viewing/categorization:**
- Backend: Add `Transaction` struct to SimpleFIN client (fields: id, posted, amount, description, pending), store in new `transactions` table, expose via `/api/transactions` endpoint with query params for date range, account, category
- Frontend: `@tanstack/react-table` for the list, `date-fns` for date range logic, `clsx` for conditional row styling
- Database: FTS5 virtual table on transaction descriptions for search, categories table with user-defined mappings

**If building data export (CSV):**
- Backend: `encoding/csv` writing to `http.ResponseWriter` with `Content-Disposition: attachment` header
- Frontend: Simple `<a>` link or `window.location` redirect to the export endpoint. No special library needed.
- Endpoints: `GET /api/export/balances?format=csv&from=...&to=...`, `GET /api/export/transactions?format=csv&...`

**If building spending analytics/budgeting:**
- Backend: SQL aggregation queries (GROUP BY category, month), returned as JSON
- Frontend: Recharts 3.8.0 already supports bar charts, composed charts, treemaps, pie charts -- no new charting library needed
- Database: `budgets` table (category, amount, period), `categories` table (id, name, parent_id for hierarchy)

**If building goal tracking:**
- Backend: New `goals` table (name, target_amount, target_date, linked_account_ids), progress calculated server-side
- Frontend: Recharts progress bars or custom SVG. `date-fns` for deadline calculations.
- No new libraries needed beyond what is already listed

**If building multi-currency:**
- Backend: `bojanz/currency` for formatting and metadata, exchange rate fetching via simple HTTP client on a schedule
- Database: `exchange_rates` table (from_currency, to_currency, rate, date), update daily
- Frontend: `Intl.NumberFormat` with currency option for display. No library needed.
- IMPORTANT: This is a large feature. The `currency` column already exists in accounts table but everything currently assumes USD. Requires touching all balance calculations, charts, and aggregations.

**If building recurring transaction detection:**
- Backend: Pure SQL + Go logic. Group transactions by normalized merchant name, check for regular intervals (weekly/monthly/yearly). No ML needed at single-user scale.
- Database: `recurring_patterns` table (merchant_name, amount, frequency, last_seen, next_expected)
- No new libraries needed

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `@tanstack/react-table` | Build custom table components | If you only need a simple, non-sortable list of fewer than 50 items. TanStack Table shines when you need sort + filter + pagination. |
| `date-fns` | Native `Date` + custom utilities | If the feature scope stays small (1-2 new date operations). Once you need month boundaries, week calculations, period diffs, date-fns pays for itself. |
| `sonner` | `react-hot-toast` | If you prefer the toast(promise) pattern for async operations. react-hot-toast (2.6.0) is equally small. Sonner wins on accessibility and route-change persistence. |
| `encoding/csv` (stdlib) | `gocarina/gocsv` | If you need struct-tag-based CSV serialization for complex types. For simple balance/transaction export, stdlib is cleaner. |
| No state management library | `zustand` | If a future feature requires genuinely shared mutable state across unrelated component trees (not just prop drilling or context). Revisit then, not now. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Moment.js | Deprecated, 329KB, mutable. The project itself recommends alternatives. | `date-fns` |
| Chart.js / Victory / Nivo | Already invested in Recharts 3.8.0. Switching charting libraries is painful and unnecessary -- Recharts covers all needed chart types. | Recharts (already in stack) |
| Prisma / TypeORM | Backend is Go, not Node.js. | Raw SQL + golang-migrate (already in stack) |
| Any Go ORM (GORM, ent) | Project uses raw SQL with golang-migrate and it works well. ORMs add abstraction for no benefit at this scale. | Raw SQL (existing pattern) |
| Redux / MobX | Massive overkill for a single-user dashboard with local-first data fetching patterns. | React Context (existing pattern) |
| Tailwind UI / headlessui | Paid component library. The project builds custom components with Tailwind utilities and that is working fine. | Custom Tailwind components (existing pattern) |

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `@tanstack/react-table@8.21.3` | React 16.8+ through 19 | Headless, no React version coupling issues |
| `date-fns@4.1.0` | Any JS environment | Pure functions, no framework dependency |
| `sonner@2.0.7` | React 18+ | Uses React 18+ APIs. Confirmed compatible with React 19. |
| `clsx@2.1.1` | Any JS environment | Zero dependencies, framework agnostic |
| `recharts@3.8.0` | React 18+ | Already in use and working with React 19.2.4 |

## Critical Finding: SimpleFIN Transaction Data

The SimpleFIN protocol provides transaction-level data that the current codebase does NOT consume. The existing client sets `balances-only=1`, which suppresses transaction details. Transaction fields available from SimpleFIN:

- `id` (string) -- unique transaction identifier
- `posted` (int64) -- Unix epoch timestamp
- `amount` (string) -- positive = deposit, negative = withdrawal
- `description` (string) -- human-readable merchant/description
- `pending` (boolean) -- whether the transaction has posted

This means transaction-based features (categorization, spending analytics, recurring detection) are feasible without any new data source. The existing `fetchAccountData()` function already supports `balancesOnly=false` for holdings -- a similar path can expose transaction data by adding a `Transaction` struct and a `transactions` field to the `Account` struct.

**This is the single most important stack finding: no new external integration is needed for transaction data.** The plumbing to fetch it already exists; it just needs to be turned on and stored.

## Sources

- [encoding/csv](https://pkg.go.dev/encoding/csv) -- Go standard library CSV package
- [@tanstack/react-table npm](https://www.npmjs.com/package/@tanstack/react-table) -- v8.21.3 verified
- [date-fns npm](https://www.npmjs.com/package/date-fns) -- v4.1.0, 34.9M weekly downloads
- [sonner npm](https://www.npmjs.com/package/sonner) -- v2.0.7, React 18+ toast component
- [bojanz/currency](https://github.com/bojanz/currency) -- Go currency handling with CLDR v48
- [recharts npm](https://www.npmjs.com/package/recharts) -- v3.8.0 confirmed latest
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) -- FTS5 compiled in, confirmed
- [SimpleFIN Protocol](https://www.simplefin.org/protocol.html) -- Transaction fields specification
- [TanStack Table docs](https://tanstack.com/table/latest) -- headless table for React
- [clsx](https://github.com/lukeed/clsx) -- 239B conditional class utility
- [SQLite FTS5](https://www.sqlite.org/fts5.html) -- Full-text search extension

---
*Stack research for: Finance Visualizer v1.2 feature additions*
*Researched: 2026-03-17*
