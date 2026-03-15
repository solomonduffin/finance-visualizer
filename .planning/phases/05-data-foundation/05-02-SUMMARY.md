---
phase: 05-data-foundation
plan: 02
subsystem: api, frontend
tags: [chi, patch, nullable-json, display-name, tdd, vitest]

# Dependency graph
requires:
  - phase: 05-data-foundation
    provides: display_name, hidden_at, account_type_override columns and COALESCE queries (Plan 01)
provides:
  - PATCH /api/accounts/{id} endpoint for renaming, hiding/unhiding, type reassignment
  - NullableString JSON unmarshaling pattern for null vs absent distinction
  - Frontend AccountItem interface extended with display_name, original_name, hidden_at, account_type_override
  - updateAccount API function for PATCH /api/accounts/:id
  - getAccountDisplayName utility for consistent display name rendering
  - PanelCard uses getAccountDisplayName for display_name-aware rendering
  - SyncResponse type with optional restored account names
affects: [05-03, 06-growth-indicators, frontend-settings-ui]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "NullableString custom JSON unmarshal for distinguishing null, absent, and string in PATCH payloads"
    - "Dynamic UPDATE query building with only-modified-columns pattern"
    - "getAccountDisplayName: display_name > org+name > name fallback chain"

key-files:
  created:
    - internal/api/handlers/update_account.go
    - internal/api/handlers/update_account_test.go
    - frontend/src/utils/account.ts
    - frontend/src/utils/account.test.ts
  modified:
    - internal/api/router.go
    - frontend/src/api/client.ts
    - frontend/src/components/PanelCard.tsx
    - frontend/src/components/PanelCard.test.tsx

key-decisions:
  - "NullableString struct with Set+Value fields for JSON null/absent/string distinction instead of **string or json.RawMessage"
  - "Handler validates account_type_override against allowed types server-side before DB query (defense in depth, not relying solely on CHECK constraint)"
  - "PATCH response returns balance as '0' since balance is not the concern of this endpoint"
  - "PATCH added to CORS AllowedMethods for frontend compatibility"

patterns-established:
  - "NullableString pattern: custom UnmarshalJSON with Set bool + Value *string for partial-update PATCH handlers"
  - "Dynamic UPDATE pattern: build SET clauses array and args slice, only include provided fields"
  - "getAccountDisplayName: single source of truth for account display name rendering across all components"

requirements-completed: [ACCT-01, ACCT-02]

# Metrics
duration: 7min
completed: 2026-03-15
---

# Phase 5 Plan 2: Account Management API Summary

**PATCH /api/accounts/{id} endpoint with NullableString JSON pattern, frontend updateAccount function, and getAccountDisplayName utility for consistent display name rendering**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-15T22:28:29Z
- **Completed:** 2026-03-15T22:35:30Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- PATCH /api/accounts/{id} handler supports display_name set/clear, hide/unhide toggle, and account_type_override with validation
- NullableString custom JSON unmarshaling cleanly distinguishes null (clear), absent (skip), and string (set) in PATCH payloads
- Frontend AccountItem interface matches backend response shape with all new metadata fields
- getAccountDisplayName utility provides single source of truth for display name logic across all components
- PanelCard now renders display_name when present, falls back to "OrgName - Name" format
- 9 Go tests + 10 frontend tests, all passing with zero regressions (75 frontend tests total)

## Task Commits

Each task was committed atomically (TDD: test then feat):

1. **Task 1: PATCH /api/accounts/{id} endpoint**
   - `0ddd859` (test: failing tests for PATCH endpoint)
   - `2f48014` (feat: implement UpdateAccount handler with route wiring)
2. **Task 2: Frontend API client extension and PanelCard display name rendering**
   - `67a694a` (test: failing tests for getAccountDisplayName)
   - `3e14c28` (feat: implement frontend API client extension and PanelCard rendering)

_Note: TDD tasks have two commits each (RED test then GREEN implementation)_

## Files Created/Modified
- `internal/api/handlers/update_account.go` - PATCH handler with NullableString, dynamic UPDATE, validation
- `internal/api/handlers/update_account_test.go` - 9 test cases covering all PATCH operations
- `internal/api/router.go` - PATCH route wired into protected group, PATCH added to CORS methods
- `frontend/src/api/client.ts` - Extended AccountItem, UpdateAccountRequest, updateAccount function, SyncResponse type
- `frontend/src/utils/account.ts` - getAccountDisplayName utility with 3-branch fallback logic
- `frontend/src/utils/account.test.ts` - 3 tests covering all display name branches
- `frontend/src/components/PanelCard.tsx` - Uses getAccountDisplayName instead of inline ternary
- `frontend/src/components/PanelCard.test.tsx` - 2 new tests for display_name rendering (7 total)

## Decisions Made
- Used NullableString struct with custom UnmarshalJSON over alternatives (**string, json.RawMessage, interface{}) for clearest intent and type safety
- Server-side validation of account_type_override values before DB query provides defense in depth beyond the SQLite CHECK constraint
- PATCH response returns balance as "0" since the endpoint's concern is metadata, not financial data
- Added PATCH to CORS AllowedMethods to support frontend cross-origin requests in dev

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added PATCH to CORS AllowedMethods**
- **Found during:** Task 1 (route wiring)
- **Issue:** CORS middleware only allowed GET, POST, PUT, DELETE, OPTIONS -- PATCH requests from frontend would be blocked
- **Fix:** Added "PATCH" to AllowedMethods in cors.Options
- **Files modified:** internal/api/router.go
- **Verification:** Route registration compiles and handler tests pass
- **Committed in:** 2f48014

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** CORS fix necessary for frontend PATCH requests to work. No scope creep.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- PATCH endpoint ready for Settings UI (Plan 03) to consume for account management
- Frontend types and API function ready for account rename/hide/type-change UI
- getAccountDisplayName utility available for any component rendering account names
- All patterns (NullableString, dynamic UPDATE) documented for future PATCH handlers

## Self-Check: PASSED

All 8 key files verified present. All 4 task commits verified in git log.

---
*Phase: 05-data-foundation*
*Completed: 2026-03-15*
