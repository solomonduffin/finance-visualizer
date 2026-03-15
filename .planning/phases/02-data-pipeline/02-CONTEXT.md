# Phase 2: Data Pipeline - Context

**Gathered:** 2026-03-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Real financial account data flows from SimpleFIN into SQLite on a daily schedule with full history on first sync. Includes: SimpleFIN HTTP client, daily cron goroutine, append-only snapshot storage, settings UI for token configuration, and manual sync trigger. The schema already exists from Phase 1 (accounts, balance_snapshots, sync_log, settings tables).

</domain>

<decisions>
## Implementation Decisions

### Token Configuration
- SimpleFIN access URL provided via a UI settings page (not environment variable)
- Stored plain text in the settings table — local-only DB behind password auth, encryption adds unnecessary complexity
- App starts and runs normally without a configured token — cron is a no-op, dashboard shows empty state
- User adds the access URL when ready via the settings page

### Settings UI
- Settings page with access URL text input + save button
- Includes read-only status indicators: whether token is configured, last sync status
- "Sync now" button lives on the settings page alongside token config and status

### First Sync Behavior
- First sync fires immediately when the access URL is saved — user sees data within seconds of configuring
- First sync pulls up to one month of historical balance snapshots (per requirements)

### Daily Cron Schedule
- Cron hour configurable via SYNC_HOUR environment variable (e.g., SYNC_HOUR=6)
- Default to a sensible morning hour if not set
- Cron goroutine runs inside the Go process (no external cron dependency)

### Manual Sync
- "Sync now" button on the settings page triggers an on-demand sync
- Should respect SimpleFIN rate limits internally

### Claude's Discretion
- Account type auto-detection / mapping from SimpleFIN metadata to schema enum (checking, savings, credit, investment, other)
- Error handling and retry strategy for failed account fetches
- Sync logging granularity in sync_log table
- SimpleFIN HTTP client implementation details
- Settings page styling and layout

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/db/db.go`: SQLite connection with WAL mode, busy_timeout, foreign keys — ready for sync writes
- `internal/config/config.go`: Config struct loaded from env vars — extend with SYNC_HOUR
- `internal/api/router.go`: Chi router with auth middleware — add settings API endpoints
- `internal/auth/auth.go`: JWT middleware — settings endpoints go behind this

### Established Patterns
- modernc.org/sqlite (CGo-free) — custom SimpleFIN client must not introduce CGo deps
- Config from environment variables via `config.Load()`
- Test helpers use `t.TempDir()` temp file DB
- shopspring/decimal for financial values (balance stored as TEXT in schema)

### Integration Points
- Settings table: `key=simplefin_access_url` for token storage
- Accounts table: SimpleFIN account data maps to existing schema (id, name, account_type, currency, org_name, org_slug)
- Balance snapshots: UNIQUE(account_id, balance_date) prevents duplicate daily entries
- Sync log: Track each sync run with started_at, finished_at, accounts_fetched, accounts_failed, error_text
- Router: New `/api/settings` and `/api/sync` endpoints behind JWT auth
- Frontend: New settings page route in React app

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-data-pipeline*
*Context gathered: 2026-03-15*
