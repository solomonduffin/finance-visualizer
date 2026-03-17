# Roadmap: Finance Visualizer

## Overview

The v1.0 milestone (Phases 1-4) established the full stack: SQLite schema, SimpleFIN data pipeline, REST API, and React dashboard with panels and charts. The v1.1 milestone (Phases 5-9) extends the product with account management, operational visibility, analytics depth, threshold-based alerts with email notifications, and forward-looking financial projections. Phase ordering is driven by a hard dependency chain: soft-delete and display_name (Phase 5) must exist before any feature that stores per-account user configuration.

## Milestones

- Shipped **v1.0 MVP** - Phases 1-4 (shipped 2026-03-15)
- Active **v1.1 Enhancements** - Phases 5-9 (in progress)

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

<details>
<summary>v1.0 MVP (Phases 1-4) - SHIPPED 2026-03-15</summary>

- [x] **Phase 1: Foundation** - SQLite schema with migrations, bcrypt auth, JWT middleware, and Docker dev environment (completed 2026-03-15)
- [x] **Phase 2: Data Pipeline** - SimpleFIN HTTP client, daily cron goroutine, append-only snapshot storage (completed 2026-03-15)
- [x] **Phase 3: Backend API** - Full REST API (accounts, balances, net worth, sync status) with layered service architecture (completed 2026-03-15)
- [x] **Phase 4: Frontend Dashboard** - React SPA with liquid/savings/investments panels, charts, and UX polish (completed 2026-03-15)

</details>

### v1.1 Enhancements

- [x] **Phase 5: Data Foundation** - Soft-delete migration and account display name system (schema prerequisites for all v1.1 features) (completed 2026-03-15)
- [ ] **Phase 6: Operational Quick Wins** - Sync failure diagnostics in settings and growth rate indicators on panel cards
- [ ] **Phase 7: Analytics Expansion** - Crypto account aggregation by institution and dedicated net worth drill-down page
- [ ] **Phase 8: Alert System** - Expression-based alert rules with 3-state machine and email notifications via SMTP
- [ ] **Phase 9: Projection Engine** - Forward-looking net worth projections with per-account APY, reinvestment, and income modeling

## Phase Details

<details>
<summary>v1.0 MVP (Phases 1-4) - SHIPPED 2026-03-15</summary>

### Phase 1: Foundation
**Goal**: Users can authenticate into a running app backed by a correct, migration-managed SQLite schema
**Depends on**: Nothing (first phase)
**Requirements**: AUTH-01, DEPLOY-01
**Success Criteria** (what must be TRUE):
  1. Visiting the app URL redirects unauthenticated users to a login page
  2. Entering the correct password grants access; wrong password is rejected
  3. `docker compose up` starts the full stack (Go backend, React frontend, Nginx) with no manual steps
  4. SQLite database initializes with all tables via golang-migrate on first start
  5. WAL mode is active and concurrent reads are not blocked during writes
**Plans:** 3/3 plans complete

Plans:
- [x] 01-01-PLAN.md — Go scaffold, SQLite connection with WAL mode, migrations with full upfront schema, config management
- [x] 01-02-PLAN.md — Auth system: bcrypt password verification, JWT issuance, chi router with rate-limited login and protected routes
- [x] 01-03-PLAN.md — React frontend scaffold with login page, Docker dev/prod environment (Dockerfile, Compose, Nginx)

### Phase 2: Data Pipeline
**Goal**: Real financial account data flows from SimpleFIN into SQLite on a daily schedule with full history on first sync
**Depends on**: Phase 1
**Requirements**: DATA-01, DATA-02, DATA-03, DATA-04
**Success Criteria** (what must be TRUE):
  1. After providing a SimpleFIN access URL, the app fetches and stores all account balances in the database
  2. On first sync the app stores up to one month of historical balance snapshots
  3. The daily cron goroutine runs automatically and appends a new snapshot row per account each day
  4. Each account has at most one snapshot per day (duplicate fetches do not clobber or duplicate data)
  5. Sync failures for individual accounts are logged and do not abort the entire sync run
**Plans:** 3/3 plans complete

Plans:
- [x] 02-01-PLAN.md — SimpleFIN HTTP client + sync orchestration engine (SyncOnce, RunScheduler, account upsert, idempotent snapshots)
- [x] 02-02-PLAN.md — Settings/sync API handlers, config extension (SYNC_HOUR), router wiring, scheduler goroutine in main.go
- [x] 02-03-PLAN.md — React settings page (token config, sync status, Sync Now button), client-side routing with react-router-dom

### Phase 3: Backend API
**Goal**: All financial data in SQLite is accessible via a typed, authenticated REST API that the frontend can consume
**Depends on**: Phase 2
**Requirements**: DASH-01, DASH-02, DASH-03, DASH-04
**Success Criteria** (what must be TRUE):
  1. `GET /api/summary` returns liquid balance (checking minus credit cards including pending), total savings, and total investments as separate fields
  2. `GET /api/accounts` returns all accounts grouped by type with individual balances
  3. `GET /api/balance-history` returns daily snapshot series for each panel (liquid, savings, investments)
  4. All API endpoints return 401 when the request lacks a valid JWT
**Plans:** 2/2 plans complete

Plans:
- [x] 03-01-PLAN.md — Summary and accounts handlers (GET /api/summary with panel totals, GET /api/accounts with grouped account lists)
- [x] 03-02-PLAN.md — Balance history handler (GET /api/balance-history with per-panel time series) and route wiring for all three endpoints

### Phase 4: Frontend Dashboard
**Goal**: The user sees a complete, polished finance dashboard with all panels, charts, and UX details in one glance
**Depends on**: Phase 3
**Requirements**: VIZ-01, VIZ-02, UX-01, UX-02, UX-03, UX-04
**Success Criteria** (what must be TRUE):
  1. The dashboard displays liquid balance, savings total, and investments total in distinct panels with individual account lists beneath each
  2. Each panel shows a balance-over-time line chart populated from daily snapshots
  3. A net worth donut chart shows the proportional split between liquid, savings, and investments
  4. A "Last updated" indicator shows how long ago the most recent sync ran
  5. Before the first sync completes, the app shows a clear empty/loading state rather than blank panels or errors
  6. The user can toggle between dark and light mode, and the layout is usable on a mobile screen
**Plans:** 3/3 plans complete

Plans:
- [x] 04-01-PLAN.md — Dark mode infrastructure, API client extensions, reusable components (PanelCard, SkeletonDashboard, EmptyState), utilities
- [x] 04-02-PLAN.md — Dashboard page with data fetching, panel rendering, freshness indicator, loading/empty/error states, NavBar dark mode toggle
- [x] 04-03-PLAN.md — Balance line chart (tabbed AreaChart) and net worth donut chart, wired into Dashboard with visual verification

</details>

### Phase 5: Data Foundation
**Goal**: Accounts survive SimpleFIN outages with all user-owned metadata intact, and users can rename any account with a display name that appears globally
**Depends on**: Phase 4 (v1.0 complete)
**Requirements**: ACCT-01, ACCT-02, OPS-03
**Success Criteria** (what must be TRUE):
  1. User can set a custom display name for any account in the settings page, and it persists across sessions
  2. Custom display names appear everywhere the account is shown: dashboard panels, charts, and any future dropdowns
  3. When a connected account disappears from SimpleFIN (outage or removal), the account is hidden rather than deleted, preserving its display name and balance history
  4. When a previously hidden account reappears in a subsequent sync, it is automatically restored with all its metadata intact
**Plans:** 3/3 plans complete

Plans:
- [x] 05-01-PLAN.md — Schema migration (display_name, hidden_at, account_type_override), handler COALESCE updates, sync engine soft-delete and auto-restore
- [x] 05-02-PLAN.md — PATCH /api/accounts/:id endpoint, frontend API client extension, display name utility, PanelCard rendering update
- [x] 05-03-PLAN.md — Settings page Accounts section with inline rename, hide/unhide, drag-and-drop type reassignment, toast notifications

### Phase 6: Operational Quick Wins
**Goal**: Users can diagnose sync problems from the settings UI and see at-a-glance growth trends on every panel card
**Depends on**: Phase 5
**Requirements**: OPS-01, OPS-02, INSIGHT-01, INSIGHT-06
**Success Criteria** (what must be TRUE):
  1. Settings page shows a log of recent sync attempts with timestamps, success/failure status, and how many accounts were synced
  2. Failed sync entries can be expanded to reveal sanitized error details (no credentials or tokens leaked)
  3. Each panel card (liquid, savings, investments) shows a percentage change badge over the last 30 days with green for positive and red for negative
  4. User can toggle the growth rate badge on/off from the settings page
**Plans:** 3/3 plans complete

Plans:
- [ ] 06-01-PLAN.md — Backend API: sync log endpoint with sanitization, growth calculation endpoint with shopspring/decimal, settings toggle, TypeScript client types
- [ ] 06-02-PLAN.md — Settings page: SyncHistory timeline component with expand/collapse errors, DashboardPreferences toggle section
- [ ] 06-03-PLAN.md — Dashboard: GrowthBadge component with tooltip and invisible placeholder, PanelCard integration, parallel data fetching

### Phase 7: Analytics Expansion
**Goal**: Users can create custom account groups to organize accounts (e.g., combining multiple Coinbase wallets into one "Coinbase" group), and all users can explore detailed net worth history on a dedicated page
**Depends on**: Phase 5
**Requirements**: ACCT-03, ACCT-04, ACCT-05, INSIGHT-02, INSIGHT-03, INSIGHT-04, INSIGHT-05
**Success Criteria** (what must be TRUE):
  1. User can create a named account group in Settings (e.g., "Coinbase") and assign accounts to it
  2. Account groups appear as a single combined line in their panel, showing the summed balance of member accounts
  3. User can expand a group to see individual account balances beneath it
  4. Clicking the net worth donut chart navigates to a dedicated net worth page
  5. The net worth page shows a historical line chart with per-panel breakdown, summary statistics (current value, period change in dollars and percent, all-time high), and a time range selector (30d, 90d, 6m, 1y, all)
**Plans:** 3 plans

Plans:
- [ ] 07-01-PLAN.md — Account groups backend: database migration, CRUD API endpoints, extend GetAccounts/GetGrowth with group data
- [ ] 07-02-PLAN.md — Net worth page: backend API endpoint with time-series and stats, frontend page with stacked area chart, stats bar, time range selector, donut click navigation, nav link
- [ ] 07-03-PLAN.md — Account groups frontend: GroupRow component, PanelCard group support, AccountsSection group management with drag-and-drop, Dashboard data flow

### Phase 8: Alert System
**Goal**: Users define threshold-based alert rules that fire an email exactly once when crossed and once when recovered, with full SMTP configuration in settings
**Depends on**: Phase 5
**Requirements**: ALERT-01, ALERT-02, ALERT-03, ALERT-04, ALERT-05, ALERT-06, ALERT-07
**Success Criteria** (what must be TRUE):
  1. User can create an alert rule using an expression builder that combines buckets (liquid, savings, investments) and/or individual accounts with arithmetic, compared against a threshold
  2. When a sync causes a rule's computed value to cross its threshold, the user receives exactly one email notification with rule name, computed value, threshold, and crossing direction
  3. When the value recovers back across the threshold, the user receives exactly one recovery email (no repeated alerts on subsequent syncs while the condition holds)
  4. User can configure SMTP or API email provider in settings and verify it works with a test email button
  5. User can create, edit, enable/disable, and delete alert rules from a dedicated alerts management page
**Plans:** 2/4 plans executed

Plans:
- [ ] 08-01-PLAN.md — Core alert engine: database migration (alert_rules, alert_history), Go dependencies (expr-lang/expr, go-mail), expression evaluator, AES-256-GCM crypto, 3-state machine engine, SMTP notifier with tests
- [ ] 08-02-PLAN.md — Backend API: alert CRUD handlers, email config/test handlers, router registration, post-sync evaluation hook
- [ ] 08-03-PLAN.md — Frontend: API client alert/email types, AlertRuleForm expression builder, AlertRuleCard with status/toggle/history, Alerts management page
- [ ] 08-04-PLAN.md — Frontend integration: App.tsx route and nav link, Settings Email Configuration section, visual verification

### Phase 9: Projection Engine
**Goal**: Users see a forward-looking net worth projection chart driven by per-account growth rates, compound/simple interest toggle, and income allocation modeling
**Depends on**: Phase 5
**Requirements**: PROJ-01, PROJ-02, PROJ-03, PROJ-04, PROJ-05, PROJ-06, PROJ-07, PROJ-08
**Success Criteria** (what must be TRUE):
  1. User can set APY or expected growth rate per account, toggle between compound and simple interest, and include/exclude individual accounts from the projection
  2. User can model income by entering annual amount, monthly savings percentage, and allocating savings across accounts
  3. A projection chart shows projected net worth over a user-selected time horizon with a dashed line distinguishing projected from historical values
  4. All projection settings (rates, toggles, income allocations) persist in the database across sessions
  5. The projections page is accessible from the main navigation and investment accounts display available holdings detail from SimpleFIN where supported
**Plans:** 3/5 plans executed

Plans:
- [ ] 09-01-PLAN.md — Backend data layer: database migration (projection settings, holdings tables), SimpleFIN holdings fetch extension, sync process holdings persistence
- [ ] 09-02-PLAN.md — Backend API: projection settings CRUD handlers (GET/PUT accounts with holdings, PUT income settings), frontend API client types and functions
- [ ] 09-03-PLAN.md — Projection engine: client-side calculation (compound/simple interest, income contributions), ProjectionChart with solid-to-dashed transition, HorizonSelector
- [ ] 09-04-PLAN.md — Configuration UI: RateConfigTable with per-account/per-holding rate controls, HoldingsRow expansion, IncomeModelingSection with allocation validation
- [ ] 09-05-PLAN.md — Page assembly: Projections page with state management, debounced auto-save, live chart recalculation, App.tsx route and NavBar integration, visual verification

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9
(Phases 6, 7, 8, 9 all depend on Phase 5 but are independent of each other.)

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Foundation | v1.0 | 3/3 | Complete | 2026-03-15 |
| 2. Data Pipeline | v1.0 | 3/3 | Complete | 2026-03-15 |
| 3. Backend API | v1.0 | 2/2 | Complete | 2026-03-15 |
| 4. Frontend Dashboard | v1.0 | 3/3 | Complete | 2026-03-15 |
| 5. Data Foundation | v1.1 | 3/3 | Complete | 2026-03-15 |
| 6. Operational Quick Wins | 3/3 | Complete   | 2026-03-16 | - |
| 7. Analytics Expansion | v1.1 | 0/3 | Planned | - |
| 8. Alert System | 2/4 | In Progress|  | - |
| 9. Projection Engine | 3/5 | In Progress|  | - |
