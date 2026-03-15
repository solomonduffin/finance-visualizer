# Architecture Research: v1.1 Feature Integration

**Domain:** Self-hosted personal finance dashboard -- 7 new features integrating into existing Go/React/SQLite stack
**Researched:** 2026-03-15
**Confidence:** HIGH -- all 7 features are well-understood patterns; existing codebase is clean and straightforward to extend

## Existing Architecture Snapshot

Before detailing integration, here is the current system as-built in v1.0:

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Docker Network                              │
│                                                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                    Nginx (port 80/443)                         │  │
│  │          /api/* -> Go:8080    /* -> React static               │  │
│  └──────────────────────────┬────────────────────────────────────┘  │
│                              │                                      │
│  ┌──────────────────────────▼────────────────────────────────────┐  │
│  │                    Go Backend (chi router)                     │  │
│  │                                                                │  │
│  │  cmd/server/main.go           internal/config/config.go        │  │
│  │                                                                │  │
│  │  internal/api/                                                 │  │
│  │    router.go                  (7 routes, JWT middleware)        │  │
│  │    handlers/                                                   │  │
│  │      auth.go       summary.go      accounts.go                 │  │
│  │      settings.go   history.go      health.go                   │  │
│  │                                                                │  │
│  │  internal/sync/sync.go        (SyncOnce, RunScheduler)         │  │
│  │  internal/simplefin/client.go (FetchAccounts, ClaimSetupToken) │  │
│  │  internal/db/db.go            (Open, WAL mode)                 │  │
│  │  internal/db/migrations.go    (go-migrate with embed)          │  │
│  │  internal/auth/auth.go        (JWT init + TokenAuth)           │  │
│  └──────────────────────────┬────────────────────────────────────┘  │
│                              │                                      │
│  ┌──────────────────────────▼────────────────────────────────────┐  │
│  │              SQLite (data/finance.db) -- WAL mode              │  │
│  │                                                                │  │
│  │  settings             key/value config store                   │  │
│  │  accounts             id, name, account_type, org_name, ...    │  │
│  │  balance_snapshots    account_id, balance, balance_date        │  │
│  │  sync_log             started_at, finished_at, error_text      │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │              React SPA (Vite + TypeScript)                     │  │
│  │                                                                │  │
│  │  pages/Dashboard.tsx    pages/Settings.tsx    pages/Login.tsx   │  │
│  │  components/PanelCard   BalanceLineChart      NetWorthDonut    │  │
│  │  api/client.ts          hooks/useDarkMode     utils/format     │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

### Key Architectural Facts (from code analysis)

1. **No service layer exists.** Handlers query SQLite directly via `database.QueryContext`. Business logic (account type grouping, balance aggregation) lives in handler functions.
2. **Sync is standalone.** `internal/sync/sync.go` owns the entire sync flow: read settings, call SimpleFIN, upsert accounts, insert snapshots, log results. Uses a package-level `syncMu` mutex for concurrency safety.
3. **`*sql.DB` is passed directly** to all handler constructors and sync functions. No repository interfaces.
4. **Migration files** use golang-migrate with embedded SQL (`//go:embed migrations/*.sql`). Only one migration exists (`000001_init`).
5. **Frontend has no router for sub-pages.** BrowserRouter exists but only routes `/` (Dashboard), `/settings`, and catch-all redirect.
6. **API client** is a single file with typed fetch wrappers. No state management library (no Redux/Zustand) -- components fetch directly.

## v1.1 Feature Integration Map

### Feature-by-Feature Analysis

Each feature is assessed for: schema changes, backend changes (new vs modified files), frontend changes (new vs modified files), new API endpoints, and dependencies on other features.

---

### Feature 1: Crypto Account Aggregation

**What:** Group crypto accounts by institution (e.g., all Coinbase wallets show as one summed line in the Investments panel).

**Schema changes:** None. The `accounts` table already has `org_name` and `org_slug`. Aggregation is a query-time concern.

**Backend changes (MODIFY):**
- `internal/api/handlers/accounts.go` -- Add query parameter `?group_crypto=true` or make it the default. When grouping, use `GROUP BY org_slug` for investment accounts where `org_name` matches known crypto institutions, summing balances.
- `internal/api/handlers/summary.go` -- No change needed. Summary already sums all investment accounts regardless of grouping.

**Alternative approach (preferred):** Add a new `is_crypto` boolean column to `accounts` via migration, set it during sync based on heuristics (org name contains "coinbase", "kraken", etc.), and group by `org_slug` where `is_crypto = true` in the accounts endpoint. This is more reliable than runtime heuristics.

**Frontend changes (MODIFY):**
- `frontend/src/components/PanelCard.tsx` -- Modify the account list rendering to show grouped entries. When the API returns a grouped crypto item, render it as a single line with the institution name.
- `frontend/src/api/client.ts` -- Update `AccountItem` type to include optional `grouped_count` field.

**New API endpoints:** None. Modify existing `GET /api/accounts`.

**Dependencies:** None. This feature is standalone.

---

### Feature 2: Account Renaming

**What:** Users set custom display names for accounts in Settings. These names are used globally (dashboard, drill-down, alerts).

**Schema changes:**
- **Migration 000002:** `ALTER TABLE accounts ADD COLUMN display_name TEXT;`

**Backend changes (MODIFY):**
- `internal/api/handlers/accounts.go` -- Return `display_name` alongside `name`. Frontend decides which to show.
- `internal/api/handlers/summary.go` -- No change (sums by type, doesn't use names).
- `internal/sync/sync.go` -- The upsert in `processAccount` must NOT overwrite `display_name` on sync. Modify the `ON CONFLICT` clause to exclude `display_name` from the UPDATE set.

**Backend changes (NEW):**
- `internal/api/handlers/account_settings.go` -- New handler: `PUT /api/accounts/{id}/display-name` accepting `{"display_name": "My Checking"}`. Updates the `display_name` column.

**Frontend changes (MODIFY):**
- `frontend/src/components/PanelCard.tsx` -- Use `display_name || name` for rendering.
- `frontend/src/api/client.ts` -- Add `display_name` to `AccountItem`, add `updateAccountDisplayName()` function.

**Frontend changes (NEW):**
- `frontend/src/pages/Settings.tsx` -- Add an "Account Names" section listing all accounts with inline edit fields. (Extend existing page, not a new page.)

**New API endpoints:**
- `PUT /api/accounts/{id}/display-name`

**Dependencies:** None. Standalone, but should be built before features that display account names (drill-down, alerts) so they all use display_name from the start.

---

### Feature 3: Growth Rate Indicators

**What:** Show percentage change (e.g., +2.3% this month) on each PanelCard.

**Schema changes:** None. Derived from existing `balance_snapshots`.

**Backend changes (MODIFY):**
- `internal/api/handlers/summary.go` -- Extend the summary response to include growth fields. Query the previous period's totals (e.g., 30 days ago) alongside current totals and compute `(current - previous) / |previous| * 100`. Return `liquid_growth`, `savings_growth`, `investments_growth` as string percentages.

**Alternative approach:** Compute growth on the frontend from balance-history data already fetched. This avoids a backend change but couples the Dashboard component to history data for a summary display. **Recommendation: compute in backend** because the summary endpoint is the canonical source and the growth values should be consistent regardless of which frontend page requests them.

**Frontend changes (MODIFY):**
- `frontend/src/api/client.ts` -- Add growth fields to `SummaryResponse`.
- `frontend/src/components/PanelCard.tsx` -- Add a growth badge (green up-arrow / red down-arrow) below the total. Accept `growth` prop.
- `frontend/src/pages/Dashboard.tsx` -- Pass growth values from summary to PanelCard.

**New API endpoints:** None. Extend existing `GET /api/summary`.

**Dependencies:** None. Standalone.

---

### Feature 4: Net Worth Drill-Down Page

**What:** New page with detailed historical graphs, per-account breakdowns, and pattern analysis.

**Schema changes:** None. All data exists in `balance_snapshots` and `accounts`.

**Backend changes (NEW):**
- `internal/api/handlers/drilldown.go` -- New handler: `GET /api/balance-history/detailed` returning per-account time-series data (not aggregated by panel type). Returns `{ accounts: [{ id, name, display_name, type, org_name, history: [{date, balance}] }] }`.

**Backend changes (MODIFY):**
- `internal/api/router.go` -- Register new route.

**Frontend changes (NEW):**
- `frontend/src/pages/NetWorthDrillDown.tsx` -- New page with:
  - Stacked area chart showing all accounts over time (using Recharts AreaChart with multiple Area elements)
  - Account-level line charts (toggle which accounts are visible)
  - Summary stats (total net worth, 30d/90d/1y growth, best/worst performing account)
- `frontend/src/components/StackedAreaChart.tsx` -- New chart component for the stacked view.

**Frontend changes (MODIFY):**
- `frontend/src/App.tsx` -- Add route `/net-worth` pointing to new page.
- `frontend/src/App.tsx` (NavBar) -- Add navigation link.
- `frontend/src/api/client.ts` -- Add `getDetailedHistory()` function and types.

**New API endpoints:**
- `GET /api/balance-history/detailed?days=N`

**Dependencies:** Feature 2 (account renaming) should be done first so drill-down uses display names from day one.

---

### Feature 5: Sync Diagnostics

**What:** Show sync history and failure details in Settings page.

**Schema changes:** None. `sync_log` table already captures `started_at`, `finished_at`, `accounts_fetched`, `accounts_failed`, `error_text`.

**Backend changes (NEW):**
- `internal/api/handlers/sync_diagnostics.go` -- New handler: `GET /api/sync/log?limit=N` returning recent sync_log entries as JSON array. Each entry includes all sync_log columns.

**Backend changes (MODIFY):**
- `internal/api/router.go` -- Register new route.

**Frontend changes (MODIFY):**
- `frontend/src/pages/Settings.tsx` -- Add a "Sync History" section below the existing Sync Status card. Show a table/list of recent syncs with timestamp, accounts fetched, accounts failed, and error text (expandable).
- `frontend/src/api/client.ts` -- Add `getSyncLog()` function and `SyncLogEntry` type.

**New API endpoints:**
- `GET /api/sync/log?limit=N`

**Dependencies:** None. Standalone.

---

### Feature 6: Alert Rules with Email Notifications

**What:** User defines rules like "notify me when liquid < $5000" or "investments drop > 10% in a week". Alerts fire once on threshold crossing, once on recovery. Email delivery via SMTP or API provider.

**Schema changes:**
- **Migration 000003 (or combined with others):**

```sql
CREATE TABLE alert_rules (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    expression  TEXT NOT NULL,           -- e.g., "liquid < 5000"
    enabled     INTEGER NOT NULL DEFAULT 1,
    last_state  TEXT DEFAULT 'normal',   -- 'normal' | 'triggered'
    last_eval   DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE alert_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id     INTEGER NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    state       TEXT NOT NULL,           -- 'triggered' | 'recovered'
    eval_result TEXT,                    -- the computed value at evaluation time
    notified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Backend changes (NEW):**
- `internal/alerts/evaluator.go` -- Expression evaluation engine. Use [expr-lang/expr](https://github.com/expr-lang/expr) for safe, sandboxed expression evaluation. Define an environment struct with fields: `liquid`, `savings`, `investments`, `net_worth`, plus per-account balances accessible as `account["Display Name"]`. Evaluate the user's expression string against this environment, returning a boolean.
- `internal/alerts/engine.go` -- Alert evaluation orchestrator. Called after each sync completes. Loads all enabled rules, evaluates each against current data, compares result to `last_state`, fires notification on state transition (normal->triggered or triggered->normal), updates `last_state` and writes to `alert_history`.
- `internal/alerts/notifier.go` -- Email notification sender. Uses [wneessen/go-mail](https://github.com/wneessen/go-mail) for SMTP delivery. Reads SMTP config from settings table. Sends a formatted email with rule name, current value, threshold, and timestamp.
- `internal/api/handlers/alerts.go` -- CRUD handlers for alert rules:
  - `GET /api/alerts` -- list all rules with last_state
  - `POST /api/alerts` -- create rule (validates expression with expr.Compile before saving)
  - `PUT /api/alerts/{id}` -- update rule
  - `DELETE /api/alerts/{id}` -- delete rule
  - `GET /api/alerts/{id}/history` -- get alert history for a rule

**Backend changes (MODIFY):**
- `internal/sync/sync.go` -- After `finalize(fetched, failed, nil)` in `SyncOnce`, call `alerts.EvaluateAll(ctx, db)` to trigger alert evaluation on every successful sync.
- `internal/api/handlers/settings.go` -- Extend `SaveSettings` / `GetSettings` to handle SMTP configuration keys (`smtp_host`, `smtp_port`, `smtp_user`, `smtp_pass`, `smtp_from`, `alert_email_to`). Store in existing `settings` table as key/value pairs.
- `internal/api/router.go` -- Register alert routes.
- `internal/config/config.go` -- No change needed. SMTP config lives in the settings table (user-configurable at runtime), not in environment variables.

**Frontend changes (NEW):**
- `frontend/src/pages/Alerts.tsx` -- New page for managing alert rules. Includes:
  - Rule list with enable/disable toggle, last state indicator, delete button
  - "Add Rule" form with expression input, name field
  - Expression syntax help text (available variables, operators)
- `frontend/src/components/AlertRuleForm.tsx` -- Form component for creating/editing rules.

**Frontend changes (MODIFY):**
- `frontend/src/App.tsx` -- Add route `/alerts` and nav link.
- `frontend/src/pages/Settings.tsx` -- Add "Email Notifications" section for SMTP configuration.
- `frontend/src/api/client.ts` -- Add alert CRUD functions and types.

**New API endpoints:**
- `GET /api/alerts`
- `POST /api/alerts`
- `PUT /api/alerts/{id}`
- `DELETE /api/alerts/{id}`
- `GET /api/alerts/{id}/history`

**Dependencies:** Feature 2 (account renaming) should be built first so expressions can reference display names. Feature 3 (growth indicators) is useful context but not a hard dependency.

---

### Feature 7: Projected Net Worth with Income Modeling

**What:** A projections page showing future net worth based on per-account growth rates (APY), reinvestment toggles, income allocation, and a configurable time horizon.

**Schema changes:**
- **Migration 000004 (or combined):**

```sql
CREATE TABLE projection_config (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id      TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    apy             TEXT NOT NULL DEFAULT '0',       -- annual percentage yield as decimal string
    reinvest        INTEGER NOT NULL DEFAULT 1,      -- 1 = compound, 0 = simple
    UNIQUE(account_id)
);

CREATE TABLE income_allocations (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,                    -- e.g., "Monthly Salary"
    amount          TEXT NOT NULL,                    -- monthly contribution amount
    account_id      TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    enabled         INTEGER NOT NULL DEFAULT 1
);
```

**Backend changes (NEW):**
- `internal/projections/engine.go` -- Projection computation engine. Takes current balances, per-account APY, reinvestment flags, income allocations, and time horizon (months). Computes month-by-month projected balances using compound interest formula: `balance * (1 + apy/12)^months` for reinvest, or `balance + (balance * apy/12 * months)` for simple. Adds monthly income allocations. Returns a time-series of projected values per account and total net worth.
- `internal/api/handlers/projections.go` -- Handlers:
  - `GET /api/projections/config` -- returns all projection_config and income_allocation rows
  - `PUT /api/projections/config` -- bulk save projection configs (all accounts at once)
  - `POST /api/projections/income` -- add income allocation
  - `DELETE /api/projections/income/{id}` -- remove income allocation
  - `GET /api/projections/compute?months=N` -- runs the projection engine and returns the time-series result

**Backend changes (MODIFY):**
- `internal/api/router.go` -- Register projection routes.

**Frontend changes (NEW):**
- `frontend/src/pages/Projections.tsx` -- New page with:
  - Configuration panel: per-account APY inputs, reinvestment toggles, income allocation manager
  - Projection chart: line chart showing projected net worth over time horizon
  - Time horizon selector (1y, 3y, 5y, 10y)
  - Comparison lines (with vs without income, with vs without reinvestment)
- `frontend/src/components/ProjectionChart.tsx` -- Recharts line chart with multiple series.

**Frontend changes (MODIFY):**
- `frontend/src/App.tsx` -- Add route `/projections` and nav link.
- `frontend/src/api/client.ts` -- Add projection API functions and types.

**New API endpoints:**
- `GET /api/projections/config`
- `PUT /api/projections/config`
- `POST /api/projections/income`
- `DELETE /api/projections/income/{id}`
- `GET /api/projections/compute?months=N`

**Dependencies:** Feature 2 (account renaming) for display names in the configuration UI. No functional dependency on other features.

---

## Combined Schema Migration Plan

All schema changes should be delivered as separate numbered migrations for clean rollback:

```
internal/db/migrations/
  000001_init.up.sql              (existing)
  000001_init.down.sql            (existing)
  000002_account_display_name.up.sql
  000002_account_display_name.down.sql
  000003_alert_rules.up.sql
  000003_alert_rules.down.sql
  000004_projections.up.sql
  000004_projections.down.sql
```

**Migration 000002:**
```sql
ALTER TABLE accounts ADD COLUMN display_name TEXT;
```

**Migration 000003:**
```sql
CREATE TABLE alert_rules (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    expression  TEXT NOT NULL,
    enabled     INTEGER NOT NULL DEFAULT 1,
    last_state  TEXT NOT NULL DEFAULT 'normal',
    last_eval   DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE alert_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id     INTEGER NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    state       TEXT NOT NULL,
    eval_result TEXT,
    notified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_alert_history_rule ON alert_history(rule_id, notified_at DESC);
```

**Migration 000004:**
```sql
CREATE TABLE projection_config (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id  TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    apy         TEXT NOT NULL DEFAULT '0',
    reinvest    INTEGER NOT NULL DEFAULT 1,
    UNIQUE(account_id)
);

CREATE TABLE income_allocations (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    amount      TEXT NOT NULL,
    account_id  TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    enabled     INTEGER NOT NULL DEFAULT 1
);
```

Optional migration 000005 for crypto grouping (if the `is_crypto` column approach is chosen):
```sql
ALTER TABLE accounts ADD COLUMN is_crypto INTEGER NOT NULL DEFAULT 0;
```

## New Backend File Map

```
internal/
  api/
    router.go                          MODIFY (add 12+ new routes)
    handlers/
      accounts.go                      MODIFY (crypto grouping, display_name)
      summary.go                       MODIFY (growth indicators)
      settings.go                      MODIFY (SMTP config, account renaming section)
      history.go                       unchanged
      auth.go                          unchanged
      health.go                        unchanged
      account_settings.go              NEW (PUT display name)
      drilldown.go                     NEW (detailed balance history)
      sync_diagnostics.go              NEW (sync log endpoint)
      alerts.go                        NEW (alert CRUD + history)
      projections.go                   NEW (projection config + compute)
  sync/
    sync.go                            MODIFY (call alert evaluation after sync)
  alerts/                              NEW PACKAGE
    evaluator.go                       NEW (expr-lang expression evaluation)
    engine.go                          NEW (evaluate all rules, detect transitions)
    notifier.go                        NEW (email sending via go-mail)
  projections/                         NEW PACKAGE
    engine.go                          NEW (compound interest, income modeling)
  db/
    migrations/
      000002_account_display_name.*    NEW
      000003_alert_rules.*             NEW
      000004_projections.*             NEW
```

## New Frontend File Map

```
frontend/src/
  App.tsx                              MODIFY (3 new routes, nav links)
  api/client.ts                        MODIFY (new types + API functions)
  pages/
    Dashboard.tsx                      MODIFY (pass growth to PanelCard)
    Settings.tsx                       MODIFY (sync diagnostics, SMTP config, account names)
    NetWorthDrillDown.tsx              NEW
    Alerts.tsx                         NEW
    Projections.tsx                    NEW
  components/
    PanelCard.tsx                      MODIFY (growth badge, display_name, crypto grouping)
    StackedAreaChart.tsx               NEW
    ProjectionChart.tsx                NEW
    AlertRuleForm.tsx                  NEW
```

## New API Endpoints Summary

| Endpoint | Method | Feature | Handler File |
|----------|--------|---------|--------------|
| `/api/accounts/{id}/display-name` | PUT | Account Renaming | account_settings.go |
| `/api/balance-history/detailed` | GET | Net Worth Drill-Down | drilldown.go |
| `/api/sync/log` | GET | Sync Diagnostics | sync_diagnostics.go |
| `/api/alerts` | GET | Alert Rules | alerts.go |
| `/api/alerts` | POST | Alert Rules | alerts.go |
| `/api/alerts/{id}` | PUT | Alert Rules | alerts.go |
| `/api/alerts/{id}` | DELETE | Alert Rules | alerts.go |
| `/api/alerts/{id}/history` | GET | Alert Rules | alerts.go |
| `/api/projections/config` | GET | Projections | projections.go |
| `/api/projections/config` | PUT | Projections | projections.go |
| `/api/projections/income` | POST | Projections | projections.go |
| `/api/projections/income/{id}` | DELETE | Projections | projections.go |
| `/api/projections/compute` | GET | Projections | projections.go |

**Modified existing endpoints:**
- `GET /api/summary` -- adds growth fields
- `GET /api/accounts` -- adds display_name, crypto grouping
- `GET /api/settings` / `POST /api/settings` -- adds SMTP config keys

## Data Flow Changes

### Current Sync Flow (v1.0)

```
Timer fires -> SyncOnce() -> SimpleFIN fetch -> upsert accounts -> insert snapshots -> update sync_log -> done
```

### New Sync Flow (v1.1 with alerts)

```
Timer fires -> SyncOnce() -> SimpleFIN fetch -> upsert accounts -> insert snapshots -> update sync_log
                                                                                          |
                                                                          alerts.EvaluateAll()
                                                                                |
                                                             for each enabled rule:
                                                               evaluate expression
                                                               compare to last_state
                                                               if state changed:
                                                                 insert alert_history
                                                                 update rule.last_state
                                                                 send email notification
```

### Dashboard Load Flow (v1.1)

```
Browser -> GET /api/summary (now includes growth %)
        -> GET /api/accounts (now includes display_name, grouped crypto)
        -> GET /api/balance-history?days=30 (unchanged)
```

### New Page Flows

```
Net Worth Drill-Down:
  Browser -> GET /api/balance-history/detailed?days=365
  Response: per-account time-series (not aggregated)
  Frontend renders stacked area chart + per-account toggles

Alerts:
  Browser -> GET /api/alerts (list rules with state)
  User creates rule -> POST /api/alerts (backend validates expression with expr.Compile)
  Browser -> GET /api/alerts/{id}/history (view past triggers)

Projections:
  Browser -> GET /api/projections/config (load per-account APY, income allocations)
  User edits -> PUT /api/projections/config
  Browser -> GET /api/projections/compute?months=120 (10 years)
  Response: month-by-month projected balances per account + total
```

## Recommended Build Order

The order is driven by dependency analysis and incremental value delivery:

### Phase 1: Foundation (Account Renaming + Sync Diagnostics + Crypto Aggregation)

Build order within phase:
1. **Account Renaming** (Feature 2) -- First because it introduces `display_name` which all later features should use from day one. Migration + sync fix + settings UI + API endpoint.
2. **Sync Diagnostics** (Feature 5) -- Simplest feature. Read-only endpoint over existing `sync_log` table. One new handler, one Settings UI section. Quick win.
3. **Crypto Aggregation** (Feature 1) -- Modify accounts endpoint query logic. May or may not need a migration depending on approach. Touches one handler + PanelCard component.

**Rationale:** These three are independent, low-risk, and establish the data foundation (display names) for later features. Sync diagnostics gives immediate operational value.

### Phase 2: Insights (Growth Indicators + Net Worth Drill-Down)

Build order within phase:
4. **Growth Indicators** (Feature 3) -- Extends summary endpoint. Modify one handler + PanelCard. Small scope.
5. **Net Worth Drill-Down** (Feature 4) -- New page with detailed charts. One new handler + one new page + new chart component. Medium scope. Benefits from display_name (Phase 1) and growth indicators (conceptual alignment).

**Rationale:** Both features are read-only analytics over existing data. No new tables, no background processing, no external integrations. Natural to build together.

### Phase 3: Alert System (Alert Rules + Email Notifications)

Build order within phase:
6. **Alert Rules + Email** (Feature 6) -- This is the most complex feature. Requires: new tables (migration), new Go package (`internal/alerts/`), expr-lang dependency, go-mail dependency, SMTP configuration in settings, sync hook, CRUD endpoints, frontend page. Build in sub-phases:
   a. Alert rules CRUD (no evaluation) -- table + handlers + frontend
   b. Expression evaluator with expr-lang -- environment setup, compilation, evaluation
   c. Engine wiring into sync flow -- post-sync evaluation, state transition detection
   d. Email notifier -- go-mail integration, SMTP config in settings

**Rationale:** Isolated as its own phase because it introduces two new external dependencies, a new background processing hook, and is the only feature that sends data out (email). Needs focused testing.

### Phase 4: Projections (Projected Net Worth + Income Modeling)

Build order within phase:
7. **Projections** (Feature 7) -- New tables (migration), new Go package (`internal/projections/`), new endpoints, new frontend page. Build in sub-phases:
   a. Projection config CRUD -- tables + handlers + settings UI
   b. Projection engine -- compound interest computation, income allocation
   c. Projection page -- chart rendering, time horizon selector

**Rationale:** This is the most standalone feature. It reads current balances but otherwise works with its own data (APY rates, income allocations). No dependency on alerts or other v1.1 features. Saves the most complex UI for last.

### Dependency Graph

```
Feature 2 (Account Renaming) ─────┬──> Feature 4 (Drill-Down)
                                   ├──> Feature 6 (Alerts -- for display names in expressions)
                                   └──> Feature 7 (Projections -- for display names in config)

Feature 3 (Growth Indicators) ──> (nice-to-have context for Drill-Down, not blocking)

Feature 1 (Crypto Aggregation) ──> (standalone)
Feature 5 (Sync Diagnostics) ──> (standalone)
Feature 6 (Alerts) ──> (standalone, but sync hook must be careful)
Feature 7 (Projections) ──> (standalone)
```

## Architectural Patterns to Follow

### Pattern 1: Keep Handlers Thin

**What:** The existing codebase puts SQL queries directly in handlers. For v1.1, resist the urge to add more complexity there. For features with real business logic (alerts, projections), create dedicated packages.

**When to use:** Any feature with logic beyond "query DB, serialize JSON."

**Trade-offs:** More files, but testable without HTTP. The alert evaluator and projection engine should be pure functions that accept data and return results.

**Example:**
```go
// internal/alerts/evaluator.go
type Environment struct {
    Liquid      float64
    Savings     float64
    Investments float64
    NetWorth    float64
    Accounts    map[string]float64 // display_name -> balance
}

func Evaluate(expression string, env Environment) (bool, error) {
    program, err := expr.Compile(expression, expr.Env(env), expr.AsBool())
    if err != nil {
        return false, err
    }
    result, err := expr.Run(program, env)
    if err != nil {
        return false, err
    }
    return result.(bool), nil
}
```

### Pattern 2: Post-Sync Hook Pattern

**What:** After sync completes successfully, run additional processing (alert evaluation). Do not embed alert logic inside sync -- use a callback/hook pattern.

**When to use:** When background processing needs to chain.

**Example:**
```go
// In sync.go, after finalize:
if failed == 0 || fetched > 0 {
    if err := alerts.EvaluateAll(ctx, db); err != nil {
        slog.Error("sync: alert evaluation failed", "err", err)
        // Don't fail the sync for alert errors
    }
}
```

### Pattern 3: Expression Validation at Write Time

**What:** When the user saves an alert rule, compile the expression with `expr.Compile()` before storing it. Reject invalid expressions with a 400 error. This ensures every stored expression is guaranteed to be evaluable at runtime.

**When to use:** Alert rule creation/update.

**Trade-offs:** Slightly slower write (compile step), but prevents runtime failures during sync.

### Pattern 4: Decimal Strings for Money

**What:** The existing codebase stores balances as TEXT (decimal strings) and uses `shopspring/decimal` for arithmetic. All new features must follow this convention. Never use float64 for money storage or comparison.

**When to use:** Always, for any money-related field.

**Note for projections:** The projection engine can use float64 internally for compound interest math (acceptable for projections which are inherently approximate), but should format results back to 2-decimal strings for the API response.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Fat Migration Files

**What people do:** Put all schema changes in one migration file.

**Why it's wrong:** Can't roll back individual features. If alert tables need adjustment, you have to redo projections tables too.

**Do this instead:** One migration per logical feature. Account renaming = 000002. Alerts = 000003. Projections = 000004.

### Anti-Pattern 2: Evaluating Expressions with string matching / regex

**What people do:** Parse alert expressions like "liquid < 5000" with string splitting and if/else chains.

**Why it's wrong:** Fragile, limited operators, hard to extend. Users will want compound expressions like `liquid < 5000 AND savings > 10000`.

**Do this instead:** Use expr-lang/expr. It handles parsing, type checking, and safe evaluation. It prevents code injection and guarantees termination.

### Anti-Pattern 3: Sending Email Synchronously in Sync Flow

**What people do:** Send the email notification inline during sync, blocking the sync goroutine.

**Why it's wrong:** SMTP can be slow (2-10 seconds). If the SMTP server is down, the entire sync blocks or fails.

**Do this instead:** Alert evaluation and email sending should be best-effort. Log errors but never fail the sync. Consider sending email in a separate goroutine with a timeout, or at minimum wrap email sending with a 10-second context deadline.

### Anti-Pattern 4: Overwriting display_name on Sync

**What people do:** The upsert in `processAccount` uses `ON CONFLICT DO UPDATE SET name=excluded.name, ...` which would clobber `display_name` if included.

**Why it's wrong:** User's carefully chosen display names disappear on next sync.

**Do this instead:** Explicitly exclude `display_name` from the `ON CONFLICT DO UPDATE` clause. The sync should never touch user-configured fields.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| SimpleFIN Bridge | Existing -- no change for v1.1 | Unaffected by new features |
| SMTP Server | New -- outbound email via go-mail | User-configured in Settings. Store credentials in `settings` table (encrypted at rest is out of scope for single-user self-hosted). Support standard SMTP (port 587/465) with STARTTLS. |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| sync -> alerts | Direct function call after sync completion | `alerts.EvaluateAll(ctx, db)` called at end of `SyncOnce` |
| alerts -> email | Direct function call within alert engine | `notifier.Send(ctx, rule, state)` with context deadline |
| handlers -> alerts pkg | Import for expression validation | `alerts.Validate(expression)` called in POST/PUT handlers |
| handlers -> projections pkg | Import for computation | `projections.Compute(config, months)` called in GET handler |

### New Go Dependencies

| Package | Purpose | Import Path |
|---------|---------|-------------|
| expr-lang/expr | Safe expression evaluation for alert rules | `github.com/expr-lang/expr` |
| go-mail | SMTP email sending | `github.com/wneessen/go-mail` |

No new frontend dependencies are needed. Recharts (already installed) handles all new chart types.

## Sources

- [expr-lang/expr](https://github.com/expr-lang/expr) -- Expression language for Go, safe evaluation, used by Google Cloud, Uber, ByteDance
- [wneessen/go-mail](https://github.com/wneessen/go-mail) -- Modern Go email library, fork of stdlib net/smtp with extensions
- [golang-migrate/migrate](https://github.com/golang-migrate/migrate) -- Already used in project for SQLite migrations
- [shopspring/decimal](https://github.com/shopspring/decimal) -- Already used in project for money arithmetic
- Existing codebase analysis (all files read directly from repository)

---
*Architecture research for: v1.1 feature integration into existing Go/React/SQLite personal finance dashboard*
*Researched: 2026-03-15*
