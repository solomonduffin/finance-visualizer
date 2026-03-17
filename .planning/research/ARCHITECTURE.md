# Architecture Research

**Domain:** Personal finance dashboard -- next-step feature integration
**Researched:** 2026-03-17
**Confidence:** HIGH (existing codebase fully inspected, SimpleFIN protocol verified)

## Current System Overview

```
                         Existing Architecture (v1.1)
+----------------------------------------------------------------------+
|                        React Frontend (Vite)                          |
|  Pages: Dashboard, NetWorth, Alerts, Projections, Settings, Login     |
|  Components: AccountsSection, BalanceLineChart, StackedAreaChart,     |
|              AlertRuleForm, ProjectionChart, RateConfigTable, etc.     |
+----------------------------------+-----------------------------------+
                                   | REST API (fetch w/ JWT cookie)
+----------------------------------+-----------------------------------+
|                     Go Backend (chi router)                           |
|  Handlers: auth, summary, accounts, history, growth, networth,        |
|            alerts, projections, groups, settings, synclog, email       |
|  Middleware: httplog, CORS, JWT (jwtauth), rate-limit (httprate)       |
+----------------------------------+-----------------------------------+
                                   | database/sql
+----------------------------------+-----------------------------------+
|                     SQLite (WAL mode)                                  |
|  Tables: accounts, balance_snapshots, holdings, settings,             |
|          account_groups, group_members, alert_rules, alert_history,    |
|          projection_account_settings, projection_holding_settings,     |
|          projection_income_settings, sync_log                         |
+----------------------------------------------------------------------+
       ^
       | Goroutine cron (daily) + on-demand POST /api/sync/now
+----------------------------------------------------------------------+
|                     Sync Pipeline                                      |
|  SimpleFIN -> FetchAccountsWithHoldings -> processAccount (upsert     |
|  account + snapshot) -> persistHoldings -> soft-delete stale ->        |
|  restore returning -> EvaluateAll alerts                              |
+----------------------------------------------------------------------+
```

### Component Responsibilities

| Component | Responsibility | Key Pattern |
|-----------|----------------|-------------|
| `internal/sync/sync.go` | Orchestrate fetch, upsert, soft-delete lifecycle | `SyncOnce()` with mutex serialization |
| `internal/simplefin/client.go` | HTTP client for SimpleFIN protocol | Returns `AccountSet` with `[]Account` |
| `internal/api/handlers/*.go` | One file per feature domain, closure-based handlers | `func HandlerName(db *sql.DB) http.HandlerFunc` |
| `internal/api/router.go` | Route registration, middleware stack | Public vs protected route groups |
| `internal/alerts/` | Expression evaluation engine | `expr-lang/expr` sandboxed evaluation |
| `internal/db/` | Migration runner, connection setup | golang-migrate with numbered SQL files |
| `frontend/src/api/client.ts` | Typed fetch wrapper for all endpoints | One function per endpoint, credentials: 'include' |
| `frontend/src/pages/*.tsx` | Full-page views (Dashboard, NetWorth, etc.) | Fetch on mount, loading/error states |
| `frontend/src/components/*.tsx` | Reusable UI pieces (charts, cards, forms) | Props-driven, test file alongside |

## Feature Integration Analysis

### Feature 1: Transaction Data Ingestion

**What:** Capture and store transaction data from SimpleFIN alongside existing balance snapshots.

**Critical discovery:** SimpleFIN already provides transactions. The current codebase calls `FetchAccountsWithHoldings()` which does NOT set `balances-only=1`. The `Account` struct already has a `Holdings` field parsed from the response. Transaction data is present in the JSON response but the Go struct does NOT include a `Transactions` field -- it is silently dropped during `json.Decode`. Enabling transaction capture requires adding a `Transactions` field to the `Account` struct and persisting the data.

**SimpleFIN Transaction object** (from protocol spec):
```json
{
  "id": "12394832938403",
  "posted": 793090572,
  "amount": "-33293.43",
  "description": "Uncle Frank's Bait Shop",
  "pending": true,
  "extra": { "category": "food" }
}
```

**New table (migration 000007):**

```sql
CREATE TABLE IF NOT EXISTS transactions (
    id              TEXT NOT NULL,
    account_id      TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    posted          DATETIME NOT NULL,
    amount          TEXT NOT NULL,        -- shopspring/decimal string
    description     TEXT NOT NULL DEFAULT '',
    pending         INTEGER NOT NULL DEFAULT 0,
    category        TEXT,                 -- from SimpleFIN extra.category
    user_category   TEXT,                 -- user override (future)
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, account_id)          -- SimpleFIN IDs are unique per account
);

CREATE INDEX IF NOT EXISTS idx_transactions_account_posted
    ON transactions(account_id, posted DESC);

CREATE INDEX IF NOT EXISTS idx_transactions_posted
    ON transactions(posted DESC);

CREATE INDEX IF NOT EXISTS idx_transactions_category
    ON transactions(COALESCE(user_category, category));
```

**Modified files:**

| File | Change |
|------|--------|
| `internal/simplefin/client.go` | Add `Transaction` struct, add `Transactions []Transaction` to `Account` |
| `internal/sync/sync.go` | Add `persistTransactions()` function, call from `SyncOnce` loop |
| `internal/api/handlers/transactions.go` | NEW: `GetTransactions`, search handler |
| `internal/api/router.go` | Add `GET /api/transactions` |
| `frontend/src/api/client.ts` | Add `TransactionItem`, `TransactionsResponse`, `getTransactions()` |

**Sync pipeline change:**

```
Current:  processAccount -> persistHoldings
Proposed: processAccount -> persistHoldings -> persistTransactions
```

`persistTransactions` should use `INSERT ... ON CONFLICT(id, account_id) DO UPDATE` (upsert), not DELETE + INSERT. Unlike holdings (which are replace-all because SimpleFIN always returns the full set), transactions have a 90-day rolling window. Using upsert preserves older transactions that SimpleFIN no longer returns.

**New endpoints:**

| Method | Path | Query Params | Purpose |
|--------|------|-------------|---------|
| GET | `/api/transactions` | `?account_id=X&days=N&limit=50&offset=0` | Paginated transaction list |
| GET | `/api/transactions/search` | `?q=term&days=90` | Text search on description |

**New frontend:**

| Component | Type | Purpose |
|-----------|------|---------|
| `pages/Transactions.tsx` | Page | Transaction list with filters |
| `components/TransactionRow.tsx` | Component | Single transaction display |
| `components/TransactionFilters.tsx` | Component | Account/date/search filter bar |

**Data volume estimate:** Single user, ~10-50 transactions/day across all accounts = ~1,500/month = ~18,000/year. SQLite handles this trivially.

---

### Feature 2: Spending Analytics

**What:** Categorize transactions and show spending breakdowns (by category, by merchant, over time).

**Depends on:** Transaction data (Feature 1).

**Architecture approach:** Server-side aggregation, client-side charting. The existing pattern of returning pre-aggregated JSON (like `GetSummary`, `GetNetWorth`) works well. Do NOT send raw transactions to the frontend for aggregation -- the handler should run SQL GROUP BY queries.

**New table (migration 000008):**

```sql
CREATE TABLE IF NOT EXISTS category_rules (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    pattern     TEXT NOT NULL,            -- substring match on description
    category    TEXT NOT NULL,            -- target category name
    priority    INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
    name        TEXT PRIMARY KEY,
    icon        TEXT,                     -- emoji or icon identifier
    color       TEXT,                     -- hex color for charts
    sort_order  INTEGER NOT NULL DEFAULT 0
);
```

**Why server-side categorization over client-side:**
- Consistent across all views (spending page, cashflow, alerts)
- Can run during sync (batch categorize new transactions)
- `user_category` column on transactions table acts as override
- Category rules are simple substring match, evaluated in priority order
- No ML needed -- manual rules + corrections work well for single users

**New endpoints:**

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/spending` | `?days=30&group_by=category` -- aggregated spending |
| GET | `/api/spending/trends` | `?months=6` -- monthly spending by category |
| GET | `/api/categories` | List all categories with rules |
| POST | `/api/categories` | Create/update category |
| PUT | `/api/transactions/{id}/category` | Override transaction category |
| POST | `/api/category-rules` | Create auto-categorization rule |

**New frontend:**

| Component | Type | Purpose |
|-----------|------|---------|
| `pages/Spending.tsx` | Page | Spending analytics dashboard |
| `components/SpendingDonut.tsx` | Component | Category breakdown (Recharts PieChart) |
| `components/SpendingTrends.tsx` | Component | Monthly trends (Recharts BarChart, stacked) |
| `components/CategoryManager.tsx` | Component | Category rules CRUD |

**SQL aggregation pattern:**

```sql
SELECT
    strftime('%Y-%m', posted) AS month,
    COALESCE(user_category, category, 'Uncategorized') AS cat,
    SUM(CAST(amount AS REAL)) AS total
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE a.hidden_at IS NULL
  AND CAST(t.amount AS REAL) < 0
  AND t.posted >= date('now', '-6 months')
GROUP BY month, cat
ORDER BY month, total;
```

---

### Feature 3: Investment Performance Tracking

**What:** Track investment returns over time using holdings history snapshots.

**Depends on:** None (builds on existing holdings table). Independent of transactions.

**Key insight:** The current `persistHoldings` function does DELETE + INSERT (full replacement). Historical holdings data is lost every sync. To track performance, holdings snapshots must be preserved over time, just like `balance_snapshots` preserves daily balances.

**New table (migration 000009):**

```sql
CREATE TABLE IF NOT EXISTS holdings_snapshots (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    holding_id      TEXT NOT NULL,
    account_id      TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    symbol          TEXT,
    description     TEXT NOT NULL,
    shares          TEXT,
    market_value    TEXT NOT NULL,
    cost_basis      TEXT,
    snapshot_date   DATE NOT NULL,
    UNIQUE(holding_id, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_holdings_snapshots_account_date
    ON holdings_snapshots(account_id, snapshot_date DESC);

CREATE INDEX IF NOT EXISTS idx_holdings_snapshots_symbol_date
    ON holdings_snapshots(symbol, snapshot_date DESC);
```

**Modified files:**

| File | Change |
|------|--------|
| `internal/sync/sync.go` | `persistHoldings` also inserts into `holdings_snapshots` (upsert on holding_id + date) |
| `internal/api/handlers/holdings.go` | NEW: `GetHoldings`, `GetHoldingsHistory` |

**Performance calculation:** Simple return, not time-weighted. TWR requires tracking cash flows (purchases/sales) that SimpleFIN does not explicitly provide. The user already uses ProjectionLab for sophisticated analysis.

```
Return % = (current_market_value - cost_basis) / cost_basis * 100
Daily change = today_market_value - yesterday_market_value
```

**New endpoints:**

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/holdings` | Current holdings with return calculations |
| GET | `/api/holdings/history` | `?symbol=VTSAX&days=90` -- value over time |

**New frontend:**

| Component | Type | Purpose |
|-----------|------|---------|
| `pages/Investments.tsx` | Page | Holdings overview with performance |
| `components/HoldingsTable.tsx` | Component | Current value, cost basis, return % |
| `components/HoldingChart.tsx` | Component | Market value over time (LineChart) |

---

### Feature 4: Data Export

**What:** Export financial data as CSV/JSON for external tools, tax prep, or backup.

**Depends on:** Feature 1 (transactions) for full value, but balance/account export works independently.

**Architecture:** Export handlers are stateless -- query DB, stream response. No new tables. Use `Content-Disposition: attachment` for browser download.

**New endpoints:**

| Method | Path | Content-Type | Purpose |
|--------|------|-------------|---------|
| GET | `/api/export/balances` | `text/csv` | Balance history CSV |
| GET | `/api/export/transactions` | `text/csv` | Transaction history CSV |
| GET | `/api/export/holdings` | `text/csv` | Current holdings CSV |
| GET | `/api/export/all` | `application/json` | Full database export as JSON |

**New handler file:** `internal/api/handlers/export.go`

**CSV streaming pattern:**

```go
func ExportBalances(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/csv")
        w.Header().Set("Content-Disposition",
            `attachment; filename="balances.csv"`)
        writer := csv.NewWriter(w)
        defer writer.Flush()
        writer.Write([]string{"date", "account", "balance"})
        rows, _ := db.Query(`SELECT ...`)
        defer rows.Close()
        for rows.Next() { /* scan and write */ }
    }
}
```

**Frontend:** Download buttons in Settings page. `window.location.href = '/api/export/balances'` triggers download with JWT cookie. No new page needed.

| Component | Type | Purpose |
|-----------|------|---------|
| `components/ExportSection.tsx` | Component | Export buttons within Settings page |

---

### Feature 5: Goal Tracking

**What:** Set financial goals (emergency fund, down payment, debt payoff) and track progress.

**Depends on:** None (uses existing account/balance data).

**New table (migration 000010):**

```sql
CREATE TABLE IF NOT EXISTS goals (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    target_amount   TEXT NOT NULL,
    target_date     DATE,
    goal_type       TEXT NOT NULL DEFAULT 'savings'
                    CHECK(goal_type IN ('savings', 'debt_payoff', 'net_worth')),
    operands        TEXT NOT NULL DEFAULT '[]',
    icon            TEXT,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Architectural win -- reuse the alert operand engine:** The existing alert system has a flexible operand structure (bucket/group/account selectors with +/- operators) and `EvaluateExpression` logic. Goals use the same `operands` JSON format. Progress is `current_value / target_amount * 100`. This avoids building a second account-selection mechanism. Consider extracting the shared evaluation logic to `internal/eval/` so both alerts and goals import it.

**New endpoints:**

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/goals` | List goals with current progress |
| POST | `/api/goals` | Create a goal |
| PUT | `/api/goals/{id}` | Update a goal |
| DELETE | `/api/goals/{id}` | Delete a goal |

**New frontend:**

| Component | Type | Purpose |
|-----------|------|---------|
| `pages/Goals.tsx` | Page | Goals dashboard with progress bars |
| `components/GoalCard.tsx` | Component | Single goal with progress ring |
| `components/GoalForm.tsx` | Component | Goal form (reuse operand selector from AlertRuleForm) |

---

### Feature 6: Recurring Transaction Detection

**What:** Identify recurring transactions (subscriptions, bills, income) from transaction history.

**Depends on:** Feature 1 (transactions). Needs 60+ days of history for reliable detection.

**Architecture:** Batch detection during sync, stored in a table. NOT real-time.

**Detection algorithm:**
1. Group transactions by normalized description (lowercase, strip variable parts like dates/reference numbers)
2. For each group with 3+ occurrences, compute average interval between postings
3. If interval is consistent (within +/- 5 days) and near a known period (weekly/biweekly/monthly/quarterly/annual), flag as recurring
4. Store with confidence score; user can confirm or dismiss

**New table (migration 000011):**

```sql
CREATE TABLE IF NOT EXISTS recurring_patterns (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    description_pattern TEXT NOT NULL,
    account_id          TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    avg_amount          TEXT NOT NULL,
    frequency           TEXT NOT NULL CHECK(frequency IN
                        ('weekly','biweekly','monthly','quarterly','annual')),
    confidence          REAL NOT NULL DEFAULT 0.0,
    next_expected       DATE,
    is_income           INTEGER NOT NULL DEFAULT 0,
    user_confirmed      INTEGER NOT NULL DEFAULT 0,
    user_dismissed      INTEGER NOT NULL DEFAULT 0,
    last_seen_at        DATETIME,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_recurring_patterns_account
    ON recurring_patterns(account_id);
```

**Sync pipeline integration:**

```
processAccount -> persistHoldings -> persistTransactions
  -> (after all accounts) detectRecurringPatterns -> EvaluateAll alerts
```

**New endpoints:**

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/recurring` | List detected recurring transactions |
| PATCH | `/api/recurring/{id}` | Confirm or dismiss a pattern |

**Frontend:** `components/RecurringList.tsx` -- lives within Spending or Transactions page.

---

### Feature 7: Cashflow Forecasting

**What:** Project future cash position based on recurring transactions and income.

**Depends on:** Feature 1 (transactions), Feature 6 (recurring detection). Capstone feature.

**Architecture:** Client-side calculation, same pattern as Projections page. Server provides inputs; frontend computes the forecast with `useMemo`. Enables interactive "what-if" (toggle subscriptions on/off).

**No new tables.** Consumes: `recurring_patterns`, `balance_snapshots`, optionally `goals`.

**New endpoint:**

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/cashflow/inputs` | Liquid balance + active recurring patterns + goals |

**New frontend:**

| Component | Type | Purpose |
|-----------|------|---------|
| `pages/Cashflow.tsx` | Page | Cashflow forecast with interactive chart |
| `components/CashflowChart.tsx` | Component | Projected balance (Recharts AreaChart) |
| `components/CashflowAssumptions.tsx` | Component | Toggle recurring items for what-if |

---

## Recommended Project Structure (New Files)

```
internal/
  api/handlers/
    transactions.go        # GET /api/transactions, search
    spending.go            # GET /api/spending, /spending/trends
    categories.go          # CRUD for categories and rules
    holdings_perf.go       # GET /api/holdings, /holdings/history
    export.go              # GET /api/export/*
    goals.go               # CRUD for goals
    recurring.go           # GET/PATCH /api/recurring
    cashflow.go            # GET /api/cashflow/inputs
  sync/
    transactions.go        # persistTransactions()
    recurring.go           # detectRecurringPatterns()
  eval/                    # NEW: shared expression evaluation
    eval.go                # Extracted from internal/alerts
  db/migrations/
    000007_transactions.{up,down}.sql
    000008_spending_categories.{up,down}.sql
    000009_holdings_snapshots.{up,down}.sql
    000010_goals.{up,down}.sql
    000011_recurring_patterns.{up,down}.sql

frontend/src/
  pages/
    Transactions.tsx
    Spending.tsx
    Investments.tsx
    Goals.tsx
    Cashflow.tsx
  components/
    TransactionRow.tsx
    TransactionFilters.tsx
    SpendingDonut.tsx
    SpendingTrends.tsx
    CategoryManager.tsx
    HoldingsTable.tsx
    HoldingChart.tsx
    GoalCard.tsx
    GoalForm.tsx
    RecurringList.tsx
    CashflowChart.tsx
    CashflowAssumptions.tsx
    ExportSection.tsx
```

### Structure Rationale

- **One handler file per feature domain:** Matches existing pattern (accounts.go, alerts.go, projections.go).
- **Sync sub-modules:** transactions.go and recurring.go run as part of the sync pipeline.
- **Shared eval package:** Alert operand evaluation is useful for goals too. Extract to avoid circular dependency between alerts and goals.
- **One page per top-level nav item:** Matches current structure.
- **Flat component structure:** Same as existing codebase.

## Data Flow

### Transaction Ingestion Flow

```
SimpleFIN /accounts (without balances-only=1)
    |
    v
simplefin.FetchAccountsWithHoldings()  -- already called, zero extra API calls
    |
    v
AccountSet.Accounts[].Transactions      -- NEW: parse from JSON (currently dropped)
    |
    v
sync.persistTransactions()              -- NEW: upsert to transactions table
    |
    v
sync.categorizeNewTransactions()        -- NEW: apply category rules
    |
    v
sync.detectRecurringPatterns()          -- NEW: batch pattern analysis
    |
    v
alerts.EvaluateAll()                    -- existing: unchanged
```

### Spending Analytics Flow

```
GET /api/spending?days=30&group_by=category
    |
    v
SQL: GROUP BY COALESCE(user_category, category)
     WHERE amount < 0 AND account hidden_at IS NULL
    |
    v
JSON: { categories: [{name, total, count}], total, period }
    |
    v
SpendingDonut + SpendingTrends components
```

### Goal Progress Flow

```
GET /api/goals
    |
    v
For each goal: eval.EvaluateOperands(goal.operands)
    -> progress_pct = current / target * 100
    -> if target_date: days_remaining, on_track boolean
    |
    v
JSON: [{ name, target, current, progress_pct, on_track }]
    |
    v
GoalCard components with progress bars
```

### Cashflow Forecast Flow (Client-Side)

```
GET /api/cashflow/inputs
    |
    v
JSON: { liquid_balance, recurring: [...], goals: [...] }
    |
    v
useMemo(() => {
  for each day in forecast_horizon:
    balance += sum(recurring income due today)
    balance -= sum(recurring expenses due today)
    points.push({ date, balance })
})
    |
    v
CashflowChart with goal milestone markers
```

## Suggested Build Order

Dependencies drive the order. You cannot analyze what you have not captured.

```
Phase 1: Transaction Ingestion (Feature 1)
    |     Foundation -- spending, recurring, cashflow all need this
    |
    +---> Phase 2a: Data Export (Feature 4)            [independent]
    |     Low effort, immediate user value, deferred from v1.1
    |
    +---> Phase 2b: Investment Performance (Feature 3) [independent]
    |     Uses existing holdings, independent of transactions
    |
    +---> Phase 2c: Goal Tracking (Feature 5)          [independent]
    |     Reuses alert operand engine, no transaction dep
    |
Phase 3: Spending Analytics (Feature 2)
    |     Requires transactions + introduces categorization
    |
Phase 4: Recurring Detection (Feature 6)
    |     Requires 60+ days of transaction history
    |
Phase 5: Cashflow Forecasting (Feature 7)
          Capstone -- needs recurring patterns + balances + goals
```

**Rationale:**
1. **Transaction Ingestion first** -- three features are blocked without it.
2. **Export, Investment Performance, Goal Tracking are parallel** -- no dependencies between them. Export is low-effort; Investment Performance builds on existing holdings; Goal Tracking reuses the alert engine.
3. **Spending Analytics before Recurring Detection** -- categorization infrastructure enriches recurring detection; recurring also needs sufficient history.
4. **Cashflow Forecasting last** -- integrates everything. Building it before its inputs exist means rebuilding it later.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| Single user (current) | No changes. SQLite handles everything trivially. |
| 5 years (~90K transactions) | Ensure `LIMIT/OFFSET` on all list endpoints. Still fast with indexes. |
| 10+ years | Spending aggregation may slow. Pre-compute monthly aggregates in a cache table updated during sync. |

### Scaling Priorities

1. **First bottleneck:** Transaction table size. Mitigation: proper indexes + pagination.
2. **Second bottleneck:** Spending GROUP BY over full table. Mitigation: monthly aggregate cache.

For a single-user self-hosted app, these are years away.

## Anti-Patterns

### Anti-Pattern 1: Fetching All Transactions Client-Side

**What people do:** Return all transactions and paginate/filter in JavaScript.
**Why it is wrong:** Transaction volume grows indefinitely. 50K+ rows to the browser is slow.
**Do this instead:** Server-side pagination, filtering, and sorting via query params.

### Anti-Pattern 2: Real-Time Recurring Detection

**What people do:** Run pattern detection on every page load.
**Why it is wrong:** O(n) scan of all transactions on the read path. Results change only when new transactions arrive.
**Do this instead:** Batch detection during daily sync. Store in `recurring_patterns`. Read path is a simple SELECT.

### Anti-Pattern 3: Client-Side Category Aggregation

**What people do:** Send raw transactions to frontend, aggregate with JavaScript reduce().
**Why it is wrong:** Duplicates logic (export needs same aggregation). Wastes bandwidth.
**Do this instead:** SQL GROUP BY in spending handler. Return pre-aggregated data. Matches existing pattern (GetSummary, GetNetWorth).

### Anti-Pattern 4: Separate SimpleFIN Fetch for Transactions

**What people do:** Second API call to SimpleFIN for transactions.
**Why it is wrong:** 24 requests/day quota. The existing `FetchAccountsWithHoldings()` already receives transactions (balances-only is NOT set). Data is in the response, just not parsed.
**Do this instead:** Add `Transactions` field to Go struct. Zero additional API calls.

### Anti-Pattern 5: Float64 for Financial Amounts

**What people do:** Store amounts as REAL or use float64.
**Why it is wrong:** Rounding errors. This project already uses `shopspring/decimal` and TEXT storage.
**Do this instead:** Continue existing pattern. TEXT in SQLite, shopspring/decimal in Go, strings to frontend.

### Anti-Pattern 6: Deep Category Hierarchies

**What people do:** Build nested category trees (Food > Restaurants > Fast Food > McDonald's).
**Why it is wrong:** Over-engineering for single user. Recursive CTEs for rollups. Users do not maintain deep trees.
**Do this instead:** Single-level categories. "Food & Dining" is sufficient. Optional parent_id if two levels ever needed.

## Integration Points

### SimpleFIN Protocol (Extended)

| Data | Current Status | Change Needed |
|------|---------------|---------------|
| Account balances | Captured daily in `balance_snapshots` | None |
| Holdings | Captured (replace-all) in `holdings` | Also insert into `holdings_snapshots` |
| Transactions | **Available in response but ignored** | Add `Transaction` struct, persist in sync |
| `extra.category` | Not captured | Map to `category` column |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| Sync -> Transaction storage | Direct SQL in same goroutine | Follow `persistHoldings` pattern |
| Sync -> Recurring detection | Function call after all accounts | `detectRecurringPatterns(ctx, db)` |
| Alert engine <-> Goal engine | Shared `internal/eval/` package | Extract operand evaluation |
| Spending handler -> Categories | SQL COALESCE at query time | `COALESCE(user_category, category)` |
| Cashflow page -> Multiple APIs | Parallel frontend fetch | `Promise.all([...])` |

### Navigation Restructuring

Adding 5 pages to existing 5 requires nav reorganization:

```
Overview:    Dashboard, Net Worth
Money:       Transactions, Spending, Cashflow
Investments: Holdings, Projections
Planning:    Goals, Alerts
System:      Settings
```

Plan this restructuring for the first phase that adds a new page.

## Sources

- [SimpleFIN Protocol Specification](https://www.simplefin.org/protocol.html) -- Transaction fields: id, posted, amount, description, pending, extra
- [SimpleFIN Protocol on GitHub](https://github.com/simplefin/simplefin.github.com/blob/master/protocol.md) -- Account includes `transactions` array
- [SimpleFIN Developer Guide](https://beta-bridge.simplefin.org/info/developers) -- 24 req/day quota, 90-day window
- [SQL Habit: Recurring Payment Detection](https://www.sqlhabit.com/blog/how-to-detect-recurring-payments-with-sql)
- Existing codebase: `internal/simplefin/client.go`, `internal/sync/sync.go`, `internal/api/router.go`, all 6 migrations, `frontend/src/api/client.ts`

---
*Architecture research for: finance-visualizer v1.2 feature integration*
*Researched: 2026-03-17*
