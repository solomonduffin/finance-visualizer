---
phase: 09-projection-engine
plan: 02
subsystem: api
tags: [go, chi, sqlite, projection, settings, typescript, api-client]

# Dependency graph
requires:
  - phase: 09-projection-engine
    plan: 01
    provides: "Migration 000005 with projection_account_settings, projection_holding_settings, holdings, projection_income_settings tables"
  - phase: 01-foundation
    provides: "chi router, JWT auth, handler patterns"
provides:
  - "GetProjectionSettings handler: accounts with holdings nested under investment types, income defaults"
  - "SaveProjectionSettings handler: transactional upsert for account and holding rates"
  - "SaveIncomeSettings handler: singleton upsert for income modeling configuration"
  - "Route registration: GET/PUT /api/projections/settings, PUT /api/projections/income"
  - "Frontend TypeScript types and fetch functions for all projection endpoints"
affects: [09-projection-engine]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "investmentTypes map for account type gating (brokerage, retirement, crypto, investment)"
    - "Nested holdings query per investment account with LEFT JOIN to projection_holding_settings"
    - "Income settings singleton pattern with ON CONFLICT upsert"

key-files:
  created:
    - "internal/api/handlers/projections.go"
    - "internal/api/handlers/projections_test.go"
  modified:
    - "internal/api/router.go"
    - "frontend/src/api/client.ts"

key-decisions:
  - "investmentTypes map includes brokerage, retirement, crypto, investment for holdings query gating"
  - "Holdings query uses COALESCE(h.symbol, '') to handle nullable symbol field"
  - "SaveProjectionSettings wraps all upserts in single transaction for atomicity"

patterns-established:
  - "Projection settings handler pattern: LEFT JOIN to settings table with COALESCE defaults"
  - "Nested resource query: per-account holdings loop after closing main rows cursor"

requirements-completed: [PROJ-01, PROJ-02, PROJ-03, PROJ-04, PROJ-06]

# Metrics
duration: 4min
completed: 2026-03-17
---

# Phase 9 Plan 2: Projection Settings API & Frontend Client Summary

**Three projection API handlers (GET settings with nested holdings, PUT account/holding rates, PUT income) plus typed TypeScript client with 5 interfaces and 3 fetch functions**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-17T02:59:46Z
- **Completed:** 2026-03-17T03:04:06Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Created GET /api/projections/settings handler returning accounts grouped by type with projection rates, holdings nested under investment accounts, and income settings with sensible defaults
- Created PUT /api/projections/settings handler with transactional upsert for account and holding rate changes
- Created PUT /api/projections/income handler with singleton upsert for income modeling configuration
- Extended frontend API client with 5 exported TypeScript interfaces and 3 exported async fetch functions matching backend response shapes

## Task Commits

Each task was committed atomically:

1. **Task 1: Projection settings API handlers (TDD)**
   - `4fe1ed6` (test) - Failing tests for projection settings handlers
   - `ac8607e` (feat) - Implement projection settings handlers and register routes
2. **Task 2: Frontend API client projection types and functions**
   - `52c5dcd` (feat) - Add frontend projection API types and functions

_TDD tasks have RED (test) and GREEN (feat) commits_

## Files Created/Modified
- `internal/api/handlers/projections.go` - Three handler functions: GetProjectionSettings, SaveProjectionSettings, SaveIncomeSettings
- `internal/api/handlers/projections_test.go` - 7 comprehensive handler tests covering defaults, holdings, income, save/read round-trips, hidden exclusion
- `internal/api/router.go` - Route registration for /api/projections/settings (GET, PUT) and /api/projections/income (PUT)
- `frontend/src/api/client.ts` - TypeScript interfaces and fetch functions for all projection endpoints

## Decisions Made
- Used `investmentTypes` map with brokerage, retirement, crypto, and investment as keys for determining which accounts query holdings -- covers all possible investment-like account types
- Holdings query uses `COALESCE(h.symbol, '')` since symbol is nullable in the holdings table schema
- SaveProjectionSettings wraps all account and holding upserts in a single database transaction for atomicity
- Frontend types match backend JSON shapes exactly for type-safe integration

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All three projection API endpoints registered and tested
- Frontend API client ready for consumption by Projections page components
- Settings persist across sessions via projection_account_settings, projection_holding_settings, and projection_income_settings tables
- 7 handler tests provide regression coverage

## Self-Check: PASSED

All 4 created/modified files verified present. All 3 task commits verified in git log.

---
*Phase: 09-projection-engine*
*Completed: 2026-03-17*
