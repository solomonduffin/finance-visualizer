---
phase: 09-projection-engine
plan: 01
subsystem: database, api
tags: [sqlite, migrations, simplefin, holdings, sync, projection]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: "accounts table schema, SimpleFIN client, sync process"
  - phase: 02-data-pipeline
    provides: "SimpleFIN data flow and balance snapshot persistence"
provides:
  - "Migration 000005 with projection_account_settings, projection_holding_settings, holdings, projection_income_settings tables"
  - "Holding struct and FetchAccountsWithHoldings function in SimpleFIN client"
  - "Holdings persistence in sync process for investment-type accounts"
  - "fetchAccountData shared helper eliminating FetchAccounts code duplication"
affects: [09-projection-engine]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "fetchAccountData helper pattern for balances-only vs full-holdings fetch variants"
    - "Transactional DELETE+INSERT for holdings refresh (atomicity)"
    - "InferAccountType gating for investment-only holdings persistence"

key-files:
  created:
    - "internal/db/migrations/000005_projection_settings.up.sql"
    - "internal/db/migrations/000005_projection_settings.down.sql"
  modified:
    - "internal/simplefin/client.go"
    - "internal/simplefin/client_test.go"
    - "internal/sync/sync.go"
    - "internal/sync/sync_test.go"

key-decisions:
  - "fetchAccountData private helper extracts shared logic between FetchAccounts and FetchAccountsWithHoldings"
  - "Holdings persistence uses DELETE+INSERT in transaction rather than individual UPSERTs for simplicity"
  - "persistHoldings only called for investment-type accounts (InferAccountType == investment) with non-empty Holdings"

patterns-established:
  - "fetchAccountData(balancesOnly bool): shared SimpleFIN fetch logic with parameterized query"
  - "persistHoldings transactional refresh: DELETE all then INSERT new within single tx"

requirements-completed: [PROJ-06, PROJ-08]

# Metrics
duration: 6min
completed: 2026-03-17
---

# Phase 9 Plan 1: Projection Data Layer Summary

**Migration 000005 with four projection tables, SimpleFIN holdings fetch, and sync-time holdings persistence for investment accounts**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-17T02:49:40Z
- **Completed:** 2026-03-17T02:55:48Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Created migration 000005 with four new tables: projection_account_settings, projection_holding_settings, holdings, and projection_income_settings
- Extended SimpleFIN client with Holding struct, Holdings field on Account, and FetchAccountsWithHoldings function
- Integrated holdings persistence into sync process with transactional refresh for investment accounts
- Achieved full TDD coverage with 12 new tests across simplefin and sync packages

## Task Commits

Each task was committed atomically:

1. **Task 1: Database migration and SimpleFIN holdings extension**
   - `0a003eb` (test) - Failing tests for holdings struct and FetchAccountsWithHoldings
   - `be56bff` (feat) - Holdings struct, FetchAccountsWithHoldings, migration 000005
2. **Task 2: Sync process holdings persistence**
   - `e779526` (test) - Failing tests for holdings persistence in sync
   - `54cff4a` (feat) - Holdings persistence in sync process

_TDD tasks have RED (test) and GREEN (feat) commits_

## Files Created/Modified
- `internal/db/migrations/000005_projection_settings.up.sql` - Four new projection/holdings tables
- `internal/db/migrations/000005_projection_settings.down.sql` - Rollback migration
- `internal/simplefin/client.go` - Holding struct, Holdings field, fetchAccountData helper, FetchAccountsWithHoldings
- `internal/simplefin/client_test.go` - 7 new tests for holdings and FetchAccountsWithHoldings
- `internal/sync/sync.go` - persistHoldings function, SyncOnce uses FetchAccountsWithHoldings
- `internal/sync/sync_test.go` - 5 new tests for holdings persistence

## Decisions Made
- Extracted `fetchAccountData` private helper to share logic between `FetchAccounts` and `FetchAccountsWithHoldings`, differing only by `balancesOnly` parameter
- Holdings persistence uses DELETE+INSERT in a transaction rather than individual UPSERTs -- simpler logic for complete refresh semantics
- `persistHoldings` only called when `InferAccountType(acct.Name) == "investment"` AND `len(acct.Holdings) > 0` -- avoids unnecessary work for checking/savings/credit accounts

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Holdings table ready for projection engine calculations
- Projection settings tables ready for per-account and per-holding rate configuration
- Income settings table ready for annual income and savings allocation
- All 8 backend packages pass their test suites

## Self-Check: PASSED

All 6 created/modified files verified present. All 4 task commits verified in git log.

---
*Phase: 09-projection-engine*
*Completed: 2026-03-17*
