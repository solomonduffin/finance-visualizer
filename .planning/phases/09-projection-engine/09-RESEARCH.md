# Phase 9: Projection Engine - Research

**Researched:** 2026-03-17
**Domain:** Financial projection engine (client-side calculation), Recharts charting, SQLite persistence, SimpleFIN holdings data
**Confidence:** HIGH

## Summary

Phase 9 builds a forward-looking net worth projection page with per-account growth rates, compound/simple interest toggles, income allocation modeling, and holdings detail for investment accounts. The implementation spans four domains: (1) a new database schema for projection settings and holdings, (2) REST API endpoints for CRUD on projection settings and fetching accounts with holdings, (3) a client-side projection calculation engine, and (4) a Recharts-based projection chart with supporting UI components.

The existing codebase provides strong foundations: Recharts 3.8.0 already supports `ComposedChart`, `ReferenceLine`, `Area`, and `Line` with `strokeDasharray` for the solid-to-dashed chart transition. The Go backend follows a clear handler pattern with `chi` router, SQLite migrations via `golang-migrate`, and `shopspring/decimal` for financial arithmetic. The frontend uses Vitest + React Testing Library for tests, and the backend uses Go's built-in `testing` package with `httptest`.

**Primary recommendation:** Structure implementation in four waves: (1) database migration + API endpoints for projection settings and holdings, (2) SimpleFIN client extension to fetch holdings data, (3) client-side projection calculation engine, (4) frontend page with chart and configuration UI. The projection calculation engine should run entirely client-side (no server-side calculation endpoint) since the math is straightforward and this enables the instant-feedback "what-if" interaction pattern.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Single aggregate net worth line chart (not per-panel stacked area)
- Historical data shown as solid line (recent ~6 months), projection continues as dashed line from today's date
- Clear visual break at the "now" boundary -- solid to dashed transition
- Time horizon selector: segmented control with presets (1y, 5y, 10y, 20y) plus a "Custom" button that reveals a year input field
- Chart includes historical context for a "where you've been to where you're going" narrative
- Built with Recharts (consistent with existing StackedAreaChart from Phase 7)
- Configuration table lives directly on the /projections page below the chart (self-contained workflow)
- Single table grouped by panel type (Liquid / Savings / Investments) with columns: account name, APY input, compound/simple toggle, include/exclude checkbox
- APY input is annual percentage only (no monthly/quarterly unit selector)
- Per-account compound/simple toggle (not global)
- Accounts with 0% or no rate set appear as flat lines at current balance in projection (not excluded)
- Chart updates live with ~500ms debounce as user adjusts any rate or toggle
- Investment accounts WITH holdings data from SimpleFIN: each holding gets its own APY/rate input and compound/simple toggle. Account row expands to reveal holdings.
- Investment accounts WITHOUT holdings data: rate input is at the account level (fallback behavior)
- Savings and Liquid accounts: always account-level rate (no holdings concept)
- Expand/collapse pattern reuses chevron pattern from account groups (Phase 7)
- Separate collapsible "Income Modeling" section below the account rate table
- Toggle to enable/disable income modeling -- when off, projection uses growth rates only
- Inputs: annual income and monthly savings percentage
- Allocation via percentage inputs -- must sum to 100%
- For investment accounts WITH holdings: allocation targets individual holdings
- Allocation only shows accounts/holdings that are checked "include" in rate table
- New route: /projections
- "Projections" link added to top nav bar between Alerts and Settings

### Claude's Discretion
- Exact chart styling (colors, gradients, line thickness, tooltip design) -- UI-SPEC already defines these
- Rate table row styling and responsive layout -- UI-SPEC already defines these
- Percentage slider/input component design for income allocation -- UI-SPEC already defines these
- Database schema for projection settings (rates, toggles, income config) -- researched below
- API endpoint design for projection settings CRUD -- researched below
- How to fetch and store holdings data from SimpleFIN -- researched below
- Loading and error state designs -- UI-SPEC already defines these
- Projection calculation engine (monthly compounding math, income distribution logic) -- researched below

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| PROJ-01 | User can set APY per savings account and expected growth rate per investment account | Database schema for `projection_account_settings` and `projection_holding_settings` tables; API endpoints for GET/PUT projection settings; RateConfigTable component from UI-SPEC |
| PROJ-02 | User can toggle reinvestment (compound vs simple) per account | `compound` boolean column in settings tables; miniature toggle component from UI-SPEC |
| PROJ-03 | User can enable/disable which accounts are included in the projection | `included` boolean column in settings tables; checkbox in RateConfigTable |
| PROJ-04 | User can model income: annual amount, monthly savings %, and per-account allocation | `projection_income_settings` table with JSON allocation field; IncomeModelingSection component from UI-SPEC |
| PROJ-05 | Projection chart shows projected net worth over a custom time horizon | Client-side projection engine using compound/simple interest formulas; ProjectionChart component with Recharts ComposedChart; HorizonSelector component |
| PROJ-06 | All projection settings persist in the database across sessions | Migration 000005 creates projection settings tables; auto-save via API PATCH with 500ms debounce |
| PROJ-07 | Projections page accessible from main navigation | NavBar modification in App.tsx; new /projections route |
| PROJ-08 | Investment accounts display available holdings detail from SimpleFIN where supported | SimpleFIN client extension to remove `balances-only=1` or add separate holdings fetch; `holdings` table in SQLite; sync process extension |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Recharts | 3.8.0 (installed) | Projection chart with ComposedChart, Line, Area, ReferenceLine | Already used for BalanceLineChart and StackedAreaChart; supports all needed features |
| React | 19.2.4 (installed) | UI framework | Already in use |
| react-router-dom | 7.13.1 (installed) | /projections route | Already in use |
| Tailwind CSS | 4.2.1 (installed) | Styling | Already in use |
| Go chi | 5.2.5 (installed) | API routing | Already in use |
| shopspring/decimal | 1.4.0 (installed) | Financial arithmetic for stored rates | Already in use for all financial calculations |
| golang-migrate | 4.19.1 (installed) | Database migrations | Already in use |
| modernc.org/sqlite | 1.46.1 (installed) | SQLite driver | Already in use |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Vitest | 3.2.1 (installed) | Frontend unit testing | All component and utility tests |
| @testing-library/react | 16.3.2 (installed) | Component rendering in tests | All frontend component tests |
| @testing-library/user-event | 14.6.1 (installed) | User interaction simulation in tests | Toggle, input, checkbox interaction tests |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Client-side projection calculation | Server-side projection endpoint | Server-side would add latency to the "what-if" loop; not worth it for deterministic math |
| Recharts ComposedChart | Separate charts for historical/projected | Single ComposedChart handles the seamless solid-to-dashed transition correctly |
| JSON blob for all settings | Normalized tables | Normalized tables (per-account rows) enable easier querying and partial updates |

**Installation:**
No new packages needed. All libraries are already installed.

## Architecture Patterns

### Recommended Project Structure
```
internal/
  api/handlers/
    projections.go          # GET/PUT projection settings, GET accounts-with-holdings
    projections_test.go     # Handler tests
  db/migrations/
    000005_projection_settings.up.sql    # New migration
    000005_projection_settings.down.sql  # Rollback
  simplefin/
    client.go               # Extended: add Holdings struct, modify FetchAccounts
  sync/
    sync.go                 # Extended: persist holdings data during sync

frontend/src/
  pages/
    Projections.tsx          # Main page component
    Projections.test.tsx     # Page tests
  components/
    ProjectionChart.tsx      # Recharts ComposedChart with solid/dashed lines
    ProjectionChart.test.tsx
    HorizonSelector.tsx      # Segmented control for time horizon
    HorizonSelector.test.tsx
    RateConfigTable.tsx      # Account rate configuration table
    RateConfigTable.test.tsx
    HoldingsRow.tsx          # Expandable holdings for investment accounts
    HoldingsRow.test.tsx
    IncomeModelingSection.tsx # Collapsible income modeling
    IncomeModelingSection.test.tsx
    AllocationRow.tsx         # Single allocation target row
    AllocationRow.test.tsx
  utils/
    projection.ts            # Pure projection calculation engine
    projection.test.ts       # Math unit tests (critical)
  api/
    client.ts                # Extended: projection settings endpoints
```

### Pattern 1: Database Schema for Projection Settings

**What:** Three new tables store projection configuration (account rates, holding rates, income settings). Settings are per-account/per-holding with auto-save semantics.

**When to use:** All projection settings persistence (PROJ-06).

**Schema design:**

```sql
-- Migration 000005_projection_settings.up.sql

-- Per-account projection settings
CREATE TABLE IF NOT EXISTS projection_account_settings (
    account_id  TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    apy         TEXT NOT NULL DEFAULT '0',        -- annual percentage yield (decimal string)
    compound    INTEGER NOT NULL DEFAULT 1,       -- 1=compound, 0=simple
    included    INTEGER NOT NULL DEFAULT 1,       -- 1=included, 0=excluded
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Per-holding projection settings (investment accounts with holdings)
CREATE TABLE IF NOT EXISTS projection_holding_settings (
    holding_id  TEXT PRIMARY KEY,                  -- SimpleFIN holding ID
    account_id  TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    apy         TEXT NOT NULL DEFAULT '0',
    compound    INTEGER NOT NULL DEFAULT 1,
    included    INTEGER NOT NULL DEFAULT 1,
    allocation  TEXT NOT NULL DEFAULT '0',         -- income allocation percentage (decimal string)
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_projection_holding_settings_account
    ON projection_holding_settings(account_id);

-- Holdings data from SimpleFIN (synced during daily fetch)
CREATE TABLE IF NOT EXISTS holdings (
    id          TEXT PRIMARY KEY,                  -- SimpleFIN holding ID
    account_id  TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    symbol      TEXT,                              -- ticker symbol (e.g., "VOO")
    description TEXT NOT NULL,                     -- human-readable name
    shares      TEXT,                              -- number of shares (decimal string)
    market_value TEXT NOT NULL,                    -- current market value (decimal string)
    cost_basis  TEXT,                              -- total cost basis (decimal string)
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_holdings_account
    ON holdings(account_id);

-- Income modeling settings (single row, key-value in settings table is too fragile)
CREATE TABLE IF NOT EXISTS projection_income_settings (
    id                  INTEGER PRIMARY KEY CHECK(id = 1),  -- singleton row
    enabled             INTEGER NOT NULL DEFAULT 0,
    annual_income       TEXT NOT NULL DEFAULT '0',
    monthly_savings_pct TEXT NOT NULL DEFAULT '0',
    allocation_json     TEXT NOT NULL DEFAULT '{}',          -- JSON: { "account_or_holding_id": "percentage", ... }
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Design rationale:**
- `projection_account_settings` uses account_id as PK with CASCADE delete -- auto-cleanup when account is removed
- `projection_holding_settings` stores per-holding rates and allocation percentages
- `holdings` table stores synced SimpleFIN holdings data (refreshed on each sync)
- `projection_income_settings` is a singleton row (CHECK constraint `id = 1`) -- avoids key-value fragility
- All financial values stored as TEXT (decimal strings) consistent with existing `balance_snapshots.balance` pattern
- `allocation_json` in income settings stores the allocation mapping as a JSON object for flexibility since allocation targets can be accounts or holdings

### Pattern 2: SimpleFIN Holdings Fetch

**What:** Extend the SimpleFIN client to fetch holdings data for investment accounts.

**Critical discovery:** The current `FetchAccounts` function sets `balances-only=1` query parameter, which strips holdings (and transaction) data from the response. To get holdings, either:
1. Remove `balances-only=1` entirely (fetches more data but includes holdings), or
2. Make a separate fetch without `balances-only=1` for investment accounts specifically

**Recommended approach:** Add a new `FetchAccountsWithHoldings` function that omits `balances-only=1`. The sync process can call the regular `FetchAccounts` for daily balance snapshots and periodically call `FetchAccountsWithHoldings` to update holdings data.

**SimpleFIN Holdings data structure (from API):**
```go
type Holding struct {
    ID            string `json:"id"`
    Created       int64  `json:"created"`
    Currency      string `json:"currency"`
    CostBasis     string `json:"cost_basis"`
    Description   string `json:"description"`
    MarketValue   string `json:"market_value"`
    PurchasePrice string `json:"purchase_price"`
    Shares        string `json:"shares"`
    Symbol        string `json:"symbol"`
}

// Extended Account struct
type Account struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Currency    string    `json:"currency"`
    Balance     string    `json:"balance"`
    BalanceDate int64     `json:"balance-date"`
    Org         Org       `json:"org"`
    Holdings    []Holding `json:"holdings,omitempty"` // NEW: only populated without balances-only=1
}
```

**Important caveat:** SimpleFIN's investment/holdings support is undocumented and varies by institution. Some accounts will have complete holdings data (symbol, shares, market_value), others will have partial data, and many will have none. The implementation must handle missing holdings gracefully.

### Pattern 3: Client-Side Projection Calculation

**What:** Pure TypeScript functions that compute projected balances month-by-month.

**When to use:** Every time a rate, toggle, checkbox, or income parameter changes (after 500ms debounce).

**Core formulas:**

```typescript
// projection.ts -- pure functions, no side effects

interface AccountProjection {
  id: string;
  currentBalance: number;
  apy: number;           // annual percentage (e.g., 5.0 for 5%)
  compound: boolean;
  included: boolean;
}

interface HoldingProjection {
  id: string;
  accountId: string;
  currentValue: number;
  apy: number;
  compound: boolean;
  included: boolean;
  allocation: number;    // percentage of income allocated (0-100)
}

interface IncomeSettings {
  enabled: boolean;
  annualIncome: number;
  monthlySavingsPct: number;  // percentage (e.g., 20 for 20%)
  // allocation lives on each account/holding
}

interface ProjectionPoint {
  date: string;       // ISO date string
  value: number;      // projected net worth
}

/**
 * Compound interest: balance * (1 + monthlyRate) for each month
 * Simple interest: balance + (principal * monthlyRate) for each month
 * Monthly rate = APY / 100 / 12
 */
function projectBalance(
  principal: number,
  apy: number,
  compound: boolean,
  months: number,
  monthlyContribution: number = 0
): number {
  const monthlyRate = apy / 100 / 12;

  if (compound) {
    // Compound: A = P(1 + r)^n + C * ((1 + r)^n - 1) / r
    // Where C = monthly contribution
    let balance = principal;
    for (let m = 0; m < months; m++) {
      balance = balance * (1 + monthlyRate) + monthlyContribution;
    }
    return balance;
  } else {
    // Simple: A = P + P*r*n + C*n
    return principal + (principal * monthlyRate * months) + (monthlyContribution * months);
  }
}

/**
 * Generate monthly projection points for the entire portfolio.
 * Sums all included account/holding projections per month.
 */
function calculateProjection(
  accounts: AccountProjection[],
  holdings: HoldingProjection[],
  income: IncomeSettings,
  horizonYears: number
): ProjectionPoint[] {
  const totalMonths = horizonYears * 12;
  const monthlySavings = income.enabled
    ? (income.annualIncome / 12) * (income.monthlySavingsPct / 100)
    : 0;

  const points: ProjectionPoint[] = [];
  const now = new Date();

  for (let m = 0; m <= totalMonths; m++) {
    const date = new Date(now);
    date.setMonth(date.getMonth() + m);

    let total = 0;

    // Accounts without holdings
    for (const acct of accounts) {
      if (!acct.included) continue;
      const contribution = monthlySavings * (getAccountAllocation(acct.id) / 100);
      total += projectBalance(acct.currentBalance, acct.apy, acct.compound, m, contribution);
    }

    // Holdings (for investment accounts with holdings)
    for (const h of holdings) {
      if (!h.included) continue;
      const contribution = monthlySavings * (h.allocation / 100);
      total += projectBalance(h.currentValue, h.apy, h.compound, m, contribution);
    }

    points.push({
      date: date.toISOString().split('T')[0],
      value: total,
    });
  }

  return points;
}
```

**Critical implementation detail:** For investment accounts WITH holdings, the account balance is NOT projected separately. Instead, each holding within that account is projected individually, and the sum of holding projections represents the account's projected value. This avoids double-counting.

### Pattern 4: Recharts ComposedChart for Solid-to-Dashed Transition

**What:** Use Recharts `ComposedChart` (or `LineChart`) with two `Line` series and one `Area` series, plus a `ReferenceLine` for the "Now" marker.

**Key technique:** The data array has both `historical` and `projected` fields. Historical points have `historical` set and `projected` null. The "now" point has BOTH set (bridging the two lines). Projection points have `projected` set and `historical` null. Recharts renders each Line only where its data is non-null, creating the seamless transition.

```typescript
import {
  ComposedChart,
  Line,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
  ResponsiveContainer,
  Label,
} from 'recharts';

// ComposedChart allows mixing Line and Area in the same chart
// Line with strokeDasharray="8 4" creates the dashed projection line
// ReferenceLine at the "now" date creates the vertical marker
// Area under projected line only (gradient fill)
```

**Recharts 3.8.0 verification:** ComposedChart, ReferenceLine, Line with strokeDasharray, and Area are all available. The `connectNulls={false}` prop (default) ensures lines don't bridge null gaps, which is essential for the historical/projected split.

### Pattern 5: Auto-Save with Debounce

**What:** Settings changes are auto-saved to the server after 500ms debounce. No explicit "Save" button.

**Implementation:**
```typescript
// Custom hook for debounced auto-save
function useDebouncedSave(saveFn: (data: unknown) => Promise<void>, delay = 500) {
  const timeoutRef = useRef<number>();

  const save = useCallback((data: unknown) => {
    clearTimeout(timeoutRef.current);
    timeoutRef.current = window.setTimeout(() => {
      saveFn(data).catch(() => {
        // Show toast on failure
      });
    }, delay);
  }, [saveFn, delay]);

  useEffect(() => () => clearTimeout(timeoutRef.current), []);

  return save;
}
```

**State management:** The Projections page component owns all projection state (rates, toggles, income). Child components receive values and onChange handlers. The page component debounces both:
1. Chart recalculation (runs projection engine)
2. API save call (persists to server)

### Anti-Patterns to Avoid
- **Double-counting investment accounts with holdings:** Never project the account-level balance AND the per-holding balances. When holdings exist, only project holdings.
- **Server-side projection calculation:** Adds unnecessary latency for "what-if" scenarios. Keep it client-side.
- **Separate save button for each field:** Auto-save with debounce is the correct UX for this "what-if" tool pattern.
- **Over-normalizing income allocation:** A JSON blob for allocation mapping is simpler than a junction table, since the allocation targets can be either accounts or holdings.
- **Fetching holdings on every page load from SimpleFIN:** Holdings should be synced during the daily sync process and read from the local database. The projections page reads from local DB only.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Charting | Custom SVG projection chart | Recharts ComposedChart | ComposedChart handles the solid-to-dashed transition, tooltips, axes, reference lines, and responsive sizing |
| Financial arithmetic (backend) | Raw float64 math | shopspring/decimal | Float precision errors accumulate in financial calculations |
| Database migrations | Manual CREATE TABLE | golang-migrate with embedded SQL files | Consistent with existing migration pattern (migrations 000001-000004) |
| Date formatting on charts | Custom date formatting | Date.toLocaleDateString + custom formatting per horizon length | Different horizons need different tick formats (months vs years) |

**Key insight:** The projection math itself is simple enough to hand-roll (compound/simple interest formulas), but it MUST be extracted into pure functions in `utils/projection.ts` for testability. The math is the most critical thing to unit test in this phase.

## Common Pitfalls

### Pitfall 1: SimpleFIN Holdings Data Gaps
**What goes wrong:** Assuming all investment accounts will have holdings data. Many SimpleFIN integrations return only account-level balance with no holdings breakdown.
**Why it happens:** Holdings support in SimpleFIN is undocumented and institution-dependent.
**How to avoid:** Always design UI to gracefully handle investment accounts with zero holdings. The fallback is account-level rate configuration (identical to savings/liquid accounts). Never show "No holdings" messages -- just render the account as a standard row without the expand chevron.
**Warning signs:** Investment accounts with no holdings appearing broken in the UI.

### Pitfall 2: balances-only=1 Stripping Holdings
**What goes wrong:** The existing `FetchAccounts` function sets `balances-only=1`, which strips holdings data from the SimpleFIN response.
**Why it happens:** The original implementation only needed balances, not holdings.
**How to avoid:** Either add a new function `FetchAccountsWithHoldings` that omits `balances-only=1`, or modify `FetchAccounts` to accept a parameter controlling this. Be aware that removing `balances-only=1` also fetches transaction data, increasing response size.
**Warning signs:** Holdings array always empty despite the account having holdings at the institution.

### Pitfall 3: Double-Counting Investment Account Balances
**What goes wrong:** Projecting both the account-level balance AND summing the per-holding projections for accounts that have holdings.
**Why it happens:** The account balance already includes all holdings. Projecting both creates 2x the actual value.
**How to avoid:** When an investment account has holdings, SKIP the account-level projection entirely. Only project individual holdings. The sum of holding projections should approximate the account balance.
**Warning signs:** Projected values are suspiciously high for investment accounts with holdings.

### Pitfall 4: Allocation Sum Not Equaling 100%
**What goes wrong:** Income allocation percentages don't sum to 100% after adding/removing accounts from the "included" set, leading to incorrect projection amounts.
**Why it happens:** User excludes an account that had allocation percentage, breaking the 100% sum.
**How to avoid:** When the sum is not 100%, suppress income contribution in the projection (use last valid state). Show clear validation error. Do NOT auto-redistribute -- let the user fix it manually.
**Warning signs:** Projection values jumping unexpectedly when include/exclude checkboxes change.

### Pitfall 5: Floating Point Accumulation in Monthly Projection Loop
**What goes wrong:** Running compound interest calculation for 240 months (20 years) with floating-point numbers accumulates rounding errors.
**Why it happens:** JavaScript number type is IEEE 754 double precision.
**How to avoid:** For the projection chart display, this is acceptable -- errors will be sub-cent after 20 years. For stored rates and settings, use decimal strings (consistent with existing balance storage). Do NOT use a BigDecimal library client-side for projection calculation -- the performance cost is not worth the negligible accuracy gain for a visual projection.
**Warning signs:** None -- this is a known acceptable trade-off for projection visualization.

### Pitfall 6: CASCADE DELETE on Account Removal
**What goes wrong:** When an account is soft-deleted (hidden_at set), projection settings should be preserved (per OPS-03 requirement).
**Why it happens:** ON DELETE CASCADE would remove settings when the account is hard-deleted, but soft-delete preserves the account row.
**How to avoid:** Use ON DELETE CASCADE on the foreign key (handles true deletion cleanly), but since the app uses soft-delete, settings are naturally preserved. The projections page should include only non-hidden accounts (WHERE hidden_at IS NULL) when loading settings.
**Warning signs:** Settings disappearing after an account goes stale and returns.

## Code Examples

### Example 1: API Endpoint Pattern (GET /api/projections/settings)

```go
// Source: follows existing handler patterns (handlers/alerts.go, handlers/networth.go)

type projectionAccountSetting struct {
    AccountID   string  `json:"account_id"`
    AccountName string  `json:"account_name"`   // COALESCE(display_name, name)
    AccountType string  `json:"account_type"`    // effective type
    Balance     string  `json:"balance"`         // latest balance
    APY         string  `json:"apy"`
    Compound    bool    `json:"compound"`
    Included    bool    `json:"included"`
    Holdings    []projectionHoldingSetting `json:"holdings,omitempty"`
}

type projectionHoldingSetting struct {
    HoldingID   string `json:"holding_id"`
    Symbol      string `json:"symbol"`
    Description string `json:"description"`
    MarketValue string `json:"market_value"`
    APY         string `json:"apy"`
    Compound    bool   `json:"compound"`
    Included    bool   `json:"included"`
    Allocation  string `json:"allocation"`
}

type projectionIncomeSettings struct {
    Enabled          bool   `json:"enabled"`
    AnnualIncome     string `json:"annual_income"`
    MonthlySavingsPct string `json:"monthly_savings_pct"`
    AllocationJSON   string `json:"allocation_json"`
}

type projectionSettingsResponse struct {
    Accounts []projectionAccountSetting `json:"accounts"`
    Income   projectionIncomeSettings   `json:"income"`
}
```

### Example 2: SQL Query for Accounts with Projection Settings

```sql
-- Fetch accounts with their projection settings (LEFT JOIN for accounts without settings yet)
SELECT
    a.id,
    COALESCE(a.display_name, a.name) as account_name,
    COALESCE(a.account_type_override, a.account_type) as effective_type,
    COALESCE(bs.balance, '0') as balance,
    COALESCE(ps.apy, '0') as apy,
    COALESCE(ps.compound, 1) as compound,
    COALESCE(ps.included, 1) as included
FROM accounts a
LEFT JOIN projection_account_settings ps ON a.id = ps.account_id
LEFT JOIN (
    SELECT account_id, balance
    FROM balance_snapshots
    WHERE (account_id, balance_date) IN (
        SELECT account_id, MAX(balance_date) FROM balance_snapshots GROUP BY account_id
    )
) bs ON a.id = bs.account_id
WHERE a.hidden_at IS NULL
ORDER BY COALESCE(a.account_type_override, a.account_type), COALESCE(a.display_name, a.name)
```

### Example 3: Recharts ComposedChart Setup

```typescript
// Source: Recharts 3.8.0 documentation, verified against existing StackedAreaChart pattern
import {
  ComposedChart, Line, Area, XAxis, YAxis,
  CartesianGrid, Tooltip, ReferenceLine,
  ResponsiveContainer, Label
} from 'recharts';

// Data point: { date: "Mar 15", historical: 50000, projected: null }
// or: { date: "Mar 17", historical: 50500, projected: 50500 } -- the "now" bridge point
// or: { date: "Apr 17", historical: null, projected: 52000 }

<ResponsiveContainer width="100%" height={400}>
  <ComposedChart data={chartData}>
    <CartesianGrid strokeDasharray="3 3" strokeOpacity={0.2} />
    <XAxis dataKey="date" tick={{ fontSize: 12 }} />
    <YAxis tickFormatter={formatYAxis} tick={{ fontSize: 12 }} />
    <Tooltip content={<ProjectionTooltip />} />

    {/* Subtle gradient fill under projection only */}
    <defs>
      <linearGradient id="projGradient" x1="0" y1="0" x2="0" y2="1">
        <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.2} />
        <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
      </linearGradient>
    </defs>
    <Area dataKey="projected" fill="url(#projGradient)" stroke="none" />

    {/* Historical line: solid, gray */}
    <Line
      dataKey="historical"
      stroke="#374151"
      strokeWidth={2}
      dot={false}
      connectNulls={false}
    />

    {/* Projection line: dashed, blue */}
    <Line
      dataKey="projected"
      stroke="#3b82f6"
      strokeWidth={2}
      strokeDasharray="8 4"
      dot={false}
      connectNulls={false}
    />

    {/* "Now" vertical marker */}
    <ReferenceLine x={nowDateString} stroke="#9ca3af" strokeDasharray="4 4">
      <Label value="Now" position="top" fill="#6b7280" fontSize={12} fontWeight={600} />
    </ReferenceLine>
  </ComposedChart>
</ResponsiveContainer>
```

### Example 4: Auto-Save Debounce Pattern

```typescript
// Source: follows existing debounce patterns in the codebase
const saveTimeoutRef = useRef<number>();

const debouncedSave = useCallback((settings: ProjectionSettings) => {
  clearTimeout(saveTimeoutRef.current);
  saveTimeoutRef.current = window.setTimeout(async () => {
    try {
      await saveProjectionSettings(settings);
    } catch {
      showToast('Failed to save settings', 'error');
    }
  }, 500);
}, []);

// On any setting change:
const handleApyChange = (accountId: string, apy: string) => {
  setSettings(prev => {
    const next = updateAccountApy(prev, accountId, apy);
    debouncedSave(next);  // auto-save
    return next;          // triggers recalculation via useMemo/useEffect
  });
};
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Server-side projection rendering | Client-side calculation with live updates | Standard practice for "what-if" tools | Enables instant feedback without API round-trips |
| Separate historical and projection charts | Single ComposedChart with solid/dashed transition | Recharts has supported this for years | Cleaner UX, single visual context |
| SimpleFIN balances-only | Full account data with holdings | Holdings support exists but is undocumented | Can fetch investment positions but with data gap risk |

**Deprecated/outdated:**
- None relevant to this phase.

## Open Questions

1. **SimpleFIN Holdings Data Completeness**
   - What we know: SimpleFIN returns holdings with fields (id, symbol, description, market_value, shares, cost_basis) for some investment accounts
   - What's unclear: Which institutions provide complete holdings data and which return empty/partial data. The SimpleFIN developer confirmed investment/holdings support is "not as good as the banking side."
   - Recommendation: Implement holdings support optimistically but design all UI to work gracefully without holdings data. Test with the user's actual SimpleFIN accounts to validate.

2. **Holdings Sync Frequency**
   - What we know: Holdings data should be refreshed during the daily sync
   - What's unclear: Whether fetching without `balances-only=1` significantly increases response time or data volume
   - Recommendation: Make a single call without `balances-only=1` during sync (simplest approach). If performance is an issue, optimize later by using two calls (one for balances, one for full data).

3. **Account Allocation for Accounts Without Holdings**
   - What we know: Investment accounts without holdings get account-level allocation; accounts with holdings get per-holding allocation
   - What's unclear: How to handle the allocation_json format for mixed account/holding targets
   - Recommendation: Use a flat JSON object where keys are either account IDs or holding IDs, with a type prefix: `{"acct:ABC123": "30", "hold:XYZ789": "20", ...}`. This avoids ID collision between accounts and holdings.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (frontend) | Vitest 3.2.1 + @testing-library/react 16.3.2 |
| Framework (backend) | Go testing + net/http/httptest |
| Config file (frontend) | `frontend/vitest.config.ts` |
| Config file (backend) | none (Go standard) |
| Quick run command (frontend) | `cd frontend && npx vitest run --reporter=verbose` |
| Quick run command (backend) | `go test ./internal/...` |
| Full suite command | `cd frontend && npx vitest run && cd .. && go test ./internal/...` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PROJ-01 | APY setting per account/holding persists | unit (backend) | `go test ./internal/api/handlers/ -run TestProjectionSettings -v` | Wave 0 |
| PROJ-02 | Compound/simple toggle per account | unit (frontend) | `cd frontend && npx vitest run src/components/RateConfigTable.test.tsx` | Wave 0 |
| PROJ-03 | Include/exclude accounts from projection | unit (frontend) | `cd frontend && npx vitest run src/utils/projection.test.ts` | Wave 0 |
| PROJ-04 | Income modeling with allocation | unit (frontend) | `cd frontend && npx vitest run src/components/IncomeModelingSection.test.tsx` | Wave 0 |
| PROJ-05 | Projection chart renders with correct data | unit (frontend) | `cd frontend && npx vitest run src/components/ProjectionChart.test.tsx` | Wave 0 |
| PROJ-06 | Settings persist across sessions | unit (backend) | `go test ./internal/api/handlers/ -run TestProjectionSettingsPersistence -v` | Wave 0 |
| PROJ-07 | Projections page accessible from nav | unit (frontend) | `cd frontend && npx vitest run src/pages/Projections.test.tsx` | Wave 0 |
| PROJ-08 | Holdings data displayed for investment accounts | unit (backend) | `go test ./internal/api/handlers/ -run TestAccountsWithHoldings -v` | Wave 0 |
| MATH | Compound interest calculation correctness | unit (frontend) | `cd frontend && npx vitest run src/utils/projection.test.ts` | Wave 0 |
| MATH | Simple interest calculation correctness | unit (frontend) | `cd frontend && npx vitest run src/utils/projection.test.ts` | Wave 0 |
| MATH | Income contribution projection correctness | unit (frontend) | `cd frontend && npx vitest run src/utils/projection.test.ts` | Wave 0 |

### Sampling Rate
- **Per task commit:** `cd frontend && npx vitest run --reporter=verbose && go test ./internal/... -v`
- **Per wave merge:** Full suite (same as above)
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `frontend/src/utils/projection.test.ts` -- covers PROJ-03, PROJ-05, MATH (compound, simple, income)
- [ ] `frontend/src/components/ProjectionChart.test.tsx` -- covers PROJ-05
- [ ] `frontend/src/components/HorizonSelector.test.tsx` -- covers PROJ-05
- [ ] `frontend/src/components/RateConfigTable.test.tsx` -- covers PROJ-01, PROJ-02, PROJ-03
- [ ] `frontend/src/components/HoldingsRow.test.tsx` -- covers PROJ-08
- [ ] `frontend/src/components/IncomeModelingSection.test.tsx` -- covers PROJ-04
- [ ] `frontend/src/components/AllocationRow.test.tsx` -- covers PROJ-04
- [ ] `frontend/src/pages/Projections.test.tsx` -- covers PROJ-07
- [ ] `internal/api/handlers/projections_test.go` -- covers PROJ-01, PROJ-06, PROJ-08

## Sources

### Primary (HIGH confidence)
- Existing codebase: `internal/api/router.go`, `internal/simplefin/client.go`, `internal/sync/sync.go`, `internal/db/migrations/*.sql` -- architecture patterns and integration points
- Existing codebase: `frontend/src/components/StackedAreaChart.tsx`, `frontend/src/components/BalanceLineChart.tsx` -- Recharts usage patterns
- Existing codebase: `frontend/src/api/client.ts` -- API client patterns
- Existing codebase: `frontend/src/App.tsx` -- routing and NavBar patterns
- Existing codebase: `frontend/src/components/GroupRow.tsx` -- expand/collapse chevron pattern
- Recharts 3.8.0 (installed) -- ComposedChart, ReferenceLine, Line with strokeDasharray verified
- Phase 9 UI-SPEC: `.planning/phases/09-projection-engine/09-UI-SPEC.md` -- complete visual specification

### Secondary (MEDIUM confidence)
- SimpleFIN Holdings structure: verified via [SimpleFIN/Wealthfolio integration issue](https://github.com/afadil/wealthfolio/issues/197) -- holdings fields: id, created, currency, cost_basis, description, market_value, purchase_price, shares, symbol
- SimpleFIN developer statement (from the same issue): "Investment/holdings support for SimpleFIN is definitely not as good as the banking side, which is why it's not yet documented."
- Recharts dashed line examples: [Recharts Dashed Line Chart](https://recharts.github.io/en-US/examples/DashedLineChart/)
- Compound interest formula: A = P(1 + r/n)^(n*t) for compound; A = P(1 + r*t) for simple -- standard financial mathematics

### Tertiary (LOW confidence)
- SimpleFIN `balances-only=1` parameter behavior: inferred from parameter name and the absence of holdings in current API responses. Not explicitly documented. Needs validation with an actual API call without this parameter.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries already installed and proven in the codebase
- Architecture: HIGH -- follows established patterns from Phases 5-8
- Projection math: HIGH -- standard compound/simple interest formulas
- SimpleFIN holdings: MEDIUM -- holdings structure confirmed via third-party integration, but data completeness varies by institution
- Pitfalls: HIGH -- identified through codebase analysis and domain knowledge

**Research date:** 2026-03-17
**Valid until:** 2026-04-17 (stable -- no fast-moving dependencies)
