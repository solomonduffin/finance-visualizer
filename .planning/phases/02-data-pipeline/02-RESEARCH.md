# Phase 2: Data Pipeline - Research

**Researched:** 2026-03-15
**Domain:** SimpleFIN HTTP client, Go cron scheduler, SQLite upsert patterns, React settings UI
**Confidence:** HIGH (core stack confirmed against official docs; SimpleFIN rate limits MEDIUM — not formally specified)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Token Configuration**
- SimpleFIN access URL provided via a UI settings page (not environment variable)
- Stored plain text in the settings table — local-only DB behind password auth, encryption adds unnecessary complexity
- App starts and runs normally without a configured token — cron is a no-op, dashboard shows empty state
- User adds the access URL when ready via the settings page

**Settings UI**
- Settings page with access URL text input + save button
- Includes read-only status indicators: whether token is configured, last sync status
- "Sync now" button lives on the settings page alongside token config and status

**First Sync Behavior**
- First sync fires immediately when the access URL is saved — user sees data within seconds of configuring
- First sync pulls up to one month of historical balance snapshots (per requirements)

**Daily Cron Schedule**
- Cron hour configurable via SYNC_HOUR environment variable (e.g., SYNC_HOUR=6)
- Default to a sensible morning hour if not set
- Cron goroutine runs inside the Go process (no external cron dependency)

**Manual Sync**
- "Sync now" button on the settings page triggers an on-demand sync
- Should respect SimpleFIN rate limits internally

### Claude's Discretion
- Account type auto-detection / mapping from SimpleFIN metadata to schema enum (checking, savings, credit, investment, other)
- Error handling and retry strategy for failed account fetches
- Sync logging granularity in sync_log table
- SimpleFIN HTTP client implementation details
- Settings page styling and layout

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DATA-01 | App connects to SimpleFIN and fetches account data via read-only token | SimpleFIN protocol flow, custom HTTP client with Basic Auth from embedded access URL |
| DATA-02 | Daily cron goroutine fetches data automatically and stores snapshots in SQLite | Pure-Go ticker/time pattern with SYNC_HOUR env var, no external cron library needed |
| DATA-03 | First sync pulls up to one month of historical data from SimpleFIN | `start-date` Unix epoch query param on `/accounts` endpoint, 30 days back |
| DATA-04 | Each daily fetch creates append-only balance snapshots (one row per account per day) | `INSERT OR IGNORE` using existing `UNIQUE(account_id, balance_date)` constraint |
</phase_requirements>

---

## Summary

Phase 2 adds the core data pipeline: a custom SimpleFIN HTTP client, a daily cron goroutine, settings API endpoints, and a React settings page. The schema (accounts, balance_snapshots, sync_log, settings tables) is already fully migrated from Phase 1. This phase populates those tables.

The SimpleFIN protocol uses HTTP Basic Auth credentials embedded directly in the access URL. There is no external SDK needed — a ~60-80 line custom client handles token claiming, authenticated requests, and JSON decoding. The existing project constraint (no CGo, modernc.org/sqlite) means the client must use only stdlib net/http, which is correct.

The cron goroutine uses stdlib `time` only — no external library. A goroutine computes the duration until the next target hour each day, sleeps until then, fires, then resets. This is idiomatic Go and keeps the binary dependency-free. The goroutine accepts a context so it shuts down cleanly with the server.

**Primary recommendation:** Custom SimpleFIN client using stdlib `net/http` + `encoding/json`; pure stdlib scheduler using `time.Timer` with nightly recalculation; `INSERT OR IGNORE` for idempotent snapshot writes; React settings page with `useState`/`useEffect` following the existing Login page pattern.

---

## Standard Stack

### Core (all existing or stdlib — no new Go dependencies required)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `net/http` (stdlib) | Go 1.25 | SimpleFIN HTTP client | No CGo, no external dep, fully sufficient |
| `encoding/json` (stdlib) | Go 1.25 | Decode SimpleFIN JSON response | Already used throughout codebase |
| `encoding/base64` (stdlib) | Go 1.25 | Decode setup token to claim URL | Stdlib, one-liner |
| `time` (stdlib) | Go 1.25 | Daily cron scheduler goroutine | No external dep, handles SYNC_HOUR logic |
| `context` (stdlib) | Go 1.25 | Goroutine cancellation / graceful shutdown | Already used in db package |
| `shopspring/decimal` | already in go.sum (indirect) | Parse SimpleFIN balance strings | Prevents float64 precision loss; already decided |
| `modernc.org/sqlite` | v1.46.1 (existing) | SQLite writes | Existing CGo-free driver |
| React + Vitest | existing | Settings page + tests | Established in Phase 1 |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `log/slog` (stdlib) | Go 1.25 | Structured sync error logging | Already the project logger |
| `github.com/go-chi/chi/v5` | existing | New `/api/settings` and `/api/sync` routes | Already wired in router |
| `@testing-library/react` | existing | Settings page component tests | Already configured via vitest |

### Note: shopspring/decimal
`shopspring/decimal` appears in go.sum as an indirect dependency. It needs to be promoted to a direct dependency in go.mod:
```bash
go get github.com/shopspring/decimal
```
Use `decimal.NewFromString(balance)` to parse SimpleFIN balance strings before storing as TEXT.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| stdlib ticker | `robfig/cron` | cron is more expressive but overkill for a single daily job; adds a dependency; SYNC_HOUR pattern is simpler with stdlib |
| stdlib ticker | `go-co-op/gocron` | Same — unnecessary dependency for one job |
| custom client | `github.com/jazzboME/simplefin` | Pre-alpha (v0.1.2), no v1 stability, 7 transitive deps; custom client is ~60 lines and dependency-free |

---

## Architecture Patterns

### Recommended Package Structure

```
internal/
├── simplefin/          # SimpleFIN HTTP client + data types
│   ├── client.go       # Client struct: Claim(), FetchAccounts()
│   └── client_test.go  # httptest.Server mocks
├── sync/               # Sync orchestration: SyncOnce(), RunScheduler()
│   ├── sync.go         # SyncOnce writes accounts + snapshots + sync_log
│   └── sync_test.go    # Unit tests with temp DB
├── config/
│   └── config.go       # Add SyncHour int field + SYNC_HOUR env var
└── api/
    └── handlers/
        ├── settings.go  # GET/POST /api/settings, POST /api/sync/now
        └── settings_test.go
frontend/src/
├── pages/
│   └── Settings.tsx    # Settings page component
└── api/
    └── client.ts       # Add getSettings(), saveSettings(), triggerSync()
```

### Pattern 1: SimpleFIN Client — Claim + Fetch

The SimpleFIN protocol is a two-credential flow. A **setup token** is a Base64-encoded URL. You POST to the decoded URL to exchange it for an **access URL** (which contains embedded Basic Auth credentials). The access URL is stored in the settings table and reused for all future requests.

**Claiming the access URL (one-time):**
```go
// Source: https://beta-bridge.simplefin.org/info/developers
// Source: https://www.simplefin.org/protocol.html
func Claim(setupToken string) (string, error) {
    // 1. Base64-decode the token to get the claim URL
    claimURL, err := base64.StdEncoding.DecodeString(setupToken)
    if err != nil {
        return "", fmt.Errorf("simplefin: invalid setup token: %w", err)
    }

    // 2. POST to the claim URL with Content-Length: 0
    req, _ := http.NewRequest(http.MethodPost, string(claimURL), nil)
    req.Header.Set("Content-Length", "0")
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("simplefin: claim POST failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("simplefin: claim returned %d", resp.StatusCode)
    }

    // 3. Response body IS the access URL
    body, _ := io.ReadAll(resp.Body)
    return strings.TrimSpace(string(body)), nil
}
```

**Fetching accounts (daily):**
```go
// Source: https://www.simplefin.org/protocol.html
type AccountSet struct {
    Errors   []string  `json:"errors"`
    Accounts []Account `json:"accounts"`
}

type Account struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Currency    string `json:"currency"`
    Balance     string `json:"balance"`
    BalanceDate int64  `json:"balance-date"` // Unix epoch — when balance was current
    Org         struct {
        Name string `json:"name"`
        ID   string `json:"id"`
    } `json:"org"`
}

func FetchAccounts(accessURL string, startDate *time.Time) (*AccountSet, error) {
    reqURL, err := url.Parse(accessURL + "/accounts")
    if err != nil {
        return nil, err
    }

    q := reqURL.Query()
    q.Set("balances-only", "1") // skip transactions — not needed for dashboard
    if startDate != nil {
        q.Set("start-date", fmt.Sprintf("%d", startDate.Unix()))
    }
    reqURL.RawQuery = q.Encode()

    req, _ := http.NewRequest(http.MethodGet, reqURL.String(), nil)
    // Basic Auth credentials are embedded in the access URL itself
    if u := reqURL.User; u != nil {
        password, _ := u.Password()
        req.SetBasicAuth(u.Username(), password)
        // Strip credentials from URL path for the actual request
        reqURL.User = nil
        req.URL = reqURL
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("simplefin: GET /accounts failed: %w", err)
    }
    defer resp.Body.Close()

    var result AccountSet
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("simplefin: JSON decode failed: %w", err)
    }
    return &result, nil
}
```

**Critical detail — access URL credentials extraction:**
The access URL is in the form `https://user:password@host/simplefin`. Use `url.Parse()` to extract username and password, then call `req.SetBasicAuth()`. Clear `reqURL.User` before assigning back to `req.URL` so credentials aren't sent in the URL path.

### Pattern 2: Daily Cron Goroutine (stdlib only)

The goroutine recalculates the sleep duration on each iteration so it correctly targets the next occurrence of `SYNC_HOUR:00` local time.

```go
// Source: Go stdlib time package docs
func RunScheduler(ctx context.Context, syncHour int, db *sql.DB) {
    for {
        next := nextRunTime(syncHour)
        timer := time.NewTimer(time.Until(next))

        select {
        case <-ctx.Done():
            timer.Stop()
            return
        case <-timer.C:
            if err := SyncOnce(ctx, db); err != nil {
                slog.Error("scheduled sync failed", "error", err)
            }
        }
    }
}

func nextRunTime(hour int) time.Time {
    now := time.Now()
    next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.Local)
    if !next.After(now) {
        next = next.Add(24 * time.Hour)
    }
    return next
}
```

Start it from main.go after server startup:
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
go sync.RunScheduler(ctx, cfg.SyncHour, database)
```

### Pattern 3: Idempotent Snapshot Write (INSERT OR IGNORE)

The `UNIQUE(account_id, balance_date)` constraint in the existing schema enforces one-per-day semantics. Use `INSERT OR IGNORE` so duplicate fetches silently no-op:

```go
// Source: SQLite docs — INSERT OR IGNORE conflict clause
_, err = db.ExecContext(ctx,
    `INSERT OR IGNORE INTO balance_snapshots (account_id, balance, balance_date)
     VALUES (?, ?, ?)`,
    accountID,
    balance.String(), // shopspring/decimal → string
    balanceDate,      // DATE string "YYYY-MM-DD" derived from Unix epoch
)
```

`balance_date` derivation from SimpleFIN `balance-date` (Unix epoch int64):
```go
// balance-date is Unix epoch; convert to DATE string for the DB column
balanceDate := time.Unix(account.BalanceDate, 0).UTC().Format("2006-01-02")
```

### Pattern 4: First Sync — 30 Days of History

On first sync (immediately after saving the access URL), pass a `startDate` 30 days in the past:

```go
startDate := time.Now().UTC().AddDate(0, 0, -30)
accounts, err := simplefin.FetchAccounts(accessURL, &startDate)
```

For subsequent daily syncs, pass `nil` for `startDate` (SimpleFIN returns the current balance snapshot by default).

**Critical detail:** SimpleFIN does not store historical balances natively — the `balance-date` is when the current balance was captured. Passing `start-date` restricts *transactions*, not balance history. For balance snapshots, we create one row per sync run. The "first sync pulls one month of history" means the very first row is date-stamped with `balance-date` from the response (which may reflect an older capture if SimpleFIN hasn't updated recently). For daily snapshots going forward, each day's sync inserts today's row.

### Pattern 5: Sync Orchestration with Per-Account Error Isolation

```go
// Source: Phase requirements — sync failures for individual accounts must not abort the run
func SyncOnce(ctx context.Context, db *sql.DB) error {
    logID, err := insertSyncLog(ctx, db) // started_at = NOW()
    if err != nil {
        return err
    }

    accessURL := getAccessURL(ctx, db) // reads settings table
    if accessURL == "" {
        return nil // no-op if not configured
    }

    accounts, err := simplefin.FetchAccounts(accessURL, nil)
    fetched, failed := 0, 0

    for _, account := range accounts.Accounts {
        if err := upsertAccount(ctx, db, account); err != nil {
            slog.Error("failed to upsert account", "id", account.ID, "error", err)
            failed++
            continue
        }
        if err := insertSnapshot(ctx, db, account); err != nil {
            slog.Error("failed to insert snapshot", "id", account.ID, "error", err)
            failed++
            continue
        }
        fetched++
    }

    return finalizeSyncLog(ctx, db, logID, fetched, failed, err)
}
```

### Pattern 6: Account Type Mapping (Claude's Discretion)

SimpleFIN accounts have no explicit type field. Use keyword matching on account name as a best-effort heuristic:

```go
func inferAccountType(name string) string {
    lower := strings.ToLower(name)
    switch {
    case strings.Contains(lower, "saving"):
        return "savings"
    case strings.Contains(lower, "credit"), strings.Contains(lower, "card"):
        return "credit"
    case strings.Contains(lower, "invest"), strings.Contains(lower, "brokerage"),
         strings.Contains(lower, "ira"), strings.Contains(lower, "401"):
        return "investment"
    case strings.Contains(lower, "check"):
        return "checking"
    default:
        return "other"
    }
}
```

Use `INSERT OR REPLACE` for account upserts (account name/currency may change over time):
```go
_, err = db.ExecContext(ctx,
    `INSERT INTO accounts (id, name, account_type, currency, org_name, org_slug, updated_at)
     VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
     ON CONFLICT(id) DO UPDATE SET
       name=excluded.name, account_type=excluded.account_type,
       currency=excluded.currency, org_name=excluded.org_name,
       org_slug=excluded.org_slug, updated_at=CURRENT_TIMESTAMP`,
    account.ID, account.Name, inferAccountType(account.Name),
    account.Currency, account.Org.Name, account.Org.ID,
)
```

### Pattern 7: Settings API Endpoints

Follow the existing handler factory pattern (closure over `*sql.DB`):

```
GET  /api/settings      → {simplefin_configured: bool, last_sync_at: string|null, last_sync_status: string|null}
POST /api/settings      → {access_url: string} body; saves to settings table; triggers first sync if new
POST /api/sync/now      → triggers SyncOnce in background goroutine; returns {ok: true}
```

All three go behind JWT auth middleware (existing pattern from router.go).

### Pattern 8: Settings React Page

Follow the Login page pattern — controlled form with `useState`, API call in handler, error display:

```tsx
// New route in App.tsx after authenticated check
// Settings page has:
//   - Access URL text input (type="password" or text) with Save button
//   - Status badge: "Configured" / "Not configured"
//   - Last sync time + status from GET /api/settings
//   - "Sync Now" button — calls POST /api/sync/now
```

Add `react-router-dom` for navigation between Dashboard and Settings, or use a simple tab/state approach if routing is deferred to Phase 3.

**Routing note:** The CONTEXT.md does not specify whether to use react-router-dom. Given Phase 3 adds a dashboard page, it is appropriate to introduce react-router-dom now to avoid refactoring. However, if Claude's discretion prefers minimal change, a simple `page` state in App.tsx works for Phase 2 and can be migrated later.

### Anti-Patterns to Avoid

- **Storing credentials in the URL sent to the server** — extract `url.User` before issuing the HTTP GET request; otherwise Basic Auth appears in server logs
- **Using `float64` for balance parsing** — SimpleFIN returns `"1234.56"` as a string; always `decimal.NewFromString()` before storing
- **Global sync mutex omission** — the "Sync now" button and the cron goroutine can race; protect `SyncOnce` with a `sync.Mutex` or channel-based guard to prevent concurrent syncs
- **Blocking the HTTP handler on sync** — `POST /api/sync/now` should launch `SyncOnce` in a goroutine and immediately return `{ok: true}`; do not block the request handler waiting for sync completion
- **Fetching transactions on daily sync** — use `balances-only=1` query param; transactions are out of scope and increase response payload size significantly

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTP Basic Auth from embedded URL credentials | Manual string splitting on `:` | `url.Parse()` + `u.User.Password()` + `req.SetBasicAuth()` | stdlib handles URL encoding, edge cases |
| JSON decoding with field name mapping | Manual string manipulation | `encoding/json` with struct tags | Already in codebase, handles all cases |
| Idempotent snapshot insert | UPDATE + SELECT + conditional INSERT | `INSERT OR IGNORE` | Single-statement, atomic, uses existing UNIQUE constraint |
| Account upsert | DELETE + INSERT | `INSERT ... ON CONFLICT DO UPDATE` (SQLite upsert) | Atomic, preserves existing data on conflict |
| Daily scheduling | External cron daemon or system crontab | stdlib `time.Timer` goroutine | No external dependency; cleaner lifecycle management |

**Key insight:** The SQLite schema's `UNIQUE(account_id, balance_date)` constraint already encodes the "one snapshot per account per day" invariant. The application layer just needs `INSERT OR IGNORE` — the database enforces correctness.

---

## Common Pitfalls

### Pitfall 1: Setup Token vs Access URL Confusion
**What goes wrong:** Storing the user-provided setup token (Base64 string) instead of the claimed access URL. Subsequent sync calls fail because the setup token is one-time-use.
**Why it happens:** The distinction between setup token (user gives this to the app) and access URL (app fetches this by POSTing to the decoded token) is subtle.
**How to avoid:** The settings page should accept only a setup token from users, immediately claim it server-side (POST to decoded URL), then store the resulting access URL. Display only whether the access URL is configured — never show the raw URL to the user.
**Warning signs:** 403 responses on subsequent `/accounts` requests.

**Alternative user flow:** The protocol docs also describe apps accepting the access URL directly (already claimed elsewhere). Both flows are valid. The simpler approach for this app is to accept only the access URL (user visits SimpleFIN Bridge, claims it themselves, pastes the access URL). This avoids one round-trip and the one-time token storage problem. Decision is Claude's discretion per CONTEXT.md — **recommend accepting the access URL directly** (simpler, no server-side claim needed).

### Pitfall 2: Basic Auth Credentials Leak in HTTP Logs
**What goes wrong:** Passing the full access URL (with embedded credentials) to `http.NewRequest()` without stripping the `url.User` portion. Request logs expose `user:password@host` in plain text.
**How to avoid:** Always `url.Parse()` the access URL, extract `u.User` credentials, set `req.SetBasicAuth()`, then set `reqURL.User = nil` before assigning to the request.

### Pitfall 3: balance-date Is Not Today
**What goes wrong:** Assuming SimpleFIN's `balance-date` is today's date. It's "when the balance became what it is" — it may be a day old if the bank hasn't refreshed.
**How to avoid:** Derive `balance_date` from the `balance-date` field (convert Unix epoch to `YYYY-MM-DD`), not from `time.Now()`. The UNIQUE constraint means if two syncs return the same `balance-date`, only the first is stored (idempotent by design).

### Pitfall 4: Concurrent Sync Race Condition
**What goes wrong:** User clicks "Sync Now" while cron fires simultaneously; two SyncOnce goroutines write to sync_log and balance_snapshots concurrently. Even though SQLite has WAL+busy_timeout, double sync_log rows appear and confusion ensues.
**How to avoid:** Protect SyncOnce with a `sync.Mutex` or use a buffered channel of capacity 1 as a semaphore. If sync is already running, `POST /api/sync/now` returns `{ok: true, running: true}` without launching a second goroutine.

### Pitfall 5: SimpleFIN Rate Limits (MEDIUM confidence)
**What goes wrong:** Exceeding ~24 requests/day causes access tokens to be disabled by SimpleFIN Bridge.
**Source:** beta-bridge.simplefin.org developer guide — "24 requests or fewer per day" per account.
**How to avoid:** One sync per day (cron) + manual trigger = well within limits. Do not retry failed accounts in a tight loop. Log failures and move on.
**Warning signs:** HTTP 402 or 403 responses after previously successful syncs.

### Pitfall 6: App Fails to Start Without SimpleFIN Token
**What goes wrong:** Config validation rejects startup if `simplefin_access_url` is not set.
**How to avoid (locked decision):** The cron scheduler checks for a configured access URL each run and no-ops if absent. No validation at startup. Settings page shows "Not configured" state.

---

## Code Examples

### Read Settings Key from SQLite

```go
// Source: internal/api/handlers/auth.go — existing pattern
func getSettingValue(ctx context.Context, db *sql.DB, key string) (string, error) {
    var value string
    err := db.QueryRowContext(ctx,
        `SELECT value FROM settings WHERE key = ?`, key,
    ).Scan(&value)
    if err == sql.ErrNoRows {
        return "", nil // key not set
    }
    return value, err
}
```

### Upsert Setting Value

```go
_, err = db.ExecContext(ctx,
    `INSERT INTO settings (key, value) VALUES (?, ?)
     ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
    "simplefin_access_url", accessURL,
)
```

### Insert Sync Log Row

```go
// Insert at start; update at end
res, err := db.ExecContext(ctx,
    `INSERT INTO sync_log (started_at) VALUES (CURRENT_TIMESTAMP)`)
logID, _ := res.LastInsertId()
// ... sync work ...
_, err = db.ExecContext(ctx,
    `UPDATE sync_log SET finished_at=CURRENT_TIMESTAMP,
     accounts_fetched=?, accounts_failed=?, error_text=?
     WHERE id=?`,
    fetched, failed, errText, logID,
)
```

### decimal.NewFromString for Balance

```go
// Source: github.com/shopspring/decimal README
import "github.com/shopspring/decimal"

balance, err := decimal.NewFromString(account.Balance)
if err != nil {
    return fmt.Errorf("invalid balance %q: %w", account.Balance, err)
}
// Store as TEXT
balanceStr := balance.String() // "1234.56"
```

---

## Validation Architecture

`nyquist_validation` is `true` in `.planning/config.json` — this section is required.

### Test Framework

| Property | Value |
|----------|-------|
| Framework (Go) | `go test` (stdlib) |
| Framework (Frontend) | Vitest 3.2.1 |
| Go config file | none — stdlib test runner |
| Go quick run command | `go test ./internal/... -count=1 -timeout 30s` |
| Go full suite command | `go test ./... -count=1 -timeout 60s` |
| Frontend quick run | `cd frontend && npx vitest run --reporter=verbose` |
| Frontend full suite | `cd frontend && npx vitest run` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DATA-01 | SimpleFIN client fetches accounts from mock server | unit | `go test ./internal/simplefin/... -run TestFetchAccounts -v` | Wave 0 |
| DATA-01 | SimpleFIN client handles HTTP 403 / 402 errors | unit | `go test ./internal/simplefin/... -run TestFetchAccounts_Error -v` | Wave 0 |
| DATA-01 | Settings API GET returns configured/not-configured | unit | `go test ./internal/api/handlers/... -run TestGetSettings -v` | Wave 0 |
| DATA-01 | Settings API POST saves access URL to settings table | unit | `go test ./internal/api/handlers/... -run TestSaveSettings -v` | Wave 0 |
| DATA-02 | Scheduler fires SyncOnce at the configured hour | unit | `go test ./internal/sync/... -run TestNextRunTime -v` | Wave 0 |
| DATA-02 | POST /api/sync/now triggers SyncOnce | unit | `go test ./internal/api/handlers/... -run TestSyncNow -v` | Wave 0 |
| DATA-03 | SyncOnce passes start-date=30days-ago on first sync | unit | `go test ./internal/sync/... -run TestSyncOnce_FirstSync -v` | Wave 0 |
| DATA-04 | Duplicate snapshot insert is silently ignored | unit | `go test ./internal/sync/... -run TestInsertSnapshot_Duplicate -v` | Wave 0 |
| DATA-04 | Account upsert updates name/currency if changed | unit | `go test ./internal/sync/... -run TestUpsertAccount -v` | Wave 0 |
| DATA-01 | Settings React page renders token form + status | unit | `cd frontend && npx vitest run src/pages/Settings.test.tsx` | Wave 0 |
| DATA-01 | "Sync Now" button calls POST /api/sync/now | unit | `cd frontend && npx vitest run src/pages/Settings.test.tsx` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/... -count=1 -timeout 30s`
- **Per wave merge:** `go test ./... -count=1 && cd frontend && npx vitest run`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/simplefin/client_test.go` — covers DATA-01 (mock httptest.Server for /accounts)
- [ ] `internal/sync/sync_test.go` — covers DATA-02, DATA-03, DATA-04 (temp file DB)
- [ ] `internal/api/handlers/settings_test.go` — covers DATA-01 settings API
- [ ] `frontend/src/pages/Settings.test.tsx` — covers DATA-01 settings UI

No new test framework install needed — Go stdlib test runner and Vitest are already configured.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| External cron daemon | In-process Go goroutine with `time.Timer` | Go 1.0+ | No system dependency, clean context-based cancellation |
| `INSERT OR REPLACE` | `INSERT OR IGNORE` / `ON CONFLICT DO UPDATE` | SQLite 3.24.0 (2018) | Fine-grained conflict control; `INSERT OR IGNORE` preferred for snapshots, `ON CONFLICT DO UPDATE` for accounts |
| mattn/go-sqlite3 (CGo) | modernc.org/sqlite (pure Go) | Already decided in Phase 1 | CGo-free Docker builds |
| float64 for money | shopspring/decimal | Already decided pre-planning | Exact decimal arithmetic |

---

## Open Questions

1. **Setup token vs access URL in the UI**
   - What we know: The locked decision says "SimpleFIN access URL provided via settings page." This implies the user already has the access URL (claimed themselves from SimpleFIN Bridge).
   - What's unclear: Should the app also support one-click claiming from a setup token? The CONTEXT.md says "access URL" specifically.
   - Recommendation: Accept the access URL directly (simpler, matches locked decision wording). If user has only a setup token, they can claim it at beta-bridge.simplefin.org. No server-side claim logic needed.

2. **SimpleFIN rate limits under concurrent requests**
   - What we know: ~24 requests/day is the informal guidance from SimpleFIN Bridge developer docs (MEDIUM confidence — not formally specified in protocol).
   - What's unclear: Whether a single daily sync with a "Sync Now" button push counts as 1 or N requests (one per account or one total).
   - Recommendation: Treat each GET /accounts call as 1 request; the daily cron + occasional manual sync stays well within limits. No retry loops on failure.

3. **React routing approach**
   - What we know: Phase 3 adds a Dashboard page. Phase 2 adds a Settings page.
   - What's unclear: Introduce `react-router-dom` now or use simple state?
   - Recommendation: Introduce `react-router-dom` (v6) now to avoid refactoring in Phase 3. Low cost, established pattern.

---

## Sources

### Primary (HIGH confidence)
- [SimpleFIN Protocol Specification](https://www.simplefin.org/protocol.html) — complete API flow, response structure, query params, error codes, balance-date semantics
- [SimpleFIN Bridge Developer Guide](https://beta-bridge.simplefin.org/info/developers) — claim flow details, Content-Length: 0 requirement, rate limit guidance
- Go stdlib `net/http`, `encoding/json`, `encoding/base64`, `time`, `context` — language-level facts, Go 1.25
- Existing codebase: `internal/db/migrations/000001_init.up.sql` — confirmed schema (UNIQUE constraint, column names)
- Existing codebase: `internal/api/handlers/auth.go`, `router.go` — confirmed handler pattern, router structure

### Secondary (MEDIUM confidence)
- [jazzboME/simplefin Go package](https://pkg.go.dev/github.com/jazzboME/simplefin) — confirmed struct shape matches SimpleFIN spec; library not used (pre-stable), structs are reference
- SimpleFIN Bridge: ~24 requests/day rate limit — from developer guide prose, not formal spec

### Tertiary (LOW confidence)
- SimpleFIN rate limit specifics (exact count per day, per-account vs total) — informal guidance only, needs validation on real account if rate errors appear
- Account type inference by name keyword — heuristic; actual SimpleFIN accounts may have additional `extra` metadata not documented

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries are existing project deps or stdlib; no new library decisions
- Architecture: HIGH — SimpleFIN protocol formally documented; handler/router patterns directly observable in codebase
- SimpleFIN rate limits: MEDIUM — referenced in developer guide but not formally specified in protocol
- Account type mapping: LOW — inferred from schema constraints; SimpleFIN has no formal type field

**Research date:** 2026-03-15
**Valid until:** 2026-06-15 (SimpleFIN protocol is stable; Go stdlib is stable)
