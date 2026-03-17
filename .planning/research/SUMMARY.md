# Project Research Summary

**Project:** Finance Visualizer v1.2
**Domain:** Personal finance dashboard — transaction-based feature expansion
**Researched:** 2026-03-17
**Confidence:** HIGH

## Executive Summary

Finance Visualizer v1.2 is an incremental expansion of a working Go/React/SQLite personal finance dashboard that already handles balance tracking, net worth, alerts, and projections. The defining characteristic of this expansion is that the most important unlock — transaction data — requires no new external integration. SimpleFIN already returns transaction arrays in its API response; the current Go client simply drops them during JSON decoding because no `Transactions` field exists on the `Account` struct. Enabling transaction capture is a contained change to `internal/simplefin/client.go` and `internal/sync/sync.go`, and it unblocks every transaction-dependent feature (spending analytics, categorization, recurring detection, cashflow forecasting). This means the v1.2 feature set is largely a question of build order, not feasibility.

The recommended approach is to layer features in dependency order: establish transaction ingestion first, then build analytics on top of that data, then layer planning features on top of analytics. Three features are independent of transactions — data export for balance history, investment performance tracking, and goal tracking — and can be built in parallel once the transaction foundation exists. The stack requires minimal additions: `encoding/csv` (stdlib, zero install), and four small frontend libraries (`@tanstack/react-table`, `date-fns`, `sonner`, `clsx`) totaling ~45KB gzipped. The existing stack handles all charting, state management, and routing without additions.

The primary risks are deduplication integrity and data model discipline. Transaction deduplication is harder than it appears: SimpleFIN transaction IDs are not stable across pending-to-cleared transitions, and real-world integrations (Actual Budget, Firefly III) have documented duplicate bugs. The second risk is architectural discipline around the balance model — once transactions exist, there is pressure to derive balances from transaction sums, which will produce wrong numbers because SimpleFIN only provides a 90-day rolling window. Balance snapshots must remain the single source of truth; transactions are supplementary. Both risks must be resolved in the very first transaction phase before analytics are built on top.

## Key Findings

### Recommended Stack

The existing stack requires no changes for core functionality. The additions are small and targeted. Backend needs only `encoding/csv` (already in Go stdlib) for CSV export. Frontend needs `@tanstack/react-table` for sortable/filterable transaction lists, `date-fns` for date range logic and period comparisons, `sonner` for toast notifications, and `clsx` for conditional CSS composition. No state management library is needed — React Context handles the single-user scale. No new charting library is needed — Recharts 3.8.0 (already in use) supports bar charts, pie charts, area charts, and composed charts, covering all planned analytics views.

**Core technologies (additions only):**
- `encoding/csv` (stdlib): CSV export — zero-dependency streaming writer, no install needed
- `@tanstack/react-table ^8.21.3`: Transaction list table — headless, Tailwind-compatible, sort/filter/pagination built-in
- `date-fns ^4.1.0`: Date manipulation — tree-shakeable, functional API, needed for month boundaries and period comparisons
- `sonner ^2.0.7`: Toast notifications — 5KB, React 19 compatible, zero hooks setup
- `clsx ^2.1.1`: Conditional class composition — 239 bytes, incremental adoption

**Critical stack finding:** The SimpleFIN client already receives transaction data in every `FetchAccountsWithHoldings()` response because `balances-only` is NOT set in that call. Adding a `Transactions []Transaction` field to the Go `Account` struct is sufficient to begin capturing transaction data. Zero additional API calls are needed, and the existing 24-request/day rate limit is unaffected.

### Expected Features

All feature dependencies flow from transactions as the root. Goal tracking and investment performance tracking are independent branches that can be built in parallel.

**Must have (table stakes):**
- Transaction list view — users see balance changes but cannot see why; every competitor shows transactions
- Data export (CSV) — explicitly deferred from v1.1; users expect data portability
- Transaction search — once transactions exist, search is expected; FTS5 already compiled into the SQLite driver
- Toast/notification feedback — operations like sync, export, settings save currently have no visual feedback

**Should have (differentiators):**
- Transaction categorization (rule-based) — turns raw transactions into spending insights; keyword rules + user overrides, no ML needed
- Spending analytics (monthly category breakdown) — shows where money goes; SQL aggregation + Recharts bar charts
- Recurring transaction detection — surfaces subscription/bill awareness; batch SQL detection during sync
- Goal tracking — save toward specific targets; reuses the existing alert operand engine for account selection
- Investment performance tracking — returns on holdings over time; requires `holdings_snapshots` table to preserve history

**Defer to v1.3+:**
- Multi-currency support — large blast radius, touches every balance calculation and chart
- Budget tracking (monthly limits per category) — depends on categorization; useful but complex
- Cashflow forecasting — capstone feature, requires 60+ days of recurring pattern history to be meaningful
- OFX/QFX import for historical backfill — useful but not critical for launch
- Year-over-year spending comparison — needs >12 months of transaction data to be meaningful

### Architecture Approach

The existing architecture is a clean pattern worth preserving: one handler file per feature domain, raw SQL with `database/sql`, React pages that fetch on mount with loading/error states, and a sync pipeline (`SyncOnce`) that serializes all writes behind a mutex. New features slot into this pattern cleanly. The critical structural addition is the sync pipeline extension: `processAccount -> persistHoldings -> persistTransactions -> (post-all-accounts) detectRecurringPatterns -> EvaluateAll`. An `internal/eval/` package should be extracted to share operand evaluation logic between alerts and goals, avoiding duplication and circular imports.

**Major components (new):**
1. `internal/simplefin/client.go` (modified) — add `Transaction` struct and `Transactions []Transaction` to `Account`; this is the zero-API-call transaction unlock
2. `internal/sync/transactions.go` (new) — `persistTransactions()` using upsert (not DELETE+INSERT) because SimpleFIN transactions are a rolling window, not a full replacement set
3. `internal/sync/recurring.go` (new) — `detectRecurringPatterns()` batch-run after all accounts are processed, stored in `recurring_patterns` table
4. `internal/eval/eval.go` (new) — extracted from `internal/alerts/`, shared by both alert evaluation and goal progress calculation
5. `internal/api/handlers/export.go` (new) — stateless streaming CSV handlers for balances, transactions, holdings
6. Five new frontend pages: `Transactions`, `Spending`, `Investments`, `Goals`, `Cashflow`
7. Navigation restructuring into groups (Overview, Money, Investments, Planning, System) — required when adding 5 pages to existing 5

### Critical Pitfalls

1. **Transaction deduplication failure** — SimpleFIN transaction IDs are not stable across pending-to-cleared transitions. Use `UNIQUE(account_id, external_id)` as primary dedup, but add a secondary fuzzy-match layer (same account + date ±1 day + same amount) to catch ID changes. Store `pending` boolean and UPDATE (not INSERT) when a pending transaction clears. Must be solved in the first transaction phase — bolting on dedup later requires a cleanup migration against potentially corrupted data.

2. **Balance model corruption** — Once transactions exist, the temptation to derive balances from transaction sums is strong and wrong. SimpleFIN provides only a 90-day rolling window; a transaction sum will never equal the authoritative balance. `balance_snapshots` must remain the single source of truth. Enforce this in code comments and schema design; add a reconciliation warning log but never a balance override.

3. **SimpleFIN 90-day window creates permanent data gaps** — Transaction history starts from first sync and never reaches further back. All analytics must be designed around "data available since [first sync date]" rather than arbitrary history depth. The analytics UI must show explicit date range indicators and graceful empty states for the first weeks of operation.

4. **Categorization accuracy ceiling without a feedback loop** — Rule-based categorization achieves 70-85% accuracy at best. The `COALESCE(user_category, auto_category)` pattern (already used for `display_name` in accounts) is the right model. "Apply rule to all matching past transactions" is required for the feature to feel correct; without it, users spend hours on manual correction and abandon the feature.

5. **Schema migration risk on a live database** — Six migrations exist with real user data. Each new migration must: use `CREATE TABLE IF NOT EXISTS`, have a corresponding down migration, be tested against a copy of the production database, and be preceded by an automatic pre-migration backup. The transaction phase adds the most schema surface area and is the highest risk.

## Implications for Roadmap

Based on the dependency graph from ARCHITECTURE.md and the build order confirmed in both FEATURES.md and ARCHITECTURE.md, the following phase structure is clear and well-motivated.

### Phase 1: Transaction Foundation

**Rationale:** Three planned features are entirely blocked without this. Starting with anything else means building analytics without data. This is the single most load-bearing change in v1.2.

**Delivers:** Full transaction ingestion pipeline, `transactions` table with proper dedup, `GET /api/transactions` with pagination, `GET /api/transactions/search` using FTS5, `pages/Transactions.tsx` with filters, and navigation restructuring for the new page.

**Addresses:** "Transaction list view" and "Transaction search" from table stakes.

**Avoids:** Deduplication failures, balance model corruption, and rate limit overrun (single consolidated sync call).

**Research flag:** SKIP — patterns are fully specified in ARCHITECTURE.md with exact SQL schema, struct changes, and data flow. Standard Go upsert pattern. No additional research needed.

### Phase 2: Data Export + Toast Notifications

**Rationale:** Data export was explicitly deferred from v1.1 and is low-effort (stateless streaming, no new tables). Toast notifications improve the entire app, including Phase 1's sync feedback. Both are independent of transaction analytics and ship usable value immediately after Phase 1.

**Delivers:** `GET /api/export/{balances,transactions,holdings,all}` with streaming CSV, `ExportSection` component in Settings, and Sonner toast system wired to sync/export/settings operations.

**Addresses:** "Data export (CSV)" and "Toast/notification feedback" from table stakes.

**Avoids:** Export security pitfalls (JWT auth required on all endpoints, no on-disk file persistence, settings table excluded from export).

**Research flag:** SKIP — CSV export uses stdlib, documented streaming pattern in ARCHITECTURE.md. Sonner is trivial setup.

### Phase 3: Investment Performance Tracking

**Rationale:** Independent of transactions; builds on existing holdings data. Requires adding `holdings_snapshots` table to preserve history that currently gets overwritten on each sync (`persistHoldings` does DELETE+INSERT today). Delivers meaningful value for investment-heavy users before spending analytics is complete.

**Delivers:** `holdings_snapshots` table (migration 000009), modified `persistHoldings()` that also upserts snapshots, `GET /api/holdings` and `/api/holdings/history`, `pages/Investments.tsx` with `HoldingsTable` and `HoldingChart`.

**Addresses:** Investment performance tracking from differentiator features.

**Avoids:** Contribution-conflation pitfall — display "Balance change" not "Return" unless manual cash flows are tracked; label must be honest about methodology.

**Research flag:** SKIP — architecture fully specified. The only design decision (simple return vs. TWR) is resolved: use balance change with clear labeling, flag large single-day jumps as likely contributions.

### Phase 4: Goal Tracking

**Rationale:** Independent of transactions; reuses the alert operand engine for account selection. Delivers forward-looking value while transaction history accumulates. The `internal/eval/` extraction happens here, which benefits the existing alerts feature too.

**Delivers:** `goals` table (migration 000010), `internal/eval/` package extracted from alerts, CRUD at `/api/goals`, `pages/Goals.tsx` with `GoalCard` progress rings and `GoalForm` (reuses operand selector from `AlertRuleForm`).

**Addresses:** Goal tracking from differentiator features.

**Avoids:** "Goal set and forgotten" UX pitfall — progress shown on dashboard, alert trigger when goal is reached (reuses existing alert engine).

**Research flag:** SKIP — operand engine already built, extraction is refactoring. Goal schema fully specified in ARCHITECTURE.md.

### Phase 5: Spending Analytics + Categorization

**Rationale:** Requires Phase 1 transactions. Highest-value analytics phase. Categorization infrastructure must precede charts because charts depend on categorized data. Server-side aggregation (`SQL GROUP BY`) is the correct pattern here — consistent with existing `GetSummary`/`GetNetWorth` handlers, avoids sending raw transaction volumes to the browser.

**Delivers:** `category_rules` and `categories` tables (migration 000008), `COALESCE(user_category, auto_category)` pattern wired throughout, `GET /api/spending` and `/spending/trends`, `pages/Spending.tsx` with `SpendingDonut` (Recharts PieChart) and `SpendingTrends` (Recharts stacked BarChart), `CategoryManager` for rule CRUD.

**Addresses:** Transaction categorization and spending analytics from differentiator features.

**Avoids:** Categorization accuracy pitfall — user overrides persist across re-syncs, "apply to all matching" bulk action required before shipping.

**Research flag:** SKIP — SQL aggregation patterns specified, category data model specified, all charting done with existing Recharts.

### Phase 6: Recurring Detection + Cashflow Forecasting

**Rationale:** Recurring detection requires 60+ days of transaction history (from Phase 1). Cashflow forecasting is the capstone — it consumes recurring patterns, balance data, and goals together. Build together because the forecast is the primary value delivery; recurring detection alone is table stakes for the forecast.

**Delivers:** `recurring_patterns` table (migration 000011), `detectRecurringPatterns()` batch-run during sync, `GET/PATCH /api/recurring`, `GET /api/cashflow/inputs`, `pages/Cashflow.tsx` with interactive `CashflowChart` (Recharts AreaChart) and `CashflowAssumptions` toggle component.

**Addresses:** Recurring transaction detection and cashflow forecasting from differentiator features. Dashboard summary widgets (spending badge, upcoming bills) as part of this phase.

**Avoids:** Real-time detection anti-pattern — detection runs once per sync and stores results; the read path is a simple SELECT against `recurring_patterns`.

**Research flag:** NEEDS RESEARCH — the merchant description normalization algorithm (stripping variable reference numbers, dates, transaction IDs) is described conceptually but not implemented. Research how Actual Budget and similar tools normalize merchant names. Plan for a tuning pass after the first 60 days of real transaction data.

### Phase Ordering Rationale

- Phases 1-4 are ordered by the transaction dependency: foundation first, then independent value-adds that can be built while history accumulates toward the 60-day recurring detection threshold.
- Phase 5 is gated on having enough transaction data to make categorization meaningful (30+ days); starting after Phase 1 ships means real data exists by the time Phase 5 launches.
- Phase 6 is explicitly last because recurring detection requires 60+ days of history. Building it sooner means shipping a page that shows "no patterns detected yet" for the first two months.
- Navigation restructuring (grouping into Overview/Money/Investments/Planning/System) should happen in Phase 1 when the first new page (`Transactions`) is added, not retrofitted later when 3-4 more pages exist.

### Research Flags

Phases needing deeper research during planning:
- **Phase 6 (Recurring Detection):** Description normalization algorithm needs validation against real transaction data. Research how Actual Budget normalizes merchant names before pattern matching. Plan a tuning phase after initial deployment.

Phases with standard patterns (skip research-phase):
- **Phase 1:** Architecture fully specified with exact SQL, Go structs, and data flow.
- **Phase 2:** Stdlib CSV streaming + one-line Sonner setup.
- **Phase 3:** Fully specified in ARCHITECTURE.md.
- **Phase 4:** Refactoring of existing alert engine, no new patterns.
- **Phase 5:** SQL GROUP BY + existing Recharts, established patterns.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Existing stack verified from running codebase. Additions verified via npm/pkg.go.dev with exact versions. Alternatives explicitly considered and rejected with rationale. |
| Features | MEDIUM-HIGH | Feature expectations validated against competitor analysis and SimpleFIN protocol spec. Transaction availability confirmed from protocol documentation. Complexity estimates are informed guesses. |
| Architecture | HIGH | Existing codebase fully inspected. SimpleFIN protocol verified. SQL schemas, Go structs, data flows, and file structure fully specified. Exact migration numbers assigned. |
| Pitfalls | HIGH | Sourced from real-world issues in Actual Budget and Firefly III SimpleFIN integrations. shopspring/decimal edge cases verified from package docs. SQLite WAL behavior verified from SQLite documentation. |

**Overall confidence:** HIGH

### Gaps to Address

- **Recurring detection normalization:** The algorithm for normalizing merchant names (stripping variable reference numbers, dates, transaction IDs) is described conceptually but not implemented. It will require iteration against real transaction descriptions once syncing starts. Budget for a tuning pass after first 60 days.
- **SimpleFIN transaction ID stability per institution:** The deduplication strategy accounts for ID instability, but actual behavior varies by financial institution. The user's specific banks may or may not exhibit pending-to-cleared ID changes. Monitor for duplicates in the first weeks after Phase 1 ships and tune the fuzzy matcher if needed.
- **Investment cash flow tracking:** If the user wants true return calculations (not just balance change), manual contribution entry is needed. The current design defers this. If the user relies on ProjectionLab for serious investment analysis (noted in codebase comments), balance-change display may be sufficient and the gap may never matter.
- **Navigation restructuring UX:** Adding 5 pages to an existing 5-page app requires grouped navigation. The restructuring is described in ARCHITECTURE.md but the exact grouping should be confirmed with the user before Phase 1 ships the first new page.

## Sources

### Primary (HIGH confidence)
- [SimpleFIN Protocol Specification](https://www.simplefin.org/protocol.html) — transaction fields, rate limits, 90-day window
- [SimpleFIN Developer Guide](https://beta-bridge.simplefin.org/info/developers) — 24 req/day quota, daily update cadence
- Existing codebase: `internal/simplefin/client.go`, `internal/sync/sync.go`, `internal/api/router.go`, migrations 000001-000006, `frontend/src/api/client.ts`
- [encoding/csv](https://pkg.go.dev/encoding/csv) — Go stdlib CSV package
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — FTS5 compiled-in confirmed
- [shopspring/decimal](https://pkg.go.dev/github.com/shopspring/decimal) — division-by-zero panic behavior verified

### Secondary (MEDIUM confidence)
- [@tanstack/react-table npm](https://www.npmjs.com/package/@tanstack/react-table) — v8.21.3 verified, React 19 compatible
- [date-fns npm](https://www.npmjs.com/package/date-fns) — v4.1.0, 34.9M weekly downloads
- [sonner npm](https://www.npmjs.com/package/sonner) — v2.0.7, React 18+ required, React 19 confirmed compatible
- [Actual Budget SimpleFIN duplicate transactions](https://github.com/actualbudget/actual/issues/2519) — real-world dedup failure modes
- [Actual Budget cross-account mirror transactions](https://github.com/actualbudget/actual/issues/7015) — Mercury bank behavior
- [Stripe: Transaction Categorization Guide](https://stripe.com/resources/more/what-is-transaction-categorization-a-guide-to-transaction-taxonomy-and-its-benefits) — 70-85% accuracy ceiling
- [SQL Habit: Recurring Payment Detection](https://www.sqlhabit.com/blog/how-to-detect-recurring-payments-with-sql) — detection algorithm approach
- [Kitces.com: TWR vs IRR calculations](https://www.kitces.com/blog/twr-dwr-irr-calculations-performance-reporting-software-methodology-gips-compliance/) — return methodology comparison

### Tertiary (LOW confidence)
- [bojanz/currency](https://github.com/bojanz/currency) — multi-currency library, deferred feature, not yet validated against project's decimal usage patterns

---
*Research completed: 2026-03-17*
*Ready for roadmap: yes*
