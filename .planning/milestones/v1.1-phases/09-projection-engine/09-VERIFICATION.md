---
phase: 09-projection-engine
verified: 2026-03-17T00:00:00Z
status: human_needed
score: 13/13 must-haves verified
human_verification:
  - test: "Navigate to /projections, set a non-zero APY on any account, observe chart"
    expected: "Projection line updates within ~500ms (debounce), chart shows dashed projected line to the right of Now marker"
    why_human: "Live reactivity and visual chart rendering cannot be verified programmatically"
  - test: "Set APY, reload the page"
    expected: "APY value persists — settings survive a full page reload (PROJ-06 end-to-end)"
    why_human: "Round-trip persistence requires a running database and server"
  - test: "Toggle compound/simple on an account"
    expected: "Chart recalculates immediately, projected values change (compound higher than simple for same APY)"
    why_human: "Visual chart difference between interest modes requires live environment"
  - test: "Open Income Modeling, enable it, enter $100,000 income, 20% savings, allocate 100% to one account"
    expected: "Chart projection line rises noticeably compared to growth-only; Total shows green '100%'"
    why_human: "Income contribution visual effect and allocation validation feedback require live environment"
  - test: "Click HorizonSelector presets: 1y, 5y, 10y, 20y, then Custom (enter 15)"
    expected: "Chart x-axis adjusts to show the selected time range for each preset; custom year input appears on Custom click"
    why_human: "Chart axis range and UI transitions require browser rendering"
  - test: "For an investment account with holdings: click the chevron to expand"
    expected: "Holdings rows slide out with per-holding APY inputs and compound toggles"
    why_human: "Expand/collapse animation and holding data (requires SimpleFIN with holdings) needs live environment"
  - test: "Verify NavBar order"
    expected: "Links appear in order: Net Worth — Alerts — Projections — Settings"
    why_human: "Visual nav bar layout confirmation"
---

# Phase 9: Projection Engine Verification Report

**Phase Goal:** Users see a forward-looking net worth projection chart driven by per-account growth rates, compound/simple interest toggle, and income allocation modeling
**Verified:** 2026-03-17
**Status:** human_needed — all automated checks pass; 7 items require human/browser verification
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths (from ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|---------|
| 1 | User can set APY per account, toggle compound/simple, include/exclude accounts | VERIFIED | `RateConfigTable.tsx`: APY inputs, `role="switch"` compound toggles, include checkboxes. `projection.ts`: `projectBalance` handles compound/simple branching. `projections.go`: persists via `INSERT OR REPLACE INTO projection_account_settings` |
| 2 | User can model income: annual amount, monthly savings %, per-account allocation | VERIFIED | `IncomeModelingSection.tsx`: income inputs, `AllocationRow.tsx`: percentage bars. `Projections.tsx`: allocation_json state, `saveIncomeSettings` debounced save. `projection.ts`: `calculateProjection` applies `monthlySavings * (allocation/100)` per target |
| 3 | Projection chart shows projected net worth over user-selected time horizon with dashed projected line | VERIFIED | `ProjectionChart.tsx`: `ComposedChart` with `strokeDasharray="8 4"` on projected Line, `ReferenceLine` "Now" marker, `connectNulls={false}`. `HorizonSelector.tsx`: 1y/5y/10y/20y/Custom presets |
| 4 | All projection settings persist across sessions | VERIFIED | Migration 000005 creates `projection_account_settings`, `projection_holding_settings`, `projection_income_settings` tables. `SaveProjectionSettings` and `SaveIncomeSettings` handlers use `INSERT OR REPLACE`. `Projections.tsx`: debounced auto-save with 500ms timeout on every state change |
| 5 | Projections page accessible from main navigation; investment accounts display holdings detail | VERIFIED | `App.tsx`: `to="/projections"` nav link between Alerts and Settings, `path="/projections"` route. Holdings: migration `holdings` table, `FetchAccountsWithHoldings` in sync, `HoldingsRow.tsx` expandable per-holding controls |

**Score:** 5/5 success criteria verified

---

## Required Artifacts

### Plan 01 — Backend data layer

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/db/migrations/000005_projection_settings.up.sql` | 4 projection tables | VERIFIED | All 4 tables created: `projection_account_settings`, `projection_holding_settings`, `holdings`, `projection_income_settings`. `CHECK(id = 1)` singleton constraint present. `REFERENCES accounts(id) ON DELETE CASCADE` on all tables |
| `internal/db/migrations/000005_projection_settings.down.sql` | Rollback migration | VERIFIED | Drops all 4 tables in correct reverse-dependency order |
| `internal/simplefin/client.go` | `Holding` struct, `FetchAccountsWithHoldings`, `fetchAccountData` helper | VERIFIED | `type Holding struct` present with all SimpleFIN fields. `Account` struct has `Holdings []Holding` field with `json:"holdings,omitempty"`. Both `FetchAccountsWithHoldings` and `fetchAccountData` present |
| `internal/sync/sync.go` | `persistHoldings`, calls `FetchAccountsWithHoldings` | VERIFIED | `persistHoldings` deletes then re-inserts holdings in transaction. `SyncOnce` calls `simplefin.FetchAccountsWithHoldings` (line 126). `INSERT INTO holdings` confirmed |
| `internal/simplefin/client_test.go` | Tests for holdings fetch | VERIFIED | `TestFetchAccountsWithHoldings_ReturnsHoldings`, `TestFetchAccountsWithHoldings_OmitsBalancesOnly`, `TestAccount_UnmarshalWithHoldings` all present |
| `internal/sync/sync_test.go` | `TestPersistHoldings` | VERIFIED | `TestPersistHoldings_InsertsHoldings`, `TestPersistHoldings_ReplacesStaleHoldings`, `TestSyncOnce_NoHoldingsForNonInvestment`, `TestSyncOnce_EmptyHoldingsGraceful`, `TestSyncOnce_UsesWithHoldings` all present |

### Plan 02 — Backend API + frontend client

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/api/handlers/projections.go` | 3 handler functions | VERIFIED | `GetProjectionSettings`, `SaveProjectionSettings`, `SaveIncomeSettings` all present. Queries `projection_account_settings`, `FROM holdings`, `projection_income_settings` |
| `internal/api/handlers/projections_test.go` | Handler tests | VERIFIED | 6 test functions: `TestGetProjectionSettings_Defaults`, `TestGetProjectionSettings_HoldingsNested`, `TestGetProjectionSettings_InvestmentNoHoldings`, `TestGetProjectionSettings_IncomeDefaults`, `TestSaveProjectionSettings`, `TestSaveIncomeSettings`, `TestGetProjectionSettings_ExcludesHidden` |
| `internal/api/router.go` | Routes `/api/projections/*` | VERIFIED | `GET /api/projections/settings`, `PUT /api/projections/settings`, `PUT /api/projections/income` all registered |
| `frontend/src/api/client.ts` | 5 interfaces + 3 async functions | VERIFIED | `ProjectionHoldingSetting`, `ProjectionAccountSetting`, `ProjectionIncomeSettings`, `ProjectionSettingsResponse`, `SaveProjectionSettingsRequest`. Functions `getProjectionSettings`, `saveProjectionSettings`, `saveIncomeSettings`. All fetch `/api/projections/*` with `credentials: 'include'` |

### Plan 03 — Projection engine + chart components

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/utils/projection.ts` | `projectBalance`, `calculateProjection`, 4 interfaces | VERIFIED | All 4 interfaces exported. `projectBalance`: compound loop or linear simple formula. `calculateProjection`: 500ms debounce path, `hasHoldings` double-counting prevention, allocation validation within 0.01 of 100% |
| `frontend/src/utils/projection.test.ts` | 10+ test cases | VERIFIED | 12 test cases covering: compound, simple, contributions, 0% flat, 0% with contributions, multi-account sum, exclusion, income allocation, per-holding, point count, growth-only, allocation validation |
| `frontend/src/components/ProjectionChart.tsx` | Recharts ComposedChart, dashed projected line, Now marker | VERIFIED | `ComposedChart`, `strokeDasharray="8 4"` on projected Line, `strokeDasharray="4 4"` on ReferenceLine, `connectNulls={false}` on both Lines, `role="img"`, "No data to project" empty state, gradient fill |
| `frontend/src/components/HorizonSelector.tsx` | 1y/5y/10y/20y/Custom presets, accessible | VERIFIED | `role="radiogroup"`, `aria-checked`, `type="number"`, `min={1}`, `max={50}`. Custom input with debounce 500ms |

### Plan 04 — Configuration UI components

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/components/RateConfigTable.tsx` | Account table with APY/compound/include controls | VERIFIED | "Projection Settings" heading, `grid grid-cols-[1fr_80px_80px_64px]`, `role="switch"`, `aria-checked`, `aria-expanded`, "No accounts found" empty state. Groups by panel type (liquid/savings/investments) |
| `frontend/src/components/HoldingsRow.tsx` | Expandable holdings sub-rows | VERIFIED | `pl-8` indent, `motion-reduce:transition-none`, `transition-[max-height]` animation |
| `frontend/src/components/IncomeModelingSection.tsx` | Collapsible income section with validation | VERIFIED | "Income Modeling", `role="switch"`, `aria-label="Enable income modeling"`, "Savings Allocation", "Must total 100%", `role="status"` on sum display, `motion-reduce:transition-none` |
| `frontend/src/components/AllocationRow.tsx` | Allocation input with visual bar | VERIFIED | `rounded-full` bar track and fill, `transition-all duration-200` on fill width |

### Plan 05 — Page assembly + routing

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/pages/Projections.tsx` | Full page with state, auto-save, chart wiring | VERIFIED | Imports `getProjectionSettings`, `calculateProjection`, `saveProjectionSettings`, `saveIncomeSettings`. `useMemo` for projectionData. `saveTimeoutRef` debounced save. `getNetWorth` for historical data. All 4 sub-components rendered. "No accounts to project" empty state. "Something went wrong loading projection data" error state |
| `frontend/src/pages/Projections.test.tsx` | Page integration tests | VERIFIED | File present with test content |
| `frontend/src/App.tsx` | Route and nav link | VERIFIED | `import Projections from './pages/Projections'`. `to="/projections"` nav link at line 92 (between Alerts at 86 and Settings at 98). `path="/projections"` route at line 129 |

---

## Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/sync/sync.go` | `internal/simplefin/client.go` | `FetchAccountsWithHoldings` call | WIRED | Line 126: `simplefin.FetchAccountsWithHoldings(accountsURL, startDate)` |
| `internal/db/migrations/000005...up.sql` | `accounts` table | `REFERENCES accounts(id) ON DELETE CASCADE` | WIRED | All 3 child tables reference accounts with CASCADE |
| `internal/api/handlers/projections.go` | `projection_account_settings` table | SQL LEFT JOIN | WIRED | `LEFT JOIN projection_account_settings ps ON a.id = ps.account_id` |
| `internal/api/handlers/projections.go` | `holdings` table | SQL query | WIRED | `FROM holdings h` per-account holdings query |
| `frontend/src/api/client.ts` | `/api/projections/settings` | fetch calls | WIRED | GET and PUT both fetch `/api/projections/settings` with `credentials: 'include'` |
| `frontend/src/components/ProjectionChart.tsx` | `frontend/src/utils/projection.ts` | `ProjectionPoint` type prop | WIRED | `import type { ProjectionPoint } from '../utils/projection'`; prop `projectionData: ProjectionPoint[]` |
| `frontend/src/components/HorizonSelector.tsx` | Projections page | `onChange` callback | WIRED | `onChange: (years: number) => void` prop; Projections.tsx passes `setHorizonYears` |
| `frontend/src/components/RateConfigTable.tsx` | `frontend/src/api/client.ts` | `ProjectionAccountSetting` type | WIRED | `import type { ProjectionAccountSetting } from '../api/client'` |
| `frontend/src/components/RateConfigTable.tsx` | `frontend/src/components/HoldingsRow.tsx` | Renders HoldingsRow | WIRED | `import { HoldingsRow }` and renders `<HoldingsRow>` for investment accounts with holdings |
| `frontend/src/components/IncomeModelingSection.tsx` | `frontend/src/components/AllocationRow.tsx` | Renders AllocationRow | WIRED | `import { AllocationRow }` and renders `<AllocationRow>` per allocation target |
| `frontend/src/pages/Projections.tsx` | `frontend/src/api/client.ts` | API function calls | WIRED | `getProjectionSettings`, `saveProjectionSettings`, `saveIncomeSettings` all imported and called |
| `frontend/src/pages/Projections.tsx` | `frontend/src/utils/projection.ts` | `calculateProjection` in useMemo | WIRED | `import { calculateProjection }` and called inside `useMemo` block |
| `frontend/src/pages/Projections.tsx` | `frontend/src/components/ProjectionChart.tsx` | Props `historicalData` and `projectionData` | WIRED | `<ProjectionChart historicalData={historicalData} projectionData={projectionData} isDark={isDark} />` |
| `frontend/src/pages/Projections.tsx` | `frontend/src/components/RateConfigTable.tsx` | accounts + handler props | WIRED | `<RateConfigTable accounts={accounts} onApyChange={...} ... />` |
| `frontend/src/pages/Projections.tsx` | `frontend/src/components/IncomeModelingSection.tsx` | income settings + handler props | WIRED | `<IncomeModelingSection enabled={income.enabled} ... />` |
| `frontend/src/App.tsx` | `frontend/src/pages/Projections.tsx` | `path="/projections"` route | WIRED | `<Route path="/projections" element={<Projections />} />` |

---

## Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|---------|
| PROJ-01 | 09-02, 09-04 | User can set APY per savings account and expected growth rate per investment account | SATISFIED | `RateConfigTable`: APY text input per account; holdings: per-holding APY via `HoldingsRow`. Persisted via `SaveProjectionSettings` → `projection_account_settings` |
| PROJ-02 | 09-03, 09-04 | User can toggle reinvestment (compound vs simple) per account | SATISFIED | `projectBalance` branches on `compound` flag. `RateConfigTable`: `role="switch"` compound toggle per account and per holding. `SaveProjectionSettings` persists `compound` field |
| PROJ-03 | 09-02, 09-04 | User can enable/disable which accounts are included in the projection | SATISFIED | `RateConfigTable`: include checkbox per account. `calculateProjection` skips `!acct.included` and `!h.included`. Persisted in `projection_account_settings.included` |
| PROJ-04 | 09-02, 09-04 | User can model income: annual amount, monthly savings %, and per-account allocation | SATISFIED | `IncomeModelingSection`: annual income + monthly savings % inputs, `AllocationRow` per target, "Must total 100%" validation with `role="status"`. `projection_income_settings` table. `SaveIncomeSettings` handler |
| PROJ-05 | 09-03, 09-05 | Projection chart shows projected net worth over a custom time horizon | SATISFIED | `ProjectionChart`: ComposedChart with dashed projected line, solid historical line, "Now" ReferenceLine. `HorizonSelector`: 1y/5y/10y/20y/Custom. `calculateProjection` takes `horizonYears` parameter |
| PROJ-06 | 09-01, 09-02, 09-05 | All projection settings persist in the database across sessions | SATISFIED | Schema in migration 000005. `SaveProjectionSettings` and `SaveIncomeSettings` upsert to DB. Projections page debounces save on every change. Loads from DB via `getProjectionSettings` on mount |
| PROJ-07 | 09-05 | Projections page accessible from main navigation | SATISFIED | `App.tsx` line 92: `to="/projections"` NavBar link (between Alerts and Settings). Line 129: `path="/projections"` route renders `<Projections />` |
| PROJ-08 | 09-01, 09-02, 09-04 | Investment accounts display available holdings detail from SimpleFIN where supported | SATISFIED | `holdings` table in migration 000005. `FetchAccountsWithHoldings` returns `Holdings []Holding`. `persistHoldings` in sync. `GetProjectionSettings` nests holdings under investment accounts. `HoldingsRow` provides expandable per-holding rate controls |

**All 8 requirements accounted for. No orphaned requirements.**

---

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | — | — | No anti-patterns found |

No TODOs, FIXMEs, stub returns, or empty implementations found in any phase-09 artifact.

The two `return {}` occurrences in `Projections.tsx` are both inside error-handling catch blocks of the `parseAllocationJson` helper — correct fallback behavior, not stubs.

---

## Human Verification Required

### 1. Live chart reactivity

**Test:** Set a non-zero APY on any savings account (e.g., 4.5%), observe chart within 500ms
**Expected:** Projection dashed line curves upward; chart recalculates after brief debounce delay
**Why human:** Debounce timing and visual chart update require browser rendering

### 2. Settings persistence across reload (PROJ-06 end-to-end)

**Test:** Set APY and compound values, reload the page (hard reload)
**Expected:** All APY values, compound toggles, and income settings retained exactly as set
**Why human:** Requires running Go server + SQLite database; cannot verify in static analysis

### 3. Compound vs. simple interest visual difference

**Test:** Set 10% APY on an account, observe chart; toggle from compound to simple
**Expected:** Chart projection line drops noticeably (simple interest yields less growth)
**Why human:** Visual chart difference requires live rendering

### 4. Income modeling end-to-end

**Test:** Enable income modeling, enter $120,000 annual income, 20% savings, allocate 100% to one checking account; observe chart
**Expected:** Chart shows steeper projection; allocation Total shows green "100%"; if allocation is 90% it shows red "Total: 90% (must equal 100%)"
**Why human:** Income contribution magnitude and allocation validation UI require live environment

### 5. Time horizon selector behavior

**Test:** Click 1y, 5y, 10y, 20y presets; click Custom and enter 15
**Expected:** Chart x-axis range adjusts correctly for each; custom input appears and accepts 15; chart updates after debounce
**Why human:** Chart axis scaling and custom input behavior require browser

### 6. Holdings expansion (if SimpleFIN data includes holdings)

**Test:** In Investments panel, click chevron on an account that has holdings
**Expected:** Holdings rows slide out showing per-holding description, market value, APY input, compound toggle, include checkbox
**Why human:** Requires live SimpleFIN data with holdings and browser rendering of CSS transitions

### 7. NavBar visual order

**Test:** Look at the NavBar
**Expected:** Links in order: Net Worth — Alerts — Projections — Settings (Projections between Alerts and Settings)
**Why human:** Visual nav layout confirmation; code order verified but browser rendering checked by eye

---

## Summary

All 13 automated must-haves are verified across 5 plans and 8 requirements (PROJ-01 through PROJ-08):

- **Backend data layer** (Plan 01): Migration 000005 creates all 4 tables with correct foreign keys and constraints. SimpleFIN client extended with `Holding` struct, `Holdings []Holding` on Account, and `FetchAccountsWithHoldings`. Sync calls the new function and persists holdings for investment accounts. 7 new test functions cover all holdings persistence scenarios.

- **Backend API** (Plan 02): Three handlers (`GetProjectionSettings`, `SaveProjectionSettings`, `SaveIncomeSettings`) registered on router, querying all required tables including holdings nesting under investment accounts. Frontend API client has 5 typed interfaces and 3 async functions fetching the correct endpoints.

- **Projection engine** (Plan 03): Pure math functions `projectBalance` and `calculateProjection` with correct compound/simple branching, income contribution distribution, and double-counting prevention via `hasHoldings`. 12 test cases with floating-point precision. `ProjectionChart` renders solid/dashed line transition with "Now" marker. `HorizonSelector` provides full preset and custom input with accessibility.

- **Configuration UI** (Plan 04): `RateConfigTable` groups accounts by panel type with APY inputs, compound toggles, include checkboxes, and chevron-expandable holdings rows. `IncomeModelingSection` has collapsible income inputs, `AllocationRow` components, and 100% validation with `role="status"`. All 4 components have test files.

- **Page assembly** (Plan 05): `Projections.tsx` wires all components with proper state management, `useMemo` for projection recalculation, debounced auto-save (500ms), loading/error/empty states. `App.tsx` has NavBar link (Alerts → Projections → Settings order) and route at `/projections`.

7 items require human browser verification (live reactivity, persistence, visual correctness, holdings data).

---

_Verified: 2026-03-17_
_Verifier: Claude (gsd-verifier)_
