---
phase: 02-data-pipeline
plan: "03"
subsystem: ui
tags: [react, typescript, simplefin, routing, react-router-dom, tailwind, vitest]

# Dependency graph
requires:
  - phase: 02-data-pipeline-02
    provides: Settings and sync REST API endpoints (/api/settings, /api/sync/now)
  - phase: 01-foundation
    provides: Login page pattern, Tailwind setup, auth cookie infrastructure
provides:
  - Settings page React component with URL input, status badge, and Sync Now button
  - Extended API client with getSettings, saveSettings, triggerSync
  - Client-side routing via react-router-dom between Dashboard and Settings
  - Nav bar visible when authenticated
  - SimpleFIN setup token claiming (ClaimSetupToken, IsSetupToken)
  - /accounts path appended to access URL per SimpleFIN spec
affects: [03-transactions, 04-ui-polish]

# Tech tracking
tech-stack:
  added: [react-router-dom v7]
  patterns:
    - BrowserRouter wrapping authenticated routes in App.tsx
    - API client functions with credentials include for HttpOnly cookie auth
    - vi.mock for unit testing API client in Settings.test.tsx
    - timeAgo helper using pure Date arithmetic (no date library)
    - IsSetupToken auto-detection — saves users from manual token-to-URL conversion

key-files:
  created:
    - frontend/src/pages/Settings.tsx
    - frontend/src/pages/Settings.test.tsx
  modified:
    - frontend/src/api/client.ts
    - frontend/src/App.tsx
    - frontend/package.json
    - frontend/package-lock.json
    - internal/api/handlers/settings.go
    - internal/simplefin/client.go
    - internal/simplefin/client_test.go
    - internal/sync/sync.go
    - internal/api/router.go
    - internal/api/router_test.go

key-decisions:
  - "Use react-router-dom v7 for client-side routing — keeps navigation in React, avoids full-page reloads"
  - "timeAgo helper implemented inline without date-fns or moment — no dependency for simple relative time"
  - "ClaimSetupToken added to simplefin client to exchange base64 setup tokens for access URLs before storage"
  - "IsSetupToken auto-detection — detect token vs URL input so user does not need to know the difference"
  - "SyncOnce appends /accounts to access URL — per SimpleFIN spec, stored URL is base, fetch target is /accounts"
  - "context.Background() for background sync goroutines — HTTP request context cancels on response, would abort sync"

patterns-established:
  - "API client pattern: exported async functions with credentials include for all fetch calls"
  - "Settings page pattern: useEffect for initial data load, controlled input, inline loading state per action"
  - "Router pattern: BrowserRouter in AuthenticatedApp component, Login rendered outside router when not authenticated"

requirements-completed: [DATA-01, DATA-02]

# Metrics
duration: 30min
completed: 2026-03-15
---

# Phase 2 Plan 03: Settings Page and End-to-End Data Pipeline Summary

**React Settings page with SimpleFIN setup token support, react-router-dom routing, and full end-to-end sync verified via Docker (48 accounts fetched)**

## Performance

- **Duration:** ~30 min
- **Started:** 2026-03-15T04:34:00Z
- **Completed:** 2026-03-15T04:57:42Z
- **Tasks:** 3 (including 1 human-verify checkpoint)
- **Files modified:** 10

## Accomplishments
- Settings page with access URL input, save-triggers-sync flow, status badge (Configured / Not configured), and Sync Now button
- Extended API client with getSettings, saveSettings, triggerSync — all using credentials:include for cookie auth
- react-router-dom v7 navigation between Dashboard (/) and Settings (/settings) with sticky nav bar
- SimpleFIN setup token claiming added to Go backend — users can paste either a base64 setup token or a full access URL
- Fixed SyncOnce to append /accounts to stored access URL per SimpleFIN spec
- End-to-end Docker verification passed: 48 accounts fetched, sync status updated, scheduler running

## Task Commits

Each task was committed atomically:

1. **Task 1 (TDD RED): Add failing Settings tests** - `02f9e18` (test)
2. **Task 1 (TDD GREEN): Settings page and API client extensions** - `121c222` (feat)
3. **Task 2: App routing with nav bar** - `dded984` (feat)
4. **Post-checkpoint: SimpleFIN setup token support and /accounts path fix** - `3718e0d` (feat)

## Files Created/Modified
- `frontend/src/pages/Settings.tsx` - Settings page component: URL input, status badge, Sync Now, navigation
- `frontend/src/pages/Settings.test.tsx` - 7 unit tests covering render, configured state, save, sync interactions
- `frontend/src/api/client.ts` - Added getSettings, saveSettings, triggerSync, SettingsResponse interface
- `frontend/src/App.tsx` - BrowserRouter routing, NavBar, AuthenticatedApp wrapper
- `frontend/package.json` - Added react-router-dom dependency
- `internal/api/handlers/settings.go` - SaveSettings handler calls ClaimSetupToken before storing
- `internal/simplefin/client.go` - Added ClaimSetupToken() and IsSetupToken() functions
- `internal/simplefin/client_test.go` - Tests for setup token claiming and detection
- `internal/sync/sync.go` - Appends /accounts to access URL; adds sync start/complete logging
- `internal/api/router.go` - Router test coverage additions

## Decisions Made
- Used react-router-dom v7 for client-side routing — keeps navigation in React, avoids full-page reloads
- timeAgo helper implemented inline without date-fns or moment — lightweight, no new dependency
- ClaimSetupToken exchanges base64 setup tokens for access URLs before storage so the stored value is always a valid HTTPS access URL
- IsSetupToken auto-detects whether input is a setup token (starts with "https://" prefix absent) so users don't need to distinguish
- SyncOnce appends /accounts to stored access URL — per SimpleFIN spec, the stored URL is the base, actual accounts endpoint is /accounts

## Deviations from Plan

### Post-Checkpoint Additions (orchestrator commits after human-verify)

**1. [Rule 1 - Bug] Fixed /accounts path missing from SyncOnce**
- **Found during:** End-to-end Docker verification (after Task 3 checkpoint)
- **Issue:** SimpleFIN spec requires appending /accounts to the access URL; SyncOnce was using the bare access URL, causing 0 accounts to be fetched
- **Fix:** SyncOnce now appends /accounts to the stored access URL before fetching
- **Files modified:** internal/sync/sync.go
- **Verification:** Docker sync showed 48 accounts fetched after fix
- **Committed in:** 3718e0d

**2. [Rule 2 - Missing Critical] Added SimpleFIN setup token claiming**
- **Found during:** End-to-end Docker verification (after Task 3 checkpoint)
- **Issue:** SimpleFIN provides a one-time base64 setup token to exchange for an access URL; without ClaimSetupToken, users had to manually exchange tokens before pasting into Settings
- **Fix:** Added ClaimSetupToken() to simplefin client, IsSetupToken() detector, SaveSettings handler auto-claims tokens before storing
- **Files modified:** internal/simplefin/client.go, internal/simplefin/client_test.go, internal/api/handlers/settings.go, frontend/src/pages/Settings.tsx
- **Verification:** Settings page accepts setup token, exchanges it, and stores access URL; Docker verified 48 accounts fetched
- **Committed in:** 3718e0d

---

**Total deviations:** 2 (1 bug fix, 1 missing critical functionality)
**Impact on plan:** Both fixes essential for the feature to work in production. /accounts path fix was a spec compliance bug; setup token claiming is required for the SimpleFIN onboarding flow.

## Issues Encountered
- SimpleFIN /accounts path omission caused 0 accounts fetched during initial Docker test — identified and fixed post-checkpoint
- Setup token claiming not in original plan scope but required for real-world usage — added as Rule 2 deviation

## User Setup Required
None — SimpleFIN configuration is done through the Settings UI. No environment variables needed beyond those established in Phase 2.

## Next Phase Readiness
- Settings page and sync pipeline complete; accounts are fetched and stored in SQLite
- Phase 3 (Transactions) can read from accounts table to display balances on the Dashboard
- Nav bar stub is in place for Phase 4 to style and extend
- No blockers for Phase 3

---
*Phase: 02-data-pipeline*
*Completed: 2026-03-15*
