# Phase 9: Projection Engine - Context

**Gathered:** 2026-03-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Forward-looking net worth projection page with per-account (and per-holding for investment accounts) growth rates, compound/simple interest toggle, income allocation modeling, and holdings detail display. Users configure rates and income on the projections page itself and see the chart update live. All settings persist in the database across sessions. A new "Projections" nav entry provides access.

</domain>

<decisions>
## Implementation Decisions

### Projection Chart Design
- Single aggregate net worth line chart (not per-panel stacked area)
- Historical data shown as solid line (recent ~6 months), projection continues as dashed line from today's date
- Clear visual break at the "now" boundary — solid → dashed transition
- Time horizon selector: segmented control with presets (1y, 5y, 10y, 20y) plus a "Custom" button that reveals a year input field
- Chart includes historical context for a "where you've been → where you're going" narrative
- Built with Recharts (consistent with existing StackedAreaChart from Phase 7)

### Rate & Toggle Configuration
- Configuration table lives directly on the /projections page below the chart (self-contained workflow)
- Single table grouped by panel type (Liquid / Savings / Investments) — columns: account name, APY input, compound/simple toggle, include/exclude checkbox
- APY input is annual percentage only (no monthly/quarterly unit selector)
- Per-account compound/simple toggle (not global) — savings typically compound, some investments may not
- Accounts with 0% or no rate set appear as flat lines at current balance in the projection (not excluded)
- Chart updates live with ~500ms debounce as user adjusts any rate or toggle — instant feedback loop

### Per-Holding Rates for Investment Accounts
- Investment accounts WITH holdings data from SimpleFIN: each holding gets its own APY/rate input and compound/simple toggle. The account row expands to reveal holdings with individual rate controls. No account-level rate needed.
- Investment accounts WITHOUT holdings data: rate input is at the account level (fallback behavior)
- Savings and Liquid accounts: always account-level rate (no holdings concept)
- Expand/collapse pattern reuses the chevron pattern from account groups (Phase 7)

### Income Modeling
- Separate collapsible "Income Modeling" section below the account rate table
- Toggle to enable/disable income modeling — when off, projection uses growth rates only
- Inputs: annual income and monthly savings percentage
- Allocation via percentage sliders/inputs per account — must sum to 100%
- Allocation only shows accounts that are checked "include" in the rate table — excluded accounts don't receive income
- Chart updates live as income parameters change (same debounce as rate changes)

### Holdings Display
- Investment accounts with holdings: expandable row in the rate table reveals holdings underneath
- Each holding shows: fund/stock name and current dollar value
- Accounts without holdings data: no expand chevron, just shows account name + balance + rate input like any other account (no "N/A" clutter)

### Navigation & Page Structure
- New route: /projections
- "Projections" link added to top nav bar alongside Dashboard, Net Worth, Alerts, Settings
- Page layout: chart at top (with time horizon selector), rate configuration table below, income modeling section at bottom (collapsible)
- Empty state if no accounts exist (uses EmptyState component pattern)

### Claude's Discretion
- Exact chart styling (colors, gradients, line thickness, tooltip design)
- Rate table row styling and responsive layout
- Percentage slider/input component design for income allocation
- Database schema for projection settings (rates, toggles, income config)
- API endpoint design for projection settings CRUD
- How to fetch and store holdings data from SimpleFIN
- Loading and error state designs
- Projection calculation engine (monthly compounding math, income distribution logic)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

No external specs — requirements are fully captured in REQUIREMENTS.md (PROJ-01 through PROJ-08) and the decisions above.

### Requirements
- `.planning/REQUIREMENTS.md` §Financial Projections — PROJ-01 through PROJ-08 define the acceptance criteria

### Prior Phase Context
- `.planning/phases/07-analytics-expansion/07-CONTEXT.md` — Net Worth page patterns (StackedAreaChart, TimeRangeSelector, nav bar addition, panelColors.ts)
- `.planning/phases/08-alert-system/08-CONTEXT.md` — Inline form builder pattern, Alerts page nav integration, EmptyState usage
- `.planning/phases/05-data-foundation/05-CONTEXT.md` — Display name rendering rules (ACCT-02), account grouping by panel type in Settings

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `StackedAreaChart` component: Recharts-based chart, can inform projection chart implementation (though projection uses a single line, not stacked area)
- `TimeRangeSelector` component: segmented control pattern — adaptable for projection time horizon selector
- `EmptyState` component: empty page pattern for /projections when no accounts exist
- `panelColors.ts`: panel type color scheme for grouping accounts in the rate table
- `Toast` component: success/error feedback for saving projection settings
- `GrowthBadge` component: percentage display pattern (may inform rate input styling)
- Account group expand/collapse pattern (`GroupRow` component): reusable for holdings expand in rate table

### Established Patterns
- Recharts for all charting (BalanceLineChart, StackedAreaChart, NetWorthDonut)
- Go + chi router for API endpoints with JWT auth
- SQLite for persistence with migrations in `internal/db/migrations.go`
- Display names via COALESCE(display_name, name) across all queries
- Inline form patterns from alert expression builder

### Integration Points
- Router: new `/api/projections/*` endpoints in `internal/api/router.go`
- Nav bar: add "Projections" link (same pattern as Net Worth and Alerts additions)
- SimpleFIN client: may need extension to fetch holdings data (`internal/simplefin/client.go`)
- Frontend routing: add /projections route alongside Dashboard, NetWorth, Alerts, Settings

</code_context>

<specifics>
## Specific Ideas

- Per-holding growth rates are critical for investment accounts — a Vanguard account might hold VTSAX at 10% and bonds at 4%, so per-holding rates give accurate projections
- The page should feel exploratory — adjust a rate, see the line move. Interactive "what-if" tool.
- Income modeling is optional (toggled) so users who just want growth rate projections aren't overwhelmed

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 09-projection-engine*
*Context gathered: 2026-03-16*
