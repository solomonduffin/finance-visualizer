# Roadmap: Finance Visualizer

## Overview

Four phases build up the stack in dependency order: the foundation establishes the schema, auth, and Docker skeleton before any feature code touches them; the data pipeline wires SimpleFIN and proves real data flows into SQLite; the backend API exposes that data as a typed REST contract; the React frontend consumes the API and delivers the complete dashboard experience. Each phase is fully testable before the next begins.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation** - SQLite schema with migrations, bcrypt auth, JWT middleware, and Docker dev environment (completed 2026-03-15)
- [x] **Phase 2: Data Pipeline** - SimpleFIN HTTP client, daily cron goroutine, append-only snapshot storage (completed 2026-03-15)
- [ ] **Phase 3: Backend API** - Full REST API (accounts, balances, net worth, sync status) with layered service architecture
- [ ] **Phase 4: Frontend Dashboard** - React SPA with liquid/savings/investments panels, charts, and UX polish

## Phase Details

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
- [ ] 02-01-PLAN.md — SimpleFIN HTTP client + sync orchestration engine (SyncOnce, RunScheduler, account upsert, idempotent snapshots)
- [ ] 02-02-PLAN.md — Settings/sync API handlers, config extension (SYNC_HOUR), router wiring, scheduler goroutine in main.go
- [ ] 02-03-PLAN.md — React settings page (token config, sync status, Sync Now button), client-side routing with react-router-dom

### Phase 3: Backend API
**Goal**: All financial data in SQLite is accessible via a typed, authenticated REST API that the frontend can consume
**Depends on**: Phase 2
**Requirements**: DASH-01, DASH-02, DASH-03, DASH-04
**Success Criteria** (what must be TRUE):
  1. `GET /api/summary` returns liquid balance (checking minus credit cards including pending), total savings, and total investments as separate fields
  2. `GET /api/accounts` returns all accounts grouped by type with individual balances
  3. `GET /api/balance-history` returns daily snapshot series for each panel (liquid, savings, investments)
  4. All API endpoints return 401 when the request lacks a valid JWT
**Plans:** 1/2 plans executed

Plans:
- [ ] 03-01-PLAN.md — Summary and accounts handlers (GET /api/summary with panel totals, GET /api/accounts with grouped account lists)
- [ ] 03-02-PLAN.md — Balance history handler (GET /api/balance-history with per-panel time series) and route wiring for all three endpoints

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
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation | 3/3 | Complete   | 2026-03-15 |
| 2. Data Pipeline | 3/3 | Complete   | 2026-03-15 |
| 3. Backend API | 1/2 | In Progress|  |
| 4. Frontend Dashboard | 0/TBD | Not started | - |
