# Pitfalls Research

**Domain:** Adding 7 features to existing Go/React/SQLite personal finance dashboard (v1.1)
**Researched:** 2026-03-15
**Confidence:** HIGH (based on direct codebase analysis of all Go handlers, sync logic, schema, and frontend components + domain research with web search verification)

---

## Critical Pitfalls

### Pitfall 1: Stale Account Deletion Destroys Renamed Account Data and Configurations

**What goes wrong:**
The existing `removeStaleAccounts()` in `internal/sync/sync.go` (lines 198-222) hard-deletes any account not returned by the latest SimpleFIN fetch, including all its `balance_snapshots`. When v1.1 adds user-owned metadata -- display names, APY settings, alert rules, aggregation groups -- a temporary SimpleFIN outage or token refresh causes permanent loss of all that configuration. The account reappears on the next successful sync as a fresh entry with no history, no custom name, no alert rules, and no projection settings.

**Why it happens:**
The v1.0 design correctly treated SimpleFIN as the sole source of truth for account existence -- accounts had no user-owned state. But v1.1 introduces at least 4 types of user-owned per-account data (display_name, aggregation group membership, APY/growth rate, alert rule references). The existing delete-and-recreate model is incompatible with any user-owned state.

**How to avoid:**
- Replace hard-delete with soft-delete: add a `hidden_at DATETIME` column to `accounts`. When an account disappears from SimpleFIN, set `hidden_at = CURRENT_TIMESTAMP` instead of deleting rows.
- If a hidden account reappears in SimpleFIN on a subsequent sync, automatically clear `hidden_at` -- the display_name, alert rules, and all history are preserved.
- Filter hidden accounts from dashboard queries with `WHERE hidden_at IS NULL`, but keep them visible in a "Hidden Accounts" section in settings for manual permanent deletion.
- Cascade concern: every new table that references `accounts.id` (alert_rules, account_settings, account_groups) must NOT use `ON DELETE CASCADE` with the current hard-delete behavior, or it will silently destroy user config. Soft-delete avoids this entirely.

**Warning signs:**
- Test: disconnect an institution in SimpleFIN Bridge, run sync, reconnect, sync again -- is the display_name intact? Is historical data preserved?
- If any migration adds `ON DELETE CASCADE` to a foreign key referencing `accounts.id` without first switching to soft-delete, user data is at risk.

**Phase to address:**
Phase 1 (Account Renaming / first migration) -- this must be the very first schema change because every subsequent feature that stores per-account config depends on accounts surviving SimpleFIN volatility.

---

### Pitfall 2: Expression Evaluation Injection in Alert Rules

**What goes wrong:**
The alert rules feature needs users to define conditions like `investments > 100000`. If the expression evaluator uses a general-purpose interpreter (e.g., `goja`, `otto`, `expr`, or `govaluate` without restrictions), a crafted expression could access Go runtime internals, read environment variables (which contain `JWT_SECRET`, `PASSWORD_HASH`), or cause denial of service. CVE-2025-68613 (CVSS 9.9) in n8n demonstrates exactly this: expression evaluation that escaped its sandbox led to full RCE with process privileges, affecting 100,000+ instances.

**Why it happens:**
Developers reach for flexible expression libraries because they want expressive user-facing rules. But this project's alert conditions are structurally simple: compare a balance metric against a numeric threshold. A full expression engine is massive overkill and introduces unnecessary attack surface -- even in a single-user app, the environment contains secrets that should not be exfiltrable.

**How to avoid:**
- Do NOT import a general-purpose expression engine. Build a restricted DSL with a finite grammar.
- Valid left-hand operands: panel keys (`liquid`, `savings`, `investments`, `net_worth`) and specific account references by ID.
- Valid operators: `>`, `<`, `>=`, `<=`, `==`, `!=`.
- Right-hand side: always a numeric literal.
- Store as structured JSON in the database: `{"metric": "investments", "operator": ">", "value": 100000}`. Never store free-text expression strings.
- Evaluate by looking up the current value from the database, then doing a `shopspring/decimal` comparison. No interpreter or `eval()` needed.
- On the frontend, use a structured form (dropdown for metric, dropdown for operator, numeric input for value) that generates the JSON. Never accept free-text expressions.

**Warning signs:**
- If the codebase imports `goja`, `otto`, or any JavaScript runtime for alert evaluation, it is overengineered.
- If the alert expression is passed to any `Eval()` function, there is an injection vector.

**Phase to address:**
Phase 5 (Alert Rules) -- the expression storage format must be decided at schema design time, before any alert data is persisted.

---

### Pitfall 3: Alert Flooding from Threshold Oscillation

**What goes wrong:**
A checking balance that hovers around a $5,000 alert threshold (oscillating between $4,900 and $5,100 due to pending transactions) triggers an email on every sync cycle where the condition is true. With daily sync, that is a daily email. If the user triggers manual syncs, it gets worse. The user either disables alerts entirely or ignores them (alert fatigue), defeating the feature's purpose.

**Why it happens:**
Naive threshold checking (`if currentValue > threshold then sendEmail`) has no memory of prior state. Every evaluation is independent, so the system cannot distinguish "just crossed the threshold" from "still above the threshold from yesterday."

**How to avoid:**
- Implement a state machine per alert rule with three states: `NORMAL`, `TRIGGERED`, `NOTIFIED`.
  - `NORMAL` -> condition becomes true -> send "threshold crossed" email -> transition to `NOTIFIED`.
  - `NOTIFIED` -> condition still true -> do nothing (no repeat emails).
  - `NOTIFIED` -> condition becomes false -> send "recovered" email -> transition to `NORMAL`.
- This exactly matches the "fire once on cross, once on recovery" requirement in PROJECT.md.
- Store the current state in an `alert_state TEXT DEFAULT 'NORMAL'` column on the alert rules table.
- Add `last_notified_at DATETIME` and enforce a minimum cooldown (e.g., 1 hour) as a safety net.
- Edge case: do NOT send a recovery email if the alert has never fired. Initial state should be determined by evaluating the condition once without sending, to establish the baseline.

**Warning signs:**
- If the alert evaluation code does not read the previous alert state before deciding whether to send, it will flood.
- Test: set threshold exactly at current balance, run sync twice -- must NOT send two emails.

**Phase to address:**
Phase 5 (Alert Rules) -- the state machine must be part of the initial alert schema, not retrofitted after users complain about flooding.

---

### Pitfall 4: Sync Error Messages Leaking SimpleFIN Credentials to Frontend

**What goes wrong:**
The sync diagnostics feature will expose `sync_log.error_text` in the settings UI. The existing `SyncOnce()` stores raw error messages from Go's error wrapping chain. While `client.go` (lines 108-109) correctly strips credentials from the request URL before making HTTP calls, the raw access URL is read from the settings table at sync.go line 72-77, and error paths could include it. For example, a database error during the settings read, or an error message from `url.Parse()` that includes the full URL with credentials, would end up in `sync_log.error_text` and then displayed in the browser.

**Why it happens:**
Go's `fmt.Errorf("context: %w", err)` preserves the full error chain. Any error that includes the access URL (which contains embedded `user:password@host` Basic Auth) in its string representation will propagate credentials into the sync log. The current code does not sanitize errors before storage.

**How to avoid:**
- Write a `sanitizeError(msg string) string` function that strips URL credentials: replace patterns matching `https?://[^@]+@` with `https://***@`.
- Apply this sanitization in the `finalize()` helper (sync.go line 96-104) before writing to `sync_log.error_text`.
- In the new sync diagnostics API endpoint, return structured error categories ("Connection failed", "Authentication failed", "Data parse error") rather than raw error text. Map common error patterns to user-friendly messages server-side.
- Audit every `fmt.Errorf` and `slog` call in `sync.go` and `client.go` to ensure `accessURL` never appears in error messages.

**Warning signs:**
- Search: any `slog.Error` or `fmt.Errorf` that includes `accessURL` as a value.
- Test: set an intentionally broken SimpleFIN URL with credentials, trigger sync, check what `sync_log.error_text` contains.

**Phase to address:**
Phase 4 (Sync Diagnostics) -- must be solved before any sync_log data is exposed to the frontend.

---

### Pitfall 5: Crypto Aggregation Incorrectly Merges Non-Crypto Accounts at Same Institution

**What goes wrong:**
The crypto aggregation feature groups accounts by `org_name` to sum sub-accounts (e.g., 4 Coinbase wallets -> 1 line). But `org_name` is the institution name, not the account type. A user with a Coinbase crypto wallet AND a Coinbase USD checking account (Coinbase offers banking services) sees them incorrectly merged into one aggregated balance.

**Why it happens:**
The aggregation logic treats `org_name` as a proxy for "accounts that should be grouped." But `org_name` identifies the institution, not the product. The existing `InferAccountType()` already classifies accounts into `checking`, `savings`, `credit`, `investment`, `other` -- but a naive `GROUP BY org_name` ignores this.

**How to avoid:**
- Aggregation must respect `account_type` boundaries: group by `(org_slug, account_type)`, not just `org_name` or `org_slug`.
- Make aggregation opt-in, not automatic. Provide a UI in settings where the user explicitly selects which accounts to aggregate into a group. Store groups as a dedicated `account_groups` table with member account IDs.
- Display the aggregated line with sub-account count: "Coinbase (4 wallets)" so the user can verify correctness.
- If automatic grouping is desired as a default, present it as a suggestion that the user confirms, not a fait accompli.

**Warning signs:**
- If the aggregation SQL is `GROUP BY org_name` without also considering `account_type`, it is wrong.
- Test: create test data with two accounts at the same org but different types (e.g., "Coinbase" investment + "Coinbase" checking) -- verify they are NOT merged.

**Phase to address:**
Phase 1 (Crypto Aggregation) -- the grouping logic must be correct from the start since it directly affects how balances display on the dashboard.

---

### Pitfall 6: org_name Instability Breaking Aggregation Groups Over Time

**What goes wrong:**
SimpleFIN's `org.name` is a human-readable string that can change between syncs. If SimpleFIN Bridge updates how it labels an institution (e.g., "Coinbase" becomes "Coinbase, Inc." or "Coinbase Global"), aggregation grouping based on `org_name` silently breaks: previously aggregated accounts split into two groups, or one group disappears and another appears.

The existing `processAccount()` upsert (sync.go lines 169-179) overwrites `org_name` on every sync via `ON CONFLICT(id) DO UPDATE SET org_name=excluded.org_name`, so the change propagates immediately.

**Why it happens:**
`org.name` is a display label, not a stable identifier. The SimpleFIN protocol also provides `org.id` (stored as `org_slug` in the current schema), which is a domain-like identifier (e.g., `coinbase.com`) that is more stable. Developers naturally reach for the human-readable name for grouping because it is what the user sees.

**How to avoid:**
- Use `org_slug` (the `org.id` / domain from SimpleFIN) as the grouping key in all aggregation logic. Use the latest `org_name` as the display label.
- Store aggregation group configuration keyed by `org_slug`, not `org_name`.
- Handle NULL `org_slug`: the SimpleFIN protocol allows null org domain. If `org_slug IS NULL`, fall back to `org_name` but log a warning. The Actual Budget project encountered this exact issue (PR #2836) and had to add null-domain handling.

**Warning signs:**
- If aggregation SQL or Go code uses `GROUP BY org_name` anywhere, it is fragile.
- Test: manually change an account's `org_name` in the database (simulate a SimpleFIN label change), run sync -- does the aggregation group survive?

**Phase to address:**
Phase 1 (Crypto Aggregation) -- the choice of grouping key is foundational and cannot be changed later without migrating user configuration.

---

### Pitfall 7: Financial Projections Using float64 Instead of shopspring/decimal

**What goes wrong:**
Compound interest calculation involves repeated multiplication: `balance * (1 + rate/periods)^(periods*years)`. If done with `float64`, IEEE 754 rounding errors accumulate over long projection horizons. A 30-year projection at 7% APY on $100,000 can be off by hundreds of dollars. The user sees a specific dollar amount and trusts it for financial planning.

**Why it happens:**
Go's `math.Pow()` only works with `float64`. `shopspring/decimal` does not have a built-in `Pow()` for fractional exponents. The temptation is to convert to float64, do the exponentiation, and convert back. The project already uses `shopspring/decimal` everywhere else (sync.go, handlers), so the inconsistency may not be noticed.

**How to avoid:**
- Use iterative compounding with `shopspring/decimal`: loop through each compounding period, multiplying by `(1 + rate/periods)` using `decimal.Mul()` and `decimal.Div()`. This avoids `math.Pow()` entirely.
- For monthly compounding over 30 years, that is 360 iterations -- trivially fast, not a performance concern.
- Store APY/growth rates as `TEXT` in the database (matching how balances are stored), not as `REAL`. Parse with `decimal.NewFromString()`.
- Always display projections with a visual and textual disclaimer: "Projected values are estimates."
- JSON serialization: when returning decimal projection values to the frontend, use `decimal.StringFixed(2)` as already done in `summary.go`, not `MarshalJSON()` which can lose precision with large numbers.

**Warning signs:**
- Any import of `math` (except `math.Min`/`math.Max` for clamping) in projection code is suspicious.
- Any `float64` cast or `.InexactFloat64()` call in financial calculation code is a bug.
- Test: project $100,000 at 7% APY for 30 years monthly compounding. Correct answer: $761,225.50. If the result differs by more than $0.01, the implementation is wrong.

**Phase to address:**
Phase 6 (Financial Projections) -- must be correct from day one since users make financial decisions based on these numbers.

---

### Pitfall 8: Growth Indicators Misleading with Division by Zero and Small Balances

**What goes wrong:**
Growth rate calculation (`(current - previous) / previous * 100`) produces three failure modes:
1. **Division by zero:** Previous balance is $0 (new account, or account was at zero). Result is infinity or NaN, which renders as "Infinity%" or crashes the frontend.
2. **Misleading percentages:** A $1 account growing to $10,001 shows "+1,000,000%" -- technically correct but meaningless for a freshly funded account.
3. **Insufficient history:** An account with only one snapshot has no previous balance to compare against. The `UNIQUE(account_id, balance_date)` constraint means the account might have a snapshot, but only for today.

**Why it happens:**
Percentage change is a ratio that breaks at boundary values. Developers test with "normal" balances ($50,000 -> $51,000 = +2%) and miss the edges. The existing `balance_snapshots` table uses `INSERT OR IGNORE` which means the first sync creates only one snapshot per account -- there is no "previous" value to compare against until the next day's sync.

**How to avoid:**
- Guard division by zero: if previous balance is zero, display "New" or show the absolute dollar change instead of a percentage.
- Set a minimum previous-balance threshold (e.g., $100) below which percentage display is suppressed. Show dollar change instead.
- Require at least 2 snapshots on different dates before showing any growth indicator. Query: `SELECT COUNT(DISTINCT balance_date) FROM balance_snapshots WHERE account_id = ?` must be >= 2.
- Use `shopspring/decimal` for the calculation (already in the project). The growth calculation should be `current.Sub(previous).Div(previous).Mul(decimal.NewFromInt(100))`. Do NOT convert to float64.
- Cap displayed percentage at a reasonable bound (e.g., +/- 999%) to prevent layout-breaking values.

**Warning signs:**
- Any `float64` cast in growth calculation code is a bug.
- Test: account with previous balance $0, current balance $500 -- should show "New" or "$500.00", not "Inf%".
- Test: account with only 1 snapshot -- should show nothing or "Tracking...".
- Test: credit card with negative balance -- percentage calculation must handle negative previous values correctly (going from -$500 to -$200 is improvement, not "60% loss").

**Phase to address:**
Phase 2 (Growth Indicators) -- edge cases must be handled in the initial implementation.

---

### Pitfall 9: SQLite Migrations Failing on Existing v1.0 Data

**What goes wrong:**
v1.1 needs multiple new columns and tables. SQLite does not support `ALTER TABLE ... ALTER COLUMN` or `DROP COLUMN` (the project uses modernc.org/sqlite). A migration that adds a `NOT NULL` column without a `DEFAULT` value will fail on tables that already have rows. `golang-migrate` wraps each SQLite migration in an implicit transaction; a failed migration marks the schema version as "dirty," and all subsequent migrations refuse to run. The application fails to start.

**Why it happens:**
Developers write migrations that pass on empty test databases (created fresh in `db_test.go`) but fail on production databases with existing data. The v1.0 `000001_init.up.sql` created tables from scratch -- there was no existing data to worry about. v1.1 migrations operate on populated tables for the first time.

**How to avoid:**
- Every `ALTER TABLE ... ADD COLUMN` must have a `DEFAULT` value. Even if the intent is NOT NULL, add the column as nullable first, backfill data, then (if absolutely needed) rebuild the table with the constraint.
- Keep migrations small: one concern per file (`000002_add_display_name.up.sql`, `000003_add_hidden_at.up.sql`, etc.).
- Always write corresponding `.down.sql` files. For `ADD COLUMN` on SQLite, the down migration requires the table-rebuild pattern: `CREATE TABLE new_table ...`, `INSERT INTO new_table SELECT ... FROM old_table`, `DROP TABLE old_table`, `ALTER TABLE new_table RENAME TO old_table`.
- NEVER put explicit `BEGIN`/`COMMIT` in migration files -- golang-migrate wraps them automatically for SQLite.
- Add a migration integration test that runs all migrations against a database seeded with v1.0 realistic data (a few accounts, ~30 balance_snapshots, some sync_log entries).

**Warning signs:**
- A migration file containing `ALTER TABLE ... ALTER COLUMN` -- will fail on SQLite.
- A migration adding a column with `NOT NULL` and no `DEFAULT` -- will fail on non-empty tables.
- `db_test.go` only tests migrations on `:memory:` with no seed data.

**Phase to address:**
Phase 1 (first migration) -- establish the migration discipline that all subsequent phases follow.

---

### Pitfall 10: SMTP Credentials Exposed Through API Responses or Logs

**What goes wrong:**
The existing `settings` table stores key-value pairs as plaintext. The `GET /api/settings` handler already returns sync status. When SMTP config is added, the natural pattern is to store `smtp_host`, `smtp_port`, `smtp_user`, `smtp_pass` in the same table and return them in the same API response. This exposes the SMTP password to the browser, visible in DevTools Network tab, and potentially cached in browser history.

Additionally, Go's `slog` structured logging will include SMTP credentials in log output if they are passed as field values during connection attempts. Docker logs (`docker logs backend`) persist these indefinitely.

**Why it happens:**
The settings table pattern from v1.0 makes it easy to add new key-value pairs. The SimpleFIN access URL is already stored in plaintext there -- so there is precedent. But SMTP passwords are often reused credentials (unlike the SimpleFIN token), making exposure more damaging.

**How to avoid:**
- Preferred approach: store SMTP config as environment variables (matching `JWT_SECRET` and `PASSWORD_HASH` pattern). Add `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM` to `internal/config/config.go`. Pass via `docker-compose.prod.yml`.
- If database storage is needed for UI configurability: store SMTP password as a separate settings key that is NEVER returned in `GET /api/settings`. Return `smtp_configured: true/false` and `smtp_host: "smtp.example.com"` but never the password. The update endpoint accepts the password but the read endpoint masks it.
- Never pass SMTP password to `slog`. Log `smtp_host`, `smtp_port`, `smtp_user` for debugging, but mask the password field.

**Warning signs:**
- If `GET /api/settings` response JSON contains an `smtp_password` or `smtp_pass` field, it is a security bug.
- If any `slog.Info` or `slog.Error` call includes `"password"` as a key, it is a security bug.

**Phase to address:**
Phase 5 (Alert Rules / Email Setup) -- SMTP credential handling must be designed before the alert email feature is built.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Storing alert expressions as free-text strings instead of structured JSON | More flexible, easier to extend syntax later | Must parse on every evaluation; validation harder; schema evolution painful; injection risk | Never -- use structured JSON from day one |
| Storing SMTP credentials in the settings table (plaintext) | Quick to implement, matches SimpleFIN URL pattern | Credentials visible to filesystem access; exposed if API endpoint returns them | Acceptable for v1.1 if the API never returns the password and env var alternative is documented |
| Hard-coding panel keys (`liquid`, `savings`, `investments`) in both Go handlers and TypeScript types | Simple, no dynamic config needed | Every new panel requires changes in 5+ files (accounts.go, summary.go, history.go, client.ts, Dashboard.tsx, PanelCard.tsx) | Acceptable -- panel types are unlikely to change for a personal finance dashboard |
| Using `SetMaxOpenConns(1)` for all operations | Prevents "database is locked" entirely | Alert evaluation + sync + API reads all serialize; alert evaluation during sync blocks API responses | Acceptable now (single user, sub-second queries) but becomes a bottleneck if alert evaluation or projection calculation is slow. Revisit if API response times exceed 200ms |
| Computing growth indicators on every API request instead of caching | No cache invalidation logic needed | Growth calc runs on every dashboard load; involves 2 snapshot queries per account | Acceptable for single user with <20 accounts. Cache only if profiling shows it matters |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| SimpleFIN + Account Renaming | Overwriting display_name on sync because the upsert updates `name` from SimpleFIN every time | Two separate columns: `name` (SimpleFIN-managed, always updated by sync upsert) and `display_name` (user-managed, never touched by sync). Display logic everywhere: `COALESCE(display_name, name)` |
| SimpleFIN + Stale Account Cleanup | Hard-deleting accounts that temporarily disappear, destroying display_name, alert rules, APY settings | Soft-delete with `hidden_at` timestamp. Auto-restore when account reappears. Manual "permanently delete" in settings |
| SimpleFIN + Aggregation Grouping | Using `org_name` (human-readable, can change) as the grouping key | Use `org_slug` (domain-like, stable) as grouping key. Display `org_name` as label. Handle NULL `org_slug` with fallback |
| Alert Emails + Sync Cycle | Evaluating alerts and sending emails inside the sync transaction / mutex, blocking dashboard API | Evaluate alerts AFTER sync completes and mutex is released. Collect all alerts to fire, then send emails asynchronously in a goroutine. If SMTP times out, sync data is already committed |
| Docker + SMTP | Hardcoding SMTP config in docker-compose, requiring rebuild to change | Pass as environment variables in `docker-compose.prod.yml`, matching the existing pattern for `JWT_SECRET` and `PASSWORD_HASH` |
| Net Worth Drill-Down + Balance History | Reusing the existing `GET /api/balance-history` endpoint which returns panel-grouped data, not per-account data | The drill-down needs per-account time series, not just panel totals. Create a new endpoint that returns individual account histories with the account's display_name |
| Growth Indicators + Credit Cards | Applying percentage change to credit card balances without considering sign semantics | Credit card balances are already negative in the database. Going from -$500 to -$200 is a $300 improvement, but naive `(current-previous)/previous*100` gives `-60%`. Use absolute value of previous for the denominator, or display "Paid down $300" instead of a percentage |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Net worth drill-down loading all snapshots for all accounts into the browser | Browser freezes rendering thousands of data points in Recharts | Server-side aggregation: accept a `resolution` parameter (`daily`, `weekly`, `monthly`). Return weekly averages for ranges > 90 days, monthly for > 1 year | At ~1000 data points per series. Recharts handles ~500 smoothly |
| Correlated subquery for "latest balance" repeated in every new handler | Each new endpoint (growth, alerts, projections) re-queries the same data pattern | Extract a shared `latestBalances(ctx, db)` Go function used by summary, accounts, growth, and alert evaluation. The existing `idx_balance_snapshots_account_date` index covers it | Not a real performance issue at this scale, but a maintainability issue with copy-pasted queries |
| Alert evaluation querying the database once per rule per sync | N+1 query pattern if evaluating 20 rules individually | Batch: read all current balances once (reuse `latestBalances()`), compute all panel totals once, then evaluate all rules against that snapshot in-memory | Not a performance concern for <50 rules, but the batch pattern is simpler code anyway |
| Projection calculation done on every page load with no caching | Noticeable delay on the projection tab, especially for long horizons with many accounts | Cache projections and invalidate when: (a) a sync updates balances, (b) user changes APY/growth settings. For v1.1, recalculating on each load is acceptable -- 360 iterations per account is microseconds | Never at this scale |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Returning raw `sync_log.error_text` to frontend without sanitization | Error messages may contain SimpleFIN access URL with embedded Basic Auth credentials | Sanitize errors before storage: strip URL credentials. Return error categories in API, not raw text |
| Storing SMTP password in settings table and returning it in `GET /api/settings` | SMTP password exposed to authenticated browser sessions via DevTools | Never return SMTP password in API responses. Return `smtp_configured: true/false` only |
| Alert rule expressions evaluated with a general-purpose engine | Expression injection leads to env var access (JWT_SECRET, PASSWORD_HASH, SMTP credentials) | Restricted DSL with structured JSON. No `eval()`. No JavaScript runtime |
| Logging SMTP credentials in slog during email send attempts | Credentials appear in Docker logs, persisted indefinitely | Log `smtp_host` and `smtp_user` but always mask password. Never pass password to slog fields |
| Email alert subject line containing account balances | Financial data exposed in email notifications, lock screens, forwarded messages | Subject: "Finance Alert: Threshold Crossed." Balance details in body only |
| Alert rule referencing a deleted/hidden account continues to fire with stale data | Phantom alerts based on the last known balance of an account that no longer exists | Skip evaluation for alert rules that reference hidden/deleted accounts. Mark such rules as "paused" in the UI |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Growth indicators on accounts with <7 days of history | User sees "+5,000%" on a newly synced account due to small initial balance | Require minimum 7 days of history AND previous balance > $100. Show "Tracking..." badge otherwise |
| Projection page showing exact dollar amounts without disclaimers | User treats projections as financial promises | Show projections with explicit "Estimates based on assumed growth rates" disclaimer. Use dashed/muted line styling for projected portions of charts |
| Account renaming not reflected in all UI surfaces | User renames "Personal Checking" to "Main Account" but old name appears in chart tooltips, alert displays, or projection settings | Use `COALESCE(display_name, name)` in EVERY query that returns account names. Create a shared Go helper. Audit all existing handlers: `accounts.go`, `summary.go`, `history.go` |
| Alert rules UI requiring raw expression syntax | User doesn't know valid field names or operators, writes invalid expressions | Structured form with dropdowns: metric selector, operator selector, numeric input. Generate JSON from the form. Never accept free-text expressions |
| Net worth drill-down showing daily noise from checking account fluctuations | Small daily changes obscure long-term trends | Default to 30-day moving average or weekly resolution. Provide "Daily" as an explicit toggle |
| Sync diagnostics showing raw Go error chains | "sync: fetch accounts: simplefin: HTTP request failed: Get ..." is meaningless to a non-developer user | Map error patterns to user-friendly messages: "Could not connect to SimpleFIN", "Authentication failed (check your access URL)", "Unexpected response from SimpleFIN" |

## "Looks Done But Isn't" Checklist

- [ ] **Account Renaming:** display_name appears in PanelCard account list (`accounts.go`) -- verify it ALSO appears in BalanceLineChart tooltips, NetWorthDonut labels, the new drill-down page account selector, alert rule displays, and projection account labels
- [ ] **Crypto Aggregation:** summed balance is correct -- verify the aggregated entry also shows correct growth indicators (calculated from the sum of sub-account snapshot histories, not individual history averages)
- [ ] **Growth Indicators:** percentage shows on PanelCard -- verify it handles: (a) negative balances (credit cards), (b) $0 previous balance, (c) accounts with only 1 snapshot, (d) negative growth rendering (red color, down arrow, not just a minus sign)
- [ ] **Alert Rules:** alert fires correctly -- verify: (a) fires only ONCE per threshold crossing, (b) fires recovery notification when condition clears, (c) does NOT fire on initial rule creation if condition is already true (establish baseline first), (d) handles account disappearing mid-rule gracefully
- [ ] **Alert Emails:** email sends successfully -- verify: (a) works when SMTP server is down (graceful failure, not crash/panic), (b) SMTP timeout does not block sync completion, (c) invalid email address is logged but not fatal, (d) email body does not contain SMTP credentials in error scenarios
- [ ] **Sync Diagnostics:** errors display in settings -- verify: (a) no SimpleFIN credentials in displayed text, (b) no internal file paths exposed, (c) error messages are human-readable, (d) successful syncs also shown (not just errors)
- [ ] **Financial Projections:** compound interest is correct -- verify: (a) uses `shopspring/decimal` not float64, (b) handles 0% APY without division-by-zero, (c) matches a reference calculator to the cent for 30-year projections, (d) income modeling allocations sum correctly
- [ ] **Net Worth Drill-Down:** chart renders -- verify: (a) performs with 365+ data points per series, (b) time range selector filters server-side (not hiding client data), (c) hidden/deleted accounts excluded from totals
- [ ] **Migration Safety:** all new migrations run against a v1.0 database with real data, not just empty test databases

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Stale account deletion destroys user-owned config | HIGH | No automatic recovery -- display_name, alert rules, APY settings are gone. Historical snapshots are gone. Must re-sync from SimpleFIN (gets current balance only). Database backup is the only recovery path. **Prevention is the only real strategy** |
| Expression injection in alert rules | MEDIUM | Rotate `JWT_SECRET` and `PASSWORD_HASH`. Delete all stored alert rules. Audit sync_log for exploitation evidence. Redeploy with structured expression storage |
| Alert flooding | LOW | Delete accumulated emails. Add state machine. Reset all alert states to `NORMAL`. No data loss |
| SMTP credentials leaked via API | MEDIUM | Rotate SMTP password. Patch API to stop returning it. Audit logs for who accessed the endpoint |
| Sync error leaking SimpleFIN URL | MEDIUM | Regenerate SimpleFIN access token (re-claim setup token in SimpleFIN Bridge). Sanitize existing `sync_log` rows. Patch error handling |
| float64 projection errors | LOW | Replace with decimal arithmetic. No data loss -- projections are computed on the fly, not stored |
| Migration leaves database in dirty state | HIGH | Manually fix `schema_migrations` table: set `dirty = false` and correct `version`. Write corrective migration. Requires direct SQLite access via `docker exec` into the volume |
| org_name change breaks aggregation | MEDIUM | If using `org_slug` as key: no impact. If using `org_name`: must rebuild aggregation groups. User loses custom group names if any |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Stale account deletion destroys user data | Phase 1 (Account Renaming -- first migration) | Disconnect SimpleFIN, sync, reconnect, sync -- display_name and history intact |
| Crypto aggregation merging wrong account types | Phase 1 (Crypto Aggregation) | Test data: same org, different types -- remain separate in UI |
| org_name instability breaking aggregation | Phase 1 (Crypto Aggregation) | Change org_name in DB, sync -- aggregation grouping unchanged because keyed on org_slug |
| Migration breaking existing data | Phase 1 (first migration) | Migration test against v1.0 database backup with real data |
| Growth indicator division by zero | Phase 2 (Growth Indicators) | $0 previous -> shows "New"; 1 snapshot -> shows "Tracking..." |
| Growth indicator misleading on credit cards | Phase 2 (Growth Indicators) | Credit card -$500 -> -$200 shows "Paid down $300" or correct positive improvement % |
| Sync error leaking credentials | Phase 4 (Sync Diagnostics) | Break SimpleFIN URL, sync, check UI -- no credentials visible |
| Expression injection in alert rules | Phase 5 (Alert Rules) | Attempt to inject `os.Getenv("JWT_SECRET")` -- rejected at parse time |
| Alert threshold oscillation flooding | Phase 5 (Alert Rules) | Threshold at current balance, sync 3x -- exactly 1 email |
| SMTP credentials exposed | Phase 5 (Alert Rules / Email) | `GET /api/settings` response has no smtp_password field |
| float64 in projections | Phase 6 (Projections) | 30-year projection matches reference calculator to the cent |
| Net worth drill-down performance | Phase 3 (Net Worth Drill-Down) | Load with 1000+ data points -- no browser freeze, server-side resolution control |

## Sources

- Direct codebase analysis: `internal/sync/sync.go` (stale account deletion, sync flow), `internal/simplefin/client.go` (credential handling), `internal/db/migrations/000001_init.up.sql` (current schema), `internal/api/handlers/*.go` (API response patterns), `internal/db/db.go` (connection config), `internal/config/config.go` (env var pattern), `frontend/src/components/PanelCard.tsx` (display_name usage surface)
- [CVE-2025-68613: Critical RCE via Expression Injection in n8n](https://nvd.nist.gov/vuln/detail/CVE-2025-68613) -- CVSS 9.9, expression sandbox escape leading to full RCE
- [SimpleFIN Protocol Specification](https://www.simplefin.org/protocol.html) -- org field structure, account ID semantics
- [Handle Null Org Domain in SimpleFIN (Actual Budget PR #2836)](https://github.com/actualbudget/actual/pull/2836) -- real-world evidence that SimpleFIN org fields can be null
- [SQLite concurrent writes and "database is locked" errors](https://tenthousandmeters.com/blog/sqlite-concurrent-writes-and-database-is-locked-errors/) -- WAL mode pitfalls, checkpoint starvation
- [ShopSpring Decimal (pkg.go.dev)](https://pkg.go.dev/github.com/shopspring/decimal) -- JSON serialization precision warnings
- [Self-Hosting Email: Deliverability Risks](https://powerdmarc.com/self-hosting-email/) -- SMTP deliverability challenges for self-hosted apps
- [Mastering Database Migrations with golang-migrate and SQLite](https://dev.to/ouma_ouma/mastering-database-migrations-in-go-with-golang-migrate-and-sqlite-3jhb) -- SQLite-specific migration constraints

---
*Pitfalls research for: v1.1 feature additions to Go/React/SQLite personal finance dashboard*
*Researched: 2026-03-15*
