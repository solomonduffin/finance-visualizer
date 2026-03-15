# Pitfalls Research

**Domain:** Self-hosted personal finance dashboard (Go + React + SQLite + SimpleFIN)
**Researched:** 2026-03-15
**Confidence:** MEDIUM — WebSearch and WebFetch were unavailable; findings based on training knowledge of the SimpleFIN protocol, personal finance aggregator patterns, Go+SQLite applications, and self-hosted Docker deployment. Flagged where external verification would increase confidence.

---

## Critical Pitfalls

### Pitfall 1: Clobbering Historical Snapshots on Re-Fetch

**What goes wrong:**
The daily sync overwrites balance records for the day rather than appending a new snapshot. After a few weeks, running a backfill or re-triggering the cron deletes valid historical data. The balance-over-time chart either goes flat or shows gaps.

**Why it happens:**
`INSERT OR REPLACE` / `ON CONFLICT DO UPDATE` is the easy upsert pattern in SQLite. Developers apply it uniformly without distinguishing "current account state" (should upsert) from "daily balance snapshot" (should only write once per day per account). The two concerns get merged into one write path.

**How to avoid:**
Separate the data model into two concerns:
1. `accounts` table — current metadata, upserted on every sync (name, institution, account type, currency).
2. `balance_snapshots` table — append-only, one row per `(account_id, snapshot_date)`, with a `UNIQUE (account_id, snapshot_date)` constraint and `ON CONFLICT DO NOTHING` so re-runs never overwrite.

Never delete from `balance_snapshots` except through an explicit admin operation.

**Warning signs:**
- "Full history pull on first sync" logic shares the same write function as the daily cron.
- A single `upsert_balance()` function used everywhere.
- Chart data shows only 1–2 data points after multiple weeks of operation.

**Phase to address:**
Database schema phase (Phase 1 / foundation). The snapshot model must be correct from the first migration — retrofitting an append-only log into an existing schema that was designed for upserts is painful.

---

### Pitfall 2: Storing the SimpleFIN Access URL in the Wrong Place

**What goes wrong:**
The SimpleFIN setup token (used once to claim an access URL) and the resulting access URL are stored in plaintext in environment variables, config files, or the SQLite database without encryption. If the container is compromised or the database file is copied off the host, an attacker gains persistent read access to all connected financial accounts.

**Why it happens:**
It's a single credential for a single user — developers treat it like a DATABASE_URL and embed it in `docker-compose.yml` or `.env` without additional protection. Unlike a database password, SimpleFIN access URLs are long-lived bearer tokens with no expiry mechanism under the user's control (the institution controls expiry).

**How to avoid:**
- Store the access URL as a Docker secret or environment variable that is never written to the database or committed to version control.
- Add `.env` to `.gitignore` from day one.
- Document in the README that the access URL must be treated like a private key.
- Optionally: encrypt at rest using a key derived from the user's login password (though this adds complexity — acceptable to defer if the self-hosted model is trusted).

**Warning signs:**
- `docker-compose.yml` contains `SIMPLEFIN_ACCESS_URL=https://...` hardcoded.
- Database contains a `settings` table with the access URL in plaintext.
- The Git history includes any version of a `.env` file.

**Phase to address:**
SimpleFIN integration phase (Phase 2). Establish the credential handling pattern before writing any fetching logic.

---

### Pitfall 3: No Retry / Error Isolation for the Daily Cron

**What goes wrong:**
SimpleFIN (or the upstream institution bridge) returns a 5xx, a timeout, or malformed JSON for one account. The entire cron goroutine panics or returns early, leaving zero records for that day across all accounts. The chart shows a gap that looks like missing data but is actually a fetch failure. There is no way to tell the difference.

**Why it happens:**
The happy path is straightforward: fetch all accounts, write balances. Error handling gets deferred to "later." Goroutines that panic without a recovery harness silently die. A single bad account poisons the whole batch.

**How to avoid:**
- Fetch each account (or the entire SimpleFIN response) inside an isolated error boundary. Log failures per-account, not globally.
- Record sync attempts in a `sync_log` table: `(id, started_at, finished_at, accounts_fetched, accounts_failed, error_text)`. The dashboard can surface "last sync: 2 days ago — check logs."
- Use `recover()` in the goroutine to catch panics and convert them to logged errors, not silent deaths.
- Implement exponential backoff with a single retry before giving up for the day.

**Warning signs:**
- Cron handler is a single function with one error return: `if err != nil { log.Fatal(err) }`.
- No `sync_log` table in the schema.
- Dashboard has no "last synced at" indicator.

**Phase to address:**
SimpleFIN integration phase (Phase 2). Build the sync loop with observability from the start, not as a polish item.

---

### Pitfall 4: Pending Transactions Counted Twice in Liquid Balance

**What goes wrong:**
The "liquid balance" panel (checking minus credit card) is computed as: `posted_balance + sum(pending_transactions)`. But SimpleFIN's `balance` field on an account already includes pending transactions in many institution bridges. The result double-counts pending debits, making the user's apparent liquid cash lower than reality.

**Why it happens:**
The SimpleFIN protocol reports a `balance` field whose semantics vary by institution bridge. Some bridges include pending items in balance; others report only posted balance. Developers assume uniformity and add pending amounts on top without checking.

**How to avoid:**
- Read the SimpleFIN protocol carefully: the `balance` field is defined as "the current balance as of the data source." This typically includes pending transactions at most institutions.
- Default to using `balance` directly, without adding pending transaction amounts.
- If the drill-down view needs to show pending items, display them as informational context (listed below the balance), not as addends.
- During integration testing, cross-check the API balance against the user's actual bank app balance.

**Warning signs:**
- Liquid balance is consistently $50–$500 lower than what the bank app shows.
- Code contains `account.Balance + sum(pendingTransactions.Amount)`.
- No integration test comparing fetched balance to a known manual check.

**Phase to address:**
SimpleFIN integration phase (Phase 2) and the liquid balance calculation logic.

---

### Pitfall 5: Password Auth Implemented with No Brute-Force Protection

**What goes wrong:**
A simple password check (`bcrypt compare`) with no rate limiting, no lockout, and no timing consistency allows an attacker who reaches the login endpoint to brute-force the password offline or online. Since this app exposes financial data, a compromised login is a significant privacy breach.

**Why it happens:**
Single-user apps feel low-risk. Developers implement the simplest `POST /login` → check password → set session cookie flow and move on. Rate limiting is seen as a production concern to add later, then never added.

**How to avoid:**
- Add a middleware rate limiter on the login endpoint from day one (e.g., 5 attempts per IP per 15 minutes). Go's `golang.org/x/time/rate` or a simple in-memory token bucket is sufficient for single-user.
- Use `bcrypt` with cost factor >= 12.
- Return HTTP 429 after threshold, not just a failed login message.
- Consider HTTP Basic Auth via Nginx if the Go auth layer feels like overkill — but then ensure Nginx is never bypassed.

**Warning signs:**
- `/login` endpoint has no rate limiting middleware.
- bcrypt cost is 10 or lower.
- Login failures and successes return in the same response time (timing oracle).

**Phase to address:**
Auth phase (likely Phase 1 or 2, depending on roadmap ordering). Auth must be correct before the app is exposed on the network.

---

### Pitfall 6: SQLite WAL Mode Not Enabled — Cron Blocks Dashboard Reads

**What goes wrong:**
The daily sync goroutine holds a write lock on the SQLite database for several seconds while inserting balance rows. During this window, the Go HTTP server cannot service read requests (dashboard page loads). The user sees a hanging request exactly once a day, usually first thing in the morning.

**Why it happens:**
SQLite defaults to "rollback journal" mode, which blocks all readers during a write. WAL (Write-Ahead Log) mode allows concurrent reads during writes. Most Go SQLite tutorials don't mention WAL mode for small apps.

**How to avoid:**
- Set `PRAGMA journal_mode=WAL` immediately after opening the database connection.
- Also set `PRAGMA busy_timeout=5000` to handle the rare case where WAL itself contends.
- Use a single `*sql.DB` connection pool with `SetMaxOpenConns(1)` on the writer, or use separate reader/writer connections.

**Warning signs:**
- `db, _ := sql.Open("sqlite3", "finance.db")` with no pragmas set.
- Dashboard occasionally hangs for 2–10 seconds around the scheduled cron time.
- No WAL file (`finance.db-wal`) visible in the data directory.

**Phase to address:**
Database setup phase (Phase 1). A one-line pragma fix, but must be in the initial connection setup.

---

### Pitfall 7: Investment "Performance" Charts Based on Balance Snapshots, Not Cost Basis

**What goes wrong:**
Investment growth/loss charts show the change in account balance over time. But balance changes include both market movement AND new contributions (401k payroll deductions, IRA deposits). The chart shows "up 8% this year" which is actually "up 3% market gain plus 5% new money added." The user cannot distinguish performance from contributions.

**Why it happens:**
SimpleFIN does not provide cost basis, contribution history, or holding-level data — only account balance. Developers treat balance delta as performance. This is technically correct for display but misleading for investment performance evaluation.

**How to avoid:**
- Label charts clearly: "Account Value Over Time" not "Investment Performance."
- Include a disclaimer in the UI: "Growth includes new contributions."
- Do not label the delta as "return" or "gain/loss" unless cost basis data is available (it won't be via SimpleFIN).
- This is a UX/labeling fix, not a data fix — SimpleFIN simply doesn't provide what's needed for true performance metrics.

**Warning signs:**
- Chart Y-axis is labeled "Return (%)" or "Gain/Loss."
- Investment panel shows a "+X%" figure without a "includes contributions" caveat.

**Phase to address:**
Investment panel and charting phase. Label correctly from day one; retrofitting language after users have internalized the wrong framing is confusing.

---

### Pitfall 8: Docker Container Runs as Root with SQLite Volume Permissions

**What goes wrong:**
The Docker container runs as `root` (Go app default). The SQLite `.db` file on the host bind-mount is owned by root. When the user tries to back up, inspect, or migrate the database file from the host, they need `sudo`. If the container is compromised, the process has root on the host filesystem for the mounted volume.

**Why it happens:**
Go Dockerfiles that start from `scratch` or `alpine` don't have a non-root user pre-configured. Developers add the volume mount and ship without thinking about UID/GID.

**How to avoid:**
- Add a `USER` directive in the Dockerfile to run as a non-root UID (e.g., `RUN adduser -D -u 1001 appuser && USER appuser`).
- Document the required host UID in `docker-compose.yml` comments so the bind-mount file is accessible from the host without sudo.
- Set the data directory to `755` and the database file to `644` owned by that UID.

**Warning signs:**
- `Dockerfile` has no `USER` instruction.
- `docker-compose.yml` bind-mounts `./data:/data` without UID mapping.
- Database file on host is owned by `root`.

**Phase to address:**
Docker/deployment phase (likely the final phase). Low urgency for local dev, but fix before considering any networked self-hosting.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Hardcoding the cron schedule instead of making it configurable | Simpler code | Cannot adapt to rate limits or personal preference without redeploy | Only MVP; add env var before v1 |
| Storing all API response JSON in a blob column for later parsing | Flexible schema, fast initial build | Querying is painful; migration required to normalize | Short-term prototyping only — normalize before building charts |
| Single `db.go` file for all database operations | Fast iteration | Untestable, bloats quickly | Never beyond ~5 queries |
| No database migrations (just `CREATE TABLE IF NOT EXISTS`) | No migration tooling needed | Schema drift between versions, impossible to change columns safely | Single-developer, single-deploy — acceptable in early phases |
| Fetching all accounts and transactions on every page load | Simpler backend | Slow dashboards once history grows | Never — cache fetched data from day one |
| JWT tokens stored in localStorage | Easy to implement | XSS vulnerability exposes auth token | Never for a financial app — use HttpOnly cookies |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| SimpleFIN | Using the setup token URL as the access URL | Claim the access URL once at setup time; the setup token is one-use-only and expires |
| SimpleFIN | Not handling `errors` array on accounts | Each account in the response can have an `errors` array even if HTTP 200 — must check per-account |
| SimpleFIN | Treating `balance` as posted-only and adding pending on top | `balance` typically includes pending at most institutions; use it directly |
| SimpleFIN | Ignoring `balance-date` field | The balance may be from yesterday; store `balance_date` separately from `fetched_at` |
| SimpleFIN | Not storing raw `organization` slug | Institution name display and deduplication rely on the org slug being stable |
| SQLite + Go | Using `database/sql` without WAL pragma | Default journal mode causes write-blocking reads; set WAL on open |
| SQLite + Go | Opening multiple connections without `_busy_timeout` | Concurrent access panics with "database is locked" without a busy timeout set |
| Docker + SQLite | Bind-mounting a volume without pre-creating the directory | Docker creates it as root, causing permission errors at runtime |
| Nginx + Go | Passing `X-Forwarded-For` without configuring Go to trust the proxy | Rate limiting by IP sees all traffic from `127.0.0.1`; must parse the forwarded header |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Fetching full transaction history on every chart render | Chart load >2s, repeated DB scans | Pre-aggregate daily balance snapshots at write time | After ~90 days of history |
| No index on `(account_id, snapshot_date)` in balance_snapshots | Chart queries slow as history grows | Add composite index at schema creation | After ~365 rows per account |
| Computing net worth by summing all transactions | Gets slower linearly with transaction count | Store explicit balance snapshots; transactions are supplementary | After ~1000 transactions |
| Returning all transaction records to the frontend | Large JSON payload, slow renders | Paginate or limit to last N transactions; charts use pre-aggregated data | After ~500 transactions |
| Blocking the HTTP handler goroutine during cron | Dashboard unresponsive during sync | Run cron in a separate goroutine; never block HTTP handlers | First time sync takes >1s |

*Note: This is a single-user app with daily snapshots. At realistic scale (~5 accounts × 365 days = 1825 rows/year), performance is not a concern for queries. The traps above are about poor data modeling, not raw scale.*

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Exposing the app on a public port without TLS | Credentials and financial data transmitted in plaintext | Nginx TLS termination (Let's Encrypt or self-signed) before any network exposure |
| No rate limiting on the login endpoint | Brute-force attack succeeds in hours | 5 attempts/IP/15 min middleware; HTTP 429 response |
| Session token in localStorage | XSS attack steals session | Use HttpOnly, SameSite=Strict cookies for session storage |
| SimpleFIN access URL in Git history | Permanent exposure of read access to all accounts | `.gitignore` the `.env` file from commit 1; rotate the access URL if ever committed |
| No CSRF protection on state-changing endpoints | CSRF attack triggers unauthorized actions | Even with no write operations, protect `/logout` and any future POST endpoints |
| API endpoints reachable without auth check | Direct URL access bypasses login | Apply auth middleware to ALL `/api/*` routes, not just individual handlers |
| Password stored as MD5 or SHA-256 | Offline dictionary attack on database dump | bcrypt cost >= 12, always |
| Detailed error messages returned to client | Leaks stack traces, file paths, DB schema | Structured server-side logging; return only `{"error": "internal error"}` to client |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| No "last synced" timestamp on dashboard | User cannot tell if data is stale (e.g., SimpleFIN was down) | Always show "Last synced: X hours ago" prominently; highlight if >25 hours |
| Investment chart labeled as "performance" when it includes contributions | User misreads portfolio returns | Label as "Account Value" with note "includes contributions" |
| Pending transactions shown as confirmed | User plans spending based on incorrect liquid balance | Style pending items differently (italic, muted color, "(pending)" label) |
| No loading states during initial data fetch | Blank dashboard on first load looks broken | Skeleton loaders or spinner with "Loading your accounts..." |
| Hard-coding currency symbol as $ | International users (or future self with foreign accounts) see wrong symbol | Use the `currency` field from SimpleFIN account data; fall back to $ |
| Chart date axis shows UTC midnight instead of local date | Chart appears to show data one day off | Parse dates as local timezone or strip time component entirely for daily snapshots |
| Showing $0.00 when sync has never run | Looks like accounts have zero balance | Distinguish "no data yet" from "balance is zero" — show "Awaiting first sync" state |
| Net worth chart with no baseline reference | User cannot tell if they're on track | Optional: add a simple "starting from X date" reference point |

---

## "Looks Done But Isn't" Checklist

- [ ] **SimpleFIN setup flow:** The setup token is one-use-only — verify the app claims the access URL and persists it, not just the setup token.
- [ ] **Daily cron:** Verify it actually runs via goroutine at the configured time AND that it logs its execution — a silent no-op is indistinguishable from a working cron.
- [ ] **Historical backfill:** "Full history pull on first sync" — verify the date range parameter is sent correctly to SimpleFIN and that the app handles accounts with no historical data gracefully.
- [ ] **Liquid balance calculation:** Cross-check the computed value against the actual bank app balance; pending double-counting is invisible without manual verification.
- [ ] **Auth protection:** Hit `/api/accounts` without a session cookie — should return 401, not data.
- [ ] **Dark mode:** Financial numbers with insufficient contrast in dark mode are a common oversight — verify all numerical values pass WCAG AA contrast in both modes.
- [ ] **Empty states:** View dashboard before first sync completes — should not show null errors or blank panels with no explanation.
- [ ] **Docker volume persistence:** Restart the container and verify historical data survives — confirm the SQLite file is on a named volume or bind mount, not inside the container layer.
- [ ] **Error surface:** Deliberately provide a bad SimpleFIN access URL and verify the app logs the error, surfaces it in the sync log, and does not crash.

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Snapshot data clobbered by upsert logic | HIGH | Write a migration to add the append-only constraint; manually re-fetch historical data via SimpleFIN's date range parameters; may lose data if SimpleFIN bridge doesn't retain long history |
| Access URL committed to Git | MEDIUM | Revoke via SimpleFIN website, generate new access URL, rotate in deployment; scrub Git history with `git filter-branch` or BFG |
| WAL mode not set — DB corruption under concurrent load | MEDIUM | Stop the app, run `sqlite3 finance.db "PRAGMA journal_mode=WAL;"`, update connection code, redeploy; data is usually intact |
| No sync logging — hard to diagnose why data is missing | LOW | Add sync_log table and backfill last N sync results from application logs if available |
| Investment charts mislabeled as performance | LOW | Pure UX fix — update labels and add disclaimer text; no data migration needed |
| Password auth has no rate limiting | MEDIUM | Add middleware and redeploy; no data migration, but assess whether any brute-force occurred in access logs first |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Snapshot clobbering (append-only) | Phase 1: Database schema | Unit test: re-run sync for same date, assert row count = 1 per account |
| Access URL credential handling | Phase 2: SimpleFIN integration | Verify `.env` is gitignored; no plaintext URL in DB |
| Cron error isolation + sync log | Phase 2: SimpleFIN integration | Integration test: inject a bad account, assert others succeed and sync_log records failure |
| Pending transaction double-counting | Phase 2: SimpleFIN integration | Manual cross-check against bank app balance after first real fetch |
| Login brute-force protection | Phase 1 or 2: Auth | Test: submit 10 rapid login attempts, assert 429 on 6th |
| SQLite WAL mode | Phase 1: Database setup | Verify WAL pragma in connection init; check for `-wal` file after first write |
| Investment chart labeling | Phase 3 or 4: Charts/UI | Design review: no "return" or "performance" labels without cost basis data |
| Docker non-root user | Final phase: Deployment | `docker exec <container> whoami` must not return `root` |
| No loading / empty states | Phase 3 or 4: UI | Test with empty database; test with slow network simulation |

---

## Sources

- SimpleFIN protocol specification (https://www.simplefin.org/protocol.html) — training knowledge; **verify balance-date semantics and errors array structure against current spec**. Confidence: MEDIUM.
- SQLite WAL mode documentation (https://www.sqlite.org/wal.html) — training knowledge; HIGH confidence for core behavior.
- Go `database/sql` + SQLite WAL pragma patterns — common Go community practice; HIGH confidence.
- Personal finance aggregator post-mortems (Mint shutdown, Copilot, Monarch, YNAB community forums) — training synthesis; MEDIUM confidence for feature/UX patterns.
- OWASP session management and brute-force prevention guidelines — HIGH confidence for security patterns.
- Docker non-root user best practices — HIGH confidence; well-established pattern.
- Timezone handling in financial date display — MEDIUM confidence; known common mistake in time-series dashboards.

*Note: WebSearch and WebFetch were unavailable during this research session. All findings are based on training knowledge. Pitfalls marked MEDIUM confidence should be spot-checked against the current SimpleFIN protocol spec before implementation.*

---
*Pitfalls research for: Self-hosted personal finance dashboard (Go + React + SQLite + SimpleFIN)*
*Researched: 2026-03-15*
