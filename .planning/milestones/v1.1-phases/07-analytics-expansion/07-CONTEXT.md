# Phase 7: Analytics Expansion - Context

**Gathered:** 2026-03-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Custom account groups that act as organizational folders (e.g., combining multiple Coinbase wallets into one "Coinbase" entry) and a dedicated net worth drill-down page with historical stacked area chart, summary statistics, and time range selector. Groups appear as collapsible rows in dashboard panels with summed balances. The net worth page is the first new page beyond Dashboard and Settings.

</domain>

<decisions>
## Implementation Decisions

### Account Group Model
- Groups act as accounts — a group is assigned to a panel type (Liquid/Savings/Investments) and its summed balance appears in that panel
- Groups are NOT limited by member account type — any account can be placed in any group regardless of its original type (organizational folder model)
- Ungrouped accounts are allowed — groups are optional organization, existing accounts keep working standalone
- When the last account is removed from a group, the group is auto-deleted (no empty group persistence)
- Growth badge calculated on group's summed member balance over 30 days (consistent with panel-level growth from Phase 6)

### Group Management UX (Settings)
- "+ New Group" button in the Accounts section creates a new group with inline name entry
- Drag-and-drop to assign accounts into a group (extends existing drag pattern from Phase 5 type reassignment)
- Drag-and-drop to assign group to a panel type section (same pattern as individual accounts)
- Drag accounts out of a group to ungroup them back to standalone
- Grouped accounts no longer appear individually in panel lists — they only exist inside the group
- Mobile: drag-and-drop with existing mobile fallback patterns from Phase 5

### Group Display on Dashboard
- Groups appear as collapsible rows in their assigned panel card: group name + summed balance + expand chevron
- Collapsed by default — keeps dashboard compact, aligns with "one glance" core value
- Clicking expands to reveal indented member accounts with individual balances
- Visual distinction: bold group name + collapse/expand chevron (subtle, not heavy)
- Growth badge shown on individual group rows (alongside the panel total growth badge)

### Net Worth Page Layout
- New route: /net-worth
- Navigation: clicking the net worth donut chart navigates to /net-worth (INSIGHT-02) + "Net Worth" link added to top nav bar
- Layout: summary stats bar at top, full-width chart below
- Summary stats: current net worth, period change ($ and %), all-time high
- Period change matches selected time range (if 90d selected, shows 90-day change)
- Line chart only on this page — no donut (donut lives on dashboard, avoids redundancy)
- Time range selector: segmented control with options 30d, 90d, 6m, 1y, All

### Net Worth Chart Style
- Stacked area chart with three layers: Liquid (bottom), Savings (middle), Investments (top)
- Top line of stacked area = total net worth
- Uses same panel colors from panelColors.ts — consistent visual language across dashboard and net worth page
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

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — ACCT-03 (create groups), ACCT-04 (group display), ACCT-05 (expand group), INSIGHT-02 (donut click), INSIGHT-03 (NW line chart), INSIGHT-04 (NW stats), INSIGHT-05 (time range selector)
- `.planning/ROADMAP.md` — Phase 7 success criteria (5 criteria)

### Prior Context
- `.planning/phases/05-data-foundation/05-CONTEXT.md` — Settings page structure, drag-and-drop pattern, inline edit pattern, instant-save pattern, account display name rendering
- `.planning/phases/06-operational-quick-wins/06-CONTEXT.md` — Growth badge design, growth calculation approach, panel card structure, Settings page sections

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `frontend/src/components/PanelCard.tsx`: Panel card with account list — needs group row support (collapsible, summed balance, growth badge)
- `frontend/src/components/GrowthBadge.tsx`: Growth badge component — reusable on group rows
- `frontend/src/components/BalanceLineChart.tsx`: Line chart with Recharts — reference for stacked area chart implementation
- `frontend/src/components/NetWorthDonut.tsx`: Donut chart — needs click handler to navigate to /net-worth
- `frontend/src/components/panelColors.ts`: Panel accent colors — reuse for stacked area chart layers
- `frontend/src/components/AccountsSection.tsx`: Settings accounts section with drag-and-drop — extend with group management
- `frontend/src/utils/format.ts`: `formatCurrency` — reuse for net worth stats
- `frontend/src/utils/account.ts`: `getAccountDisplayName` — accounts inside groups still need display name rendering

### Established Patterns
- `shopspring/decimal` for all financial arithmetic (balance stored as TEXT)
- Tailwind v4 CSS-first config — all new UI uses Tailwind classes
- `COALESCE(display_name, name)` in all account queries
- `COALESCE(account_type_override, account_type)` for effective type grouping
- `WHERE hidden_at IS NULL` excludes hidden accounts
- Per-item instant save in Settings (no batch save button)
- `react-router-dom` BrowserRouter with Routes in App.tsx
- Recharts for all chart components

### Integration Points
- `frontend/src/App.tsx`: Add `/net-worth` route and "Net Worth" nav link
- `frontend/src/pages/Dashboard.tsx`: Pass group data to PanelCard, add click handler on NetWorthDonut
- `internal/api/router.go`: Register new endpoints (group CRUD, net worth history with time range)
- `internal/api/handlers/history.go`: `dayAccumulator` pattern — extend for net worth time-series with grouping
- `internal/api/handlers/accounts.go`: Account grouping affects how accounts are returned and displayed
- `internal/api/handlers/growth.go`: Growth calculation needs group-level computation
- `internal/db/migrations/`: New migration for account_groups and group_members tables

</code_context>

<specifics>
## Specific Ideas

- Groups should feel like organizational folders — "grab and drop accounts into a folder"
- User expects most/all accounts to be in groups eventually, so the UX should make grouping feel natural and lightweight
- Segmented control for time range selector — slightly more polished than pill buttons
- Net worth page is the first new page — sets the tone for future pages (alerts, projections)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 07-analytics-expansion*
*Context gathered: 2026-03-16*
