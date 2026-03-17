# Phase 5: Data Foundation - Context

**Gathered:** 2026-03-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Soft-delete migration and account display name system — schema prerequisites for all v1.1 features. Users can rename any account with a display name that appears globally, re-categorize accounts between panels, and accounts survive SimpleFIN outages with all user-owned metadata intact. When a previously hidden account reappears, it is automatically restored.

</domain>

<decisions>
## Implementation Decisions

### Account Renaming UX
- Rename accounts in a new "Accounts" section on the existing Settings page, below SimpleFIN config
- Inline text field editing: click pencil icon → name becomes editable → Enter to save, Escape to cancel
- Per-account instant save (no batch "Save All" button)
- Reset button next to renamed accounts to revert to original SimpleFIN name; original name shown as placeholder text in edit field

### Hidden Account Behavior
- When an account disappears from SimpleFIN, it is soft-deleted (hidden) — not hard-deleted
- Hidden accounts excluded from dashboard panel totals and net worth calculations
- Subtle indicator in settings: hidden accounts appear with "Hidden" badge and grayed styling
- When a hidden account reappears in a sync, show a brief toast notification (e.g., "Fidelity 401k is back online")
- Users can manually hide/unhide accounts from settings (toggle per account) — useful for closed accounts still in SimpleFIN

### Display Name Rendering
- Display name replaces everything when set: show "My Checking" not "Chase – My Checking"
- Accounts without display name keep current "Org – Name" format (e.g., "Chase – Chase Checking")
- Display names appear everywhere: dashboard panels, chart tooltips, legends, and any future dropdowns (ACCT-02)
- Original SimpleFIN name visible only in settings (shown as secondary/subtitle text below display name)

### Account Type Reassignment
- Users can re-categorize accounts between panel types (Liquid, Savings, Investments, Other)
- Desktop: drag and drop accounts between grouped sections in settings
- Mobile: dropdown fallback for type selection (touch drag-and-drop is awkward)
- Type override persists in database and affects dashboard panels, charts, and all calculations

### Account List in Settings
- Accounts grouped by panel type (Liquid / Savings / Investments) matching dashboard layout
- Each row shows: account name (display name or original), balance, rename action, hide/show action
- Hidden accounts in a separate collapsible "Hidden Accounts" section at the bottom, grayed out with "Unhide" action

### Claude's Discretion
- Toast notification implementation (lightweight, no external library needed)
- Drag-and-drop library choice for desktop account reordering
- Migration file numbering and column naming conventions
- Exact styling of account list rows, badges, and inline edit fields
- API endpoint design for rename, hide/unhide, and type reassignment
- How COALESCE(display_name, name) is applied across existing queries

</decisions>

<specifics>
## Specific Ideas

- Account list in settings should mirror the dashboard panel grouping so the mental model is consistent
- Drag and drop for re-categorizing accounts feels natural — "grab and move to the right bucket"
- Reset button for display names with original name as placeholder — user always knows what they're overriding

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/sync/sync.go`: `removeStaleAccounts()` currently hard-deletes — needs conversion to soft-delete (set hidden flag instead of DELETE)
- `internal/db/migrations/000001_init.up.sql`: Current accounts table schema — needs new migration adding `display_name`, `hidden_at`, `account_type_override` columns
- `frontend/src/api/client.ts`: `AccountItem` interface `{id, name, balance, account_type, org_name}` — needs `display_name` field
- `frontend/src/components/PanelCard.tsx`: Currently renders `account.org_name ? \`${account.org_name} – ${account.name}\` : account.name` — needs display_name-aware rendering
- `frontend/src/pages/Settings.tsx`: Existing settings page with SimpleFIN config — extend with Accounts section

### Established Patterns
- `modernc.org/sqlite` (CGo-free) — new migration must stay compatible
- `shopspring/decimal` for financial values — balance stored as TEXT
- Tailwind v4 CSS-first config — account list styling uses Tailwind classes
- API client pattern: typed async functions returning typed responses
- Settings key-value table for config storage

### Integration Points
- All account queries (summary, accounts, balance-history handlers) must use `COALESCE(display_name, name)` and filter `WHERE hidden_at IS NULL`
- Sync engine: `removeStaleAccounts()` → convert to `SET hidden_at = CURRENT_TIMESTAMP WHERE id NOT IN (...)`
- Sync engine: auto-restore → `SET hidden_at = NULL WHERE id IN (...)` for accounts that reappear
- New API endpoints needed: `PATCH /api/accounts/:id` (rename, hide/unhide, type override)
- Frontend Settings page: new Accounts section with grouped list, inline rename, drag-and-drop

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 05-data-foundation*
*Context gathered: 2026-03-15*
