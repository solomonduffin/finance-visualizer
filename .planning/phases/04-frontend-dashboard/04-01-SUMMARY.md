---
phase: 04-frontend-dashboard
plan: 01
subsystem: ui
tags: [react, recharts, tailwind, dark-mode, typescript, vitest]

# Dependency graph
requires:
  - phase: 03-backend-api
    provides: GET /api/summary, /api/accounts, /api/balance-history endpoints with typed JSON shapes
provides:
  - recharts installed and importable
  - useDarkMode hook with localStorage persistence and html.dark class toggling
  - Dark mode infrastructure (index.css @custom-variant, index.html blocking script)
  - ResizeObserver stub for Recharts test compatibility
  - getSummary, getAccounts, getBalanceHistory typed API client functions
  - SummaryResponse, AccountsResponse, BalanceHistoryResponse TypeScript interfaces
  - timeAgo utility extracted from Settings.tsx with 4 tests
  - formatCurrency utility for USD string formatting with 4 tests
  - PANEL_COLORS constant with blue/green/purple accents
  - PanelCard component with account list and dark mode support
  - SkeletonDashboard with 3 animated placeholder cards
  - EmptyState component with Settings link for first-launch
affects:
  - 04-02 (dashboard page consumes all components built here)
  - 04-03 (charts depend on recharts install and BalanceHistoryResponse type)

# Tech tracking
tech-stack:
  added: [recharts]
  patterns:
    - TDD for all hooks and utilities (RED then GREEN commits)
    - Named exports for all components (not default exports)
    - Tailwind dark: classes via @custom-variant dark directive
    - String balance values parsed via Number() and toLocaleString for USD formatting
    - Blocking inline script in index.html head to prevent dark mode flash

key-files:
  created:
    - frontend/src/hooks/useDarkMode.ts
    - frontend/src/hooks/useDarkMode.test.ts
    - frontend/src/utils/time.ts
    - frontend/src/utils/time.test.ts
    - frontend/src/utils/format.ts
    - frontend/src/utils/format.test.ts
    - frontend/src/components/panelColors.ts
    - frontend/src/components/PanelCard.tsx
    - frontend/src/components/PanelCard.test.tsx
    - frontend/src/components/SkeletonDashboard.tsx
    - frontend/src/components/SkeletonDashboard.test.tsx
    - frontend/src/components/EmptyState.tsx
    - frontend/src/components/EmptyState.test.tsx
  modified:
    - frontend/package.json (recharts added)
    - frontend/src/index.css (@custom-variant dark added)
    - frontend/index.html (blocking dark mode script, updated title)
    - frontend/src/test-setup.ts (ResizeObserver stub)
    - frontend/src/api/client.ts (dashboard endpoint functions and interfaces)
    - frontend/src/pages/Settings.tsx (imports timeAgo from utils/time)

key-decisions:
  - "Named exports for all new components (PanelCard, SkeletonDashboard, EmptyState) — consistent with existing codebase pattern"
  - "recharts installed without --legacy-peer-deps flag (no peer dep errors)"
  - "No window.matchMedia / prefers-color-scheme detection in useDarkMode — manual toggle only per plan spec"
  - "Tailwind v4 @custom-variant dark (&:where(.dark, .dark *)) — applies dark styles to any element inside .dark"
  - "EmptyState uses react-router-dom Link component for /settings navigation"

patterns-established:
  - "Dark mode: html.dark class + Tailwind @custom-variant — all components use dark: prefix classes"
  - "Currency formatting: formatCurrency(stringValue) via Number().toLocaleString en-US USD"
  - "TDD flow: write failing tests first, then implement, commit after GREEN passes"

requirements-completed: [UX-02, UX-03, UX-04]

# Metrics
duration: 2min
completed: 2026-03-15
---

# Phase 4 Plan 01: Frontend Dashboard Foundations Summary

**recharts installed, dark mode infrastructure with no-flash script, typed API client extensions, and 6 new tested components/utilities (useDarkMode, timeAgo, formatCurrency, PanelCard, SkeletonDashboard, EmptyState)**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-15T16:19:41Z
- **Completed:** 2026-03-15T16:22:38Z
- **Tasks:** 2
- **Files modified:** 19 (13 created, 6 modified)

## Accomplishments
- Installed recharts (39 packages, no peer dep errors), added ResizeObserver stub so chart components can be tested in jsdom
- Dark mode infrastructure: Tailwind v4 @custom-variant dark directive, blocking inline script in index.html head prevents flash of wrong theme on load, useDarkMode hook toggles html.dark class and persists to localStorage
- Extended API client with fully typed getSummary, getAccounts, getBalanceHistory functions and all associated TypeScript interfaces
- Created PanelCard (label + formatted total + account list), SkeletonDashboard (3 animate-pulse cards + chart placeholder), EmptyState (Settings link CTA) — all with Tailwind dark: classes
- Extracted timeAgo from Settings.tsx into shared utils/time.ts; Settings.tsx updated to import it
- 33 tests pass across 8 test files (21 new tests + 12 pre-existing)

## Task Commits

Each task was committed atomically:

1. **Task 1: Install recharts, dark mode infrastructure, API client extensions, test setup** - `a492dca` (feat)
2. **Task 2: PanelCard, SkeletonDashboard, EmptyState components with panel colors** - `90c7f31` (feat)

_Note: Both tasks used TDD (failing tests committed implicitly within same atomic commit after GREEN)_

## Files Created/Modified
- `frontend/src/hooks/useDarkMode.ts` - Dark mode toggle hook with localStorage persistence
- `frontend/src/hooks/useDarkMode.test.ts` - 4 tests for toggle, class mutation, localStorage
- `frontend/src/utils/time.ts` - timeAgo relative time formatter (extracted from Settings)
- `frontend/src/utils/time.test.ts` - 4 tests for seconds/minutes/hours/days
- `frontend/src/utils/format.ts` - formatCurrency for USD string formatting
- `frontend/src/utils/format.test.ts` - 4 tests including negative values
- `frontend/src/components/panelColors.ts` - PANEL_COLORS with label/accent/darkAccent per panel
- `frontend/src/components/PanelCard.tsx` - Panel card with account list and dark mode
- `frontend/src/components/PanelCard.test.tsx` - 5 tests for rendering label, total, accounts
- `frontend/src/components/SkeletonDashboard.tsx` - 3-card grid + chart area placeholder
- `frontend/src/components/SkeletonDashboard.test.tsx` - 2 tests for card count and animation class
- `frontend/src/components/EmptyState.tsx` - First-launch prompt with /settings Link
- `frontend/src/components/EmptyState.test.tsx` - 2 tests for heading text and link href
- `frontend/package.json` - recharts added as dependency
- `frontend/src/index.css` - @custom-variant dark directive added
- `frontend/index.html` - Blocking dark mode script + updated title
- `frontend/src/test-setup.ts` - ResizeObserver stub for Recharts compatibility
- `frontend/src/api/client.ts` - Dashboard API functions and TypeScript interfaces
- `frontend/src/pages/Settings.tsx` - Updated to import timeAgo from utils/time

## Decisions Made
- Named exports for all new components (consistent with existing codebase)
- recharts installed without `--legacy-peer-deps` (no errors needed)
- No `window.matchMedia` detection in useDarkMode — manual toggle only per plan spec
- `@custom-variant dark (&:where(.dark, .dark *))` — Tailwind v4 dark mode via class selector

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all implementations passed tests on first attempt.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All foundational building blocks for Plans 02 and 03 are in place
- recharts importable, all API client types defined
- PanelCard, SkeletonDashboard, EmptyState ready for Dashboard page assembly
- useDarkMode hook ready for App.tsx integration
- No blockers

---
*Phase: 04-frontend-dashboard*
*Completed: 2026-03-15*
