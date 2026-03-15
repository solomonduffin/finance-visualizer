---
phase: 04-frontend-dashboard
plan: 03
subsystem: ui
tags: [react, recharts, charts, visualization, dashboard, typescript]

# Dependency graph
requires:
  - phase: 04-frontend-dashboard
    plan: 01
    provides: "PANEL_COLORS, useDarkMode, formatCurrency, BalanceHistoryResponse, SummaryResponse interfaces"
  - phase: 04-frontend-dashboard
    plan: 02
    provides: "Dashboard page with history/summary state and charts-section placeholder"
provides:
  - "BalanceLineChart: tabbed area chart with monotone curves, gradient fill, and custom delta tooltip"
  - "NetWorthDonut: donut chart with proportional segments, center total, and custom legend"
  - "Complete dashboard visualization layer wired below panel cards"
affects: [05-polish, future-phases]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "recharts mock pattern: vi.mock('recharts') with data-testid and data-* attrs for jsdom testing"
    - "TDD with exported pure function (prepareChartData) for direct unit testing"
    - "useDarkMode called inside Dashboard component for chart color selection"

key-files:
  created:
    - frontend/src/components/BalanceLineChart.tsx
    - frontend/src/components/BalanceLineChart.test.tsx
    - frontend/src/components/NetWorthDonut.tsx
    - frontend/src/components/NetWorthDonut.test.tsx
  modified:
    - frontend/src/pages/Dashboard.tsx
    - frontend/src/pages/Dashboard.test.tsx
    - frontend/src/api/client.ts
    - frontend/src/components/PanelCard.tsx
    - internal/api/handlers/accounts.go
    - internal/api/handlers/accounts_test.go
    - internal/sync/sync.go
    - internal/sync/sync_test.go

key-decisions:
  - "Export prepareChartData from BalanceLineChart for direct unit testing without recharts dependency"
  - "useDarkMode called inside Dashboard (not App) — two independent hook instances read same localStorage, avoids prop drilling"
  - "recharts SVG elements (defs, linearGradient, stop) cause jsdom warnings but tests pass — documented non-issue"
  - "Custom legend div instead of recharts Legend component — avoids dark mode styling issues per plan spec"
  - "/api/accounts merges checking+credit into 'liquid' key to align with frontend panel model (fix during visual verify)"
  - "InferAccountType: sapphire and platinum keywords added; org_name threaded to PanelCard for institution context"
  - "SyncOnce deletes stale accounts not present in latest SimpleFIN response to prevent orphaned data"

patterns-established:
  - "recharts mock pattern: vi.mock with data-testid captures on Pie/Area for data-count and data-stroke assertions"
  - "Chart wiring pattern: conditional render {history && summary && (<charts>)} guards API data availability"

requirements-completed: [VIZ-01, VIZ-02]

# Metrics
duration: 41min
completed: 2026-03-15
---

# Phase 4 Plan 3: Charts — BalanceLineChart + NetWorthDonut Summary

**Recharts-powered tabbed area chart and net worth donut wired into Dashboard with dark mode support and 32 passing tests, plus three backend/sync correctness fixes discovered during visual verification**

## Performance

- **Duration:** ~41 min
- **Started:** 2026-03-15T16:30:20Z
- **Completed:** 2026-03-15T17:11:14Z
- **Tasks:** 3 of 3 (Task 3 human-verify checkpoint — APPROVED)
- **Files modified:** 12

## Accomplishments
- BalanceLineChart: tabbed panel switching (Liquid/Savings/Investments), monotone AreaChart with gradient fill, custom tooltip showing date + formatted balance + delta arrows
- NetWorthDonut: donut with 3 proportional segments, formatted center total, zero-value exclusion, custom legend with colored dots and amounts
- Dashboard charts section: flex layout with ~65% line chart + ~35% donut on desktop, stacked on mobile
- 32 passing tests across all 12 test files; full suite green
- Visual verification approved; three backend/sync bugs discovered and fixed: accounts key alignment, account type inference, stale account cleanup

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: BalanceLineChart tests** - `878ab02` (test)
2. **Task 1 GREEN: BalanceLineChart implementation** - `4636b53` (feat)
3. **Task 2 RED: NetWorthDonut tests** - `8bcbcc3` (test)
4. **Task 2 GREEN: NetWorthDonut + Dashboard wiring** - `872f71d` (feat)
5. **Task 3 verify fix: /api/accounts key alignment** - `436ed91` (fix)
6. **Task 3 verify fix: InferAccountType + org_name in PanelCard** - `c4dc930` (fix)
7. **Task 3 verify fix: stale account cleanup in sync** - `d5cd061` (fix)

_Note: TDD tasks have separate RED (test) and GREEN (implementation) commits. Task 3 was a human-verify checkpoint with fixes applied during verification._

## Files Created/Modified
- `frontend/src/components/BalanceLineChart.tsx` - Tabbed area chart with prepareChartData, CustomTooltip, gradient fill, dark mode accents
- `frontend/src/components/BalanceLineChart.test.tsx` - 11 tests covering tabs, colors, empty states, delta computation, date formatting
- `frontend/src/components/NetWorthDonut.tsx` - Donut chart with segment filtering, center label, custom legend
- `frontend/src/components/NetWorthDonut.test.tsx` - 7 tests covering segments, zero exclusion, colors, legend, empty state
- `frontend/src/pages/Dashboard.tsx` - Added BalanceLineChart + NetWorthDonut imports, useDarkMode call, replaced placeholder with responsive charts section
- `frontend/src/pages/Dashboard.test.tsx` - Added mocks for chart components and useDarkMode; 5 new chart section tests
- `frontend/src/api/client.ts` - org_name added to AccountItem interface
- `frontend/src/components/PanelCard.tsx` - org_name displayed as "OrgName – AccountName" prefix in account list
- `internal/api/handlers/accounts.go` - checking+credit accounts merged under 'liquid' response key
- `internal/api/handlers/accounts_test.go` - Tests updated for new key structure
- `internal/sync/sync.go` - InferAccountType: sapphire/platinum keywords added; stale account deletion after upsert
- `internal/sync/sync_test.go` - Tests for new InferAccountType keywords and stale account deletion

## Decisions Made
- Exported `prepareChartData` from `BalanceLineChart.tsx` so it can be unit tested directly without recharts dependency — cleaner test coverage
- Called `useDarkMode()` directly in Dashboard rather than prop-drilling from App — two independent hook instances both read same localStorage key, no state divergence
- jsdom SVG element warnings (`<defs>`, `<linearGradient>`, `<stop>` unknown elements) are cosmetic only — tests pass, no fix needed; documented as non-issue
- `/api/accounts` restructured to use `liquid` key (merging `checking` and `credit`) matching frontend panel model
- `org_name` threaded through API to PanelCard for institution context display
- Stale account cleanup: DELETE WHERE id NOT IN (...) after upsert prevents orphaned rows

## Deviations from Plan

### Auto-fixed Issues (during Task 3 visual verification)

**1. [Rule 1 - Bug] /api/accounts response keys mismatched frontend panel model**
- **Found during:** Task 3 (visual verification — liquid panel showed empty)
- **Issue:** Backend returned `{checking, credit, savings, investments, other}` but frontend expected `{liquid, savings, investments, other}`; liquid panel always empty
- **Fix:** Merged checking and credit accounts under `liquid` key in `accounts.go`; updated handler tests
- **Files modified:** `internal/api/handlers/accounts.go`, `internal/api/handlers/accounts_test.go`
- **Verification:** Accounts endpoint returns `liquid` key; liquid panel shows checking/credit accounts correctly
- **Committed in:** `436ed91`

**2. [Rule 1 - Bug] Chase Sapphire Preferred classified as unknown account type**
- **Found during:** Task 3 (visual verification — Sapphire card missing from liquid panel)
- **Issue:** `InferAccountType` didn't recognize "sapphire" or "platinum" keywords; premium card names classified as unknown
- **Fix:** Added "sapphire" and "platinum" to credit keyword list; added org_name field to API response and PanelCard display
- **Files modified:** `internal/sync/sync.go`, `internal/sync/sync_test.go`, `frontend/src/api/client.ts`, `frontend/src/components/PanelCard.tsx`
- **Verification:** InferAccountType tests pass for new keywords; card appears in liquid panel with "Chase – Sapphire Preferred" label
- **Committed in:** `c4dc930`

**3. [Rule 2 - Missing Critical] Stale accounts not removed on sync**
- **Found during:** Task 3 (visual verification — disconnected institution left phantom accounts)
- **Issue:** SyncOnce upserting accounts but never deleting rows for accounts no longer returned by SimpleFIN; orphaned accounts persist indefinitely
- **Fix:** After upsert loop, collect seen account IDs and DELETE WHERE id NOT IN (...) for this access URL; snapshots cascade-deleted via FK
- **Files modified:** `internal/sync/sync.go`, `internal/sync/sync_test.go`
- **Verification:** Stale account cleanup test passes; orphaned rows removed on subsequent sync
- **Committed in:** `d5cd061`

---

**Total deviations:** 3 auto-fixed (2 Rule 1 bugs, 1 Rule 2 missing critical)
**Impact on plan:** All three fixes required for correct dashboard behavior with real data. No scope creep.

## Issues Encountered
- jsdom produces console warnings for SVG-specific recharts elements (`defs`, `linearGradient`, `stop`) because the recharts mock in tests renders them as lowercase HTML elements. These are warnings only — all assertions pass. No fix applied.

## User Setup Required
None - no external service configuration required. SimpleFIN token setup is unchanged from Phase 2.

## Next Phase Readiness
- Complete finance dashboard delivered: panel cards, tabbed line chart, net worth donut, dark mode, loading/empty states, responsive layout
- All four phases (Foundation, Data Pipeline, Backend API, Frontend Dashboard) are complete
- Project is at v1.0 feature completeness
- Visual verification APPROVED by user

---
*Phase: 04-frontend-dashboard*
*Completed: 2026-03-15*

## Self-Check: PASSED

All files confirmed present and all commits verified in git log (878ab02, 4636b53, 8bcbcc3, 872f71d, 436ed91, c4dc930, d5cd061).
