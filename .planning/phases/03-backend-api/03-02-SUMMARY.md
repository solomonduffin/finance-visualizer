---
phase: 03-backend-api
plan: 02
subsystem: api
tags: [go, chi, sqlite, shopspring-decimal, jwt, time-series]

# Dependency graph
requires:
  - phase: 03-backend-api-01
    provides: GetSummary and GetAccounts handlers, balance_snapshots + accounts schema

provides:
  - GetBalanceHistory handler at internal/api/handlers/history.go
  - GET /api/balance-history endpoint with per-panel time-series aggregation
  - All three new routes wired into JWT-protected router group (/api/summary, /api/accounts, /api/balance-history)
  - Complete Phase 3 REST API contract (three endpoints, all behind JWT auth)

affects: [04-frontend-dashboard, frontend, charts, dashboard]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Factory handler pattern: func GetXxx(database *sql.DB) http.HandlerFunc"
    - "Per-day accumulator map pattern for time-series aggregation in Go"
    - "DATE() SQL function to normalize balance_date to YYYY-MM-DD string"
    - "Defensive sort.Strings after SQL ORDER BY for map iteration safety"
    - "hasSomething bool flags to avoid phantom zero entries for missing panel types"

key-files:
  created:
    - internal/api/handlers/history.go
    - internal/api/handlers/history_test.go
  modified:
    - internal/api/router.go

key-decisions:
  - "DATE() applied to balance_date in SQL to strip datetime suffix (returns YYYY-MM-DD, not YYYY-MM-DDT00:00:00Z)"
  - "hasChecking/hasCredit/hasSavings/hasInvestments bool flags prevent phantom zero entries for days where a panel has no data"
  - "Accumulator map (date -> dayAccumulator struct) chosen over per-panel queries for single-pass aggregation efficiency"

patterns-established:
  - "History handler uses fmt.Sprintf with validated integer for days parameter to prevent SQL injection"
  - "Empty series initialized as []balancePoint{} not nil to guarantee JSON [] not null"

requirements-completed: [DASH-01, DASH-02, DASH-03, DASH-04]

# Metrics
duration: 3min
completed: 2026-03-15
---

# Phase 3 Plan 02: Balance History Handler and Route Wiring Summary

**GET /api/balance-history time-series handler with per-panel daily aggregation (liquid/savings/investments), plus all three new routes wired into JWT-protected chi router group completing Phase 3 REST API contract.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-15T15:16:46Z
- **Completed:** 2026-03-15T15:19:00Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- GetBalanceHistory handler aggregates balance_snapshots per calendar day into three panels: liquid (checking+credit), savings, investments
- Optional ?days=N query parameter filters to last N days; invalid/negative/zero values return all data
- "Other" account type excluded from all panels; empty state returns empty arrays not null
- All three routes (/api/summary, /api/accounts, /api/balance-history) registered in JWT-protected group — 401 without valid JWT
- Full 10-test TDD suite covering aggregation, filtering, ordering, and edge cases

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): Failing tests for GetBalanceHistory** - `ff83416` (test)
2. **Task 1 (GREEN): GetBalanceHistory handler implementation** - `56691a6` (feat)
3. **Task 2: Wire all three routes into JWT-protected router group** - `bc358a2` (feat)

**Plan metadata:** (docs commit — see below)

_Note: TDD task has two commits (test → feat)_

## Files Created/Modified
- `internal/api/handlers/history.go` - GetBalanceHistory factory handler with per-day accumulator logic
- `internal/api/handlers/history_test.go` - 10-test TDD suite covering all behavior scenarios
- `internal/api/router.go` - Added three route registrations in the JWT-protected group

## Decisions Made
- DATE() normalization in SQL: SQLite returns balance_date as `YYYY-MM-DDT00:00:00Z` when joining with datetime columns; `DATE(bs.balance_date)` normalizes to `YYYY-MM-DD` string as expected by frontend charts
- bool flags for panel presence: `hasChecking`/`hasCredit`/etc. prevent appending a liquid/savings/investments point when a panel has zero snapshots on a given day — avoids phantom "0.00" entries
- Single-pass accumulator map: one SQL query scans all rows; Go accumulates by date into a struct, then builds output slices — more efficient than three separate per-panel queries

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] DATE() applied to balance_date to normalize datetime to date string**
- **Found during:** Task 1 (GetBalanceHistory implementation — GREEN phase)
- **Issue:** SQLite returned `2024-01-01T00:00:00Z` for balance_date instead of `2024-01-01`, causing two tests to fail (LiquidIsSumOfCheckingAndCredit and TimeSeriesOrdering)
- **Fix:** Changed `SELECT bs.balance_date` to `SELECT DATE(bs.balance_date)` in the SQL query
- **Files modified:** internal/api/handlers/history.go
- **Verification:** All 10 TestGetBalanceHistory tests pass
- **Committed in:** 56691a6 (Task 1 GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Essential for correct date format in API response. No scope creep.

## Issues Encountered
- SQLite datetime normalization: `balance_date DATE` column in schema stores values, but JOIN with datetime fields caused SQLite to return the full datetime representation. Resolved with `DATE()` function in the SELECT clause.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 3 REST API contract is complete: three endpoints (/api/summary, /api/accounts, /api/balance-history), all behind JWT auth
- Frontend (Phase 4) can now fetch all dashboard data from these endpoints
- All handlers follow identical factory pattern — easy to extend with additional routes
- Full test suite (10 history + 7 accounts + 7 summary tests) provides regression safety

---
*Phase: 03-backend-api*
*Completed: 2026-03-15*
