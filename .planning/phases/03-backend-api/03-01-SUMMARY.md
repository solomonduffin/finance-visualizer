---
phase: 03-backend-api
plan: 01
subsystem: api
tags: [go, sqlite, shopspring-decimal, chi, sql, handlers, tdd]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: db.Open, db.Migrate, handler factory pattern, auth_test.go test patterns
  - phase: 02-data-pipeline
    provides: accounts table, balance_snapshots table, sync_log table schema

provides:
  - GetSummary handler (GET /api/summary) with liquid/savings/investments panel totals and last_synced_at
  - GetAccounts handler (GET /api/accounts) with accounts grouped by type and latest balance

affects: [03-backend-api-02]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Handler factory pattern: func GetXxx(database *sql.DB) http.HandlerFunc
    - Correlated subquery for latest balance per account (MAX balance_date)
    - setupFinanceTestDB + seedAccounts + seedSnapshots test helpers shared across handler tests
    - shopspring/decimal for financial arithmetic, StringFixed(2) for consistent string output
    - sql.NullString for nullable DB fields (balance, org_name, finished_at)

key-files:
  created:
    - internal/api/handlers/summary.go
    - internal/api/handlers/summary_test.go
    - internal/api/handlers/accounts.go
    - internal/api/handlers/accounts_test.go
  modified: []

key-decisions:
  - "investment account type maps to 'investments' JSON key (plural) in GetAccounts response"
  - "GetAccounts empty groups pre-initialized as []accountItem{} (not nil) to ensure JSON '[]' not 'null'"
  - "GetSummary uses StringFixed(2) for all decimal outputs — zero is '0.00' not '0'"
  - "GetAccounts balance defaults to '0' (not '0.00') for accounts with no snapshots — raw string default"

patterns-established:
  - "setupFinanceTestDB: shared test helper in handlers_test package, no password param unlike setupTestDB"
  - "seedAccounts + seedSnapshots: slice of map[string]string helpers for readable test data seeding"
  - "TDD cycle: failing test commit (test:) -> implementation commit (feat:) for each handler"

requirements-completed: [DASH-01, DASH-02, DASH-03, DASH-04]

# Metrics
duration: 4min
completed: 2026-03-15
---

# Phase 3 Plan 01: Summary and Accounts API Handlers Summary

**Two SQLite-backed JSON API handlers: GetSummary aggregates liquid/savings/investments panel totals via shopspring/decimal, GetAccounts returns all accounts grouped by type with latest snapshot balances**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-15T15:10:59Z
- **Completed:** 2026-03-15T15:14:27Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- GetSummary handler computes liquid (checking + credit), savings, investments panel totals from SQLite with correct decimal precision; excludes "other" accounts; exposes last_synced_at from sync_log
- GetAccounts handler returns all accounts grouped by type with latest snapshot balance via LEFT JOIN correlated subquery; pre-initialized slices ensure empty arrays never serialize as null
- 14 new tests (7 per handler) covering all edge cases: empty DB, no snapshots, other exclusion, null sync, JSON string validation, ordering, latest-balance selection

## Task Commits

Each task was committed atomically using TDD (test → feat):

1. **Task 1 RED: GetSummary failing tests** - `2c7c6fb` (test)
2. **Task 1 GREEN: GetSummary implementation** - `bdb74d8` (feat)
3. **Task 2 RED: GetAccounts failing tests** - `d069bd0` (test)
4. **Task 2 GREEN: GetAccounts implementation** - `298a597` (feat)

_TDD tasks have two commits each (test → feat)_

## Files Created/Modified
- `internal/api/handlers/summary.go` - GetSummary handler: panel totals with decimal arithmetic and sync metadata
- `internal/api/handlers/summary_test.go` - 7 tests covering all summary edge cases including JSON string verification
- `internal/api/handlers/accounts.go` - GetAccounts handler: accounts grouped by type with latest balance
- `internal/api/handlers/accounts_test.go` - 7 tests covering grouping, field validation, balance selection, null handling

## Decisions Made
- `investment` account_type maps to `investments` JSON key (plural) for consistent UX language
- Empty account groups pre-initialized as `[]accountItem{}` (not nil) so they serialize to `[]` not `null` in JSON
- `GetSummary` uses `decimal.StringFixed(2)` for all values — zero renders as `"0.00"` not `"0"`
- `GetAccounts` balance for accounts with no snapshots defaults to `"0"` (bare string, not `"0.00"`) — raw string default to avoid decimal import in accounts.go

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- GetSummary and GetAccounts handlers ready for route registration in Plan 02
- Handlers follow the factory pattern established in settings.go; Plan 02 wires them into the chi router
- All 26 handler tests (including pre-existing) pass; no regressions

## Self-Check: PASSED

All created files confirmed present:
- internal/api/handlers/summary.go: FOUND
- internal/api/handlers/summary_test.go: FOUND
- internal/api/handlers/accounts.go: FOUND
- internal/api/handlers/accounts_test.go: FOUND
- .planning/phases/03-backend-api/03-01-SUMMARY.md: FOUND

All task commits confirmed in git log:
- 2c7c6fb (test: GetSummary failing tests): FOUND
- bdb74d8 (feat: GetSummary implementation): FOUND
- d069bd0 (test: GetAccounts failing tests): FOUND
- 298a597 (feat: GetAccounts implementation): FOUND

---
*Phase: 03-backend-api*
*Completed: 2026-03-15*
