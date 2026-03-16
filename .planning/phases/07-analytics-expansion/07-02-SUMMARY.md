---
phase: 07-analytics-expansion
plan: 02
subsystem: api, ui
tags: [recharts, stacked-area-chart, net-worth, time-series, shopspring-decimal, react-router]

# Dependency graph
requires:
  - phase: 04-frontend-dashboard
    provides: Dashboard page with NetWorthDonut, BalanceLineChart, panelColors
  - phase: 05-data-foundation
    provides: COALESCE(account_type_override, account_type) pattern, hidden_at filtering
  - phase: 06-operational-quick-wins
    provides: Growth endpoint pattern, queryPanelTotals helper, computeGrowth
provides:
  - GET /api/net-worth endpoint with per-panel time-series and summary statistics
  - Net Worth page with stacked area chart, stats bar, time range selector
  - Donut-to-drill-down navigation pattern (click donut -> /net-worth)
  - TimeRangeSelector reusable component
  - NetWorthStats reusable component
  - StackedAreaChart reusable component with prepareNetWorthData utility
affects: [07-03, 08-alerts]

# Tech tracking
tech-stack:
  added: []
  patterns: [carry-forward missing panel data, stacked area chart with gradient fills, segmented radio control]

key-files:
  created:
    - internal/api/handlers/networth.go
    - internal/api/handlers/networth_test.go
    - frontend/src/components/TimeRangeSelector.tsx
    - frontend/src/components/TimeRangeSelector.test.tsx
    - frontend/src/components/NetWorthStats.tsx
    - frontend/src/components/NetWorthStats.test.tsx
    - frontend/src/components/StackedAreaChart.tsx
    - frontend/src/components/StackedAreaChart.test.tsx
    - frontend/src/pages/NetWorth.tsx
    - frontend/src/pages/NetWorth.test.tsx
  modified:
    - internal/api/router.go
    - frontend/src/api/client.ts
    - frontend/src/components/NetWorthDonut.tsx
    - frontend/src/components/NetWorthDonut.test.tsx
    - frontend/src/App.tsx

key-decisions:
  - "Carry-forward logic fills missing panel data with last known value (not zero) for continuous stacked chart"
  - "period_change_pct is null (not zero) when first-point total is zero to avoid division by zero"
  - "TimeRangeSelector uses role=radiogroup/radio pattern for accessibility"
  - "NetWorthDonut wraps in clickable div with role=link for drill-down navigation"

patterns-established:
  - "Carry-forward pattern: track last known values per panel, fill gaps on dates with missing data"
  - "Time range selector: segmented radio control with 30d/90d/6m/1y/All options"
  - "Drill-down navigation: donut click -> detail page via useNavigate"

requirements-completed: [INSIGHT-02, INSIGHT-03, INSIGHT-04, INSIGHT-05]

# Metrics
duration: 8min
completed: 2026-03-16
---

# Phase 07 Plan 02: Net Worth Drill-Down Summary

**Full-stack net worth drill-down: backend time-series API with carry-forward and stats, frontend stacked area chart page with time range selector and donut click navigation**

## Performance

- **Duration:** 8 min
- **Started:** 2026-03-16T05:08:48Z
- **Completed:** 2026-03-16T05:16:54Z
- **Tasks:** 2
- **Files modified:** 15

## Accomplishments
- GET /api/net-worth endpoint with per-panel time-series, carry-forward logic, and computed stats (current NW, period change, all-time high)
- Net Worth page at /net-worth with stacked area chart (3 layers), summary statistics bar, and 5-option time range selector
- Donut chart on Dashboard now navigates to /net-worth on click (SPA navigation, no reload)
- Nav bar updated with "Net Worth" link between dark mode toggle and Settings
- 12 backend tests + 165 frontend tests all passing

## Task Commits

Each task was committed atomically (TDD: test -> feat):

1. **Task 1: Net worth API endpoint** - `9dfc32c` (test: failing tests) -> `3f5c8e0` (feat: implementation)
2. **Task 2: Net Worth frontend page** - `d58ed34` (test: failing tests) -> `4695563` (feat: implementation)

_Note: TDD tasks have RED (test) and GREEN (feat) commits_

## Files Created/Modified
- `internal/api/handlers/networth.go` - GetNetWorth handler with time-series + stats
- `internal/api/handlers/networth_test.go` - 12 test cases for endpoint
- `internal/api/router.go` - Added /api/net-worth route
- `frontend/src/api/client.ts` - NetWorthPoint, NetWorthStatsData, NetWorthResponse types + getNetWorth function
- `frontend/src/components/TimeRangeSelector.tsx` - Segmented radio control (30d/90d/6m/1y/All)
- `frontend/src/components/NetWorthStats.tsx` - Stats bar (current NW, period change, ATH)
- `frontend/src/components/StackedAreaChart.tsx` - Recharts stacked area with gradient fills
- `frontend/src/pages/NetWorth.tsx` - Net Worth page with loading/empty/error/data states
- `frontend/src/components/NetWorthDonut.tsx` - Added useNavigate click handler to /net-worth
- `frontend/src/App.tsx` - Added /net-worth route + "Net Worth" nav link

## Decisions Made
- Carry-forward logic fills missing panel data with last known value (not zero) ensuring continuous stacked chart display
- period_change_pct returns null when first-point total is zero (avoids division by zero, consistent with growth endpoint pattern)
- TimeRangeSelector uses role=radiogroup/radio for accessibility compliance
- NetWorthDonut wraps chart in clickable div with role=link and group-hover:shadow-lg for drill-down UX
- Settings link updated from font-medium to font-semibold for consistency with Net Worth link (2-weight typography budget)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Net worth drill-down page complete, ready for Plan 07-03 (spending/category analytics)
- TimeRangeSelector and StackedAreaChart components available for reuse in future analytics pages
- Stacked area chart pattern established for any multi-series financial visualization

## Self-Check: PASSED

All 11 files verified present. All 4 task commits verified in git log.

---
*Phase: 07-analytics-expansion*
*Completed: 2026-03-16*
