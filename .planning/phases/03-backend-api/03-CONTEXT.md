# Phase 3: Backend API - Context

**Gathered:** 2026-03-15
**Status:** Ready for planning

<domain>
## Phase Boundary

All financial data in SQLite is accessible via a typed, authenticated REST API that the frontend can consume. Three endpoints: GET /api/summary (liquid/savings/investments totals), GET /api/accounts (grouped by type with balances), GET /api/balance-history (daily snapshot series per panel). All behind existing JWT auth middleware. No frontend changes in this phase — just the API contract.

</domain>

<decisions>
## Implementation Decisions

### Liquid Balance Calculation
- Liquid = sum(checking balances) minus sum(credit card balances owed)
- Use the raw `balance` field from SimpleFIN (already includes pending per prior decision)
- Credit card balances from SimpleFIN are negative (amount owed), so liquid = sum(checking) + sum(credit) effectively subtracts debt
- If no checking or credit accounts exist, liquid is 0

### "Other" Account Type Handling
- Accounts with type `other` are included in GET /api/accounts response (visible to the user)
- Accounts with type `other` are excluded from summary panel totals (liquid, savings, investments)
- This prevents data loss while keeping panel semantics clean

### Balance History Shape
- GET /api/balance-history returns per-panel aggregated daily totals, not per-account series
- Three series returned: liquid, savings, investments — each as an array of {date, balance} pairs
- Aggregation mirrors the summary calculation: liquid = sum(checking) + sum(credit) per day
- Optional `days` query parameter to limit range (default: all available history)
- Per-account drill-down is deferred to v2 (DRILL-01)

### Balance Representation in JSON
- All balance values returned as JSON strings (e.g., `"1234.56"`) to preserve decimal precision
- Consistent with DB storage format (TEXT column with shopspring/decimal)
- Frontend parses with `Number()` or a decimal library as needed

### Sync Metadata in Summary
- GET /api/summary includes a `last_synced_at` field (ISO 8601 timestamp from sync_log)
- Directly supports UX-01 (data freshness indicator) in Phase 4 without an extra endpoint
- If no sync has ever run, field is null

### Claude's Discretion
- Exact SQL query construction and optimization
- Whether to introduce a service/repository layer or keep queries in handlers
- Response DTO struct naming and organization
- Error message wording for edge cases (no accounts, no snapshots)
- Test data setup and fixture design
- Whether to batch-query or query per-type for summary calculations

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. Follow existing handler patterns (factory functions returning http.HandlerFunc, JSON responses, inline error handling).

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/api/router.go`: Chi router with JWT auth group — add new routes inside existing protected group
- `internal/api/handlers/`: Established handler package — add summary.go, accounts.go, history.go
- `internal/db/db.go`: SQLite connection with WAL mode — ready for read queries
- `internal/sync/sync.go`: `InferAccountType()` function defines the type mapping (checking/savings/credit/investment/other)
- `shopspring/decimal`: Already a dependency — use for balance arithmetic in aggregation

### Established Patterns
- Handler factory pattern: `func HandlerName(database *sql.DB) http.HandlerFunc`
- JSON responses with `json.NewEncoder(w).Encode()`
- Error responses: `http.Error(w, message, statusCode)` or `{"error":"message"}`
- Tests: `httptest.NewRequest` + `httptest.NewRecorder`, temp DB with migrations via `t.TempDir()`
- modernc.org/sqlite (CGo-free) — all queries must be compatible

### Integration Points
- Routes registered in `router.go` inside the `r.Group` that applies `jwtauth.Verifier` + `jwtauth.Authenticator`
- Queries hit `accounts` table (type grouping) and `balance_snapshots` table (balance data)
- `sync_log` table queried for `last_synced_at` (latest `finished_at` where `error_text IS NULL`)
- `balance_snapshots` index on `(account_id, balance_date DESC)` enables efficient latest-balance and history queries
- `UNIQUE(account_id, balance_date)` constraint guarantees one snapshot per account per day
- Account types constrained by CHECK: `('checking', 'savings', 'credit', 'investment', 'other')`

</code_context>

<deferred>
## Deferred Ideas

- Per-account balance history drill-down — v2 requirement (DRILL-01)
- Account APY display — v2 requirement (DRILL-02)
- Investment growth/loss calculations — v2 requirement (DRILL-03)

</deferred>

---

*Phase: 03-backend-api*
*Context gathered: 2026-03-15*
