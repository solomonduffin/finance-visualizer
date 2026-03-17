# Phase 6: Operational Quick Wins - Research

**Researched:** 2026-03-16
**Domain:** Sync diagnostics UI, growth rate calculation, settings key-value toggle
**Confidence:** HIGH

## Summary

Phase 6 extends two existing pages (Settings and Dashboard) with no new routes or pages. The Settings page gains a "Sync History" section (timeline of last 7 sync attempts with expandable error details) and a "Dashboard Preferences" section (global toggle for growth badges). The Dashboard's PanelCard component gains an inline growth badge showing 30-day percentage change.

The backend work centers on two new API endpoints: `GET /api/sync-log` (returns last 7 sync_log entries with sanitized error text) and `GET /api/growth` (computes per-panel 30-day growth using balance_snapshots). A third endpoint for the settings toggle uses the existing settings key-value table pattern. All financial arithmetic uses `shopspring/decimal` per project convention.

**Primary recommendation:** Compute growth server-side (avoids shipping decimal arithmetic to the browser, reuses the proven `dayAccumulator` pattern from `GetBalanceHistory`, and keeps panel total logic in one place). Deliver the toggle setting via the existing `GET /api/settings` response to avoid an extra round-trip.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Timeline list layout in Settings showing last 7 sync entries
- Each entry shows: timestamp, status indicator (green check / red X / amber warning), account count
- Successful syncs: "12 accounts synced"
- Failed syncs: red X with expandable error detail (click to expand, sanitized -- no credentials/tokens)
- Partial failures (some accounts succeeded, some failed): amber warning icon with "10 synced, 2 failed", expandable error shows per-account failure reasons
- Section placed below the Accounts section on Settings page
- New API endpoint: GET /api/sync-log returning last 7 entries
- Badge appears inline next to the total balance on each panel card: `$12,450.00  triangle_up +2.3%`
- Small triangle indicator + sign: triangle_up +2.3% (green) / triangle_down -1.5% (red)
- When badge is hidden (no data / zero base), render an invisible placeholder to prevent layout shift
- Tooltip on hover shows: dollar change + time period (e.g., "+$280 over 30 days")
- Badge only appears when there's a meaningful, calculable change (>0.0% with valid 30-day-ago baseline)
- Growth calculated per panel total (liquid, savings, investments), not per individual account
- Liquid growth uses net change: (checking - credit today) vs (checking - credit 30 days ago)
- Credit cards excluded from individual account growth display -- only contribute through liquid panel total
- New accounts with <30 days: use available data -- accounts that didn't exist 30 days ago contribute $0 to the earlier total
- When 30-day-ago panel total is $0 or doesn't exist: hide badge (with invisible placeholder)
- Use shopspring/decimal for all growth arithmetic
- Global toggle -- single on/off switch controls growth badges on all panel cards
- ON by default for new users
- Lives in a new "Dashboard Preferences" section on Settings page
- Immediate save -- toggle flips and persists instantly (consistent with Phase 5 instant-save pattern)
- Stored in existing settings key-value table

### Claude's Discretion
- Exact growth badge typography and spacing relative to the total balance
- Tooltip implementation approach (native title vs custom tooltip)
- Sync timeline entry animation/transition when expanding error details
- API response shape for sync log entries
- Whether to compute growth server-side or client-side (both have the data)
- Dashboard Preferences section styling and toggle component design

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| OPS-01 | Settings page shows log of recent sync attempts with timestamps, success/failure status, and account counts | sync_log table already exists with all needed columns; new GET /api/sync-log endpoint returns last 7 entries; SyncHistorySection component renders timeline |
| OPS-02 | Failed syncs show expandable error details (with sensitive data sanitized) | error_text stored in sync_log; sanitization function strips URL patterns and credentials; expand/collapse UI with conditional rendering |
| INSIGHT-01 | Each panel card shows percentage change over the last 30 days with green/red color coding | New GET /api/growth endpoint computes panel totals at today and 30 days ago using balance_snapshots + dayAccumulator pattern; GrowthBadge component renders inline |
| INSIGHT-06 | User can toggle the 30-day growth rate badge on/off from settings | Settings key-value table stores `growth_badge_enabled`; DashboardPreferencesSection with instant-save toggle; setting included in GET /api/settings response |
</phase_requirements>

## Standard Stack

### Core (already in project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| shopspring/decimal | v1.4.0 | Growth percentage arithmetic | Project decision -- all financial math uses decimal, not float64 |
| go-chi/chi | v5.2.5 | HTTP routing for new endpoints | Existing router, add 2-3 new routes |
| modernc.org/sqlite | latest | Database queries for sync_log and balance_snapshots | Existing DB driver (CGo-free) |
| React | 19.2.4 | Frontend components | Existing framework |
| Tailwind CSS | v4.2.1 | Styling for new sections and badge | Existing CSS-first config |
| Vitest | 3.2.1 | Frontend unit tests | Existing test framework |
| @testing-library/react | 16.3.2 | Component testing | Existing test tooling |

### Supporting (no new dependencies needed)
This phase requires zero new library installations. All features are built with existing project dependencies.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Server-side growth calculation | Client-side JS calculation | Client approach would require shipping decimal.js or using float arithmetic. Server-side is consistent with how summary/history already work and avoids floating-point rounding bugs. **Recommend: server-side.** |
| Native `title` tooltip | Custom React tooltip component | Native title is zero-dependency, accessible by default, but cannot be styled or show rich content. A lightweight custom tooltip (pure CSS + state) allows matching the design system. **Recommend: custom tooltip (no library needed, ~30 lines).** |
| CSS `transition` for expand | Animation library | CSS `max-height` transition handles expand/collapse cleanly. No library needed. |

## Architecture Patterns

### Recommended Project Structure
```
internal/api/handlers/
  synclog.go            # GET /api/sync-log handler
  growth.go             # GET /api/growth handler
  settings.go           # Extend existing -- add growth_badge_enabled to response

frontend/src/components/
  SyncHistory.tsx        # Sync history timeline section
  GrowthBadge.tsx        # Inline growth percentage badge + tooltip
  DashboardPreferences.tsx  # Toggle section for settings page
```

### Pattern 1: Settings Key-Value Extension
**What:** Add `growth_badge_enabled` to the existing settings table, include it in the `GET /api/settings` response.
**When to use:** Any boolean preference that controls UI behavior.
**Example:**
```go
// In GetSettings handler -- extend settingsResponse struct
type settingsResponse struct {
    Configured          bool    `json:"configured"`
    LastSyncAt          *string `json:"last_sync_at"`
    LastSyncStatus      *string `json:"last_sync_status"`
    GrowthBadgeEnabled  bool    `json:"growth_badge_enabled"`
}

// Query the setting (defaults to true for new users)
var growthEnabled string
err = database.QueryRowContext(r.Context(),
    `SELECT value FROM settings WHERE key='growth_badge_enabled'`,
).Scan(&growthEnabled)
if err == sql.ErrNoRows {
    resp.GrowthBadgeEnabled = true // ON by default
} else if err == nil {
    resp.GrowthBadgeEnabled = growthEnabled != "false"
}
```

### Pattern 2: Growth Calculation (Server-Side dayAccumulator)
**What:** Compute panel totals at two points in time (latest date and 30 days prior), then calculate percentage change.
**When to use:** Any per-panel metric that compares two snapshots.
**Example:**
```go
// Growth response shape
type growthResponse struct {
    Liquid      *growthData `json:"liquid"`
    Savings     *growthData `json:"savings"`
    Investments *growthData `json:"investments"`
}

type growthData struct {
    CurrentTotal string `json:"current_total"`
    PriorTotal   string `json:"prior_total"`
    DollarChange string `json:"dollar_change"`
    PctChange    string `json:"pct_change"` // e.g., "2.30"
}

// Calculation pattern (using shopspring/decimal):
// pctChange = ((current - prior) / prior) * 100
// If prior is zero: return nil (hide badge)
change := current.Sub(prior)
if prior.IsZero() {
    return nil // No meaningful growth when starting from zero
}
pct := change.Div(prior).Mul(decimal.NewFromInt(100))
```

### Pattern 3: Sync Log Entry with Three-State Status
**What:** Derive sync status from `accounts_fetched`, `accounts_failed`, and `error_text` fields.
**When to use:** Rendering sync timeline entries.
**Logic:**
```
if error_text IS NOT NULL AND accounts_fetched = 0:
    status = "failed" (red X)
elif accounts_failed > 0:
    status = "partial" (amber warning)
else:
    status = "success" (green check)
```

### Pattern 4: Instant-Save Toggle (Phase 5 Pattern)
**What:** Toggle fires API call immediately on change, no save button.
**When to use:** Any settings toggle.
**Example:**
```typescript
// Frontend toggle handler (matches Phase 5 instant-save pattern)
async function handleToggleGrowthBadge() {
  const newValue = !growthBadgeEnabled
  setGrowthBadgeEnabled(newValue) // Optimistic update
  try {
    await saveSetting('growth_badge_enabled', String(newValue))
  } catch {
    setGrowthBadgeEnabled(!newValue) // Revert on failure
    // Show error toast
  }
}
```

### Anti-Patterns to Avoid
- **Float arithmetic for financial calculations:** Never use `float64` or JavaScript `Number` for growth percentages derived from balances. Always use `shopspring/decimal` server-side and pass pre-computed string values to the frontend.
- **Storing computed growth in the database:** Growth is derived data -- compute on read from balance_snapshots, not stored separately. Avoids stale cache and extra write complexity.
- **Separate API call for the toggle setting:** Don't create a separate endpoint to GET the toggle state. Include it in the existing `GET /api/settings` response that's already fetched on the Settings page and make the Dashboard also fetch it (or include growth data with the toggle state in the growth endpoint).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Credential sanitization | Regex-based URL scrubber | Pattern-match and strip: `user:pass@`, full URLs matching `https?://...simplefin...`, base64 tokens | Error text from SimpleFIN client already strips credentials from URLs, but defense-in-depth requires sanitization at the display layer |
| Decimal percentage formatting | `toFixed()` in JS | Compute server-side with `shopspring/decimal`, send as string, format with `Number(pct).toFixed(1)` + sign prefix on frontend | Avoid float precision issues in the calculation step |
| Expand/collapse animation | Animation library | CSS `max-height` + `overflow: hidden` + `transition` | Standard CSS pattern, no JS needed for height animation |

**Key insight:** The growth calculation is the hardest part of this phase. The panel total logic already exists in `GetSummary` and `GetBalanceHistory` -- reuse the same SQL pattern with a date filter to get the 30-day-ago total. Don't invent a new aggregation approach.

## Common Pitfalls

### Pitfall 1: Liquid Panel Includes Credit (Negative Balances)
**What goes wrong:** Calculating liquid growth without including credit card balances, or double-counting credit cards.
**Why it happens:** Credit balances are negative numbers. Liquid = checking + credit (where credit is already negative). Growth from -$500 to -$300 on a credit card is positive growth for the liquid panel.
**How to avoid:** Replicate the exact `GetSummary` logic: `liquid = sum(checking) + sum(credit)`. Compute at both time points using the same aggregation.
**Warning signs:** Growth percentage doesn't match what a manual calculation would show when credit card balances change.

### Pitfall 2: Division by Zero When Prior Total is $0
**What goes wrong:** Panic or NaN when calculating percentage change with a zero denominator.
**Why it happens:** New users, new panels with no 30-day history, or all accounts hidden 30 days ago.
**How to avoid:** Check `prior.IsZero()` before division. Return `nil` for the growth data, and render invisible placeholder on frontend.
**Warning signs:** 500 errors on GET /api/growth for users with limited history.

### Pitfall 3: Layout Shift When Badge Appears/Disappears
**What goes wrong:** Panel cards jump in size when one has a badge and another doesn't.
**Why it happens:** The badge adds height or width to the total line.
**How to avoid:** User explicitly requested "invisible placeholder" -- render a `<span>` with `visibility: hidden` (or `opacity-0`) containing the same badge structure when there's no growth data. This reserves the space.
**Warning signs:** Cards misaligning or jumping when growth data loads asynchronously.

### Pitfall 4: Error Text Leaking Sensitive Data
**What goes wrong:** Sync error details shown to the user contain the access URL (which embeds credentials).
**Why it happens:** Go error wrapping chains include context from the original error.
**How to avoid:** The SimpleFIN client already strips credentials from URLs in `FetchAccounts` (line 109 of client.go). Errors stored in sync_log should be credential-free. However, add a sanitization function as defense-in-depth that strips patterns matching `https?://[^@]*@` and base64-like tokens.
**Warning signs:** Error details showing URL-like strings or long base64 strings.

### Pitfall 5: Race Condition Between Growth Fetch and Summary Fetch
**What goes wrong:** Dashboard shows growth badge data that doesn't match the current total.
**Why it happens:** Growth API and Summary API are separate calls, data could change between them.
**How to avoid:** The growth endpoint returns `current_total` along with the percentage -- the frontend can use this for display consistency. Alternatively, the Dashboard already calls `getSummary()`, `getAccounts()`, and `getBalanceHistory(30)` in `Promise.all` -- adding `getGrowth()` to the same batch keeps them in sync within the same render cycle.
**Warning signs:** Badge showing +5% but the total clearly being different from what the growth was calculated against.

### Pitfall 6: Account Type Override Not Respected in Growth Calculation
**What goes wrong:** Growth calculation uses `account_type` instead of `COALESCE(account_type_override, account_type)`, causing accounts the user re-categorized to be in the wrong panel.
**Why it happens:** Copy-pasting a simpler query without the override logic.
**How to avoid:** Always use `COALESCE(a.account_type_override, a.account_type) AS effective_type` -- same as `GetBalanceHistory` and `GetSummary`.
**Warning signs:** Growth showing for a panel the user moved accounts out of.

## Code Examples

### Sync Log API Response Shape
```go
// Source: designed to match sync_log table schema
type syncLogEntry struct {
    ID              int64   `json:"id"`
    StartedAt       string  `json:"started_at"`
    FinishedAt      *string `json:"finished_at"`
    AccountsFetched int     `json:"accounts_fetched"`
    AccountsFailed  int     `json:"accounts_failed"`
    ErrorText       *string `json:"error_text"` // sanitized
    Status          string  `json:"status"`     // derived: "success", "partial", "failed"
}

type syncLogResponse struct {
    Entries []syncLogEntry `json:"entries"`
}
```

### Sync Log Handler
```go
// Source: follows GetSettings pattern in settings.go
func GetSyncLog(database *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        rows, err := database.QueryContext(r.Context(), `
            SELECT id, started_at, finished_at, accounts_fetched, accounts_failed, error_text
            FROM sync_log
            ORDER BY id DESC
            LIMIT 7
        `)
        // ... iterate rows, derive status, sanitize error_text
    }
}
```

### Growth Badge Component (Frontend)
```tsx
// Source: extends PanelCard pattern
interface GrowthBadgeProps {
    pctChange: string | null  // e.g., "2.30" or "-1.50"
    dollarChange: string | null  // e.g., "280.00" or "-150.00"
    visible: boolean  // controlled by settings toggle
}

function GrowthBadge({ pctChange, dollarChange, visible }: GrowthBadgeProps) {
    const hasData = pctChange !== null && pctChange !== "0.00"
    const isPositive = hasData && !pctChange.startsWith('-')
    const pct = hasData ? Number(pctChange).toFixed(1) : '0.0'

    // Invisible placeholder for layout consistency
    if (!visible || !hasData) {
        return <span className="invisible text-sm ml-2">{'\u25B2'} +0.0%</span>
    }

    const tooltipText = dollarChange
        ? `${isPositive ? '+' : ''}${formatCurrency(dollarChange)} over 30 days`
        : ''

    return (
        <span
            className={`text-sm font-medium ml-2 ${
                isPositive ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'
            }`}
            title={tooltipText}
        >
            {isPositive ? '\u25B2' : '\u25BC'} {isPositive ? '+' : ''}{pct}%
        </span>
    )
}
```

### Error Text Sanitization
```go
// Source: defense-in-depth pattern
func sanitizeErrorText(raw string) string {
    // Strip URL credentials (user:pass@host pattern)
    re := regexp.MustCompile(`[a-zA-Z0-9+/=_-]+:[a-zA-Z0-9+/=_-]+@[^\s]+`)
    sanitized := re.ReplaceAllString(raw, "[redacted-url]")
    // Strip standalone base64 tokens (long alphanumeric strings)
    reToken := regexp.MustCompile(`[A-Za-z0-9+/=]{40,}`)
    sanitized = reToken.ReplaceAllString(sanitized, "[redacted-token]")
    return sanitized
}
```

### Settings Toggle Save Endpoint
```go
// Source: follows existing settings pattern
// PUT /api/settings/:key or POST /api/settings with key-value body
func SaveSetting(database *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            Key   string `json:"key"`
            Value string `json:"value"`
        }
        // Validate key is in allowed list
        allowed := map[string]bool{"growth_badge_enabled": true}
        if !allowed[req.Key] {
            http.Error(w, `{"error":"invalid setting key"}`, http.StatusBadRequest)
            return
        }
        _, err := database.ExecContext(r.Context(),
            `INSERT INTO settings (key, value) VALUES (?, ?)
             ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
            req.Key, req.Value,
        )
        // ...
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Float64 for percentages | shopspring/decimal for all financial math | Project inception | Growth calculation MUST use decimal |
| Hard-delete stale accounts | Soft-delete (hidden_at) | Phase 5 / v1.1 | Hidden accounts excluded from growth calculation via WHERE hidden_at IS NULL |
| Single account_type column | COALESCE(account_type_override, account_type) | Phase 5 / v1.1 | Growth query must use effective type, not raw type |

**Deprecated/outdated:**
- None specific to this phase. All patterns are current.

## Open Questions

1. **Should the growth endpoint be separate or merged into the summary response?**
   - What we know: Dashboard currently fetches `getSummary()` in `Promise.all` alongside accounts and history. Growth data could be appended to the summary response to save a round-trip.
   - What's unclear: Whether merging makes the summary handler too complex.
   - Recommendation: **Add growth data to the summary response.** The summary handler already queries the latest balances per panel -- adding a 30-day-ago query is a natural extension. This avoids an extra API call and keeps data in sync. If the handler gets too large, refactor the growth calculation into a helper function.

2. **Tooltip approach: native title vs custom component?**
   - What we know: Native `title` attribute is accessible, zero-dependency, but cannot be styled. Custom tooltip requires ~30 lines of React + CSS.
   - What's unclear: Whether the styled tooltip is worth the complexity.
   - Recommendation: **Start with native `title` attribute.** It works immediately, is accessible, and the tooltip content is simple text ("+$280 over 30 days"). Can upgrade to custom tooltip later if the user wants styled tooltips.

3. **How to handle the growth badge enabled state on the Dashboard?**
   - What we know: The toggle lives in Settings. The Dashboard needs to know the toggle state to hide/show badges.
   - What's unclear: Whether to include the toggle state in the growth/summary API or fetch settings separately on Dashboard.
   - Recommendation: **Include `growth_badge_enabled` in the growth/summary response.** The Dashboard doesn't currently call `GET /api/settings`, and adding that call just for one boolean is wasteful.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `go test` with `testing` package |
| Framework (Frontend) | Vitest 3.2.1 + @testing-library/react 16.3.2 |
| Config file (Go) | None needed (standard `go test`) |
| Config file (Frontend) | `frontend/vitest.config.ts` (jsdom environment) |
| Quick run command (Go) | `cd /home/solomon/finance-visualizer && go test ./internal/api/handlers/ -run TestSyncLog -v` |
| Quick run command (Frontend) | `cd /home/solomon/finance-visualizer/frontend && npx vitest run --reporter=verbose src/components/GrowthBadge.test.tsx` |
| Full suite command (Go) | `cd /home/solomon/finance-visualizer && go test ./...` |
| Full suite command (Frontend) | `cd /home/solomon/finance-visualizer/frontend && npx vitest run` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| OPS-01 | GET /api/sync-log returns last 7 entries with correct fields | unit (Go) | `go test ./internal/api/handlers/ -run TestGetSyncLog -v` | Wave 0 |
| OPS-01 | SyncHistory component renders timeline entries | unit (Frontend) | `npx vitest run src/components/SyncHistory.test.tsx` | Wave 0 |
| OPS-02 | Error text is sanitized (no credentials/tokens) | unit (Go) | `go test ./internal/api/handlers/ -run TestSanitize -v` | Wave 0 |
| OPS-02 | Expandable error detail renders and toggles | unit (Frontend) | `npx vitest run src/components/SyncHistory.test.tsx` | Wave 0 |
| INSIGHT-01 | GET /api/growth returns correct percentage for each panel | unit (Go) | `go test ./internal/api/handlers/ -run TestGetGrowth -v` | Wave 0 |
| INSIGHT-01 | Growth handles zero prior total (returns nil) | unit (Go) | `go test ./internal/api/handlers/ -run TestGetGrowth -v` | Wave 0 |
| INSIGHT-01 | GrowthBadge renders green for positive, red for negative | unit (Frontend) | `npx vitest run src/components/GrowthBadge.test.tsx` | Wave 0 |
| INSIGHT-01 | GrowthBadge shows invisible placeholder when no data | unit (Frontend) | `npx vitest run src/components/GrowthBadge.test.tsx` | Wave 0 |
| INSIGHT-06 | Settings toggle persists growth_badge_enabled | unit (Go) | `go test ./internal/api/handlers/ -run TestGrowthToggle -v` | Wave 0 |
| INSIGHT-06 | DashboardPreferences renders toggle and fires save | unit (Frontend) | `npx vitest run src/components/DashboardPreferences.test.tsx` | Wave 0 |

### Sampling Rate
- **Per task commit:** Run relevant Go handler tests + relevant frontend component tests
- **Per wave merge:** Full Go test suite (`go test ./...`) + full frontend suite (`npx vitest run`)
- **Phase gate:** Both full suites green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/api/handlers/synclog_test.go` -- covers OPS-01, OPS-02 (sync log endpoint + sanitization)
- [ ] `internal/api/handlers/growth_test.go` -- covers INSIGHT-01 (growth calculation with edge cases)
- [ ] `frontend/src/components/GrowthBadge.test.tsx` -- covers INSIGHT-01 frontend
- [ ] `frontend/src/components/SyncHistory.test.tsx` -- covers OPS-01, OPS-02 frontend
- [ ] `frontend/src/components/DashboardPreferences.test.tsx` -- covers INSIGHT-06 frontend
- [ ] `internal/api/handlers/settings_test.go` -- extend existing file with growth toggle tests

## Sources

### Primary (HIGH confidence)
- **Existing codebase** -- all research based on direct code inspection:
  - `internal/db/migrations/000001_init.up.sql` -- sync_log table schema (line 29-36)
  - `internal/sync/sync.go` -- SyncOnce with error handling, finalize pattern
  - `internal/api/handlers/settings.go` -- existing sync_log query pattern
  - `internal/api/handlers/history.go` -- dayAccumulator pattern for panel grouping
  - `internal/api/handlers/summary.go` -- panel total calculation with COALESCE
  - `internal/simplefin/client.go` -- credential stripping in FetchAccounts (line 109)
  - `frontend/src/components/PanelCard.tsx` -- component to extend with growth badge
  - `frontend/src/pages/Settings.tsx` -- page to extend with new sections
  - `frontend/src/pages/Dashboard.tsx` -- data fetching pattern (Promise.all)
  - `frontend/src/api/client.ts` -- API client pattern with typed interfaces

### Secondary (MEDIUM confidence)
- **Phase 5 CONTEXT.md** -- instant-save pattern, settings page structure decisions
- **Project memory** -- shopspring/decimal decision, growth badge toggle requirement

### Tertiary (LOW confidence)
- None -- all findings verified against existing code

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- zero new dependencies, all libraries already in project
- Architecture: HIGH -- all patterns derived from existing codebase (history.go, summary.go, settings.go)
- Pitfalls: HIGH -- identified from direct code analysis (credit balance sign, division by zero, COALESCE pattern)
- Growth calculation: HIGH -- the SQL pattern in GetBalanceHistory is directly reusable

**Research date:** 2026-03-16
**Valid until:** 2026-04-16 (stable -- no external dependencies, all internal patterns)
