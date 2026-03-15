# Phase 3: Backend API - Research

**Researched:** 2026-03-15
**Domain:** Go REST API handlers, SQLite aggregation queries, shopspring/decimal arithmetic
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **Liquid balance formula:** sum(checking balances) + sum(credit balances). Credit balances from SimpleFIN are negative (amount owed), so adding them subtracts debt. If no checking or credit accounts exist, liquid is 0.
- **Balance field:** Use raw `balance` field from SimpleFIN — already includes pending.
- **"Other" account type:** Included in GET /api/accounts response; excluded from summary panel totals (liquid, savings, investments).
- **Balance history shape:** Per-panel aggregated daily totals only — three series (liquid, savings, investments), each `[{date, balance}]`. Per-account drill-down is deferred (DRILL-01).
- **Aggregation for history:** mirrors summary calculation — liquid = sum(checking) + sum(credit) per day.
- **Optional `days` query parameter:** on GET /api/balance-history to limit range (default: all available history).
- **Balance representation:** All balance values as JSON strings (e.g. `"1234.56"`) — preserves decimal precision, consistent with DB TEXT storage.
- **Sync metadata in summary:** GET /api/summary includes `last_synced_at` (ISO 8601 from sync_log latest `finished_at` WHERE `error_text IS NULL`). Null if no successful sync.
- **Handler pattern:** Factory functions returning `http.HandlerFunc`, matching existing handlers.
- **Existing patterns:** `json.NewEncoder(w).Encode()`, `http.Error(w, message, statusCode)`.
- **Route registration:** Inside `r.Group` that applies `jwtauth.Verifier` + `jwtauth.Authenticator` in router.go.

### Claude's Discretion

- Exact SQL query construction and optimization
- Whether to introduce a service/repository layer or keep queries in handlers
- Response DTO struct naming and organization
- Error message wording for edge cases (no accounts, no snapshots)
- Test data setup and fixture design
- Whether to batch-query or query per-type for summary calculations

### Deferred Ideas (OUT OF SCOPE)

- Per-account balance history drill-down — v2 requirement (DRILL-01)
- Account APY display — v2 requirement (DRILL-02)
- Investment growth/loss calculations — v2 requirement (DRILL-03)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DASH-01 | User sees liquid balance (checking minus credit card balances including pending) | GET /api/summary `liquid` field — SQL SUM by type with decimal arithmetic |
| DASH-02 | User sees total savings across all savings accounts | GET /api/summary `savings` field — SQL SUM WHERE account_type='savings' |
| DASH-03 | User sees total investments (brokerage + retirement + crypto) | GET /api/summary `investments` field — SQL SUM WHERE account_type='investment' |
| DASH-04 | User sees individual account list with balances under each panel | GET /api/accounts — accounts grouped by type with latest snapshot balance per account |
</phase_requirements>

## Summary

Phase 3 implements three read-only JSON endpoints that surface data already in SQLite: `/api/summary` (panel totals + freshness), `/api/accounts` (grouped account list), and `/api/balance-history` (time-series per panel). All endpoints live inside the existing JWT-protected route group — the auth story is entirely solved by the current middleware.

The implementation is pure Go SQL + JSON encoding against an established, well-tested pattern. No new libraries are needed. The main technical work is: (1) writing correct aggregation SQL with shopspring/decimal for safe balance arithmetic, (2) constructing the right response structs, and (3) covering all edge cases (no data, zero balances, empty history) with tests that use the existing `setupTestDB` helper pattern.

The main pitfall domain is SQL aggregation against the two-table join (`accounts` + `balance_snapshots`). Getting the "latest snapshot per account" sub-query right, and the per-day panel aggregation for history, are the queries that need care. Everything else follows the established codebase rhythm.

**Primary recommendation:** Keep queries inline in handlers — no service/repository layer needed at this scale. Three handler files, one DTO block each. Use a single SQL query with GROUP BY account_type for summary, a correlated sub-query or window function for latest-balance in accounts list, and a date-aggregating GROUP BY for history.

## Standard Stack

### Core (all already in go.mod — zero new dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `database/sql` | stdlib | SQL queries | Already the project DB interface |
| `encoding/json` | stdlib | Response encoding | Matches all existing handlers |
| `github.com/shopspring/decimal` | v1.4.0 | Balance arithmetic/formatting | Already a direct dependency; prevents float64 precision loss |
| `modernc.org/sqlite` | v1.46.1 | Query execution | CGo-free driver; already in use |
| `github.com/go-chi/chi/v5` | v5.2.5 | Route registration | Already the router |
| `github.com/go-chi/jwtauth/v5` | v5.4.0 | JWT middleware | Already applied to protected group |

**No new packages to install.** All dependencies are already present in go.mod.

## Architecture Patterns

### Recommended Project Structure

Three new files in the existing handlers package; one new registration block in router.go:

```
internal/api/handlers/
├── auth.go             # existing
├── auth_test.go        # existing
├── health.go           # existing
├── health_test.go      # existing
├── settings.go         # existing
├── settings_test.go    # existing
├── summary.go          # NEW — GET /api/summary
├── summary_test.go     # NEW
├── accounts.go         # NEW — GET /api/accounts
├── accounts_test.go    # NEW
├── history.go          # NEW — GET /api/balance-history
└── history_test.go     # NEW
internal/api/
└── router.go           # MODIFIED — add 3 routes to protected group
```

No service/repository layer. Queries stay in handlers — the data model is simple and the handler logic is thin.

### Pattern 1: Handler Factory (existing pattern to follow exactly)

```go
// Source: internal/api/handlers/settings.go (existing)
func GetSummary(database *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ... query db using r.Context() ...
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp) //nolint:errcheck
    }
}
```

### Pattern 2: Response DTO Structs

Define per-file, unexported from the package (consistent with existing settings.go pattern):

```go
// summary.go
type summaryResponse struct {
    Liquid       string  `json:"liquid"`
    Savings      string  `json:"savings"`
    Investments  string  `json:"investments"`
    LastSyncedAt *string `json:"last_synced_at"`
}

// accounts.go
type accountItem struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Type     string `json:"type"`
    Balance  string `json:"balance"`
    Currency string `json:"currency"`
    OrgName  string `json:"org_name,omitempty"`
}
type accountsResponse struct {
    Checking    []accountItem `json:"checking"`
    Savings     []accountItem `json:"savings"`
    Credit      []accountItem `json:"credit"`
    Investments []accountItem `json:"investments"`
    Other       []accountItem `json:"other"`
}

// history.go
type balancePoint struct {
    Date    string `json:"date"`
    Balance string `json:"balance"`
}
type historyResponse struct {
    Liquid      []balancePoint `json:"liquid"`
    Savings     []balancePoint `json:"savings"`
    Investments []balancePoint `json:"investments"`
}
```

Note: All balance fields are `string`, not `float64`, per the locked decision.

### Pattern 3: SQL — Summary Aggregation

Aggregate balances for each type in a single pass using `GROUP BY`. Use shopspring/decimal for arithmetic after scanning:

```go
// Query all types' latest snapshot sums in one shot
rows, err := database.QueryContext(r.Context(), `
    SELECT a.account_type, SUM(CAST(bs.balance AS REAL))
    FROM accounts a
    JOIN balance_snapshots bs ON bs.account_id = a.id
    WHERE bs.balance_date = (
        SELECT MAX(bs2.balance_date)
        FROM balance_snapshots bs2
        WHERE bs2.account_id = a.id
    )
    GROUP BY a.account_type
`)
```

**IMPORTANT:** Do not use `CAST(balance AS REAL)` for final output values — that would introduce float64 imprecision. Use it only to produce a Go `float64` for intermediate scanning, then reconstruct with shopspring/decimal:

```go
// Better: scan balance as string from a subquery
rows, err := database.QueryContext(r.Context(), `
    SELECT a.account_type, bs.balance
    FROM accounts a
    JOIN balance_snapshots bs ON bs.account_id = a.id
    WHERE bs.balance_date = (
        SELECT MAX(bs2.balance_date)
        FROM balance_snapshots bs2
        WHERE bs2.account_id = a.id
    )
`)
// Then range rows and accumulate using decimal.Add()
```

### Pattern 4: SQL — Latest Balance Per Account (for /api/accounts)

Use a correlated subquery (compatible with all SQLite versions — no window functions needed):

```go
`SELECT a.id, a.name, a.account_type, a.currency, a.org_name,
        bs.balance
 FROM accounts a
 LEFT JOIN balance_snapshots bs ON bs.account_id = a.id
   AND bs.balance_date = (
       SELECT MAX(bs2.balance_date)
       FROM balance_snapshots bs2
       WHERE bs2.account_id = a.id
   )
 ORDER BY a.account_type, a.name`
```

`LEFT JOIN` is correct — accounts may exist before their first snapshot (edge case but valid).

### Pattern 5: SQL — Per-Panel History Aggregation

For each day, sum balances by type. The `days` parameter maps to a WHERE clause filter:

```go
query := `
    SELECT bs.balance_date,
           a.account_type,
           bs.balance
    FROM balance_snapshots bs
    JOIN accounts a ON a.id = bs.account_id
    WHERE a.account_type IN ('checking', 'credit', 'savings', 'investment')
`
if days > 0 {
    query += fmt.Sprintf(
        " AND bs.balance_date >= date('now', '-%d days')", days,
    )
}
query += " ORDER BY bs.balance_date ASC, a.account_type"
```

Then, in Go: group by date, accumulate per-panel sums using decimal, and build the three `[]balancePoint` slices.

**Critical:** The liquid aggregation for history must match the summary formula: liquid_on_day = sum(checking on that day) + sum(credit on that day).

### Pattern 6: Route Registration

```go
// router.go — inside the existing protected r.Group
r.Get("/api/summary", handlers.GetSummary(database))
r.Get("/api/accounts", handlers.GetAccounts(database))
r.Get("/api/balance-history", handlers.GetBalanceHistory(database))
```

### Pattern 7: last_synced_at Query

```go
var lastSyncedAt sql.NullString
err = database.QueryRowContext(r.Context(),
    `SELECT finished_at FROM sync_log
     WHERE error_text IS NULL AND finished_at IS NOT NULL
     ORDER BY id DESC LIMIT 1`,
).Scan(&lastSyncedAt)
```

If `lastSyncedAt.Valid`, set the string pointer; otherwise leave it nil (marshals to JSON null).

### Anti-Patterns to Avoid

- **Float64 for balances:** Never scan balance TEXT into a `float64` field for output — use shopspring/decimal arithmetic and `.String()` for output. It's fine to use float64 only when SQLite aggregation is the only arithmetic needed (e.g. for a COUNT, not balances).
- **Nil slice vs empty slice:** Initialize all slice fields to `[]accountItem{}` (not nil) before populating — JSON encodes nil slices as `null`, not `[]`. Return empty arrays, not null, when there's no data.
- **Context omission:** Always pass `r.Context()` to every `QueryContext`/`QueryRowContext` call — consistent with all existing handlers.
- **Missing Content-Type header:** Always set `w.Header().Set("Content-Type", "application/json")` before encoding — omission works but is inconsistent.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Decimal arithmetic on balances | Custom string parsing + float math | `shopspring/decimal` (already imported) | Float64 has binary precision errors; decimal.Add/Sub are exact |
| JWT validation | Token parsing | `jwtauth.Verifier` + `jwtauth.Authenticator` middleware | Already applied to the protected group — no per-handler work needed |
| SQL migrations | Schema evolution in handlers | `db.Migrate()` | No schema changes needed in this phase |
| Date filtering | String manipulation | SQLite `date('now', '-N days')` function | Correct, timezone-aware, no Go date manipulation required |

**Key insight:** Auth is a free pass — the middleware rejects unauthenticated requests with 401 before handlers run. Handlers never need to check token validity.

## Common Pitfalls

### Pitfall 1: Null Balance from LEFT JOIN on Accounts Without Snapshots

**What goes wrong:** `LEFT JOIN` on `balance_snapshots` returns NULL balance for accounts that have been upserted but not yet snapshotted (possible on first save before sync completes). Scanning a NULL string panics or returns an error.

**Why it happens:** `processAccount` in sync.go upserts accounts and inserts snapshots in two separate steps; a crash between them leaves an account with no snapshot.

**How to avoid:** Use `sql.NullString` for balance scan, default to `"0"` if null:
```go
var balanceNull sql.NullString
// ... scan ...
balance := "0"
if balanceNull.Valid {
    balance = balanceNull.String
}
```

**Warning signs:** Test panics with "sql: Scan error on column index N, name 'balance': converting NULL to string is unsupported."

### Pitfall 2: Empty History Returns Null Panels Instead of Empty Arrays

**What goes wrong:** If `[]balancePoint` fields in historyResponse are never assigned and remain nil, `json.Marshal` encodes them as `null` rather than `[]`.

**Why it happens:** Go nil slice vs empty slice JSON encoding difference.

**How to avoid:** Initialize all panel slices before the query loop:
```go
resp := historyResponse{
    Liquid:      []balancePoint{},
    Savings:     []balancePoint{},
    Investments: []balancePoint{},
}
```

**Warning signs:** Frontend receives `{"liquid":null}` instead of `{"liquid":[]}`.

### Pitfall 3: Liquid History Calculation Misses Days Where Only One Type Has Data

**What goes wrong:** If a user has checking accounts but no credit cards, the liquid panel should still show checking balances. If aggregation only processes days where BOTH types have snapshots, gaps appear.

**Why it happens:** Over-constraining the WHERE clause or joining checking + credit instead of scanning all types and computing in Go.

**How to avoid:** Scan all relevant types for each day, then compute liquid in Go: `liquid = sumChecking + sumCredit`. Missing types default to decimal zero — not an error.

### Pitfall 4: `days` Parameter Parsing Without Validation

**What goes wrong:** `strconv.Atoi` on a malicious `?days=; DROP TABLE` doesn't actually inject SQL when substituted via `fmt.Sprintf`, but negative or zero values can break the date arithmetic.

**Why it happens:** Trusting URL parameters.

**How to avoid:** Validate `days` is a positive integer before using; ignore/default on invalid input:
```go
days := 0
if dStr := r.URL.Query().Get("days"); dStr != "" {
    if d, err := strconv.Atoi(dStr); err == nil && d > 0 {
        days = d
    }
}
```

### Pitfall 5: Forgetting to Set Content-Type Before Writing Status Code

**What goes wrong:** `w.WriteHeader(http.StatusOK)` (if called explicitly) before `w.Header().Set(...)` means headers are already sent — the Content-Type won't be set.

**Why it happens:** Calling WriteHeader before setting headers.

**How to avoid:** Follow the established pattern: set headers, then encode. The `json.NewEncoder(w).Encode()` pattern implicitly sends 200, so explicit `WriteHeader(200)` is unnecessary and can be omitted.

## Code Examples

Verified patterns from existing codebase:

### Test Helper Pattern (use setupTestDB from auth_test.go)

```go
// Source: internal/api/handlers/auth_test.go
func setupTestDB(t *testing.T, password string) *sql.DB {
    t.Helper()
    dir := t.TempDir()
    dbPath := filepath.Join(dir, "test.db")
    database, err := db.Open(dbPath)
    // ...
    if err := db.Migrate(dbPath); err != nil { ... }
    t.Cleanup(func() { database.Close() })
    return database
}
```

New tests for summary/accounts/history should define a similar helper (or a shared `setupFinanceDB` that also inserts account + snapshot seed data).

### Seeding Test Data

```go
// Insert test accounts
_, _ = database.Exec(`INSERT INTO accounts(id, name, account_type, currency)
    VALUES ('chk-1', 'Checking', 'checking', 'USD'),
           ('sav-1', 'Savings',  'savings',  'USD'),
           ('crd-1', 'Credit',   'credit',   'USD')`)

// Insert today's snapshots
_, _ = database.Exec(`INSERT INTO balance_snapshots(account_id, balance, balance_date)
    VALUES ('chk-1', '1000.00', date('now')),
           ('sav-1', '5000.00', date('now')),
           ('crd-1', '-200.00', date('now'))`)
// Expected liquid = 1000.00 + (-200.00) = 800.00
```

### shopspring/decimal Accumulation

```go
// Source: go.mod — shopspring/decimal v1.4.0 already a direct dependency
import "github.com/shopspring/decimal"

liquid := decimal.Zero
// ... for each checking row: liquid = liquid.Add(d)
// ... for each credit row:   liquid = liquid.Add(d)  // credit balances already negative
result := liquid.StringFixed(2)  // "800.00"
```

### Error Response (existing pattern)

```go
// Source: internal/api/handlers/settings.go
http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
```

## State of the Art

| Old Approach | Current Approach | Notes |
|--------------|------------------|-------|
| `mattn/go-sqlite3` (CGo) | `modernc.org/sqlite` (CGo-free) | Already decided — all queries must be SQLite-compatible |
| `float64` for money | `shopspring/decimal` | Already in go.mod as direct dependency |
| Window functions (SQLite 3.25+) | Correlated sub-queries | Safer — avoids SQLite version uncertainty in Docker image |

## Open Questions

1. **Credit balance sign convention in history aggregation**
   - What we know: Summary formula is sum(checking) + sum(credit); credit values are negative in the DB.
   - What's unclear: Does every historical credit snapshot correctly carry a negative balance, or could early data have positive values?
   - Recommendation: Test with explicitly negative credit balances in test fixtures. Document the sign expectation in a comment.

2. **Accounts with no snapshots in /api/accounts response**
   - What we know: LEFT JOIN returns NULL balance; we default to "0".
   - What's unclear: Whether the user wants to see accounts with "0" or filter them out.
   - Recommendation: Show them with "0" balance — the account exists and the user should know it's there. This aligns with CONTEXT.md "visible to the user" intent for `other` type.

3. **`last_synced_at` ISO 8601 format from SQLite**
   - What we know: `CURRENT_TIMESTAMP` in SQLite stores as `"YYYY-MM-DD HH:MM:SS"` (space separator, no Z).
   - What's unclear: Whether the frontend needs strict ISO 8601 with T separator and Z.
   - Recommendation: Return the raw SQLite timestamp string in Phase 3; Phase 4 can add formatting if needed. Document the format in a comment.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` stdlib + `net/http/httptest` |
| Config file | none — `go test ./internal/...` is the runner |
| Quick run command | `go test ./internal/api/handlers/... -run TestSummary\|TestAccounts\|TestHistory -v` |
| Full suite command | `go test ./internal/...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DASH-01 | GET /api/summary returns correct liquid balance | unit | `go test ./internal/api/handlers/... -run TestGetSummary -v` | Wave 0 |
| DASH-01 | GET /api/summary returns 401 without JWT | unit | `go test ./internal/api/... -run TestSummaryRoute_NoAuth -v` | Wave 0 |
| DASH-02 | GET /api/summary returns correct savings total | unit | `go test ./internal/api/handlers/... -run TestGetSummary -v` | Wave 0 |
| DASH-03 | GET /api/summary returns correct investments total | unit | `go test ./internal/api/handlers/... -run TestGetSummary -v` | Wave 0 |
| DASH-04 | GET /api/accounts returns accounts grouped by type | unit | `go test ./internal/api/handlers/... -run TestGetAccounts -v` | Wave 0 |
| DASH-04 | GET /api/accounts returns 401 without JWT | unit | `go test ./internal/api/... -run TestAccountsRoute_NoAuth -v` | Wave 0 |
| (all) | GET /api/balance-history returns per-panel series | unit | `go test ./internal/api/handlers/... -run TestGetBalanceHistory -v` | Wave 0 |
| (all) | GET /api/balance-history returns 401 without JWT | unit | `go test ./internal/api/... -run TestHistoryRoute_NoAuth -v` | Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/api/handlers/... -v`
- **Per wave merge:** `go test ./internal/...`
- **Phase gate:** `go test ./internal/...` green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/api/handlers/summary.go` + `summary_test.go` — covers DASH-01, DASH-02, DASH-03
- [ ] `internal/api/handlers/accounts.go` + `accounts_test.go` — covers DASH-04
- [ ] `internal/api/handlers/history.go` + `history_test.go` — covers VIZ-01 data layer
- [ ] Route registrations in `internal/api/router.go` (and corresponding router_test.go coverage)

No new test framework install needed — all test infrastructure exists.

## Sources

### Primary (HIGH confidence)

- Codebase direct read: `internal/api/router.go` — route registration pattern
- Codebase direct read: `internal/api/handlers/settings.go` — handler factory + JSON encode pattern
- Codebase direct read: `internal/api/handlers/auth_test.go` — test helper and DB setup pattern
- Codebase direct read: `internal/db/migrations/000001_init.up.sql` — exact schema, column names, types, indexes
- Codebase direct read: `internal/sync/sync.go` — `InferAccountType`, balance sign conventions, snapshot insert
- Codebase direct read: `go.mod` — confirmed all dependencies already present

### Secondary (MEDIUM confidence)

- shopspring/decimal documentation: `.String()`, `.StringFixed(N)`, `.Add()`, `.Zero` — standard API, no breaking changes in v1.x

### Tertiary (LOW confidence)

- SQLite correlated subquery vs window function performance — correlated subquery is correct and safe; window functions would also work but add SQLite version risk

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies; all in go.mod
- Architecture: HIGH — derived directly from existing codebase patterns, not speculation
- SQL queries: HIGH — derived from known schema (exact column names verified)
- Pitfalls: HIGH — derived from codebase reading (null handling, slice initialization)
- Validation architecture: HIGH — existing `go test ./internal/...` infrastructure confirmed passing

**Research date:** 2026-03-15
**Valid until:** 2026-04-14 (stable stack; only risk is schema change, which is unlikely)
