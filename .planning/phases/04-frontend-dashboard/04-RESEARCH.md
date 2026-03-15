# Phase 4: Frontend Dashboard - Research

**Researched:** 2026-03-15
**Domain:** React 19, Recharts 3.x, Tailwind CSS v4 dark mode, skeleton loading, Vitest component testing
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **Charting library:** Recharts — declarative React charting, good dark mode support, ~45KB gzipped. No other charting libraries.
- **Line chart style:** Smooth monotone curves with subtle area fill beneath each line. Tooltips show date + balance + change from previous day (e.g., "Mar 14: $4,230.50 ↑$50.00"). One tabbed chart area — tabs to switch between Liquid / Savings / Investments series.
- **Net worth donut chart:** Shows percentage in donut segments, total net worth in center, dollar amounts in legend/tooltip. Positioned beside the line chart (~35% width) on desktop; stacks below on mobile. Segment colors match panel accent colors.
- **Dashboard layout:** 3-column grid for panels on desktop, stacks vertically on mobile. Each panel card shows: category label, total balance, individual account list with balances. Below panels: tabbed line chart (~65% width) + net worth donut (~35% width) side by side.
- **Visual style:** Clean fintech aesthetic — subtle card shadows, rounded corners, muted backgrounds. Linear / Mercury look. Each panel has a distinct accent color (blue liquid, green savings, purple investments). Accent colors carry through to chart series and donut segments.
- **Dark mode:** Manual toggle only — sun/moon icon in NavBar. No OS preference detection. Preference stored in localStorage. Tailwind v4 dark mode classes.
- **Empty/loading states:** First launch (no sync): setup prompt with Settings link. Data loading: skeleton cards with shimmering animation matching panel layout. Empty panels (no accounts of a type): hide the panel entirely, remaining panels expand. API errors: inline "Something went wrong" with retry button, no toast library.

### Claude's Discretion

- Exact color values for panel accents and dark mode palette
- Skeleton animation implementation details
- Chart axis formatting and responsive breakpoints
- Typography scale and spacing values
- Account list item styling within panels
- Tab component design for line chart switcher
- Mobile breakpoint (likely md: or lg: Tailwind breakpoint)

### Deferred Ideas (OUT OF SCOPE)

- Per-account balance history drill-down — v2 requirement (DRILL-01)
- Account APY display — v2 requirement (DRILL-02)
- Investment growth/loss calculations — v2 requirement (DRILL-03)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| VIZ-01 | Balance-over-time line chart for each panel (liquid, savings, investments) | Recharts AreaChart with monotone type, tabbed series switching, ResponsiveContainer wrapping |
| VIZ-02 | Net worth breakdown pie/donut chart (liquid vs savings vs investments) | Recharts PieChart with innerRadius/outerRadius + Label in center (fixed in recharts 3.x patch) |
| UX-01 | Dashboard shows data freshness indicator ("Last updated: X ago") | timeAgo() pattern already exists in Settings.tsx — reuse in Dashboard from GET /api/summary last_synced_at |
| UX-02 | App shows appropriate loading/empty states on first run before data exists | Skeleton with animate-pulse (Tailwind v4 built-in) + empty-state guard on summary.last_synced_at |
| UX-03 | Dark/light mode toggle | Tailwind v4 @custom-variant dark + localStorage + html.dark class toggle via useDarkMode hook |
| UX-04 | Mobile-responsive layout | Tailwind responsive grid (grid-cols-1 md:grid-cols-3) + flex-col md:flex-row chart section |
</phase_requirements>

## Summary

Phase 4 is a pure React/TypeScript frontend phase. No backend work. The API is fully built: `GET /api/summary`, `GET /api/accounts`, and `GET /api/balance-history` are all live and return typed JSON. The task is to wire those endpoints into a polished dashboard UI.

The primary new dependency is **Recharts 3.x** (latest stable: 3.8.0, released March 2025). Recharts 3 supports React 19 cleanly with matching peer dependencies. The only install-time friction is that recharts may need `--legacy-peer-deps` due to a `react-is` peer dependency declaration that can lag behind React 19 minor versions — this is documented and well understood. Dark mode uses Tailwind v4's `@custom-variant dark` directive to enable class-based toggling, storing preference in localStorage and applying `dark` class to `<html>`. Skeleton loading uses Tailwind's built-in `animate-pulse`, no extra library needed.

The main complexity zones are: (1) the custom Recharts tooltip showing date + balance + delta requires a typed `TooltipContentProps` component; (2) testing Recharts in jsdom requires a ResizeObserver stub in the test setup file because `ResponsiveContainer` calls it; (3) the center `<Label>` in the donut chart had a regression in Recharts 3.0.0 that is fixed in the released codebase — install current 3.x and it works correctly.

**Primary recommendation:** Install `recharts` at latest 3.x, add `@custom-variant dark` to index.css, add ResizeObserver mock to test-setup.ts, build components as colocated files following the Settings.tsx pattern.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `recharts` | ^3.8.0 | Line chart (AreaChart) + donut chart (PieChart) | Declarative React-native charting; good SVG dark mode theming via prop colors; no canvas |
| `tailwindcss` | ^4.2.1 (already installed) | Utility CSS including dark mode, responsive, animate-pulse | Already configured via @tailwindcss/vite plugin |
| `react` | ^19.2.4 (already installed) | Component runtime | Project baseline |

### Supporting (already in project)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `@testing-library/react` | ^16.3.2 | Component rendering in tests | All Dashboard/component tests |
| `@testing-library/user-event` | ^14.6.1 | User interaction simulation | Tab switching, dark mode toggle |
| `vitest` | ^3.2.1 | Test runner | All frontend tests |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| recharts | Chart.js / react-chartjs-2 | Canvas-based; harder to theme for dark mode; larger bundle |
| recharts | nivo | Higher quality defaults but heavier (D3-heavy), more complex API |
| recharts | Victory | Less maintained, smaller ecosystem in 2025 |
| animate-pulse | tw-shimmer plugin | tw-shimmer is richer but adds a dependency; animate-pulse is sufficient for panel-shaped skeletons |

**Installation:**
```bash
# From frontend/ directory
npm install recharts
# If peer dep error with react-is:
npm install recharts --legacy-peer-deps
```

## Architecture Patterns

### Recommended Project Structure

```
frontend/src/
├── api/
│   └── client.ts          # Add getSummary(), getAccounts(), getBalanceHistory()
├── hooks/
│   └── useDarkMode.ts     # localStorage + html.dark class toggle
├── components/
│   ├── PanelCard.tsx       # Individual panel (liquid/savings/investments)
│   ├── PanelCard.test.tsx
│   ├── BalanceLineChart.tsx # Tabbed AreaChart with custom tooltip
│   ├── BalanceLineChart.test.tsx
│   ├── NetWorthDonut.tsx   # PieChart with center label
│   ├── NetWorthDonut.test.tsx
│   ├── SkeletonDashboard.tsx  # Shimmer placeholder matching panel layout
│   ├── SkeletonDashboard.test.tsx
│   └── EmptyState.tsx      # "Connect your accounts" first-launch prompt
├── pages/
│   ├── Dashboard.tsx       # Replaces placeholder Dashboard in App.tsx
│   ├── Dashboard.test.tsx
│   ├── Login.tsx           # Unchanged
│   ├── Login.test.tsx      # Unchanged
│   ├── Settings.tsx        # Unchanged
│   └── Settings.test.tsx   # Unchanged
├── App.tsx                 # Add useDarkMode, dark mode toggle to NavBar
└── index.css               # Add @custom-variant dark line
```

### Pattern 1: Tailwind v4 Class-Based Dark Mode

**What:** Override Tailwind's default media-query `dark` variant with a class-based custom variant. Toggle via JavaScript by setting `document.documentElement.classList`.

**When to use:** Manual toggle (no OS preference detection) with localStorage persistence — exactly what the user decided.

**CSS setup (index.css):**
```css
/* Source: https://tailwindcss.com/docs/dark-mode */
@import "tailwindcss";
@custom-variant dark (&:where(.dark, .dark *));
```

**Hook (hooks/useDarkMode.ts):**
```typescript
// Source: standard pattern from Tailwind v4 docs + usehooks-ts approach
import { useState, useEffect } from 'react'

export function useDarkMode() {
  const [isDark, setIsDark] = useState<boolean>(() => {
    return localStorage.getItem('theme') === 'dark'
  })

  useEffect(() => {
    const root = document.documentElement
    if (isDark) {
      root.classList.add('dark')
      localStorage.setItem('theme', 'dark')
    } else {
      root.classList.remove('dark')
      localStorage.setItem('theme', 'light')
    }
  }, [isDark])

  return { isDark, toggle: () => setIsDark(d => !d) }
}
```

**CRITICAL:** To prevent flash of wrong theme on page load, add an inline script to `index.html` `<head>` BEFORE any CSS:
```html
<script>
  if (localStorage.getItem('theme') === 'dark') {
    document.documentElement.classList.add('dark')
  }
</script>
```

### Pattern 2: Recharts AreaChart (Line Chart with Area Fill)

**What:** `AreaChart` with `type="monotone"` produces smooth curves. Wrap in `ResponsiveContainer` for fluid width. Use SVG `linearGradient` for subtle area fill that fades to transparent.

**When to use:** VIZ-01 — balance-over-time line chart for each panel series.

```typescript
// Source: recharts official docs + verified pattern
import {
  AreaChart, Area, XAxis, YAxis, Tooltip,
  ResponsiveContainer, CartesianGrid
} from 'recharts'

// data shape from GET /api/balance-history
interface DataPoint { date: string; balance: string }

function BalanceChart({ data, color }: { data: DataPoint[]; color: string }) {
  const parsed = data.map(d => ({ date: d.date, balance: Number(d.balance) }))
  return (
    <ResponsiveContainer width="100%" height={220}>
      <AreaChart data={parsed} margin={{ top: 4, right: 4, left: 4, bottom: 4 }}>
        <defs>
          <linearGradient id={`gradient-${color}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor={color} stopOpacity={0.25} />
            <stop offset="95%" stopColor={color} stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" strokeOpacity={0.3} />
        <XAxis dataKey="date" tick={{ fontSize: 11 }} />
        <YAxis tick={{ fontSize: 11 }} />
        <Tooltip content={<CustomTooltip />} />
        <Area
          type="monotone"
          dataKey="balance"
          stroke={color}
          strokeWidth={2}
          fill={`url(#gradient-${color})`}
          fillOpacity={1}
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
```

### Pattern 3: Recharts Custom Tooltip (with delta)

**What:** Custom tooltip component typed with `TooltipContentProps` from recharts. Shows date, current balance, and change from prior day.

```typescript
// Source: recharts 3.x TooltipContentProps pattern (Discussion #3677)
import type { TooltipProps } from 'recharts'

type CustomTooltipProps = TooltipProps<number, string>

function CustomTooltip({ active, payload, label }: CustomTooltipProps) {
  if (!active || !payload?.length) return null
  const current = payload[0].value as number
  // delta requires passing prev-day data in chart's `data` array
  return (
    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-md p-3 text-sm">
      <p className="font-medium text-gray-900 dark:text-gray-100">{label}</p>
      <p className="text-gray-700 dark:text-gray-300">
        ${current.toLocaleString('en-US', { minimumFractionDigits: 2 })}
      </p>
    </div>
  )
}
```

**NOTE on delta display:** To show "↑$50.00 change", each data point must carry a `delta` field. Compute this during data preparation (map over the array, subtract previous day's balance).

### Pattern 4: Recharts PieChart Donut with Center Label

**What:** `PieChart` with `Pie` having both `innerRadius` and `outerRadius`. Center total using `<Label position="center">`. Bug in Recharts 3.0.0 where center Label didn't render has been **fixed** in current 3.x releases (PR #5987 merged).

```typescript
// Source: recharts official docs + confirmed fix in 3.x
import { PieChart, Pie, Cell, Tooltip, Legend, Label } from 'recharts'

interface Segment { name: string; value: number; color: string }

function NetWorthDonut({ segments, total }: { segments: Segment[]; total: number }) {
  return (
    <PieChart width={240} height={240}>
      <Pie
        data={segments}
        cx="50%"
        cy="50%"
        innerRadius={70}
        outerRadius={100}
        paddingAngle={3}
        dataKey="value"
      >
        {segments.map(seg => (
          <Cell key={seg.name} fill={seg.color} />
        ))}
        <Label
          value={`$${(total / 1000).toFixed(0)}k`}
          position="center"
          className="text-base font-semibold fill-gray-900 dark:fill-gray-100"
        />
      </Pie>
      <Tooltip formatter={(v: number) => `$${v.toLocaleString()}`} />
      <Legend />
    </PieChart>
  )
}
```

**WARNING:** Do not wrap `NetWorthDonut` in `ResponsiveContainer` with percentage width unless you give the container a fixed pixel height parent; this is a known layout trap with PieChart.

### Pattern 5: Skeleton Loading with animate-pulse

**What:** Tailwind's built-in `animate-pulse` fades opacity 100%→50%→100% every 2 seconds. Use gray placeholder divs matching the real panel shape.

**When to use:** When data is fetching after initial mount (loading state).

```typescript
// Source: Tailwind CSS docs animation section
function SkeletonPanel() {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl shadow-md p-5 animate-pulse">
      <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/3 mb-3" />
      <div className="h-8 bg-gray-200 dark:bg-gray-700 rounded w-2/3 mb-4" />
      <div className="space-y-2">
        <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-full" />
        <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-4/5" />
        <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-3/5" />
      </div>
    </div>
  )
}
```

**For shimmer effect** (moving highlight, richer than pulse): Add a custom keyframe in index.css using Tailwind v4's `@theme` block:
```css
@theme {
  --animate-shimmer: shimmer 1.5s infinite;
  @keyframes shimmer {
    0% { background-position: -400px 0; }
    100% { background-position: 400px 0; }
  }
}
```
Then apply: `bg-gradient-to-r from-gray-200 via-gray-100 to-gray-200 bg-[length:800px_100%] animate-[shimmer_1.5s_infinite]`.

### Pattern 6: API Client Extensions

**What:** Extend `frontend/src/api/client.ts` with typed functions for the three new endpoints.

```typescript
// Source: existing client.ts pattern + Phase 3 API contracts

export interface SummaryResponse {
  liquid: string        // e.g. "4230.50"
  savings: string
  investments: string
  last_synced_at: string | null  // ISO 8601 or null
}

export interface AccountItem {
  id: string
  name: string
  balance: string
  account_type: string
}

export interface AccountsResponse {
  liquid: AccountItem[]
  savings: AccountItem[]
  investments: AccountItem[]
  other: AccountItem[]
}

export interface HistoryPoint {
  date: string      // "YYYY-MM-DD"
  balance: string   // decimal string
}

export interface BalanceHistoryResponse {
  liquid: HistoryPoint[]
  savings: HistoryPoint[]
  investments: HistoryPoint[]
}

export async function getSummary(): Promise<SummaryResponse> {
  const res = await fetch('/api/summary', { credentials: 'include' })
  return res.json()
}

export async function getAccounts(): Promise<AccountsResponse> {
  const res = await fetch('/api/accounts', { credentials: 'include' })
  return res.json()
}

export async function getBalanceHistory(days?: number): Promise<BalanceHistoryResponse> {
  const url = days ? `/api/balance-history?days=${days}` : '/api/balance-history'
  const res = await fetch(url, { credentials: 'include' })
  return res.json()
}
```

### Anti-Patterns to Avoid

- **Floating point for balances:** The API returns balance as `string`. Always parse with `Number()` or `parseFloat()` only for display/chart rendering — never for arithmetic (no arithmetic needed in this phase).
- **Using `window.matchMedia` for dark mode:** User decided no OS preference detection. Do not add `prefers-color-scheme` check.
- **Hardcoding chart colors without considering dark mode:** Use explicit `stroke`/`fill` prop color values (hex or CSS variable). Recharts SVG elements don't inherit Tailwind dark mode classes directly — pass colors as props based on current `isDark` state.
- **Passing `width` and `height` props directly to AreaChart while also using ResponsiveContainer:** Pick one. Use `ResponsiveContainer` with percentage `width` and fixed pixel `height`.
- **Rendering PieChart inside ResponsiveContainer with 100% width and no fixed height parent:** Give the donut's parent div a fixed height (e.g., `h-60`) before using `ResponsiveContainer`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Line chart with area fill | Custom SVG path with D3 | `recharts AreaChart` | D3 path math, responsive sizing, tooltip positioning are all solved |
| Donut chart with center text | Custom SVG arcs | `recharts PieChart` with `innerRadius` + `Label` | Arc math, label positioning, legend are all edge-case-heavy |
| Responsive chart resizing | ResizeObserver manual implementation | `ResponsiveContainer` from recharts | Cross-browser ResizeObserver handling already done |
| timeAgo formatting | Custom date arithmetic | Reuse `timeAgo()` from `Settings.tsx` | Already written and correct; extract to `src/utils/time.ts` |
| localStorage sync + html class | Inline localStorage in component | `useDarkMode` custom hook | Centralizes the class-toggle side effect; prevents double-apply |

**Key insight:** Recharts handles all SVG rendering, animation, responsive sizing, and accessibility. The only custom work is data transformation (string balances → numbers, computing per-day deltas) and visual styling (colors, typography).

## Common Pitfalls

### Pitfall 1: ResponsiveContainer Fails in jsdom (ResizeObserver Missing)
**What goes wrong:** Tests that render any recharts `ResponsiveContainer` throw `ResizeObserver is not defined` in jsdom.
**Why it happens:** jsdom does not implement `ResizeObserver`; `ResponsiveContainer` calls it on mount.
**How to avoid:** Add a stub to `frontend/src/test-setup.ts`:
```typescript
// Add to test-setup.ts
global.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
```
**Warning signs:** Test output: `ReferenceError: ResizeObserver is not defined`.

### Pitfall 2: Recharts Components Render 0×0 in Tests
**What goes wrong:** `ResponsiveContainer` renders as 0 width/height in jsdom because no real layout engine runs.
**Why it happens:** ResponsiveContainer needs a measured parent to set chart dimensions.
**How to avoid:** In tests, either mock recharts entirely (`vi.mock('recharts', ...)`) or render the chart directly without ResponsiveContainer and test data-driven output, not visual output. For Dashboard integration tests, test API calls and state, not chart internals.
**Warning signs:** Chart shows nothing, no assertions on chart elements pass.

### Pitfall 3: Dark Mode Flash on Page Load
**What goes wrong:** Page briefly renders in light mode before React mounts and applies `dark` class.
**Why it happens:** React runs after HTML parses; there's a render gap.
**How to avoid:** Add blocking inline `<script>` in `index.html` `<head>` that reads localStorage and sets `document.documentElement.classList` before any paint.
**Warning signs:** Visible light→dark flash on first load when dark mode is stored.

### Pitfall 4: Balance Values Displayed Without Parsing
**What goes wrong:** Displaying `summary.liquid` directly in JSX shows raw string `"4230.50"` instead of formatted `$4,230.50`.
**Why it happens:** API returns strings for decimal precision; need explicit parse + format.
**How to avoid:** Always parse balance strings before formatting:
```typescript
const formatted = Number(summary.liquid).toLocaleString('en-US', {
  style: 'currency', currency: 'USD'
})
```
**Warning signs:** Dollar amounts missing currency symbol or thousands separator.

### Pitfall 5: Empty Panel Grid Gap When One Panel Is Hidden
**What goes wrong:** Hiding a panel (no accounts of that type) leaves a visual gap in the 3-column grid.
**Why it happens:** CSS grid reserves space for hidden elements if using `visibility: hidden`.
**How to avoid:** Use conditional rendering (`panel.length > 0 && <PanelCard .../>`) not CSS hiding. The grid will auto-collapse to 2 or 1 columns of content.
**Warning signs:** Large empty space in panel row where a panel was removed.

### Pitfall 6: Tooltip delta requires prev-day data in chart data array
**What goes wrong:** Custom tooltip tries to compute delta but only has current `value` — no reference to previous day.
**Why it happens:** Recharts tooltips receive only the hovered data point, not surrounding data.
**How to avoid:** Pre-compute deltas during data prep when transforming API response:
```typescript
const enriched = historyPoints.map((pt, i) => ({
  date: pt.date,
  balance: Number(pt.balance),
  delta: i === 0 ? 0 : Number(pt.balance) - Number(historyPoints[i - 1].balance)
}))
```
**Warning signs:** Tooltip shows "↑$NaN" or always shows $0 change.

## Code Examples

Verified patterns from official sources:

### Dark Mode @custom-variant Setup
```css
/* frontend/src/index.css */
/* Source: https://tailwindcss.com/docs/dark-mode */
@import "tailwindcss";
@custom-variant dark (&:where(.dark, .dark *));
```

### ResizeObserver Test Stub
```typescript
/* frontend/src/test-setup.ts */
import '@testing-library/jest-dom'

// Required for Recharts ResponsiveContainer in jsdom
global.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
```

### Panel Accent Color Constants
```typescript
// src/components/panelColors.ts — Claude's discretion per CONTEXT.md
export const PANEL_COLORS = {
  liquid:      { accent: '#3b82f6', label: 'Liquid',      darkAccent: '#60a5fa' },
  savings:     { accent: '#22c55e', label: 'Savings',     darkAccent: '#4ade80' },
  investments: { accent: '#a855f7', label: 'Investments', darkAccent: '#c084fc' },
} as const
```

### timeAgo Utility (extract from Settings.tsx)
```typescript
// src/utils/time.ts
export function timeAgo(dateStr: string): string {
  const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000)
  if (seconds < 60) return 'just now'
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`
  return `${Math.floor(seconds / 86400)}d ago`
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Recharts 2.x internal state via cloned props | Recharts 3.x Redux-based state, public hooks | 2024 (3.0 release) | Custom tooltip typing changed: use `TooltipProps<V,N>` not internal CategoricalChartState |
| Tailwind v3 `darkMode: 'class'` in config | Tailwind v4 `@custom-variant dark` in CSS | 2024 (v4 release) | No tailwind.config.js; pure CSS config |
| `animate-pulse` for skeleton | Still `animate-pulse` | Unchanged | Shimmer is optional enhancement via custom keyframe |
| Recharts center Label bug in 3.0.0 | Fixed in 3.x patch (PR #5987) | Mar 2025 | Use current 3.x; don't pin to 3.0.0 |

**Deprecated/outdated:**
- `CategoricalChartState` (recharts): Removed in 3.0. Don't reference it.
- `darkMode: 'class'` in tailwind.config.js: Not applicable in v4 (no config file); use `@custom-variant`.
- `blendStroke` prop on `<Pie>`: Removed in recharts 3.x; use `stroke="none"` instead.

## Open Questions

1. **React 19 + recharts peer dependency conflict at install time**
   - What we know: recharts 3.x works with React 19 at runtime; peer dep declarations may flag during `npm install`
   - What's unclear: Whether the current published 3.8.x peer deps already declare React 19 support or still list `^18`
   - Recommendation: Run `npm install recharts` first; if peer dep error, use `--legacy-peer-deps`. Document which flag was needed in the commit message.

2. **`react-is` version mismatch**
   - What we know: recharts uses `react-is` internally; if it doesn't exactly match the React 19 minor version, runtime checks may warn
   - What's unclear: Whether 3.8.0 bundles its own `react-is` or relies on the peerDep
   - Recommendation: If warnings appear, add `overrides: { "react-is": "^19.0.0" }` to `frontend/package.json`.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Vitest 3.2.1 + @testing-library/react 16.3.2 |
| Config file | `frontend/vitest.config.ts` |
| Quick run command | `cd frontend && npx vitest run --reporter=verbose` |
| Full suite command | `cd frontend && npx vitest run` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| VIZ-01 | BalanceLineChart renders with series data and correct tab labels | unit | `cd frontend && npx vitest run src/components/BalanceLineChart.test.tsx` | ❌ Wave 0 |
| VIZ-02 | NetWorthDonut renders with correct segment data and center value | unit | `cd frontend && npx vitest run src/components/NetWorthDonut.test.tsx` | ❌ Wave 0 |
| UX-01 | Dashboard displays "Last updated X ago" from summary.last_synced_at | unit | `cd frontend && npx vitest run src/pages/Dashboard.test.tsx` | ❌ Wave 0 |
| UX-02 | Dashboard shows skeleton while loading; shows EmptyState when not configured | unit | `cd frontend && npx vitest run src/pages/Dashboard.test.tsx` | ❌ Wave 0 |
| UX-03 | useDarkMode toggles html.dark class and persists to localStorage | unit | `cd frontend && npx vitest run src/hooks/useDarkMode.test.ts` | ❌ Wave 0 |
| UX-04 | PanelCard renders account list and total balance (layout verified visually; responsive via Tailwind classes) | unit | `cd frontend && npx vitest run src/components/PanelCard.test.tsx` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `cd frontend && npx vitest run --reporter=verbose`
- **Per wave merge:** `cd frontend && npx vitest run`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `frontend/src/hooks/useDarkMode.test.ts` — covers UX-03
- [ ] `frontend/src/components/PanelCard.test.tsx` — covers UX-04
- [ ] `frontend/src/components/BalanceLineChart.test.tsx` — covers VIZ-01
- [ ] `frontend/src/components/NetWorthDonut.test.tsx` — covers VIZ-02
- [ ] `frontend/src/pages/Dashboard.test.tsx` — covers UX-01, UX-02
- [ ] ResizeObserver stub addition to `frontend/src/test-setup.ts`
- [ ] `recharts` package install: `cd frontend && npm install recharts [--legacy-peer-deps]`

## Sources

### Primary (HIGH confidence)

- [Tailwind CSS dark mode docs](https://tailwindcss.com/docs/dark-mode) — @custom-variant directive, class strategy
- [Tailwind CSS animation docs](https://tailwindcss.com/docs/animation) — animate-pulse definition and usage
- [Recharts 3.0 migration guide](https://github.com/recharts/recharts/wiki/3.0-migration-guide) — breaking changes from 2.x
- [Recharts GitHub releases](https://github.com/recharts/recharts/releases) — latest stable: 3.8.0 (March 6, 2025)
- [Recharts center Label fix](https://github.com/recharts/recharts/issues/5985) — confirmed fixed in PR #5987

### Secondary (MEDIUM confidence)

- [recharts peer deps discussion](https://github.com/recharts/recharts/discussions/5701) — React 19 compatibility confirmed working
- [recharts ResponsiveContainer test issues](https://github.com/recharts/recharts/issues/2268) — ResizeObserver mock pattern
- Existing codebase `Settings.tsx` / `Settings.test.tsx` — established patterns for component structure, mocking, and Tailwind usage

### Tertiary (LOW confidence)

- WebSearch: tw-shimmer plugin for Tailwind v4 — not verified via official docs; animate-pulse chosen instead

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — recharts version confirmed from GitHub releases; Tailwind version from existing package.json
- Architecture: HIGH — component structure mirrors existing Settings/Login pages which are working
- Pitfalls: HIGH — ResizeObserver and dark mode flash are documented issues with official GitHub references
- Testing: HIGH — vitest config and setup file already exist; only stubs and new test files needed

**Research date:** 2026-03-15
**Valid until:** 2026-04-15 (recharts 3.x stable; Tailwind v4 stable — unlikely to shift in 30 days)
