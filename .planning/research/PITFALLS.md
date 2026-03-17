# Pitfalls Research

**Domain:** Adding features (transactions, spending analytics, investment tracking, data export, goal tracking) to an existing Go/React/SQLite personal finance dashboard with SimpleFIN
**Researched:** 2026-03-17
**Confidence:** HIGH (existing codebase examined, SimpleFIN protocol verified, community issues surveyed)

## Critical Pitfalls

### Pitfall 1: SimpleFIN Does Not Provide Transactions Today -- the Client Must Be Extended

**What goes wrong:**
The current SimpleFIN client (`internal/simplefin/client.go`) only parses `Account` and `Holding` structs. The SimpleFIN protocol **does** include a `transactions` array on each account (with `id`, `posted`, `amount`, `description`, `pending` fields), but the codebase currently ignores it entirely -- `FetchAccounts` uses `balances-only=1` and `FetchAccountsWithHoldings` only parses holdings, not transactions. Building any transaction-dependent feature (spending analytics, categorization, budgeting, data export with line items) requires extending the client, adding a `Transaction` struct, and modifying the sync pipeline to persist individual transactions.

**Why it happens:**
The v1.0/v1.1 scope was explicitly a balance/net-worth dashboard. The SimpleFIN integration was designed around balance snapshots, not transaction history. It is easy to assume "SimpleFIN gives us everything" when in reality the client only extracts what was needed for the original scope.

**How to avoid:**
- Extend `simplefin.Account` to include a `Transactions []Transaction` field with JSON tag `"transactions"`.
- Create a `Transaction` struct with fields: `ID string`, `Posted int64`, `Amount string`, `Description string`, `Pending bool`.
- Add a new fetch mode or modify `FetchAccountsWithHoldings` to also parse transactions (they arrive when `balances-only` is NOT set).
- Create a `transactions` table in SQLite with a composite unique constraint on `(account_id, external_id)` for idempotent upserts.
- The sync pipeline must handle transactions as a separate concern from balance snapshots -- they have different cardinality (many transactions per day vs. one balance snapshot per day).

**Warning signs:**
- Any feature spec that says "use transaction data" without mentioning the client extension is incomplete.
- If the transaction table design discussion is skipped, deduplication will fail.

**Phase to address:**
The very first phase of v1.2 that touches transactions. This is the foundational prerequisite for spending analytics, categorization, and export.

---

### Pitfall 2: Transaction Deduplication is Harder Than It Looks

**What goes wrong:**
SimpleFIN provides transaction IDs, but these are institution-dependent and may not be stable. Pending transactions can change ID when they clear. Banks may report the same transaction with slightly different descriptions across fetches. Without robust deduplication, the same transaction appears multiple times, corrupting spending totals and analytics.

Actual Budget's SimpleFIN integration has documented bugs with duplicate transactions ([Issue #2519](https://github.com/actualbudget/actual/issues/2519)) and cross-account mirror transactions from banks like Mercury ([Issue #7015](https://github.com/actualbudget/actual/issues/7015)). Firefly III's integration also suffers from transaction duplication on re-import.

**Why it happens:**
Developers treat transaction IDs as globally unique and stable, when in practice: (a) pending-to-cleared transitions can change the ID, (b) some institutions provide non-unique IDs, (c) description text can change between pending and cleared states, and (d) some banks produce mirror transactions across linked accounts (e.g., Mercury creates matching debits/credits).

**How to avoid:**
- Use SimpleFIN's transaction `id` as `external_id` with a `UNIQUE(account_id, external_id)` constraint for primary dedup.
- Implement a secondary fuzzy-match layer: same account, same date (+/-1 day), same amount, similar description -- to catch pending-to-cleared transitions where the ID changes.
- Store a `pending` boolean and update (not duplicate) when a pending transaction clears.
- Handle pending-to-cleared as an UPDATE, not INSERT: match on amount + date proximity, then update the `external_id`, `description`, and `pending` flag.
- Add a `dedup_hash` column (SHA256 of account_id + amount + date) as a fallback matching key.

**Warning signs:**
- Spending totals that seem "too high" after a sync cycle.
- Transaction count grows faster than expected.
- Users see the same purchase listed twice with slight description variations.

**Phase to address:**
Must be solved in the transaction storage phase, before any analytics are built on top. If dedup is bolted on later, existing bad data requires a cleanup migration.

---

### Pitfall 3: SimpleFIN's 90-Day Window and Daily Rate Limit Constrain Historical Data

**What goes wrong:**
SimpleFIN limits date range queries to 90 days maximum, and the number of historical days actually available varies per institution. Additionally, the protocol enforces 24 requests per day. This means: (a) you cannot backfill years of transaction history for analytics, (b) if you miss syncing for > 90 days, that gap is permanent, (c) if you burn through the rate limit with aggressive syncing, your access token gets disabled.

**Why it happens:**
Feature designers imagine "show spending for the last year" without realizing the data source can only look back 90 days. The existing codebase already handles this for balances (first sync = 30 days back), but transactions have a different profile -- you need every transaction, not just a point-in-time balance.

**How to avoid:**
- Design all analytics features to work with "data available since first transaction sync" rather than assuming arbitrary history depth.
- Show clear date range indicators in the UI: "Spending data available since [first transaction date]."
- Start syncing transactions immediately so the rolling window accumulates -- every day of delay is a day of lost history.
- Never request more than 90 days in a single fetch. On first transaction sync, request 90 days (the max), then daily incremental syncs going forward.
- Stay well under the 24-request daily limit. The current daily cron approach (1 request/day) is correct. Do not add "sync now" spam for transactions.
- Consider a manual CSV/OFX import feature as a one-time backfill option for historical data that predates SimpleFIN enrollment.

**Warning signs:**
- Feature specs referencing "all-time" analytics without acknowledging the data gap.
- Users clicking "Sync Now" repeatedly, hitting rate limits.
- Analytics pages showing empty or misleading data for the first 90 days of operation.

**Phase to address:**
Transaction sync phase (earliest v1.2 phase) should document this limitation prominently. Analytics UI phases should include empty-state handling for insufficient data.

---

### Pitfall 4: Transaction Categorization Accuracy Ceiling Without User Feedback Loop

**What goes wrong:**
Automated transaction categorization based on description text achieves 70-85% accuracy at best. SimpleFIN descriptions are institution-dependent -- some banks provide "AMZN MKTP US*ABC123", others provide "AMAZON.COM". Without a mechanism for the user to correct miscategorizations and have those corrections persist and improve future classifications, spending analytics will always feel "wrong."

**Why it happens:**
Developers build a keyword-matching or rule-based categorizer, test it against their own transactions, and declare it done. But descriptions vary wildly across institutions, and edge cases accumulate: a gas station that also sells groceries, a payment app (Venmo/Zelle) that hides the actual merchant, Amazon purchases spanning multiple categories.

**How to avoid:**
- Start with simple keyword-based rules (not ML) for a single-user app. This is sufficient and maintainable.
- Store both `auto_category` and `user_category` on each transaction. Display `COALESCE(user_category, auto_category)` -- following the same pattern as `COALESCE(display_name, name)` already used for accounts.
- Build a "recategorize all matching" feature: when the user corrects "AMZN MKTP" from "Other" to "Shopping", offer to apply that rule to all past and future matching transactions.
- Store user-created rules in a `category_rules` table (pattern, category) that take priority over built-in rules.
- DO NOT attempt ML-based categorization for a single-user self-hosted app. The training data is too small, the complexity is too high, and keyword rules with user overrides will outperform ML at this scale.

**Warning signs:**
- "Uncategorized" being the largest spending category.
- User spending hours manually correcting categories.
- Category pie charts that do not match the user's mental model of their spending.

**Phase to address:**
Spending analytics phase. Categorization is the core of analytics -- it must be built before any charts or reports.

---

### Pitfall 5: Investment Performance Calculation Without Contribution Tracking

**What goes wrong:**
The existing holdings table stores `market_value`, `shares`, and `cost_basis`, but these are point-in-time snapshots from SimpleFIN. To calculate meaningful investment performance (returns vs. contributions), you need to track when money was added vs. when it grew. Without this, a 20% balance increase could mean "great returns" or "you deposited more money."

SimpleFIN does not provide contribution/withdrawal history for investment accounts -- it only provides current holdings state. The balance_snapshots table records daily balances, but cannot distinguish between growth and cash flow.

**Why it happens:**
Balance change is conflated with return. The existing projection engine assumes a static balance growing at a configured APY, which is fine for projections but wrong for retrospective performance measurement.

**How to avoid:**
- Accept that SimpleFIN-only investment performance tracking will be approximate, not precise.
- Use the simple approach: track balance changes day-over-day from `balance_snapshots`. Large single-day jumps (> threshold, e.g., 5% of balance) are likely contributions/withdrawals, not market movement. Flag these for user confirmation.
- Allow manual entry of contributions/withdrawals for investment accounts. Store in a `investment_cash_flows` table.
- Calculate Time-Weighted Return (TWR) when cash flows are known, Simple Return when they are not. Display which method is being used.
- Do NOT display a percentage return without clearly labeling whether it accounts for contributions. "Balance growth: +15%" is honest. "Return: +15%" when half was contributions is misleading.

**Warning signs:**
- Investment "returns" that seem unrealistically high (because they include contributions).
- User confusion about why their "return" does not match their brokerage statement.
- Charts showing investment growth that look like step functions (contribution events) instead of curves (market movement).

**Phase to address:**
Investment tracking phase. This requires explicit design decisions upfront -- retrofitting contribution tracking onto a balance-only system is painful.

---

### Pitfall 6: Corrupting the Balance Snapshot Model When Adding Transaction Sums

**What goes wrong:**
Once transactions exist, there is a temptation to calculate balances by summing transactions instead of using SimpleFIN's reported balance. These numbers WILL NOT match. SimpleFIN's balance is the bank's authoritative number. Transaction sums will be wrong because: (a) you do not have complete history, (b) pending transactions may be included/excluded differently, (c) some transactions (fees, interest) may not appear in the transaction list, and (d) rounding differences accumulate.

**Why it happens:**
Developers reason: "We have all transactions, so we can derive the balance." This is the classic double-entry bookkeeping assumption, but it requires COMPLETE transaction history from account inception, which SimpleFIN cannot provide.

**How to avoid:**
- Treat SimpleFIN's reported balance as the single source of truth for account balances. Never override it with a transaction sum.
- Transaction data is for categorization, analytics, and drill-down -- NOT for balance calculation.
- If displaying "transactions for this period," show them alongside the authoritative balance, not as a replacement.
- Document this separation clearly in code: balance_snapshots (authoritative) vs. transactions (supplementary).
- Add a reconciliation check: if SUM(transactions for day) differs significantly from balance delta, log a warning but do NOT "fix" the balance.

**Warning signs:**
- Balance on dashboard differs from balance on bank website.
- "Reconciliation errors" appearing after adding transaction support.
- Developer instinct to "recalculate" balances from transactions.

**Phase to address:**
Transaction storage phase -- establish this principle in the schema design and document it in code comments before any analytics work begins.

---

### Pitfall 7: Schema Migration Coupling Breaks Existing Users on Upgrade

**What goes wrong:**
The existing app has 6 migrations and real user data. Adding transactions, categories, goals, etc. will require 5-10+ new migrations. If a migration fails mid-way (e.g., adding a foreign key to a table that has orphaned rows, or an ALTER TABLE that SQLite does not support), the database is left in a partially-migrated state. SQLite's limited ALTER TABLE support (no DROP COLUMN before 3.35.0, no RENAME COLUMN before 3.25.0) makes complex schema changes risky.

**Why it happens:**
Developers test migrations against empty databases or freshly-seeded test data, but production databases have edge cases: soft-deleted accounts with no corresponding balance_snapshots, orphaned group_members from deleted accounts, NULL values in columns that a new migration assumes are NOT NULL.

**How to avoid:**
- Always test migrations against a copy of the actual production database, not just test fixtures.
- Use `CREATE TABLE IF NOT EXISTS` for new tables (already the pattern in this codebase -- good).
- Avoid ALTER TABLE for anything beyond ADD COLUMN. If you need to change a column type or constraint, use the SQLite 12-step table rebuild pattern wrapped in a transaction.
- Each migration must be independently reversible. Write down migrations even if you think you will not need them.
- Add a pre-migration backup step to the startup sequence: copy `finance.db` to `finance.db.backup-{migration_version}` before running new migrations.
- Keep migrations small and focused. One table per migration, not "add all transaction tables in one migration."

**Warning signs:**
- Migrations that reference existing table data with assumptions about its state.
- ALTER TABLE statements that do anything other than ADD COLUMN.
- No down migration files for new up migrations.

**Phase to address:**
Every phase that touches the database. But the transaction storage phase is the highest risk because it adds the most schema surface area.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Storing transaction amounts as TEXT (like balances) | Consistency with existing pattern, shopspring/decimal compatible | Slightly harder to query with SQL aggregates, must parse in Go | Always -- this is correct for financial data. Never use REAL for money. |
| Keyword-only categorization (no ML) | Simple, deterministic, debuggable | Will not adapt to new merchants automatically | Always for single-user self-hosted. ML is overkill. |
| Client-side spending chart calculations | No server round-trips, responsive UI | Large transaction volumes could slow the browser | Acceptable up to ~10K transactions per query. Add server-side aggregation if needed later. |
| Skipping OFX/QFX import in v1.2 | Faster delivery, fewer format parsers to maintain | Users cannot backfill history from before SimpleFIN enrollment | Acceptable for v1.2, add in v1.3 if users request it |
| Storing categories as flat strings instead of a hierarchy | Simple schema, easy to query | Cannot do "Food > Restaurants > Fast Food" drill-down | Acceptable for MVP. Migrate to hierarchical later if needed. |
| Using `TEXT` for dates in SQLite | Consistent with existing schema pattern | DATE comparison requires careful formatting (ISO 8601) | Always for this project -- already the established pattern and working correctly. |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| SimpleFIN transactions | Assuming transaction IDs are globally stable across pending-to-cleared transitions | Use `UNIQUE(account_id, external_id)` with a secondary fuzzy matcher for pending-to-cleared updates |
| SimpleFIN transactions | Fetching with `balances-only=1` (current behavior) and expecting transactions | Must use `balances-only=0` or omit the parameter entirely; the current `FetchAccountsWithHoldings` already does this but the `Account` struct lacks a `Transactions` field |
| SimpleFIN rate limits | Hitting the API multiple times per day for different data (balances, transactions, holdings) | Consolidate into a SINGLE daily fetch with `balances-only=0`. Parse balances, holdings, AND transactions from one response. |
| SimpleFIN date ranges | Requesting >90 days of history | Cap `start-date` to `now - 90 days`. For initial backfill, use the full 90-day window once. |
| SQLite concurrent writes | Adding transaction writes to the sync pipeline without considering WAL locking | Keep using WAL mode. Wrap the entire sync cycle (balances + transactions) in a single SQL transaction to avoid partial writes. |
| shopspring/decimal division | Dividing by zero panics (not returns error) | Always guard `decimal.Div()` with `IsZero()` check. Relevant for percentage calculations in spending analytics. |
| shopspring/decimal formatting | Using `.String()` for display, which produces variable decimal places | Use `.StringFixed(2)` for currency display. The existing codebase stores raw decimal strings, which is correct for storage but needs formatting for UI. |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Loading all transactions into browser for client-side analytics | Page load > 3 seconds, browser memory spike | Server-side aggregation endpoints (e.g., `/api/spending/summary?period=monthly`). Send summaries, not raw transactions. | > 5K transactions (roughly 6 months of active use for a single user) |
| Unindexed transaction queries by date range | Slow spending analytics queries | Add `CREATE INDEX idx_transactions_date ON transactions(account_id, posted_date)` | > 10K transactions |
| Recalculating spending categories on every page load | Redundant processing, slow analytics page | Cache category assignments on the transaction row. Only recategorize when rules change (bulk update). | > 2K transactions |
| Single large SQL transaction for bulk transaction import | WAL checkpoint starvation during initial 90-day backfill | Batch inserts in groups of 100-500 rows per SQL transaction | > 1K transactions in a single sync |
| Fetching full transaction detail for list views | Slow API responses, excessive data transfer | Paginate transaction list endpoints. Default 50 per page with cursor-based pagination. | > 500 transactions per account |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Exposing raw SimpleFIN transaction descriptions in exports | Descriptions may contain account numbers, reference codes, or PII from bank metadata | Sanitize or let user review before export. Never include raw `extra` field data from SimpleFIN. |
| Storing export files in a web-accessible directory | Anyone with the URL can download financial data | Generate exports in a temp directory, serve via authenticated endpoint, delete after download or expiry (e.g., 1 hour). |
| Including SimpleFIN access URL or credentials in data exports | Full account access leaked if export file is shared | Explicitly exclude `settings` table data from any export functionality. |
| Not rate-limiting export endpoints | Denial of service via repeated large export generation | Apply the same rate limiting pattern used for `/api/auth/login` (httprate middleware). |
| Category rules revealing spending patterns if app is accessed | Attacker learns user's merchant relationships | The existing JWT auth protects this, but ensure new endpoints follow the same `jwtauth.Authenticator` middleware pattern in the router group. |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Showing "Spending Analytics" page with < 30 days of transaction data | Misleading trends, incomplete picture, user loses trust | Show a clear onboarding state: "Collecting transaction data. Analytics will be available after [date]. Currently have [X] days of data." |
| Requiring manual categorization of every transaction | Hours of tedious work, user abandons the feature | Auto-categorize with keyword rules, show only uncategorized transactions for review. Bulk "apply to all matching" action. |
| Displaying investment "returns" that include contributions | User thinks they earned more than they did, then cross-references with brokerage and loses trust | Label clearly: "Balance change" vs. "Estimated return (excludes contributions)". If contribution tracking is not implemented, say so. |
| Goal tracking with no progress visualization | User sets a goal and forgets about it | Show goal progress on the dashboard (progress bar or percentage). Send alert when goal is reached (reuse existing alert engine). |
| Export that produces a single massive file | Large file is slow to generate, hard to open in Excel | Offer date-range selection for exports. Default to "last 12 months" not "all time." |
| Spending charts defaulting to categories the user does not care about | User sees "Uncategorized: 60%" as the dominant category | Default view should be the top N categories by amount, with "Other" grouping the rest. Let user configure which categories to show. |

## "Looks Done But Isn't" Checklist

- [ ] **Transaction sync:** Often missing pending transaction handling -- verify that pending transactions are stored with `pending=true` and updated (not duplicated) when they clear.
- [ ] **Spending analytics:** Often missing multi-currency handling -- verify that all amounts are converted to a display currency before summing (the existing codebase is USD-only, but SimpleFIN can return other currencies).
- [ ] **Category rules:** Often missing rule ordering and priority -- verify that user-created rules take precedence over built-in rules, and that more specific rules match before general ones.
- [ ] **Data export:** Often missing proper CSV escaping -- verify that transaction descriptions containing commas, quotes, and newlines are properly escaped. Test with real bank descriptions like `"PAYMENT - THANK YOU / REF #12345"`.
- [ ] **Goal tracking:** Often missing the "what happens when the goal is met" state -- verify there is a completion state, celebration/notification, and option to archive or reset.
- [ ] **Investment tracking:** Often missing cost basis tracking over time -- verify that `cost_basis` from SimpleFIN is stored historically, not just overwritten with the latest value, so gain/loss calculations are accurate.
- [ ] **Transaction search:** Often missing search across descriptions AND categories -- verify full-text search covers both fields, not just one.
- [ ] **Date handling:** Often missing timezone awareness in transaction dates -- verify that `posted` (Unix timestamp) is converted to the user's local date consistently, not sometimes UTC and sometimes local.

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Duplicate transactions in database | MEDIUM | Write a dedup migration: GROUP BY (account_id, external_id), keep the row with the latest fetched_at, delete the rest. Recalculate any cached analytics. |
| Corrupted balance data from transaction-sum override | HIGH | Restore balance_snapshots from backup. If no backup, re-sync with SimpleFIN (only gets last 90 days). Older data is permanently lost. |
| Wrong category assignments (bad rules deployed) | LOW | Store `auto_category` and `user_category` separately. Reset `auto_category` by re-running rules. User overrides (`user_category`) are preserved. |
| Migration failure leaving DB in partial state | HIGH | Pre-migration backup restores cleanly. Without backup: manually inspect `schema_migrations` table, determine which migration failed, apply fixes manually. SQLite's single-file nature makes backup/restore simpler than other DBs. |
| Investment returns displayed incorrectly (includes contributions) | LOW | UI-only fix: change label from "Return" to "Balance change." Add disclaimer text. No data migration needed. |
| SimpleFIN rate limit exceeded, token disabled | MEDIUM | Wait for token reset (typically 24 hours). If permanently disabled, user must reclaim a new setup token. Existing data is preserved locally. |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| SimpleFIN client does not parse transactions | Transaction Storage (first phase) | Unit test verifying `Transaction` struct unmarshals from SimpleFIN JSON response |
| Transaction deduplication | Transaction Storage (first phase) | Integration test: sync same transaction twice, verify only one row exists. Sync pending then cleared, verify update not duplicate. |
| 90-day data window limitation | Transaction Storage + Analytics UI | Analytics UI shows "data since [date]" label. First sync requests 90-day window. Test that date range > 90 days does not break fetch. |
| Categorization accuracy | Spending Analytics phase | User can override any category. Override persists across re-syncs. "Apply to all matching" works for historical transactions. |
| Investment performance without contributions | Investment Tracking phase | Returns display is labeled "Balance change" not "Return" unless cash flows are manually tracked. |
| Balance vs. transaction sum divergence | Transaction Storage (first phase) | Code review: no code path derives balance from transaction sums. Balance_snapshots remain the authority. |
| Schema migration risk | Every phase | Pre-migration backup runs before each new migration. Down migration exists for every up migration. Migration tested against production DB copy. |
| Export security | Data Export phase | Export endpoint requires JWT auth. No files persist on disk after download. Settings table excluded from export. |

## Sources

- [SimpleFIN Protocol Specification](https://www.simplefin.org/protocol.html) -- Transaction fields: id, posted, amount, description, pending
- [SimpleFIN Protocol on GitHub](https://github.com/simplefin/simplefin.github.com/blob/master/protocol.md) -- Full spec with JSON examples
- [SimpleFIN Developer Guide](https://beta-bridge.simplefin.org/info/developers) -- Rate limits (24 req/day), 90-day max range, daily update cadence
- [Actual Budget SimpleFIN duplicate transactions](https://github.com/actualbudget/actual/issues/2519) -- Real-world dedup issues
- [Actual Budget cross-account mirror transactions](https://github.com/actualbudget/actual/issues/7015) -- Mercury bank mirror transaction bug
- [Firefly III SimpleFIN import issues](https://github.com/firefly-iii/firefly-iii/issues/10550) -- Currency handling, duplication on re-import
- [Actual Budget transaction importing docs](https://actualbudget.org/docs/transactions/importing/) -- Dedup strategies: OFX IDs, date+amount+payee matching
- [CockroachDB: Idempotency in Finance](https://www.cockroachlabs.com/blog/idempotency-in-finance/) -- Idempotency key patterns for financial transactions
- [shopspring/decimal Go package](https://pkg.go.dev/github.com/shopspring/decimal) -- Arbitrary-precision decimal, division-by-zero panic risk
- [SQLite WAL mode documentation](https://www.sqlite.org/wal.html) -- Checkpoint starvation, concurrent read/write behavior
- [Kitces.com: TWR vs IRR calculations](https://www.kitces.com/blog/twr-dwr-irr-calculations-performance-reporting-software-methodology-gips-compliance/) -- Time-Weighted vs Money-Weighted return methodologies
- [Stripe: Transaction Categorization Guide](https://stripe.com/resources/more/what-is-transaction-categorization-a-guide-to-transaction-taxonomy-and-its-benefits) -- 70-85% accuracy ceiling, MCC code limitations
- [Uncat: 5 Common Transaction Categorization Mistakes](https://www.uncat.com/blog/transaction-categorization) -- Over-reliance on automation, evolving merchant names
- Existing codebase: `internal/simplefin/client.go`, `internal/sync/sync.go`, migration files 000001-000006

---
*Pitfalls research for: Adding features to Go/React/SQLite personal finance dashboard with SimpleFIN*
*Researched: 2026-03-17*
