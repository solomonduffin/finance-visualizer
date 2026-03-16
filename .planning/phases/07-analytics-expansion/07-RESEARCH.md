# Phase 7: Analytics Expansion - Research

**Researched:** 2026-03-16
**Domain:** Account groups (DB schema, CRUD API, drag-drop UX), collapsible group rows in dashboard, dedicated net worth page with stacked area chart
**Confidence:** HIGH

## Summary

Phase 7 adds two major capabilities: (1) account groups as organizational folders that let users combine accounts (e.g., multiple Coinbase wallets) into named groups with summed balances, and (2) a dedicated `/net-worth` page with a stacked area chart, summary statistics, and time range selector. This is the first new page beyond Dashboard and Settings.

The account groups feature requires a new database migration (account_groups + group_members tables), CRUD API endpoints, extensions to the Settings page's existing drag-and-drop system, and modifications to PanelCard to render collapsible group rows. The net worth page requires a new route in react-router-dom, a new API endpoint that returns per-panel time-series data (extending the existing `dayAccumulator` pattern from `GetBalanceHistory`), and a Recharts stacked area chart using the existing `panelColors.ts` color system.

Both features build directly on established patterns: `shopspring/decimal` for financial arithmetic, `@dnd-kit/react` for drag-and-drop, Recharts for charting, Tailwind v4 for styling, and `chi` router for API endpoints. No new dependencies are needed.

**Primary recommendation:** Implement groups backend-first (migration, CRUD API, modify accounts/summary/growth queries to respect groups), then wire groups into the Settings and Dashboard UIs. Build the net worth page as a parallel workstream since it only depends on the existing balance_history endpoint being extended with panel-level totals per date.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Groups act as accounts -- a group is assigned to a panel type (Liquid/Savings/Investments) and its summed balance appears in that panel
- Groups are NOT limited by member account type -- any account can be placed in any group regardless of its original type (organizational folder model)
- Ungrouped accounts are allowed -- groups are optional organization, existing accounts keep working standalone
- When the last account is removed from a group, the group is auto-deleted (no empty group persistence)
- Growth badge calculated on group's summed member balance over 30 days (consistent with panel-level growth from Phase 6)
- "+ New Group" button in the Accounts section creates a new group with inline name entry
- Drag-and-drop to assign accounts into a group (extends existing drag pattern from Phase 5 type reassignment)
- Drag-and-drop to assign group to a panel type section (same pattern as individual accounts)
- Drag accounts out of a group to ungroup them back to standalone
- Grouped accounts no longer appear individually in panel lists -- they only exist inside the group
- Mobile: drag-and-drop with existing mobile fallback patterns from Phase 5
- Groups appear as collapsible rows in their assigned panel card: group name + summed balance + expand chevron
- Collapsed by default -- keeps dashboard compact, aligns with "one glance" core value
- Clicking expands to reveal indented member accounts with individual balances
- Visual distinction: bold group name + collapse/expand chevron (subtle, not heavy)
- Growth badge shown on individual group rows (alongside the panel total growth badge)
- New route: /net-worth
- Navigation: clicking the net worth donut chart navigates to /net-worth (INSIGHT-02) + "Net Worth" link added to top nav bar
- Layout: summary stats bar at top, full-width chart below
- Summary stats: current net worth, period change ($ and %), all-time high
- Period change matches selected time range (if 90d selected, shows 90-day change)
- Line chart only on this page -- no donut (donut lives on dashboard, avoids redundancy)
- Time range selector: segmented control with options 30d, 90d, 6m, 1y, All
- Stacked area chart with three layers: Liquid (bottom), Savings (middle), Investments (top)
- Top line of stacked area = total net worth
- Uses same panel colors from panelColors.ts -- consistent visual language across dashboard and net worth page
- Tooltip on hover: date, total net worth, and per-panel dollar amounts with color indicators
- Built with Recharts (already in use for BalanceLineChart)

### Claude's Discretion
- Exact group row expand/collapse animation
- Database schema for account groups (group table, group_member junction, etc.)
- API endpoint design for CRUD operations on groups
- Stacked area chart opacity/gradient styling
- Summary stats card styling and responsive breakpoints
- How to handle net worth page loading/empty states
- Whether summary stats should animate on time range change
- Segmented control component implementation details

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| ACCT-03 | User can create named account groups in Settings and assign accounts to them | DB schema (account_groups + group_members), CRUD API endpoints, Settings UI extensions with drag-drop |
| ACCT-04 | Account groups appear as a single combined line in their panel with summed balance | PanelCard modification with collapsible group rows, backend returns group data in accounts response |
| ACCT-05 | User can expand an account group to see individual account balances beneath it | Collapsible row pattern in PanelCard, expand/collapse state management |
| INSIGHT-02 | User can click net worth donut to navigate to a dedicated net worth page | NetWorthDonut click handler + useNavigate, new /net-worth route in App.tsx |
| INSIGHT-03 | Net worth page shows historical net worth line chart with per-panel breakdown | Recharts stacked area chart with 3 layers, new API endpoint returning per-panel time-series |
| INSIGHT-04 | Net worth page shows summary statistics (current net worth, period change in $ and %, all-time high) | Backend computes stats from balance_snapshots, summary stats bar component |
| INSIGHT-05 | Net worth page has a time range selector (30d, 90d, 6m, 1y, all) | Segmented control component, passes days param to API, updates chart and stats |
</phase_requirements>

## Standard Stack

### Core (Already Installed)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Recharts | 3.8.0 | Stacked area chart on net worth page | Already used for BalanceLineChart and NetWorthDonut; supports stacked areas via `stackId` prop |
| @dnd-kit/react | 0.3.2 | Drag-drop for group management in Settings | Already used for account type reassignment in AccountsSection |
| react-router-dom | 7.13.1 | New /net-worth route, programmatic navigation | Already used for /, /settings, /login routes |
| shopspring/decimal | 1.4.0 | All financial arithmetic (group sums, net worth stats) | Mandated project convention for balance math |
| go-chi/chi/v5 | 5.2.5 | API routing for new group endpoints | Already used for all backend routes |
| modernc.org/sqlite | 1.46.1 | Database with foreign keys for group tables | Already used; FK support enabled via PRAGMA |
| Tailwind CSS | 4.2.1 | All UI styling | Mandated project convention |

### Supporting (Already Installed)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| golang-migrate/migrate/v4 | 4.19.1 | New migration for account_groups tables | Database schema changes |
| @testing-library/react | 16.3.2 | Frontend component tests | All new React components |
| Vitest | (devDep) | Test runner | Frontend test execution |

### No New Dependencies Required
The entire phase can be implemented with the existing stack. Recharts already supports stacked area charts, dnd-kit already supports the drag patterns needed for group management, and react-router-dom already provides useNavigate for programmatic navigation.

## Architecture Patterns

### Recommended Project Structure

```
# New files
frontend/src/pages/NetWorth.tsx                     # New page component
frontend/src/pages/NetWorth.test.tsx                # Page tests
frontend/src/components/StackedAreaChart.tsx         # Stacked area chart
frontend/src/components/StackedAreaChart.test.tsx    # Chart tests
frontend/src/components/TimeRangeSelector.tsx        # Segmented control
frontend/src/components/TimeRangeSelector.test.tsx   # Selector tests
frontend/src/components/NetWorthStats.tsx            # Summary stats bar
frontend/src/components/NetWorthStats.test.tsx       # Stats tests
frontend/src/components/GroupRow.tsx                 # Collapsible group row for PanelCard
frontend/src/components/GroupRow.test.tsx            # Group row tests
internal/api/handlers/groups.go                     # Group CRUD handlers
internal/api/handlers/groups_test.go                # Group handler tests
internal/api/handlers/networth.go                   # Net worth history + stats handler
internal/api/handlers/networth_test.go              # Net worth handler tests
internal/db/migrations/000003_account_groups.up.sql   # Groups migration
internal/db/migrations/000003_account_groups.down.sql # Groups rollback

# Modified files
frontend/src/App.tsx                    # Add /net-worth route + nav link
frontend/src/api/client.ts             # Add group API + net worth API functions
frontend/src/components/PanelCard.tsx   # Add group row support
frontend/src/components/NetWorthDonut.tsx # Add click handler for navigation
frontend/src/components/AccountsSection.tsx # Add group management UX
frontend/src/pages/Dashboard.tsx        # Pass group data to PanelCard
internal/api/router.go                 # Register new endpoints
internal/api/handlers/accounts.go      # Include group info in response
internal/api/handlers/summary.go       # Account for groups in panel totals
internal/api/handlers/growth.go        # Group-level growth computation
```

### Pattern 1: Database Schema for Account Groups

**What:** Two new tables -- `account_groups` for group metadata and `group_members` for the membership junction.
**When to use:** Any feature requiring many-to-many or one-to-many grouping.

```sql
-- Migration 000003_account_groups.up.sql
CREATE TABLE IF NOT EXISTS account_groups (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    panel_type TEXT NOT NULL CHECK(panel_type IN ('checking', 'savings', 'investment')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id   INTEGER NOT NULL REFERENCES account_groups(id) ON DELETE CASCADE,
    account_id TEXT NOT NULL REFERENCES accounts(id),
    added_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_group_members_account
    ON group_members(account_id);
```

Key design decisions:
- `panel_type` on the group determines which panel it appears in (uses same values as `account_type_override`)
- `ON DELETE CASCADE` on `group_members.group_id` ensures members are cleaned up if a group is deleted
- An account can only be in one group at a time (enforced by application logic -- check before adding)
- `panel_type` uses the raw DB values (`checking`, `savings`, `investment`) not the frontend names (`liquid`, `savings`, `investments`)
- Foreign key on `account_id` references `accounts(id)` -- PRAGMA foreign_keys = ON is already set

### Pattern 2: Group CRUD API Design

**What:** REST endpoints for group lifecycle management.
**When to use:** Group creation, updating, member management.

```
POST   /api/groups              -- Create group {name, panel_type}
PATCH  /api/groups/{id}         -- Update group {name?, panel_type?}
DELETE /api/groups/{id}         -- Delete group (cascades members)
POST   /api/groups/{id}/members -- Add account {account_id}
DELETE /api/groups/{id}/members/{account_id} -- Remove account
```

Response shapes:
```json
// POST /api/groups response
{
  "id": 1,
  "name": "Coinbase",
  "panel_type": "investment",
  "members": [],
  "total_balance": "0.00"
}

// GET /api/accounts response (extended)
{
  "liquid": [...],
  "savings": [...],
  "investments": [...],
  "other": [...],
  "groups": [
    {
      "id": 1,
      "name": "Coinbase",
      "panel_type": "investment",
      "total_balance": "5230.00",
      "members": [
        {"id": "acct1", "name": "CB Wallet 1", "balance": "3000.00", ...},
        {"id": "acct2", "name": "CB Wallet 2", "balance": "2230.00", ...}
      ]
    }
  ]
}
```

The accounts endpoint already returns grouped-by-panel data. Extend it to also return a `groups` array. Grouped accounts should NOT appear in the panel arrays (they appear only inside their group's `members` array). The dashboard uses the panel arrays for standalone accounts and the groups array for group rows.

### Pattern 3: Stacked Area Chart with Recharts

**What:** Three stacked `<Area>` components sharing a `stackId` to create a cumulative net worth visualization.
**When to use:** The /net-worth page chart.

```tsx
// Source: Recharts official stacked area chart pattern
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer,
} from 'recharts'
import { PANEL_COLORS } from './panelColors'

// Data shape: each point has date + per-panel values
interface NetWorthPoint {
  date: string       // formatted "Mar 15"
  liquid: number
  savings: number
  investments: number
}

<ResponsiveContainer width="100%" height={400}>
  <AreaChart data={data} margin={{ top: 10, right: 10, left: 10, bottom: 0 }}>
    <defs>
      <linearGradient id="gradLiquid" x1="0" y1="0" x2="0" y2="1">
        <stop offset="5%" stopColor={PANEL_COLORS.liquid.accent} stopOpacity={0.6} />
        <stop offset="95%" stopColor={PANEL_COLORS.liquid.accent} stopOpacity={0.1} />
      </linearGradient>
      {/* Similar gradients for savings, investments */}
    </defs>
    <CartesianGrid strokeDasharray="3 3" strokeOpacity={0.2} />
    <XAxis dataKey="date" tick={{ fontSize: 11 }} />
    <YAxis tickFormatter={(v) => '$' + (v / 1000).toFixed(0) + 'k'} />
    <Tooltip content={<NetWorthTooltip />} />
    <Area
      type="monotone"
      dataKey="liquid"
      stackId="networth"
      stroke={PANEL_COLORS.liquid.accent}
      fill="url(#gradLiquid)"
    />
    <Area
      type="monotone"
      dataKey="savings"
      stackId="networth"
      stroke={PANEL_COLORS.savings.accent}
      fill="url(#gradSavings)"
    />
    <Area
      type="monotone"
      dataKey="investments"
      stackId="networth"
      stroke={PANEL_COLORS.investments.accent}
      fill="url(#gradInvestments)"
    />
  </AreaChart>
</ResponsiveContainer>
```

Key insight: All three `<Area>` components share `stackId="networth"` which causes Recharts to stack them. The bottom area (liquid) starts at y=0, savings stacks on top, investments stacks on top of that. The topmost line of the chart equals total net worth.

### Pattern 4: Programmatic Navigation from Donut Chart

**What:** Make the NetWorthDonut clickable to navigate to /net-worth.
**When to use:** INSIGHT-02 requirement.

```tsx
// In NetWorthDonut.tsx -- wrap chart in a clickable container
import { useNavigate } from 'react-router-dom'

const navigate = useNavigate()

// Wrap the chart container div
<div
  className="cursor-pointer"
  onClick={() => navigate('/net-worth')}
  role="link"
  aria-label="View net worth details"
>
  {/* existing PieChart content */}
</div>
```

Note: The project currently uses `<Link>` for declarative navigation and `window.location.href` for imperative navigation. Since `useNavigate` is the idiomatic react-router-dom approach for event handler navigation (no full page reload), it should be used here. The `NetWorthDonut` component will need to be rendered inside the Router context (it already is, since Dashboard is a Route child).

### Pattern 5: Segmented Control Component

**What:** A row of mutually exclusive buttons for time range selection.
**When to use:** Time range selector on net worth page.

```tsx
const TIME_RANGES = [
  { label: '30d', days: 30 },
  { label: '90d', days: 90 },
  { label: '6m', days: 180 },
  { label: '1y', days: 365 },
  { label: 'All', days: 0 },  // 0 means no limit
] as const

interface TimeRangeSelectorProps {
  selected: number  // days value
  onChange: (days: number) => void
}

// Render as a rounded pill bar with active state highlighting
// Use role="radiogroup" with role="radio" for each option for accessibility
```

### Pattern 6: Collapsible Group Row in PanelCard

**What:** Groups render as a row with name, summed balance, chevron. Clicking toggles expansion to show member accounts.
**When to use:** ACCT-04 and ACCT-05 requirements.

```tsx
// Collapsed: [> Coinbase     $5,230.00  +3.2%]
// Expanded:  [v Coinbase     $5,230.00  +3.2%]
//              CB Wallet 1   $3,000.00
//              CB Wallet 2   $2,230.00

// Use local state for expanded group IDs
const [expandedGroups, setExpandedGroups] = useState<Set<number>>(new Set())

function toggleGroup(groupId: number) {
  setExpandedGroups(prev => {
    const next = new Set(prev)
    if (next.has(groupId)) next.delete(groupId)
    else next.add(groupId)
    return next
  })
}
```

The ChevronIcon component already exists in AccountsSection.tsx and can be extracted to a shared location or duplicated (it is only 10 lines).

### Pattern 7: Net Worth History API

**What:** API endpoint returning per-panel time-series data for the stacked area chart.
**When to use:** INSIGHT-03, INSIGHT-04, INSIGHT-05.

```go
// GET /api/net-worth?days=90
// Response:
{
  "points": [
    {"date": "2025-12-15", "liquid": "4230.50", "savings": "15000.00", "investments": "32000.00"},
    {"date": "2025-12-16", "liquid": "4280.50", "savings": "15000.00", "investments": "32100.00"},
    ...
  ],
  "stats": {
    "current_net_worth": "51330.50",
    "period_change_dollars": "3200.00",
    "period_change_pct": "6.65",
    "all_time_high": "52100.00",
    "all_time_high_date": "2026-03-10"
  }
}
```

This endpoint reuses the `dayAccumulator` pattern from `GetBalanceHistory` but returns all three panel values per date point (instead of separate arrays). The stats computation happens server-side using `shopspring/decimal`:
- `current_net_worth`: sum of latest liquid + savings + investments
- `period_change_dollars`: current minus earliest-in-range total
- `period_change_pct`: change / earliest * 100
- `all_time_high`: max of (liquid + savings + investments) across all dates in range

### Anti-Patterns to Avoid
- **Computing net worth stats client-side:** Would require shipping decimal arithmetic to the browser and duplicating panel aggregation logic. Compute server-side.
- **Separate API calls for chart data and stats:** Wasteful; the stats can be computed during the same query scan that builds the time-series. Single endpoint returns both.
- **Storing group panel_type as frontend names (liquid/investments):** Use the DB-native values (checking/savings/investment) for consistency with `account_type` and `account_type_override` columns.
- **Making group membership a column on accounts table:** A junction table (group_members) is cleaner, supports the auto-delete-empty-group rule, and avoids polluting the accounts schema.
- **Nested DragDropProviders:** The existing AccountsSection uses a single `<DragDropProvider>`. Groups should be droppable zones within the same provider, not nested providers.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Stacked area chart | Custom SVG chart | Recharts `<AreaChart>` with `stackId` | Recharts handles layout, animation, tooltips, responsive sizing. Already used in project. |
| Drag-and-drop | Custom mouse/touch handlers | @dnd-kit/react `useDraggable`/`useDroppable` | Already proven in Phase 5 for account type reassignment. Handles touch, keyboard, animations. |
| Client-side routing | window.location.href for navigation | react-router-dom `useNavigate` + `<Route>` | Avoids full page reload, keeps SPA behavior, maintains state across navigation. |
| Decimal arithmetic | JavaScript floating point | shopspring/decimal on backend | Project mandate. Floating point produces rounding errors in financial calculations. |
| Time range date math | Manual date subtraction in SQL | SQLite `DATE('now', '-N days')` | Already used in growth.go. SQLite handles DST, leap years correctly. |

**Key insight:** Every tool needed for this phase is already in the project's dependency tree. The implementation challenge is integration, not technology selection.

## Common Pitfalls

### Pitfall 1: Group Members Appearing in Both Group and Panel Lists
**What goes wrong:** Grouped accounts show up in both the group's member list AND the standalone account list in a panel, causing double-counting.
**Why it happens:** The GetAccounts query doesn't filter out accounts that belong to a group.
**How to avoid:** Add a `LEFT JOIN group_members` and `WHERE group_members.group_id IS NULL` condition to exclude grouped accounts from the standalone panel arrays. Grouped accounts only appear in the `groups[].members` arrays.
**Warning signs:** Total balance on dashboard is higher than expected; same account name appears twice.

### Pitfall 2: Net Worth Stacked Chart Data Misalignment
**What goes wrong:** Not all panels have data on every date, causing the stacked areas to have gaps or jump to zero.
**Why it happens:** Some panels may not have snapshots on every date (e.g., no savings accounts existed early on).
**How to avoid:** For each date in the series, carry forward the last known value for any panel that doesn't have a snapshot on that date. The `dayAccumulator` pattern already handles partial data per day. Ensure missing panels default to their previous value, not zero.
**Warning signs:** Stacked area chart shows sudden drops to zero or vertical spikes.

### Pitfall 3: Auto-Delete Empty Group Race Condition
**What goes wrong:** Removing the last member from a group doesn't clean up the group, or a concurrent request adds a member while deletion is in progress.
**Why it happens:** Application-level check-and-delete is not atomic.
**How to avoid:** Use a transaction: within the same TX, delete the member, check COUNT(*) of remaining members, and DELETE the group if zero. SQLite's single-writer (MaxOpenConns=1) provides serialization.
**Warning signs:** Empty groups appearing in the UI; group with 0 members showing "0.00" balance.

### Pitfall 4: Donut Click Not Working Inside PieChart
**What goes wrong:** The click handler on the wrapping div doesn't fire because Recharts' PieChart captures pointer events.
**Why it happens:** SVG elements inside the chart intercept mouse events.
**How to avoid:** Add the `onClick` to the wrapping div and ensure `pointer-events: auto` is set. Alternatively, use Recharts' `onClick` prop on the `<Pie>` component directly. Test both approaches.
**Warning signs:** Clicking the donut does nothing; no navigation occurs.

### Pitfall 5: Time Range "All" Returning Too Much Data
**What goes wrong:** "All" time range returns years of daily data points, causing slow rendering and a cluttered chart.
**Why it happens:** No downsampling for large date ranges.
**How to avoid:** For ranges longer than ~180 days, consider weekly aggregation. However, given this is a personal finance app with at most ~1 year of data (started recently), the "All" range likely has <365 points which Recharts handles fine. Monitor performance but don't prematurely optimize.
**Warning signs:** Chart render takes >500ms; X-axis labels overlap severely.

### Pitfall 6: Group Panel Type Using Wrong Values
**What goes wrong:** Frontend sends panel type as "liquid" or "investments" but the DB expects "checking" or "investment".
**Why it happens:** Frontend uses display names, DB uses raw type values. The existing `PANEL_TYPE_TO_OVERRIDE` mapping in AccountsSection shows this distinction.
**How to avoid:** Reuse the existing `PANEL_TYPE_TO_OVERRIDE` mapping. API accepts and returns the DB-native values; frontend maps for display.
**Warning signs:** Groups created with invalid panel_type; CHECK constraint violation in SQLite.

### Pitfall 7: Growth Calculation for Groups
**What goes wrong:** Group growth badge shows incorrect percentage because it computes growth on current members only, not accounting for members that were added/removed during the 30-day window.
**Why it happens:** Growth calculation uses current group membership to look up historical balances.
**How to avoid:** This is actually the correct behavior for this use case -- growth reflects the current group composition. Document this clearly. The same approach is used for panel-level growth (new accounts contribute $0 to the prior total).
**Warning signs:** None expected; this matches the panel growth convention from Phase 6.

## Code Examples

### Stacked Area Chart Data Preparation (Frontend)

```typescript
// Source: project pattern from BalanceLineChart.prepareChartData
interface NetWorthApiPoint {
  date: string      // "2025-12-15"
  liquid: string    // decimal string
  savings: string
  investments: string
}

interface ChartDataPoint {
  date: string      // formatted "Mar 15"
  liquid: number
  savings: number
  investments: number
  total: number     // for tooltip
}

function prepareNetWorthData(points: NetWorthApiPoint[]): ChartDataPoint[] {
  return points.map((p) => {
    const [year, month, day] = p.date.split('-').map(Number)
    const dateObj = new Date(year, month - 1, day)
    const liquid = parseFloat(p.liquid)
    const savings = parseFloat(p.savings)
    const investments = parseFloat(p.investments)
    return {
      date: dateObj.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
      liquid,
      savings,
      investments,
      total: liquid + savings + investments,
    }
  })
}
```

### Net Worth Stats Computation (Backend)

```go
// Source: project pattern from growth.go computeGrowth + history.go dayAccumulator
func computeNetWorthStats(points []netWorthPoint) *netWorthStats {
    if len(points) == 0 {
        return nil
    }

    last := points[len(points)-1]
    first := points[0]

    currentTotal := last.liquid.Add(last.savings).Add(last.investments)
    firstTotal := first.liquid.Add(first.savings).Add(first.investments)

    change := currentTotal.Sub(firstTotal)
    var pctChange decimal.Decimal
    if !firstTotal.IsZero() {
        pctChange = change.Div(firstTotal).Mul(decimal.NewFromInt(100))
    }

    // Find all-time high
    allTimeHigh := decimal.Zero
    allTimeHighDate := ""
    for _, p := range points {
        total := p.liquid.Add(p.savings).Add(p.investments)
        if total.GreaterThan(allTimeHigh) {
            allTimeHigh = total
            allTimeHighDate = p.date
        }
    }

    return &netWorthStats{
        CurrentNetWorth:    currentTotal.StringFixed(2),
        PeriodChangeDollar: change.StringFixed(2),
        PeriodChangePct:    pctChange.StringFixed(2),
        AllTimeHigh:        allTimeHigh.StringFixed(2),
        AllTimeHighDate:    allTimeHighDate,
    }
}
```

### Group CRUD Handler Pattern (Backend)

```go
// Source: project pattern from update_account.go
// POST /api/groups
func CreateGroup(database *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            Name      string `json:"name"`
            PanelType string `json:"panel_type"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
            return
        }
        // Validate panel_type against allowed values
        // INSERT INTO account_groups (name, panel_type) VALUES (?, ?)
        // Return the created group with empty members array
    }
}
```

### Extending GetAccounts to Include Groups

```go
// After existing account query, also query groups with their members
groupsQuery := `
    SELECT g.id, g.name, g.panel_type,
           gm.account_id,
           COALESCE(a.display_name, a.name) AS account_name,
           a.name AS original_name,
           COALESCE(a.account_type_override, a.account_type) AS effective_type,
           a.currency, a.org_name,
           a.display_name, a.hidden_at, a.account_type_override,
           COALESCE(bs.balance, '0') AS balance
    FROM account_groups g
    LEFT JOIN group_members gm ON gm.group_id = g.id
    LEFT JOIN accounts a ON a.id = gm.account_id AND a.hidden_at IS NULL
    LEFT JOIN balance_snapshots bs ON bs.account_id = a.id
      AND bs.balance_date = (
          SELECT MAX(bs2.balance_date)
          FROM balance_snapshots bs2
          WHERE bs2.account_id = a.id
      )
    ORDER BY g.id, account_name
`
// Parse results, compute group totals with shopspring/decimal
```

### Time Range to Days Mapping

```typescript
// Time range selector maps to API ?days= parameter
const TIME_RANGE_DAYS: Record<string, number> = {
  '30d': 30,
  '90d': 90,
  '6m': 180,
  '1y': 365,
  'All': 0,  // 0 = no limit
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| window.location.href for navigation | useNavigate() from react-router-dom | react-router v6+ (2021) | SPA-friendly, no full reload, preserves app state |
| Separate chart data + stats API calls | Single endpoint returns both | Current best practice | Reduces HTTP round-trips, stats computed during same DB scan |
| Group membership as column on accounts | Junction table (group_members) | Relational DB standard | Cleaner schema, supports many-to-many, easier cascade deletes |

**Deprecated/outdated:**
- `window.location.href` for in-app navigation: Still works but causes full page reload. Use `useNavigate()` for SPA navigation. The project uses it once in Settings (`onNavigateDashboard`) which could be migrated but is out of scope.

## Open Questions

1. **Carry-forward logic for missing panel data in time series**
   - What we know: Not every panel has a snapshot on every date. The `dayAccumulator` only sums what exists.
   - What's unclear: Should we carry forward the last known value, or show zero? Carry-forward is more accurate for net worth (your investments don't disappear just because there's no snapshot).
   - Recommendation: Carry forward the last known value per panel. This is how the existing BalanceLineChart effectively works (it only shows dates with data, but for a stacked chart we need all three panels on every date).

2. **Group growth badge computation endpoint**
   - What we know: Panel-level growth is served by GET /api/growth. Group-level growth needs per-group 30-day change.
   - What's unclear: Should group growth be part of the existing /api/growth endpoint or a separate call?
   - Recommendation: Extend /api/growth to include a `groups` array with per-group growth data. This keeps growth data in one endpoint and one Dashboard fetch.

3. **Handling groups in the summary endpoint**
   - What we know: GetSummary computes panel totals from all visible accounts. Groups don't change the math (a grouped account's balance still contributes to its panel's total).
   - What's unclear: Do groups affect summary at all?
   - Recommendation: Groups do NOT affect the summary endpoint. The summary computes panel totals from all visible accounts regardless of grouping. The group's `panel_type` determines display location, but the underlying account data drives the math. However, accounts in a group should use the GROUP's panel_type for panel assignment, not their own effective_type. This is a subtle but important change to how summary, growth, and history queries work.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (frontend) | Vitest + @testing-library/react + jsdom |
| Framework (backend) | Go testing with httptest |
| Config file (frontend) | `frontend/vitest.config.ts` |
| Config file (backend) | Built-in Go test runner |
| Quick run command (frontend) | `cd frontend && npx vitest run --reporter=verbose` |
| Quick run command (backend) | `cd internal && go test ./...` |
| Full suite command | `cd frontend && npx vitest run && cd ../internal && go test ./...` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ACCT-03 | Create group, add/remove members via API | unit (backend) | `go test ./internal/api/handlers/ -run TestGroup -v` | Wave 0 |
| ACCT-03 | Group management UI in Settings (drag, create) | unit (frontend) | `cd frontend && npx vitest run src/components/AccountsSection.test.tsx` | Extend existing |
| ACCT-04 | Group row renders in PanelCard with summed balance | unit (frontend) | `cd frontend && npx vitest run src/components/PanelCard.test.tsx` | Extend existing |
| ACCT-05 | Group row expand/collapse shows members | unit (frontend) | `cd frontend && npx vitest run src/components/GroupRow.test.tsx` | Wave 0 |
| INSIGHT-02 | Donut click navigates to /net-worth | unit (frontend) | `cd frontend && npx vitest run src/components/NetWorthDonut.test.tsx` | Extend existing |
| INSIGHT-03 | Stacked area chart renders 3 layers | unit (frontend) | `cd frontend && npx vitest run src/components/StackedAreaChart.test.tsx` | Wave 0 |
| INSIGHT-04 | Summary stats display correctly | unit (frontend) | `cd frontend && npx vitest run src/components/NetWorthStats.test.tsx` | Wave 0 |
| INSIGHT-04 | Net worth stats API returns correct values | unit (backend) | `go test ./internal/api/handlers/ -run TestNetWorth -v` | Wave 0 |
| INSIGHT-05 | Time range selector updates chart and stats | unit (frontend) | `cd frontend && npx vitest run src/components/TimeRangeSelector.test.tsx` | Wave 0 |

### Sampling Rate
- **Per task commit:** Quick run commands for affected files
- **Per wave merge:** Full suite (`cd frontend && npx vitest run && cd ../internal && go test ./...`)
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/api/handlers/groups_test.go` -- covers ACCT-03 (CRUD operations, auto-delete empty group, panel type validation)
- [ ] `internal/api/handlers/networth_test.go` -- covers INSIGHT-03/04/05 (time-series, stats, time range filtering)
- [ ] `frontend/src/components/GroupRow.test.tsx` -- covers ACCT-04/05 (collapse/expand, member display)
- [ ] `frontend/src/components/StackedAreaChart.test.tsx` -- covers INSIGHT-03 (renders 3 stacked areas)
- [ ] `frontend/src/components/NetWorthStats.test.tsx` -- covers INSIGHT-04 (stats display)
- [ ] `frontend/src/components/TimeRangeSelector.test.tsx` -- covers INSIGHT-05 (selection, callback)
- [ ] `frontend/src/pages/NetWorth.test.tsx` -- covers INSIGHT-02/03/04/05 (page integration)
- Existing test files to extend: `PanelCard.test.tsx`, `NetWorthDonut.test.tsx`, `AccountsSection.test.tsx`

## Sources

### Primary (HIGH confidence)
- Existing codebase files (read directly): `PanelCard.tsx`, `NetWorthDonut.tsx`, `AccountsSection.tsx`, `BalanceLineChart.tsx`, `panelColors.ts`, `App.tsx`, `Dashboard.tsx`, `client.ts`, `router.go`, `handlers/history.go`, `handlers/growth.go`, `handlers/accounts.go`, `handlers/summary.go`, `handlers/update_account.go`, `db/db.go`, `migrations/000001_init.up.sql`, `migrations/000002_account_metadata.up.sql`
- Project decisions from CONTEXT.md and STATE.md
- Package versions verified from installed node_modules and go.mod

### Secondary (MEDIUM confidence)
- [Recharts Stacked Area Chart official example](https://recharts.github.io/en-US/examples/StackedAreaChart/) -- stackId pattern verified
- [Recharts Area API](https://recharts.github.io/en-US/api/Area/) -- stackId prop documentation
- [React Router useNavigate API](https://reactrouter.com/api/hooks/useNavigate) -- programmatic navigation

### Tertiary (LOW confidence)
- None -- all findings verified against installed codebase or official docs

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries already installed and in use; no new dependencies
- Architecture: HIGH -- all patterns extend proven existing code (dayAccumulator, CRUD handlers, dnd-kit drag-drop, Recharts charts)
- Pitfalls: HIGH -- derived from direct codebase analysis and understanding of data flow
- Database schema: HIGH -- follows SQLite relational patterns with existing FK support enabled

**Research date:** 2026-03-16
**Valid until:** 2026-04-16 (stable -- no dependency changes expected)
