# Phase 5: Data Foundation - Research

**Researched:** 2026-03-15
**Domain:** SQLite schema migration, Go HTTP API endpoints, React account management UI
**Confidence:** HIGH

## Summary

Phase 5 introduces three new columns to the `accounts` table (`display_name`, `hidden_at`, `account_type_override`), converts the existing hard-delete stale account logic to soft-delete, and builds a new Accounts management section in the Settings page. This is primarily a schema migration + CRUD phase with moderate frontend complexity from inline editing and drag-and-drop type reassignment.

The existing codebase is well-structured for this work. The Go backend uses `golang-migrate/migrate/v4` with embedded SQL files (`go:embed`), chi router with handler closures, and `modernc.org/sqlite` (CGo-free). The frontend uses React 19, TypeScript, Tailwind v4, Vitest, and Recharts. All patterns are established and this phase extends them rather than introducing new paradigms.

**Primary recommendation:** Add migration `000002_account_metadata.up.sql` with three nullable columns, convert `removeStaleAccounts()` from DELETE to UPDATE, add `PATCH /api/accounts/:id` endpoint, and build the Settings Accounts section using `@dnd-kit/react` for desktop drag-and-drop type reassignment with a dropdown fallback for mobile.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Rename accounts in a new "Accounts" section on the existing Settings page, below SimpleFIN config
- Inline text field editing: click pencil icon -> name becomes editable -> Enter to save, Escape to cancel
- Per-account instant save (no batch "Save All" button)
- Reset button next to renamed accounts to revert to original SimpleFIN name; original name shown as placeholder text in edit field
- When an account disappears from SimpleFIN, it is soft-deleted (hidden) -- not hard-deleted
- Hidden accounts excluded from dashboard panel totals and net worth calculations
- Subtle indicator in settings: hidden accounts appear with "Hidden" badge and grayed styling
- When a hidden account reappears in a sync, show a brief toast notification (e.g., "Fidelity 401k is back online")
- Users can manually hide/unhide accounts from settings (toggle per account) -- useful for closed accounts still in SimpleFIN
- Display name replaces everything when set: show "My Checking" not "Chase -- My Checking"
- Accounts without display name keep current "Org -- Name" format (e.g., "Chase -- Chase Checking")
- Display names appear everywhere: dashboard panels, chart tooltips, legends, and any future dropdowns (ACCT-02)
- Original SimpleFIN name visible only in settings (shown as secondary/subtitle text below display name)
- Users can re-categorize accounts between panel types (Liquid, Savings, Investments, Other)
- Desktop: drag and drop accounts between grouped sections in settings
- Mobile: dropdown fallback for type selection (touch drag-and-drop is awkward)
- Type override persists in database and affects dashboard panels, charts, and all calculations
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

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| ACCT-01 | User can set a custom display name for any connected account in settings | Migration adds `display_name TEXT` column; PATCH endpoint accepts `display_name`; Settings UI inline edit with pencil icon, Enter/Escape, reset button |
| ACCT-02 | Custom display names appear everywhere the account is referenced (panels, charts, dropdowns, alerts, projections) | Backend returns `display_name` field; all queries use `COALESCE(display_name, name)` or equivalent; PanelCard renders display_name when present; AccountItem interface extended |
| OPS-03 | Stale accounts are soft-deleted to preserve user-owned metadata (display names, alert rules, projection rates) | Migration adds `hidden_at DATETIME` column; `removeStaleAccounts()` converted from DELETE to SET hidden_at; all dashboard queries add `WHERE hidden_at IS NULL`; auto-restore clears hidden_at when account reappears |
</phase_requirements>

## Standard Stack

### Core (Already in Project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `golang-migrate/migrate/v4` | v4.19.1 | Database migration runner | Already used; embedded SQL via `go:embed` |
| `modernc.org/sqlite` | v1.46.1 | CGo-free SQLite driver | Already used; no change needed |
| `go-chi/chi/v5` | v5.2.5 | HTTP router | Already used; add PATCH route |
| `shopspring/decimal` | v1.4.0 | Financial arithmetic | Already used; no change needed |
| React | 19.2.4 | Frontend framework | Already used |
| Tailwind CSS | v4.2.1 | Styling | Already used; CSS-first config |
| Vitest | 3.2.1 | Frontend test runner | Already used |
| Recharts | 3.8.0 | Chart library | Already used; tooltip labels need display_name |

### New (Phase 5 additions)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `@dnd-kit/react` | 0.3.2 | Drag-and-drop for account type reassignment | Desktop account re-categorization between panel groups |
| `@dnd-kit/dom` | 0.2.1 | DOM adapter for dnd-kit | Required peer dependency of @dnd-kit/react |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `@dnd-kit/react` | `@atlaskit/pragmatic-drag-and-drop` | Pragmatic is smaller (~4.7kB vs ~12kB) but headless with steeper learning curve; dnd-kit has better sortable presets, more mature docs, and good accessibility. For this use case (drag between 4 groups), dnd-kit's sortable preset is a better fit. |
| `@dnd-kit/react` | No DnD library (dropdown only) | Simpler but user explicitly wants drag-and-drop on desktop. Dropdown is the mobile fallback. |
| Custom toast | `react-hot-toast` / `sonner` | User said "lightweight, no external library needed." A simple CSS-animated div with auto-dismiss (3-4 second timeout, fixed-position bottom-right) is sufficient for the single "account restored" notification. |

**Installation:**
```bash
cd frontend && npm install @dnd-kit/react @dnd-kit/dom
```

No new Go dependencies needed.

## Architecture Patterns

### Database Migration Pattern

New migration file: `internal/db/migrations/000002_account_metadata.up.sql`

The project uses `golang-migrate/migrate` with sequential numbering (000001, 000002, ...) and embedded SQL via `go:embed`. The existing migration is `000001_init`. The next migration MUST be `000002`.

**Migration SQL:**
```sql
-- 000002_account_metadata.up.sql
ALTER TABLE accounts ADD COLUMN display_name TEXT;
ALTER TABLE accounts ADD COLUMN hidden_at DATETIME;
ALTER TABLE accounts ADD COLUMN account_type_override TEXT
    CHECK(account_type_override IN ('checking', 'savings', 'credit', 'investment', 'other'));
```

**Down migration:**
```sql
-- 000002_account_metadata.down.sql
-- SQLite does not support DROP COLUMN prior to 3.35.0.
-- modernc.org/sqlite v1.46.1 bundles SQLite 3.46+ so DROP COLUMN is safe.
ALTER TABLE accounts DROP COLUMN display_name;
ALTER TABLE accounts DROP COLUMN hidden_at;
ALTER TABLE accounts DROP COLUMN account_type_override;
```

**Key constraint:** All three columns MUST be nullable (no NOT NULL, no DEFAULT) because SQLite's ALTER TABLE ADD COLUMN requires the new column to accept NULL or have a constant default. Nullable is correct here: NULL display_name means "use original name," NULL hidden_at means "visible," NULL account_type_override means "use inferred type."

### Query Pattern: COALESCE for Display Names

Every query that returns an account name to the user must use COALESCE to prefer display_name:

```sql
-- Source: SQLite COALESCE documentation
SELECT a.id,
       COALESCE(a.display_name, a.name) AS name,
       a.name AS original_name,
       COALESCE(a.account_type_override, a.account_type) AS account_type,
       a.currency, a.org_name, a.display_name, a.hidden_at
FROM accounts a
WHERE a.hidden_at IS NULL
```

**Affected handlers:**
1. `handlers/accounts.go` - `GetAccounts()` - add COALESCE, add WHERE hidden_at IS NULL, add display_name + account_type_override to response
2. `handlers/summary.go` - `GetSummary()` - add WHERE hidden_at IS NULL, use COALESCE(account_type_override, account_type) for grouping
3. `handlers/history.go` - `GetBalanceHistory()` - add WHERE hidden_at IS NULL (via JOIN), use COALESCE(account_type_override, account_type) for grouping

### Sync Engine: Soft-Delete Pattern

Convert `removeStaleAccounts()` from hard-delete to soft-delete and add auto-restore:

```go
// Soft-delete: mark accounts not in latest fetch as hidden
func softDeleteStaleAccounts(ctx context.Context, db *sql.DB, seenIDs []string) (int64, error) {
    // SET hidden_at = CURRENT_TIMESTAMP WHERE id NOT IN (seenIDs) AND hidden_at IS NULL
}

// Auto-restore: clear hidden_at for accounts that reappear
func restoreReturningAccounts(ctx context.Context, db *sql.DB, seenIDs []string) ([]string, error) {
    // SELECT id, COALESCE(display_name, name) FROM accounts WHERE id IN (seenIDs) AND hidden_at IS NOT NULL
    // UPDATE accounts SET hidden_at = NULL WHERE id IN (seenIDs) AND hidden_at IS NOT NULL
    // Return names of restored accounts for toast notification
}
```

**Critical:** The restore function must run BEFORE the soft-delete function in SyncOnce. Order: (1) processAccount for each account, (2) restoreReturningAccounts for seenIDs, (3) softDeleteStaleAccounts for NOT IN seenIDs.

**Critical:** The existing `removeStaleAccounts()` also deletes from `balance_snapshots`. The soft-delete version must NOT delete snapshots -- the whole point of soft-delete is preserving history data.

### API Endpoint Pattern

Single PATCH endpoint for all account metadata updates:

```
PATCH /api/accounts/:id
Content-Type: application/json

// Rename:
{ "display_name": "My Checking" }

// Clear display name (revert to original):
{ "display_name": null }

// Hide:
{ "hidden": true }

// Unhide:
{ "hidden": false }

// Change type:
{ "account_type_override": "savings" }

// Combination (all fields optional):
{ "display_name": "My 401k", "account_type_override": "investment" }
```

Response: the updated account object.

Use chi's URL parameter extraction: `chi.URLParam(r, "id")`.

This follows the existing handler pattern: closure function returning `http.HandlerFunc`, accepting `*sql.DB`.

### Frontend: Account Management Section in Settings

The Settings page currently has three cards: SimpleFIN Connection, Sync Status, Actions. Add an "Accounts" card below Actions, visible only when accounts exist.

**Component structure:**
```
Settings.tsx
  +-- AccountsSection (new)
        +-- AccountGroup (one per panel type: Liquid, Savings, Investments, Other)
        |     +-- AccountRow (one per account)
        |           +-- InlineEdit (pencil icon -> editable field)
        |           +-- TypeDropdown (mobile only)
        |           +-- HideToggle
        +-- HiddenAccountsCollapsible
              +-- AccountRow (grayed, with Unhide action)
```

### Frontend: Display Name Rendering Pattern

```typescript
// Utility function used everywhere accounts are displayed
function getAccountDisplayName(account: AccountItem): string {
  if (account.display_name) return account.display_name
  return account.org_name ? `${account.org_name} – ${account.name}` : account.name
}
```

Update `PanelCard.tsx` line 48 to use this utility instead of the inline ternary. The `AccountItem` interface in `client.ts` needs `display_name?: string | null` added.

### Frontend: Toast Notification Pattern

Simple implementation without external library:

```typescript
// Lightweight toast: fixed-position div, auto-dismiss after 4 seconds
function Toast({ message, onDismiss }: { message: string; onDismiss: () => void }) {
  useEffect(() => {
    const timer = setTimeout(onDismiss, 4000)
    return () => clearTimeout(timer)
  }, [onDismiss])

  return (
    <div className="fixed bottom-4 right-4 z-50 bg-green-600 text-white px-4 py-3 rounded-lg shadow-lg
                    animate-[slideUp_0.3s_ease-out] text-sm font-medium">
      {message}
    </div>
  )
}
```

Toast is triggered after sync when the backend response includes restored account names. The sync response (`POST /api/sync/now`) should be extended to return `{ ok: true, restored_accounts: ["Fidelity 401k"] }` so the frontend can show the toast.

### Anti-Patterns to Avoid

- **Overriding account_type in the accounts row:** The `account_type` column stores the inferred type from SimpleFIN. The override must be in a SEPARATE column (`account_type_override`) so the original inference is preserved. On each sync upsert, `account_type` is still updated from SimpleFIN -- only `account_type_override` is user-controlled and persists.
- **Deleting balance_snapshots on soft-delete:** The soft-delete must preserve all history. Never delete from `balance_snapshots` for hidden accounts.
- **Using display_name in sync upsert ON CONFLICT:** The upsert in `processAccount()` updates `name`, `account_type`, etc. from SimpleFIN data. It must NOT touch `display_name`, `hidden_at`, or `account_type_override` -- these are user-owned columns.
- **Forgetting WHERE hidden_at IS NULL:** Every existing query that feeds the dashboard MUST be updated. Missing this causes hidden accounts to appear in totals.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Drag-and-drop between groups | Custom mousedown/mousemove/mouseup handlers | `@dnd-kit/react` with sortable | Touch events, accessibility (keyboard DnD), drop indicators, animation -- enormous surface area |
| Migration versioning | Manual SQL execution | `golang-migrate/migrate/v4` | Already integrated; handles version tracking, up/down, idempotency |
| URL parameter extraction | Manual path parsing | `chi.URLParam(r, "id")` | Already using chi; handles encoding, trailing slashes |
| Inline edit keyboard handling | Custom key event management | Standard input onKeyDown with Enter/Escape | Straightforward but must handle: blur on Escape (revert), blur on Enter (save), click-away (save or revert -- user decision says Enter to save, Escape to cancel) |

**Key insight:** The drag-and-drop is the only genuinely complex UI element. Everything else (inline edit, hide/unhide toggle, collapsible section) is standard React state management with Tailwind styling.

## Common Pitfalls

### Pitfall 1: Sync Upsert Overwriting User Metadata
**What goes wrong:** The existing `processAccount()` upsert uses `ON CONFLICT(id) DO UPDATE SET name=excluded.name, account_type=excluded.account_type, ...`. If `display_name` or `account_type_override` are included in the SET clause, each sync would wipe user customizations.
**Why it happens:** Copy-paste from existing column list without thinking about which columns are "system-owned" vs "user-owned."
**How to avoid:** The upsert ON CONFLICT clause must explicitly list only system columns: `name`, `account_type`, `currency`, `org_name`, `org_slug`, `updated_at`. Never include `display_name`, `hidden_at`, or `account_type_override`.
**Warning signs:** Display names reverting after a sync cycle.

### Pitfall 2: Account Type Override Not Applied in Summary/History
**What goes wrong:** The summary and history handlers currently group by `a.account_type`. If they don't use `COALESCE(a.account_type_override, a.account_type)`, overridden accounts appear in the wrong panel.
**Why it happens:** Forgetting to update the WHERE/GROUP BY clauses in handlers that don't directly return account objects.
**How to avoid:** Grep for all `account_type` references in handlers and ensure each uses the COALESCE pattern.
**Warning signs:** Moving an account to "Savings" in settings but it still shows in "Liquid" on the dashboard.

### Pitfall 3: SQLite ALTER TABLE Limitations
**What goes wrong:** Trying to add a NOT NULL column without a default value, or trying to add a column with a non-constant default.
**Why it happens:** SQLite's ALTER TABLE ADD COLUMN is more restrictive than PostgreSQL/MySQL.
**How to avoid:** All three new columns are nullable with no default. This is correct: NULL = "not set."
**Warning signs:** Migration fails with "Cannot add a NOT NULL column with default value NULL."

### Pitfall 4: Race Condition in Soft-Delete + Restore
**What goes wrong:** If restore runs after soft-delete in the same sync cycle, a returning account gets soft-deleted and then immediately restored, but the soft-delete might catch accounts that WERE just processed.
**Why it happens:** Wrong ordering of operations in SyncOnce.
**How to avoid:** Process accounts first (upsert), then restore (clear hidden_at for seen IDs), then soft-delete (set hidden_at for NOT IN seen IDs). Since seenIDs only contains successfully-processed accounts, the soft-delete correctly targets only the accounts NOT in the current fetch.
**Warning signs:** Accounts flickering between hidden and visible across syncs.

### Pitfall 5: Frontend Type Mismatch After Override
**What goes wrong:** The `AccountItem.account_type` field returns the original inferred type, but the dashboard groups by panel type. If the backend returns the original type instead of the effective type, the frontend renders the account in the wrong panel.
**Why it happens:** Backend returns raw `account_type` instead of the COALESCE result.
**How to avoid:** The `GetAccounts` handler must use `COALESCE(account_type_override, account_type)` in both the SELECT and the Go switch statement that groups accounts into response categories.
**Warning signs:** Account appears in wrong panel despite type override being set.

### Pitfall 6: Settings Page Missing Dark Mode
**What goes wrong:** The current Settings page (`Settings.tsx`) does not use dark mode classes. New UI elements would match, but existing elements won't.
**Why it happens:** Settings was built before dark mode was fully propagated.
**How to avoid:** When adding the Accounts section, also add `dark:` variants to existing Settings card elements for consistency. The page root should use `bg-gray-50 dark:bg-gray-900` matching Dashboard.
**Warning signs:** Settings page is bright white when the rest of the app is dark.

## Code Examples

### Migration SQL (verified against SQLite ALTER TABLE docs)

```sql
-- Source: https://www.sqlite.org/lang_altertable.html
-- 000002_account_metadata.up.sql

ALTER TABLE accounts ADD COLUMN display_name TEXT;
ALTER TABLE accounts ADD COLUMN hidden_at DATETIME;
ALTER TABLE accounts ADD COLUMN account_type_override TEXT
    CHECK(account_type_override IN ('checking', 'savings', 'credit', 'investment', 'other'));
```

### Updated processAccount Upsert (preserves user columns)

```go
// Source: existing internal/sync/sync.go processAccount()
// Note: ON CONFLICT SET list must NOT include display_name, hidden_at, account_type_override
_, err := db.ExecContext(ctx, `
    INSERT INTO accounts(id, name, account_type, currency, org_name, org_slug)
    VALUES(?, ?, ?, ?, ?, ?)
    ON CONFLICT(id) DO UPDATE SET
        name=excluded.name,
        account_type=excluded.account_type,
        currency=excluded.currency,
        org_name=excluded.org_name,
        org_slug=excluded.org_slug,
        updated_at=CURRENT_TIMESTAMP
`, acct.ID, acct.Name, InferAccountType(acct.Name), acct.Currency, acct.Org.Name, acct.Org.ID)
```

Note: This is IDENTICAL to the current code. The key insight is that no change is needed to the upsert -- the new columns are simply not included, so they retain their user-set values across syncs.

### Soft-Delete Conversion

```go
// Source: pattern from existing removeStaleAccounts in sync.go
func softDeleteStaleAccounts(ctx context.Context, db *sql.DB, seenIDs []string) (int64, error) {
    placeholders := make([]string, len(seenIDs))
    args := make([]interface{}, len(seenIDs))
    for i, id := range seenIDs {
        placeholders[i] = "?"
        args[i] = id
    }
    inClause := strings.Join(placeholders, ",")

    // Only soft-delete accounts that are currently visible
    res, err := db.ExecContext(ctx,
        fmt.Sprintf(`UPDATE accounts SET hidden_at = CURRENT_TIMESTAMP
                     WHERE id NOT IN (%s) AND hidden_at IS NULL`, inClause),
        args...)
    if err != nil {
        return 0, fmt.Errorf("soft-delete stale accounts: %w", err)
    }
    return res.RowsAffected()
}
```

### PATCH Handler Pattern

```go
// Source: follows existing chi handler pattern from router.go
func UpdateAccount(database *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        accountID := chi.URLParam(r, "id")
        if accountID == "" {
            http.Error(w, `{"error":"missing account id"}`, http.StatusBadRequest)
            return
        }

        var req struct {
            DisplayName *string `json:"display_name"` // pointer to distinguish null from absent
            Hidden      *bool   `json:"hidden"`
            TypeOverride *string `json:"account_type_override"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
            return
        }

        // Build dynamic UPDATE based on which fields are present
        // ... (see implementation phase)
    }
}
```

### Frontend AccountItem Extension

```typescript
// Source: existing frontend/src/api/client.ts AccountItem interface
export interface AccountItem {
  id: string
  name: string           // COALESCE(display_name, name) from backend
  original_name: string  // raw SimpleFIN name
  balance: string
  account_type: string   // effective type (COALESCE of override and inferred)
  org_name: string
  display_name: string | null  // user-set display name, null if not set
  hidden_at: string | null     // ISO timestamp if hidden, null if visible
  account_type_override: string | null // user override, null if using inferred
}
```

### Display Name Utility

```typescript
// Source: new utility based on existing PanelCard.tsx rendering logic
export function getAccountDisplayName(account: { display_name?: string | null; org_name?: string; name: string }): string {
  if (account.display_name) return account.display_name
  return account.org_name ? `${account.org_name} – ${account.name}` : account.name
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hard-delete stale accounts (`DELETE FROM accounts WHERE id NOT IN (...)`) | Soft-delete with `hidden_at` timestamp | This phase | Preserves display names, balance history, and future alert/projection config |
| Inferred `account_type` only | `COALESCE(account_type_override, account_type)` | This phase | Users can re-categorize accounts between panels |
| `react-beautiful-dnd` | `@dnd-kit/react` or `@atlaskit/pragmatic-drag-and-drop` | 2023-2024 | react-beautiful-dnd is deprecated; dnd-kit and pragmatic-drag-and-drop are the active successors |

**Deprecated/outdated:**
- `react-beautiful-dnd`: Deprecated by Atlassian, replaced by `pragmatic-drag-and-drop`. Do not use.
- `react-dnd`: Has React 19 compatibility issues. Do not use.

## Open Questions

1. **Click-away behavior on inline edit**
   - What we know: Enter saves, Escape cancels (from user decisions)
   - What's unclear: What happens when user clicks away from an active edit field without pressing Enter or Escape?
   - Recommendation: Treat click-away as "save" (same as Enter). This is the most common pattern and prevents data loss. Implement via `onBlur` handler.

2. **Sync response format for restored accounts**
   - What we know: Toast should say "Fidelity 401k is back online" when a hidden account reappears
   - What's unclear: The current `POST /api/sync/now` returns `{ ok: true }`. Need to extend to include restored account names.
   - Recommendation: Return `{ ok: true, restored: ["Fidelity 401k"] }` from the sync endpoint. The sync engine already knows which accounts were restored. Frontend shows one toast per restored account (or a combined toast if multiple).

3. **Drag-and-drop between groups vs. within groups**
   - What we know: User wants to drag accounts between panel type groups (Liquid, Savings, Investments, Other)
   - What's unclear: Whether reordering within a group is also desired
   - Recommendation: Implement cross-group drag only (for type reassignment). No within-group reordering -- accounts are ordered alphabetically within each group. This simplifies the DnD implementation significantly.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `go test` (stdlib) |
| Framework (Frontend) | Vitest 3.2.1 + jsdom + @testing-library/react |
| Config file (Go) | None needed (stdlib) |
| Config file (Frontend) | `frontend/vitest.config.ts` |
| Quick run command (Go) | `go test ./internal/... -count=1 -run TestPhase5` |
| Quick run command (Frontend) | `cd frontend && npx vitest run --reporter=verbose src/pages/Settings.test.tsx src/components/PanelCard.test.tsx` |
| Full suite command | `go test ./... -count=1 && cd frontend && npx vitest run` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ACCT-01 | PATCH /api/accounts/:id sets display_name | unit (Go) | `go test ./internal/api/handlers/ -run TestUpdateAccount_DisplayName -count=1` | No -- Wave 0 |
| ACCT-01 | Settings UI: inline edit saves display name | unit (Frontend) | `cd frontend && npx vitest run src/pages/Settings.test.tsx` | Partial -- needs new tests |
| ACCT-01 | Reset button clears display_name | unit (Go+Frontend) | `go test ./internal/api/handlers/ -run TestUpdateAccount_ClearDisplayName -count=1` | No -- Wave 0 |
| ACCT-02 | GetAccounts returns COALESCE(display_name, name) | unit (Go) | `go test ./internal/api/handlers/ -run TestGetAccounts_DisplayName -count=1` | No -- Wave 0 |
| ACCT-02 | PanelCard renders display_name when present | unit (Frontend) | `cd frontend && npx vitest run src/components/PanelCard.test.tsx` | Partial -- needs display_name test |
| ACCT-02 | GetSummary excludes hidden accounts | unit (Go) | `go test ./internal/api/handlers/ -run TestGetSummary_ExcludesHidden -count=1` | No -- Wave 0 |
| ACCT-02 | GetBalanceHistory uses overridden account_type | unit (Go) | `go test ./internal/api/handlers/ -run TestGetBalanceHistory_TypeOverride -count=1` | No -- Wave 0 |
| OPS-03 | softDeleteStaleAccounts sets hidden_at instead of deleting | unit (Go) | `go test ./internal/sync/ -run TestSoftDelete -count=1` | No -- Wave 0 |
| OPS-03 | Soft-deleted accounts preserve balance_snapshots | unit (Go) | `go test ./internal/sync/ -run TestSoftDelete_PreservesSnapshots -count=1` | No -- Wave 0 |
| OPS-03 | restoreReturningAccounts clears hidden_at | unit (Go) | `go test ./internal/sync/ -run TestRestore -count=1` | No -- Wave 0 |
| OPS-03 | Manual hide/unhide via PATCH endpoint | unit (Go) | `go test ./internal/api/handlers/ -run TestUpdateAccount_HideUnhide -count=1` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/... -count=1 && cd frontend && npx vitest run`
- **Per wave merge:** Full suite: `go test ./... -count=1 && cd frontend && npx vitest run`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/api/handlers/update_account_test.go` -- covers ACCT-01, OPS-03 (hide/unhide)
- [ ] `internal/api/handlers/accounts_test.go` -- extend with display_name and hidden_at test cases (ACCT-02)
- [ ] `internal/api/handlers/summary_test.go` -- extend with hidden account exclusion tests (ACCT-02)
- [ ] `internal/api/handlers/history_test.go` -- extend with type override and hidden exclusion tests (ACCT-02)
- [ ] `internal/sync/sync_test.go` -- extend with soft-delete and restore tests (OPS-03)
- [ ] `frontend/src/pages/Settings.test.tsx` -- extend with accounts section tests (ACCT-01)
- [ ] `frontend/src/components/PanelCard.test.tsx` -- extend with display_name rendering test (ACCT-02)
- [ ] Migration file `000002_account_metadata.up.sql` and `.down.sql` -- needed before any code changes
- [ ] `@dnd-kit/react` and `@dnd-kit/dom` npm install -- needed for account type reassignment UI

## Sources

### Primary (HIGH confidence)
- SQLite ALTER TABLE documentation: https://www.sqlite.org/lang_altertable.html -- verified ADD COLUMN constraints (nullable required, no expression defaults)
- Existing codebase files: `internal/db/migrations/000001_init.up.sql`, `internal/sync/sync.go`, `internal/api/handlers/accounts.go`, `internal/api/handlers/summary.go`, `internal/api/handlers/history.go`, `frontend/src/api/client.ts`, `frontend/src/components/PanelCard.tsx`, `frontend/src/pages/Settings.tsx`
- golang-migrate/migrate GitHub: https://github.com/golang-migrate/migrate -- verified sequential numbering convention (000001, 000002)

### Secondary (MEDIUM confidence)
- @dnd-kit/react npm: https://www.npmjs.com/package/@dnd-kit/react -- version 0.3.2 confirmed, React 19 support in progress
- @atlaskit/pragmatic-drag-and-drop npm: https://www.npmjs.com/package/@atlaskit/pragmatic-drag-and-drop -- version and activity confirmed
- dnd-kit GitHub issues: https://github.com/clauderic/dnd-kit/issues/1654 -- React 19 compatibility tracked

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All core libraries already in the project; only adding @dnd-kit/react
- Architecture: HIGH - Migration pattern, handler pattern, and frontend structure are all extensions of established v1.0 patterns
- Pitfalls: HIGH - All pitfalls identified from direct code inspection of existing handlers and sync logic
- Drag-and-drop library: MEDIUM - @dnd-kit/react is the right choice but v0.3.2 is pre-1.0; React 19 compatibility issue exists but appears actively maintained. Fallback plan: use @dnd-kit/sortable (v10.0.0, stable) with @dnd-kit/core if @dnd-kit/react has issues, or simplify to dropdown-only if DnD proves problematic

**Research date:** 2026-03-15
**Valid until:** 2026-04-15 (stable domain, low churn)
