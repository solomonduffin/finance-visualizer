---
phase: 02-data-pipeline
plan: 01
subsystem: api
tags: [simplefin, sqlite, go, sync, decimal, cron]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: db.Open, db.Migrate, SQLite schema with accounts/balance_snapshots/sync_log tables
provides:
  - SimpleFIN HTTP client (internal/simplefin) with FetchAccounts, Account, AccountSet types
  - Sync orchestration (internal/sync) with SyncOnce, RunScheduler, InferAccountType, NextRunTime
affects: [02-02-api-handlers, 02-03-frontend-dashboard]

# Tech tracking
tech-stack:
  added: [github.com/shopspring/decimal v1.4.0 (promoted to direct dependency)]
  patterns:
    - TDD with httptest.NewServer for HTTP client testing
    - INSERT OR IGNORE for idempotent balance snapshots
    - ON CONFLICT DO UPDATE for account upserts
    - t.TempDir + db.Migrate for isolated SQLite test databases
    - Package-level sync.Mutex for concurrent call serialization

key-files:
  created:
    - internal/simplefin/client.go
    - internal/simplefin/client_test.go
    - internal/sync/sync.go
    - internal/sync/sync_test.go
  modified:
    - go.mod

key-decisions:
  - "InferAccountType exported (capital I) to allow direct testing from sync_test package"
  - "NextRunTime exported to allow direct testing without running scheduler goroutine"
  - "shopspring/decimal used for balance validation only — balance stored as TEXT string in DB per schema"
  - "syncMu is package-level not struct-level — only one global sync instance needed"

patterns-established:
  - "Pattern 1: Idempotent snapshot writes via INSERT OR IGNORE on (account_id, balance_date) UNIQUE constraint"
  - "Pattern 2: First-sync detection via SELECT COUNT(*) FROM accounts — 0 rows triggers 30-day lookback"
  - "Pattern 3: Per-account error isolation: parse error increments failed counter and continues loop"

requirements-completed: [DATA-01, DATA-03, DATA-04]

# Metrics
duration: 3min
completed: 2026-03-15
---

# Phase 2 Plan 01: Data Pipeline Core Summary

**SimpleFIN HTTP client with Basic Auth extraction + sync orchestration writing idempotent account/balance snapshots to SQLite with per-account error isolation and daily cron scheduler**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-15T04:27:31Z
- **Completed:** 2026-03-15T04:30:04Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- SimpleFIN HTTP client (`internal/simplefin`) parses account+balance data, extracts Basic Auth from embedded URL credentials, always sends balances-only=1, optionally sends start-date as Unix epoch
- Sync orchestration (`internal/sync`) reads access URL from settings, upserts accounts with inferred account types, inserts idempotent balance snapshots, logs sync results to sync_log with per-account error isolation
- shopspring/decimal promoted to direct dependency for balance string validation
- RunScheduler computes next wall-clock occurrence of configured hour for daily scheduling
- 19 tests pass across both packages covering all specified behaviors

## Task Commits

Each task was committed atomically:

1. **Task 1: SimpleFIN HTTP client** - `690a1a9` (test), `feat(02-01)` already committed pre-plan
2. **Task 2: Sync orchestration — tests** - `6e6b402` (test — TDD RED)
3. **Task 2: Sync orchestration — implementation** - `0af2f2d` (feat — TDD GREEN)

_Note: Task 1 was pre-committed (690a1a9 feat, 2869d92 test) before this execution run; tests passed immediately on verification._

## Files Created/Modified

- `/home/solomon/finance-visualizer/internal/simplefin/client.go` - SimpleFIN HTTP client with FetchAccounts, Account, AccountSet, Org types
- `/home/solomon/finance-visualizer/internal/simplefin/client_test.go` - 8 tests covering success, auth, query params, error cases
- `/home/solomon/finance-visualizer/internal/sync/sync.go` - SyncOnce, RunScheduler, InferAccountType, NextRunTime, processAccount
- `/home/solomon/finance-visualizer/internal/sync/sync_test.go` - 11 tests covering all behaviors from plan spec
- `/home/solomon/finance-visualizer/go.mod` - shopspring/decimal promoted to direct dependency

## Decisions Made

- `InferAccountType` and `NextRunTime` are exported (uppercase) to support direct white-box testing from the `sync_test` package without requiring a full sync run
- Balance is validated via `decimal.NewFromString` but stored as the original TEXT string to match the DB schema — shopspring/decimal is not used for arithmetic, only validation
- `syncMu` is package-level (not on a struct) since there is only one global sync instance in this application
- `go mod tidy` was skipped due to `data/` directory permission issue; go.mod was updated manually to move shopspring/decimal to the direct `require` block

## Deviations from Plan

None — plan executed exactly as written. The pre-existing SimpleFIN client and test files matched the plan spec completely and all tests passed on first run.

## Issues Encountered

- `go mod tidy` failed with `open data: permission denied` on the data/ directory. Resolved by manually editing go.mod to move `github.com/shopspring/decimal` from the indirect require block to the direct require block. Build and all tests verified correct after manual edit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/simplefin.FetchAccounts` is ready for use by API handlers or CLI tools
- `internal/sync.SyncOnce` and `RunScheduler` are ready to be called from `main.go` or an API handler
- The sync_log table records each sync run with fetched/failed counts for dashboard display
- No blockers — proceed to Phase 2 Plan 02 (API handlers)

---
*Phase: 02-data-pipeline*
*Completed: 2026-03-15*

## Self-Check: PASSED

All files and commits verified:
- internal/simplefin/client.go: FOUND
- internal/simplefin/client_test.go: FOUND
- internal/sync/sync.go: FOUND
- internal/sync/sync_test.go: FOUND
- .planning/phases/02-data-pipeline/02-01-SUMMARY.md: FOUND
- Commit 690a1a9 (SimpleFIN client feat): FOUND
- Commit 6e6b402 (sync tests RED): FOUND
- Commit 0af2f2d (sync implementation GREEN): FOUND
