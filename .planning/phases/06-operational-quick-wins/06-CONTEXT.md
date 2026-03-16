# Phase 6: Operational Quick Wins - Context

**Gathered:** 2026-03-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Sync failure diagnostics in the settings UI and 30-day growth rate indicators on every panel card. Users can diagnose sync problems without checking logs, and see at-a-glance trends for their liquid, savings, and investment totals. This phase adds no new pages — it extends Settings and Dashboard.

</domain>

<decisions>
## Implementation Decisions

### Sync Diagnostics Display
- Timeline list layout in Settings showing last 7 sync entries
- Each entry shows: timestamp, status indicator (green check / red X / amber warning), account count
- Successful syncs: "12 accounts synced"
- Failed syncs: red X with expandable error detail (click to expand, sanitized — no credentials/tokens)
- Partial failures (some accounts succeeded, some failed): amber warning icon with "10 synced, 2 failed", expandable error shows per-account failure reasons
- Section placed **below the Accounts section** on Settings page
- New API endpoint needed: GET /api/sync-log returning last 7 entries

### Growth Badge Design
- Badge appears **inline next to the total balance** on each panel card: `$12,450.00  ▲ +2.3%`
- Small triangle indicator + sign: ▲ +2.3% (green) / ▼ -1.5% (red)
- When badge is hidden (no data / zero base), render an **invisible placeholder** to prevent layout shift across panels
- Tooltip on hover shows: dollar change + time period (e.g., "+$280 over 30 days")
- Badge only appears when there's a meaningful, calculable change (>0.0% with valid 30-day-ago baseline)

### Growth Calculation
- Growth calculated per **panel total** (liquid, savings, investments), not per individual account
- Liquid growth uses net change: (checking - credit today) vs (checking - credit 30 days ago) — credit card movement affects the trend
- Credit cards excluded from individual account growth display — only contribute through liquid panel total
- New accounts with <30 days: use available data — accounts that didn't exist 30 days ago contribute $0 to the earlier total (new account naturally shows as growth)
- When 30-day-ago panel total is $0 or doesn't exist: hide badge (with invisible placeholder)
- Use shopspring/decimal for all growth arithmetic (existing project decision)

### Settings Toggle
- **Global toggle** — single on/off switch controls growth badges on all panel cards
- **ON by default** for new users
- Lives in a new **"Dashboard Preferences"** section on Settings page
- **Immediate save** — toggle flips and persists instantly (consistent with Phase 5 per-account instant-save pattern)
- Stored in existing settings key-value table

### Claude's Discretion
- Exact growth badge typography and spacing relative to the total balance
- Tooltip implementation approach (native title vs custom tooltip)
- Sync timeline entry animation/transition when expanding error details
- API response shape for sync log entries
- Whether to compute growth server-side or client-side (both have the data)
- Dashboard Preferences section styling and toggle component design

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

No external specs — requirements are fully captured in decisions above and in the following project files:

### Requirements
- `.planning/REQUIREMENTS.md` — OPS-01 (sync log), OPS-02 (error detail), INSIGHT-01 (growth badge), INSIGHT-06 (toggle)
- `.planning/ROADMAP.md` — Phase 6 success criteria (4 criteria)

### Prior Context
- `.planning/phases/05-data-foundation/05-CONTEXT.md` — Settings page structure decisions, instant-save pattern, account display name rendering

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `frontend/src/components/PanelCard.tsx`: Panel card component — growth badge will be added inline next to the `formatCurrency(total)` render
- `frontend/src/components/Toast.tsx`: Toast notification component from Phase 5 — reusable for toggle feedback
- `frontend/src/utils/format.ts`: `formatCurrency` utility — may need `formatPercent` or similar
- `frontend/src/utils/account.ts`: `getAccountDisplayName` utility
- `frontend/src/pages/Settings.tsx`: Settings page with SimpleFIN config + AccountsSection — extend with Sync History and Dashboard Preferences sections
- `frontend/src/api/client.ts`: API client with typed async functions — extend with sync log and growth endpoints

### Established Patterns
- `shopspring/decimal` for financial arithmetic (balance stored as TEXT) — growth calculation must use this
- Tailwind v4 CSS-first config — all new UI uses Tailwind classes
- Settings key-value table for config storage — growth badge toggle stored here
- Per-item instant save (no batch save button) — Phase 5 pattern for account renames
- `COALESCE(display_name, name)` pattern in all account queries
- `WHERE hidden_at IS NULL` filter excludes hidden accounts from calculations

### Integration Points
- `sync_log` table already exists: `id, started_at, finished_at, accounts_fetched, accounts_failed, error_text`
- `balance_snapshots` table with `(account_id, balance_date, balance)` — source for 30-day growth calculation
- `internal/api/handlers/settings.go`: Already queries `sync_log` for last sync status — extend for full log
- `internal/api/handlers/history.go`: `GetBalanceHistory` with `dayAccumulator` pattern — growth calculation can follow similar panel-grouping logic
- `internal/api/router.go`: Register new endpoints (sync log, growth data, toggle setting)

</code_context>

<specifics>
## Specific Ideas

- Growth badge invisible placeholder when hidden — user explicitly wants no layout shift between panels that have data and panels that don't
- Tooltip must include both dollar amount AND time period: "+$280 over 30 days"
- Partial sync failures should feel distinct from total failures — amber/warning, not red

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 06-operational-quick-wins*
*Context gathered: 2026-03-16*
