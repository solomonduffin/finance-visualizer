---
phase: 09-projection-engine
plan: 03
subsystem: ui
tags: [recharts, projection, compound-interest, typescript, vitest]

# Dependency graph
requires:
  - phase: 04-frontend-dashboard
    provides: Recharts patterns (StackedAreaChart, BalanceLineChart), formatCurrency, TimeRangeSelector
provides:
  - projectBalance function (compound/simple interest with contributions)
  - calculateProjection function (portfolio-level aggregation with income allocation)
  - ProjectionChart component (solid historical + dashed projected ComposedChart)
  - HorizonSelector component (1y/5y/10y/20y/Custom radio group)
  - TypeScript interfaces (AccountProjection, HoldingProjection, IncomeSettings, ProjectionPoint)
affects: [09-04-PLAN, 09-05-PLAN]

# Tech tracking
tech-stack:
  added: []
  patterns: [TDD for math utilities, ComposedChart with dual-line solid/dashed transition, allocation-based income distribution]

key-files:
  created:
    - frontend/src/utils/projection.ts
    - frontend/src/utils/projection.test.ts
    - frontend/src/components/ProjectionChart.tsx
    - frontend/src/components/ProjectionChart.test.tsx
    - frontend/src/components/HorizonSelector.tsx
    - frontend/src/components/HorizonSelector.test.tsx
  modified: []

key-decisions:
  - "Compound interest test expectation corrected from plan's ~$16386 to actual $16651.05 (proper month-by-month compound with contributions)"
  - "Allocation validation: income contribution suppressed when sum != 100% (growth-only fallback)"
  - "hasHoldings flag prevents double-counting for investment accounts with per-holding projections"

patterns-established:
  - "TDD for pure math utilities: write tests first, verify RED, implement, verify GREEN"
  - "ComposedChart with bridge point connecting solid historical Line to dashed projected Line"
  - "Radiogroup accessibility pattern reused from TimeRangeSelector for HorizonSelector"

requirements-completed: [PROJ-05]

# Metrics
duration: 4min
completed: 2026-03-17
---

# Phase 9 Plan 03: Projection Engine & Chart Components Summary

**Pure projection math engine with compound/simple interest, per-holding aggregation, and income allocation; Recharts ComposedChart with solid-to-dashed historical/projected transition and accessible horizon selector**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-17T02:50:08Z
- **Completed:** 2026-03-17T02:54:46Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Projection engine with projectBalance (compound/simple + contributions) and calculateProjection (portfolio-level with income allocation, double-counting prevention)
- ProjectionChart with ComposedChart rendering solid historical line, dashed projected line, gradient fill, "Now" ReferenceLine marker, custom tooltip, and empty state
- HorizonSelector with 1y/5y/10y/20y presets and custom year input (debounced), full radiogroup accessibility
- 24 tests total across 3 test files, all passing

## Task Commits

Each task was committed atomically:

1. **Task 1: Client-side projection calculation engine** - `9b61085` (test: TDD RED), `cf8ab18` (feat: TDD GREEN)
2. **Task 2: ProjectionChart and HorizonSelector components** - `665c160` (feat)

## Files Created/Modified
- `frontend/src/utils/projection.ts` - Pure projection math: projectBalance, calculateProjection, type interfaces
- `frontend/src/utils/projection.test.ts` - 12 unit tests covering compound/simple interest, income, holdings, allocation validation
- `frontend/src/components/ProjectionChart.tsx` - Recharts ComposedChart with solid/dashed lines, gradient fill, Now marker, tooltip
- `frontend/src/components/ProjectionChart.test.tsx` - 4 component tests: role=img, empty state, chart elements, connectNulls
- `frontend/src/components/HorizonSelector.tsx` - Segmented control with presets and custom year input, debounced onChange
- `frontend/src/components/HorizonSelector.test.tsx` - 8 component tests: buttons, accessibility, interactions, custom input

## Decisions Made
- Corrected compound-with-contributions test expectation from plan's approximate ~$16,386 to actual $16,651.05 -- the plan's estimate didn't account for contributions also compounding
- Allocation validation suppresses income when sum != 100% (matches UI "last valid state" behavior)
- hasHoldings flag on AccountProjection prevents double-counting: accounts with holdings are skipped in the account loop, their value comes from the holdings loop
- Bridge point pattern: last historical point sets both historical and projected values for seamless line transition

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected compound interest test expectation**
- **Found during:** Task 1 (TDD GREEN phase)
- **Issue:** Plan specified ~$16,386 for $10,000 at 5% APY with $500/month contributions over 12 months, but actual compound calculation yields $16,651.05
- **Fix:** Updated test expectation to match mathematically correct result
- **Files modified:** frontend/src/utils/projection.test.ts
- **Verification:** All 12 math tests pass with toBeCloseTo precision 2
- **Committed in:** cf8ab18

---

**Total deviations:** 1 auto-fixed (1 bug fix)
**Impact on plan:** Test expectation corrected to match correct math. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Projection engine ready for consumption by RateConfigTable (Plan 04) and Projections page (Plan 05)
- ProjectionChart ready to receive projection data from calculateProjection
- HorizonSelector ready to drive horizonYears parameter
- Type interfaces exported for use across the projection feature

## Self-Check: PASSED

All 6 created files verified on disk. All 3 task commits (9b61085, cf8ab18, 665c160) verified in git log.

---
*Phase: 09-projection-engine*
*Completed: 2026-03-17*
