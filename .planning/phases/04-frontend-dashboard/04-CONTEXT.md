# Phase 4: Frontend Dashboard - Context

**Gathered:** 2026-03-15
**Status:** Ready for planning

<domain>
## Phase Boundary

The user sees a complete, polished finance dashboard with liquid/savings/investments panels, balance-over-time charts, net worth donut chart, and UX polish (dark mode, mobile responsive, loading/empty states, data freshness indicator). Consumes the REST API built in Phase 3. No backend changes in this phase.

</domain>

<decisions>
## Implementation Decisions

### Charting Library
- Recharts — declarative React charting, good dark mode support, ~45KB gzipped
- No other charting libraries needed (Recharts covers line charts and pie/donut)

### Line Chart Style
- Smooth monotone curves with subtle area fill beneath each line
- Tooltips show date + balance + change from previous day (e.g., "Mar 14: $4,230.50 ↑$50.00")
- One tabbed chart area — tabs to switch between Liquid / Savings / Investments series

### Net Worth Donut Chart
- Shows percentage in donut segments, total net worth in center, dollar amounts in legend/tooltip
- Positioned beside the line chart (~35% width) on desktop; stacks below on mobile
- Segment colors match panel accent colors for visual consistency

### Dashboard Layout
- 3-column grid for panels on desktop, stacks vertically on mobile
- Each panel card shows: category label, total balance, individual account list with balances
- Below panels: tabbed line chart (~65% width) + net worth donut (~35% width) side by side
- "Last synced X ago" freshness indicator near the top of the dashboard (below page title area)

### Visual Style
- Clean fintech aesthetic — subtle card shadows, rounded corners, muted backgrounds
- Think Linear / Mercury — professional but not sterile
- Each panel has a distinct accent color (e.g., blue for liquid, green for savings, purple for investments)
- Accent colors carry through to line chart series and donut segments

### Dark Mode
- Manual toggle only — sun/moon icon in the NavBar
- No OS preference detection
- Preference stored in localStorage (no API call, instant on page load)
- Tailwind v4 dark mode classes

### Empty & Loading States
- First launch (no sync): setup prompt — "Connect your accounts to get started" with button linking to Settings
- Data loading (after sync configured): skeleton cards with shimmering animation matching the panel layout
- Empty panels (no accounts of a type): hide the panel entirely, remaining panels expand to fill space
- API errors: inline in content area — "Something went wrong" with retry button, no toast library

### Claude's Discretion
- Exact color values for panel accents and dark mode palette
- Skeleton animation implementation details
- Chart axis formatting and responsive breakpoints
- Typography scale and spacing values
- Account list item styling within panels
- Tab component design for line chart switcher
- Mobile breakpoint (likely md: or lg: Tailwind breakpoint)

</decisions>

<specifics>
## Specific Ideas

- Skeleton cards should have shimmering animation showing panels "coming online" — not just static gray boxes
- Clean fintech look like Linear or Mercury — professional, not sterile or generic
- Tooltips with change indicator (↑/↓) give the line charts a trading-platform feel without being overwhelming

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `frontend/src/api/client.ts`: Typed fetch wrappers with `credentials: 'include'` — extend with getSummary(), getAccounts(), getBalanceHistory()
- `frontend/src/App.tsx`: Placeholder Dashboard component — replace with real dashboard
- `frontend/src/App.tsx`: NavBar component — extend with dark mode toggle icon
- react-router-dom v7 already configured for client-side routing

### Established Patterns
- Tailwind v4 CSS-first config (`@import "tailwindcss"` in index.css, no tailwind.config.js)
- React 19 with TypeScript, Vite 7
- Tests colocated with source files (.test.tsx beside components)
- API client pattern: typed async functions returning typed responses
- Vitest + Testing Library for component tests

### Integration Points
- `GET /api/summary` → liquid, savings, investments totals + last_synced_at
- `GET /api/accounts` → accounts grouped by type (liquid, savings, investments, other)
- `GET /api/balance-history?days=N` → per-panel daily time series (liquid, savings, investments)
- All balance values are JSON strings — parse with Number() or parseFloat()
- NavBar already exists with Settings link — add dark mode toggle
- BrowserRouter Routes already set up — Dashboard is the "/" route

</code_context>

<deferred>
## Deferred Ideas

- Per-account balance history drill-down — v2 requirement (DRILL-01)
- Account APY display — v2 requirement (DRILL-02)
- Investment growth/loss calculations — v2 requirement (DRILL-03)

</deferred>

---

*Phase: 04-frontend-dashboard*
*Context gathered: 2026-03-15*
