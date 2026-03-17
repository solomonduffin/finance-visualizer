---
phase: 05-data-foundation
plan: 03
subsystem: ui
tags: [react, dnd-kit, tailwind, drag-and-drop, settings, toast]

# Dependency graph
requires:
  - phase: 05-data-foundation/05-02
    provides: "PATCH /api/accounts/:id endpoint, getAccounts API, getAccountDisplayName utility"
provides:
  - "AccountsSection component with grouped list, inline rename, hide/unhide, drag-and-drop type reassignment"
  - "Toast notification component for sync-restored accounts"
  - "Complete Settings page with account management integrated"
affects: [06-operational-quick-wins, 07-analytics-expansion]

# Tech tracking
tech-stack:
  added: ["@dnd-kit/react", "@dnd-kit/dom"]
  patterns: ["optimistic UI updates with error rollback", "per-account instant save (no batch)", "responsive drag-and-drop with mobile dropdown fallback"]

key-files:
  created:
    - "frontend/src/components/AccountsSection.tsx"
    - "frontend/src/components/AccountsSection.test.tsx"
    - "frontend/src/components/Toast.tsx"
    - "frontend/src/components/Toast.test.tsx"
  modified:
    - "frontend/src/pages/Settings.tsx"
    - "frontend/src/pages/Settings.test.tsx"
    - "frontend/src/api/client.ts"
    - "frontend/package.json"
    - "internal/api/handlers/accounts.go"
    - "internal/api/handlers/update_account.go"

key-decisions:
  - "Optimistic state update for drag-and-drop instead of re-fetch (prevents animation glitch)"
  - "include_hidden=true query parameter for Settings page account loading"
  - "Custom account groups (e.g., Coinbase aggregation) deferred to future phase"

patterns-established:
  - "Optimistic UI: update local state immediately, revert on API error"
  - "Toast notification: lightweight, no external library, auto-dismiss after 4s"
  - "Responsive drag-and-drop: @dnd-kit on desktop, dropdown select on mobile"

requirements-completed: [ACCT-01, ACCT-02]

# Metrics
duration: 17min
completed: 2026-03-15
---

# Phase 5 Plan 3: Account Management UI Summary

**Settings page Accounts section with inline rename, hide/unhide toggle, @dnd-kit drag-and-drop type reassignment (desktop), mobile dropdown fallback, and toast notifications for restored accounts**

## Performance

- **Duration:** 17 min
- **Started:** 2026-03-15T22:46:00Z
- **Completed:** 2026-03-15T23:03:00Z
- **Tasks:** 2 (1 auto + 1 checkpoint)
- **Files modified:** 13

## Accomplishments
- AccountsSection component with accounts grouped by panel type (Liquid, Savings, Investments, Other), inline rename with Enter/Escape/onBlur, hide/unhide toggle, and reset-to-original-name button
- Desktop drag-and-drop via @dnd-kit/react for cross-group type reassignment with optimistic state updates
- Mobile dropdown fallback for type reassignment on touch devices
- Toast component with auto-dismiss and slide-up animation for sync-restored account notifications
- Settings page integration with dark mode support across all elements
- 29 tests passing across 3 test files

## Task Commits

Each task was committed atomically:

1. **Task 1: Install dnd-kit, create AccountsSection and Toast components** - `2f2130d` (feat)
2. **Bug fix: Resolve balance, hidden accounts, and drag-drop bugs** - `a6d7c9b` (fix)

Task 2 was a human-verify checkpoint -- user approved after bug fixes were applied.

## Files Created/Modified
- `frontend/src/components/AccountsSection.tsx` - Account management section with grouped list, inline edit, drag-and-drop, hide/unhide (667 lines)
- `frontend/src/components/AccountsSection.test.tsx` - 295 lines of component tests
- `frontend/src/components/Toast.tsx` - Lightweight toast notification with auto-dismiss
- `frontend/src/components/Toast.test.tsx` - Toast rendering and timeout tests
- `frontend/src/pages/Settings.tsx` - Integrated AccountsSection, toast state for restored accounts, dark mode classes
- `frontend/src/pages/Settings.test.tsx` - Extended with AccountsSection and toast tests
- `frontend/src/api/client.ts` - Added include_hidden parameter support to getAccounts
- `frontend/package.json` - Added @dnd-kit/react and @dnd-kit/dom dependencies
- `internal/api/handlers/accounts.go` - Added include_hidden query parameter support
- `internal/api/handlers/accounts_test.go` - Tests for include_hidden filtering
- `internal/api/handlers/update_account.go` - Fixed balance return from balance_snapshots
- `internal/api/handlers/update_account_test.go` - Tests for correct balance in PATCH response

## Decisions Made
- Optimistic state update for drag-and-drop instead of re-fetching accounts (prevents fly-off animation glitch when DOM re-renders after API response)
- Added `?include_hidden=true` query parameter to GET /api/accounts so Settings page can load hidden accounts alongside visible ones
- Custom account groups (e.g., aggregating multiple Coinbase wallets) noted as user request but deferred to Phase 7

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] PATCH endpoint returning hardcoded "0" balance**
- **Found during:** Task 2 (visual verification)
- **Issue:** After renaming an account, the PATCH response returned balance "0" instead of the actual balance, causing the UI to show $0.00
- **Fix:** Updated update_account handler to query latest balance from balance_snapshots table
- **Files modified:** internal/api/handlers/update_account.go, internal/api/handlers/update_account_test.go
- **Verification:** User confirmed balance displays correctly after rename
- **Committed in:** a6d7c9b

**2. [Rule 1 - Bug] Hidden accounts disappearing permanently from Settings**
- **Found during:** Task 2 (visual verification)
- **Issue:** GET /api/accounts filtered out hidden accounts, so Settings page could not display them in the "Hidden Accounts" collapsible section
- **Fix:** Added `include_hidden=true` query parameter support to GET /api/accounts handler and frontend API client
- **Files modified:** internal/api/handlers/accounts.go, internal/api/handlers/accounts_test.go, frontend/src/api/client.ts
- **Verification:** User confirmed hidden accounts appear in collapsible section
- **Committed in:** a6d7c9b

**3. [Rule 1 - Bug] Drag-and-drop fly-off animation on drop**
- **Found during:** Task 2 (visual verification)
- **Issue:** After dropping an account in a new group, re-fetching accounts from API caused the dragged item to animate back to its original position before re-rendering in the new group
- **Fix:** Switched to optimistic local state update on drop (move account in local state immediately, call API in background)
- **Files modified:** frontend/src/components/AccountsSection.tsx
- **Verification:** User confirmed smooth drop behavior without animation glitch
- **Committed in:** a6d7c9b

---

**Total deviations:** 3 auto-fixed (3 Rule 1 bugs)
**Impact on plan:** All bugs discovered during visual verification checkpoint. Fixes were essential for correct user experience. No scope creep.

## Issues Encountered
None beyond the bugs found during visual verification (documented above as deviations).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 5 (Data Foundation) is now complete: schema migration, soft-delete sync, PATCH API, and Settings UI all delivered
- Phases 6, 7, 8, and 9 can proceed -- all depend only on Phase 5
- User requested custom account groups for Coinbase -- relevant to Phase 7 (Analytics Expansion) planning

## Self-Check: PASSED

All 9 key files verified present. Both commits (2f2130d, a6d7c9b) verified in git history.

---
*Phase: 05-data-foundation*
*Completed: 2026-03-15*
