# Feature Research: v1.1 Enhancements

**Domain:** Self-hosted personal finance dashboard -- 7 new features for existing product
**Researched:** 2026-03-15
**Confidence:** HIGH -- codebase fully inspected, feature scope well-defined, domain patterns well-understood

## Context

This research covers 7 features being added to an existing, working v1.0 product. The codebase is a Go backend (go-chi, SQLite, SimpleFIN sync) with a React + TypeScript frontend (Recharts, Tailwind CSS). The existing schema has `accounts`, `balance_snapshots`, `sync_log`, and `settings` tables. The frontend has a Dashboard page with PanelCards, BalanceLineChart, NetWorthDonut, and a Settings page.

The 7 target features are: crypto aggregation, account renaming, growth rate indicators, net worth drill-down, sync failure diagnostics, alert rules with email, and projected net worth with income modeling.

---

## Feature-by-Feature Analysis

### 1. Crypto Aggregation by Institution

**Category:** Table Stakes (for crypto holders)
**Complexity:** MEDIUM
**Dependencies:** Existing accounts table (has `org_name`, `org_slug` fields)

**What users expect:**
- Multiple crypto sub-accounts from the same institution (e.g., Coinbase BTC wallet, Coinbase ETH wallet) should appear as a single combined line in the Investments panel
- The combined balance history should be a sum of the constituent accounts per day
- Individual sub-accounts should still be visible somewhere (expand/collapse or tooltip)
- Only crypto accounts are aggregated this way -- bank accounts from the same institution remain separate because they serve different purposes (checking vs savings)

**UX patterns from the domain:**
- Empower (Personal Capital) groups holdings by institution with expandable rows
- Monarch Money shows institution-level totals with expand-to-detail
- The standard pattern is a collapsible row: "Coinbase -- $12,340" that expands to show "BTC Wallet: $8,200, ETH Wallet: $4,140"

**Implementation notes:**
- The existing `accounts` table already stores `org_name` and `org_slug`. Aggregation is a query-time grouping, not a schema change.
- Need a way to identify crypto accounts specifically. Current `InferAccountType()` in sync.go maps to "investment" -- it does not distinguish crypto from brokerage/retirement. Options: (a) add a `sub_type` column to accounts, (b) infer from SimpleFIN org data or account name keywords, (c) let the user tag accounts as crypto in settings.
- Option (c) is most reliable: user marks which accounts are crypto in a new account settings UI. This pairs naturally with the account renaming feature.
- Balance history aggregation requires summing snapshots by `org_slug` + date for crypto-tagged accounts, while leaving non-crypto accounts ungrouped.

**What "done" looks like:**
- Investments panel shows "Coinbase" with combined balance instead of 3 separate wallet lines
- Clicking/expanding shows individual wallet balances
- Balance history chart for investments reflects the aggregated view
- User can designate which accounts are crypto in settings

---

### 2. Account Renaming

**Category:** Table Stakes
**Complexity:** LOW
**Dependencies:** None (standalone foundation feature)

**What users expect:**
- SimpleFIN returns institution-provided account names like "SAVINGS PLUS ACCOUNT" or "BROKERAGE-12345." Users want to rename these to "Emergency Fund" or "Vanguard Roth IRA."
- The custom name replaces the original everywhere: panel cards, charts, dropdowns, alert rule builder, projections page.
- The original institution name should still be visible somewhere (e.g., in settings or as a subtitle) so the user remembers which account maps to what.
- Renaming should persist across syncs. SimpleFIN upserts should NOT overwrite the custom name.

**UX patterns from the domain:**
- Every major finance app (Empower, Monarch, Copilot, YNAB) supports account nicknames
- The standard pattern: a settings/accounts page with an inline edit field next to each account
- The original name is shown as a secondary label or tooltip

**Implementation notes:**
- Add a `display_name` column (nullable) to the `accounts` table. When non-null, use it everywhere instead of `name`.
- The sync upsert in `processAccount()` currently overwrites `name` on every sync. The `display_name` column is separate, so syncs update `name` (the institution name) without touching `display_name`.
- Backend: modify `GetAccounts()` and `GetSummary()` (and eventually all new endpoints) to return `display_name` alongside or in place of `name`.
- Frontend: everywhere `account.name` is rendered, use `account.display_name || account.name`.
- New API endpoint: `PUT /api/accounts/:id/name` to set `display_name`.

**What "done" looks like:**
- Settings page has an "Accounts" section listing all accounts with editable name fields
- User types a custom name, saves it, and it appears everywhere immediately
- Syncs do not overwrite the custom name
- Original institution name remains visible as secondary text in settings

---

### 3. Growth Rate Indicators

**Category:** Table Stakes (for a dashboard claiming to show financial health)
**Complexity:** LOW
**Dependencies:** Existing `balance_snapshots` table with historical data

**What users expect:**
- Each panel card (Liquid, Savings, Investments) shows a percentage change indicator like "+2.3% this month" or "-$450 this week"
- Green for positive, red for negative, gray for zero/no-change
- The time period should be sensible: "this month" (comparing today's balance to the first snapshot of the current month) is the most common default
- The calculation must be clearly defined: `(current - previous) / |previous| * 100`

**UX patterns from the domain:**
- Empower shows portfolio gain/loss in dollars and percentage with red/green coloring
- Robinhood's signature feature is the prominent "+$X.XX (+Y.YY%)" badge
- Monarch Money shows monthly net worth change as a percentage
- The universal pattern: a small badge or subtitle on each card showing the change, with up/down arrow or +/- prefix and color coding

**Implementation notes:**
- Backend: new field in the summary response or a dedicated endpoint. Calculate by comparing the latest snapshot sum to the snapshot sum from 30 days ago (or start of month) for each panel group.
- The existing `balance_snapshots` table already has the data. The query is: get the sum of latest balances per panel, get the sum of balances from 30 days ago per panel, compute the delta.
- Edge cases: accounts added mid-month (new account has no 30-day-ago snapshot), accounts removed (stale account cleanup already runs). Handle by only comparing accounts that existed at both points in time, OR by comparing total panel values regardless of individual account composition (the latter is simpler and more meaningful -- "my total savings grew by X%").
- Frontend: add a subtitle element to PanelCard showing the percentage change with appropriate color.

**What "done" looks like:**
- Each of the 3 panel cards shows "+X.X% this month" or "-X.X% this month" in green/red
- Percentage is calculated from total panel value 30 days ago vs now
- Handles edge cases: no history yet (show nothing), zero previous value (show "New")

---

### 4. Net Worth Drill-Down Page

**Category:** Differentiator
**Complexity:** MEDIUM
**Dependencies:** Existing balance history endpoint, existing net worth donut

**What users expect:**
- Clicking the "Net Worth" donut on the dashboard navigates to a dedicated page
- The page shows a historical net worth line chart (total over time)
- Breakdown by panel (liquid, savings, investments) as stacked or multi-line chart
- Monthly or weekly aggregation options for the time axis
- Key statistics: all-time high, all-time low, average monthly growth rate, current net worth

**UX patterns from the domain:**
- ProjectionLab shows net worth over time with scenario overlays
- Empower has a dedicated "Net Worth" tab with a large area chart, plus a table showing asset vs liability breakdown by month
- Monarch Money shows net worth trend with monthly columns
- The standard pattern: a large time-series chart at the top, summary statistics below, and optional breakdown table

**Implementation notes:**
- Backend: new endpoint `GET /api/net-worth/history?days=N` that returns time-series data of total net worth (sum of all accounts, all types) per day, plus per-panel breakdown per day.
- This is similar to the existing `GetBalanceHistory` but adds a `netWorth` total series and potentially finer-grained breakdowns.
- Frontend: new route `/net-worth` with a page component. Reuse `BalanceLineChart` pattern (Recharts `LineChart`) but with additional series. Add a statistics card below the chart.
- Statistics: computed from the history data on the frontend -- min, max, start value, end value, percentage change over visible period.
- Time range selector: 30d, 90d, 6m, 1y, all. The existing `?days=N` pattern in `GetBalanceHistory` already supports this.

**What "done" looks like:**
- Net worth donut on dashboard is clickable, navigates to `/net-worth`
- Full-page chart showing net worth over time with multi-line breakdown
- Time range picker (30d, 90d, 6m, 1y, all)
- Statistics summary: current net worth, change over selected period ($ and %), all-time high
- Back navigation to dashboard

---

### 5. Sync Failure Diagnostics

**Category:** Table Stakes (for operational awareness)
**Complexity:** LOW
**Dependencies:** Existing `sync_log` table (already stores errors)

**What users expect:**
- The settings page shows a log of recent sync attempts (last 10-20)
- Each entry shows: timestamp, status (success/failure), accounts fetched, accounts failed, and error message if any
- Failed syncs are visually distinct (red text or warning icon)
- The user can see if syncs have been consistently failing (e.g., expired SimpleFIN token)

**UX patterns from the domain:**
- Firefly III shows import/sync logs with timestamps and error details
- Home Assistant shows integration health with a list of recent operations and their status
- The standard pattern: a simple table or timeline of recent operations with status badges

**Implementation notes:**
- The `sync_log` table already exists with `started_at`, `finished_at`, `accounts_fetched`, `accounts_failed`, and `error_text` columns. All the data is already being captured.
- Backend: new endpoint `GET /api/sync/log` returning the last N entries from `sync_log` ordered by `id DESC`.
- Frontend: add a "Sync History" section to the Settings page, below the existing Sync Status card. Render as a compact table or list.
- Enhancement: show the error_text in an expandable row for failed syncs (errors can be long).

**What "done" looks like:**
- Settings page has a "Sync History" section showing the last 10 sync operations
- Each entry shows timestamp, success/failure badge, and account counts
- Failed syncs show an expandable error message
- Users can immediately diagnose expired tokens, network issues, or SimpleFIN outages

---

### 6. Alert Rules with Email Notifications

**Category:** Differentiator
**Complexity:** HIGH
**Dependencies:** Account renaming (alerts reference accounts by name), SMTP configuration

**What users expect:**
- User builds a rule using an expression like: `Liquid > 5000` or `Savings + Investments >= 100000`
- Available operands: panel totals (Liquid, Savings, Investments) and individual accounts (by name)
- Operators between operands: `+`, `-`
- Comparison operators: `<`, `<=`, `>`, `>=`, `==`
- Compared against a numeric threshold value
- The alert fires ONCE when the threshold is crossed (not on every sync while the condition is true)
- The alert fires ONCE again when the condition recovers (crosses back)
- Email notification includes: which rule triggered, current value, threshold value, direction (crossed above/below)
- SMTP or API provider configuration in settings (host, port, username, password, from address, to address)

**UX patterns from the domain:**
- Banking apps (Chase, BofA) offer simple "balance below $X" alerts -- basic but limited
- Empower offers "spending exceeded $X in category Y" -- more sophisticated
- IFTTT/Zapier-style expression builders are familiar to power users
- The key UX insight from the domain: avoid alert fatigue. The "fire once on crossing, once on recovery" design is correct -- it prevents spam. Most finance app alerts that fire on every check get muted within a week.

**Implementation notes:**
- Schema: new `alert_rules` table with columns: `id`, `name`, `expression` (stored as JSON), `comparison_op`, `threshold`, `enabled`, `last_state` (above/below/unknown), `last_triggered_at`, `created_at`.
- Expression format as JSON: `[{"type": "bucket", "value": "liquid"}, {"op": "+"}, {"type": "account", "value": "account-id-123"}]`. This is a simple RPN or infix token list.
- Schema: new `email_config` settings (store in the existing `settings` table as key-value pairs: `smtp_host`, `smtp_port`, `smtp_user`, `smtp_pass`, `smtp_from`, `smtp_to`).
- Backend evaluation: after each sync completes, evaluate all enabled alert rules against current balances. Compare result to `last_state`. If state changed (was below, now above -- or vice versa), send email and update `last_state`.
- Email sending: use `wneessen/go-mail` for SMTP -- it is actively maintained, minimal dependencies, and handles TLS/STARTTLS properly. The Go standard library `net/smtp` is frozen and missing features.
- Frontend: expression builder UI in a new Alerts section. Could be on Settings page or a dedicated `/alerts` route. The builder is a series of dropdowns and inputs: [bucket/account dropdown] [+/-] [bucket/account dropdown] [comparison] [number].
- Testing the alert: provide a "Test Email" button that sends a test message to verify SMTP config.

**What "done" looks like:**
- Settings page has SMTP/email configuration section
- "Test Email" button to verify configuration
- Alerts page with a list of rules and a builder to create new ones
- Expression builder: select buckets/accounts, combine with +/-, compare with operator, set threshold
- Alerts evaluate after each sync, fire email on state transitions only
- Email body includes rule name, current computed value, threshold, and direction of crossing

---

### 7. Projected Net Worth with Income Modeling

**Category:** Differentiator
**Complexity:** HIGH
**Dependencies:** Account renaming (projections reference accounts by name), net worth drill-down (provides the historical context alongside projections)

**What users expect:**
- A dedicated tab or page showing projected net worth over a custom time horizon (1yr, 5yr, 10yr, 30yr)
- Per-account configuration: savings accounts get APY, investment accounts get expected growth rate
- Reinvestment toggle per account: compound (growth is reinvested) vs simple (growth is extracted)
- Income modeling: user enters annual income, monthly savings percentage, and allocation across accounts
- The projection chart shows a fan or line extending from current balances into the future
- Vanguard or brokerage holdings detail if available from SimpleFIN (this is stretch -- SimpleFIN only provides balance, not holdings)

**UX patterns from the domain:**
- ProjectionLab is the gold standard: scenario-based projections with Monte Carlo simulations, income modeling, expense modeling, tax modeling. It is a dedicated product -- our version should be dramatically simpler.
- Empower's Retirement Planner shows projected net worth based on savings rate and expected returns
- CalcXML and similar calculators show compound growth curves
- The standard pattern: input your assumptions (growth rate, savings rate), see a projected line chart extending from current values

**Implementation notes:**
- Schema: new `account_projections` table: `account_id`, `apy_or_growth_rate` (decimal), `reinvestment` (boolean, default true), `created_at`, `updated_at`.
- Schema: new `income_model` settings (stored in `settings` table or a new table): `annual_income`, `savings_pct`, and an `income_allocations` table: `account_id`, `allocation_pct`.
- Schema: `projection_settings` for time horizon default.
- Backend: `GET /api/projections?horizon_years=N` endpoint that computes projected balances per account per month (or year) over the horizon. Calculation:
  - For each account: start with current balance. Each month, apply `growth_rate / 12` (compound if reinvestment=true, simple if false). Add monthly income allocation.
  - Sum all accounts per month for total projected net worth.
- Compound formula per month: `balance * (1 + rate/12) + monthly_contribution`
- Simple formula per month: `balance + (initial_balance * rate/12) + monthly_contribution`
- Frontend: new route `/projections` with:
  - Configuration panel: per-account APY/growth rate, reinvestment toggle
  - Income section: annual income, savings %, allocation sliders
  - Time horizon selector
  - Projection chart (Recharts LineChart with projected line, optionally shaded confidence area)
- Vanguard holdings: SimpleFIN protocol is `balances-only=1` -- it does NOT provide individual holdings, cost basis, or fund detail. This is a protocol limitation. The "Vanguard holdings detail" requirement should be scoped as: if the user has multiple Vanguard accounts (e.g., Roth IRA, Traditional IRA, Brokerage), each appears separately and can have its own growth rate. Do NOT attempt to fetch individual fund holdings -- that data is not available via SimpleFIN.
- All projection parameters must be persisted in the database so the user does not re-enter them on each visit.

**What "done" looks like:**
- Projections page at `/projections` accessible from nav bar
- Per-account growth rate / APY configuration (editable table)
- Reinvestment toggle per account (compound vs simple)
- Income modeling: annual income, savings %, allocation across accounts
- Time horizon selector (1yr, 5yr, 10yr, 30yr)
- Projection chart showing current balances extending into the future
- All settings persisted in database

---

## Feature Landscape Summary

### Table Stakes (Users Expect These)

Features that are baseline expectations for a v1.1 of a finance dashboard.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Account renaming | Every finance app lets you nickname accounts. Institutional names are cryptic. | LOW | Foundation feature -- adds `display_name` column. Build first since other features reference account names. |
| Growth rate indicators | A dashboard without trend indicators feels static. "+2.3% this month" is the minimal trend signal. | LOW | Query compares current panel totals to 30 days ago. Display as colored badge on PanelCard. |
| Sync failure diagnostics | Users WILL encounter expired SimpleFIN tokens. Without diagnostics, they see stale data with no explanation. | LOW | The `sync_log` table already captures everything. This is purely a new endpoint + frontend display. |
| Crypto aggregation | Crypto users with multiple wallets at one exchange expect them grouped. Showing 5 Coinbase lines clutters the panel. | MEDIUM | Requires a way to tag accounts as crypto. Query-time grouping by `org_slug` for crypto-tagged accounts. |

### Differentiators (Competitive Advantage)

Features that go beyond expectations and add distinctive value.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Net worth drill-down | Most self-hosted tools show net worth as a single number. A dedicated page with historical trends and statistics turns a number into insight. | MEDIUM | New `/net-worth` route. Reuses existing snapshot data with a new aggregation query. |
| Alert rules with email | No self-hosted finance dashboard does threshold-based alerts with expression building. This is power-user territory that commercial tools charge for. | HIGH | Requires: expression evaluator, state machine for threshold crossing, SMTP integration, builder UI. Most complex feature in the set. |
| Projected net worth | Turns a backward-looking dashboard into a forward-looking planning tool. ProjectionLab charges $100/yr for this capability. | HIGH | Compound/simple growth modeling, income allocation, per-account rates. Significant frontend work for the configuration UI and projection chart. |

### Anti-Features (Explicitly Avoid)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Real-time crypto price feeds | "My crypto balance should update every minute." | SimpleFIN provides daily snapshots, not real-time prices. Adding a second data source (CoinGecko, etc.) creates sync conflicts, stale data races, and external API dependency for a self-hosted tool. | Daily snapshot cadence is sufficient. Crypto aggregation groups the wallets; the balance updates daily like everything else. |
| Monte Carlo simulation for projections | "Show me confidence bands and probability of success." | Monte Carlo requires statistical modeling (standard deviation of returns, correlation between assets, inflation modeling). This is a dedicated product scope (ProjectionLab). It dramatically increases complexity without proportional value for a simple dashboard. | Deterministic projection with user-specified growth rates. Clear, simple, honest. Users who want Monte Carlo already use ProjectionLab. |
| Holdings-level investment detail | "Show me my individual stocks/funds within each brokerage account." | SimpleFIN protocol with `balances-only=1` does not return holdings data. Even without this flag, holdings support varies wildly by institution. Building a holdings UI for inconsistent data creates a half-broken feature. | Show account-level balances (which are reliable). User sets growth rate per account in projections rather than per-holding. |
| SMS notifications for alerts | "I want text messages, not just email." | SMS requires a paid service (Twilio, etc.), adds a recurring cost dependency to a free self-hosted tool, and adds significant complexity (phone number validation, carrier issues, international support). | Email is universal, free (with any SMTP server), and sufficient for daily-cadence financial alerts. Document that users can use email-to-SMS gateways if they want texts. |
| Alert rule templates / presets | "Give me one-click 'emergency fund alert' setup." | Templates assume financial patterns that vary widely by user. A "low balance" template for one user is normal for another. Templates create a false sense of completeness and add maintenance burden. | The expression builder is simple enough that creating rules is fast. Document a few example rules in the UI help text instead. |

---

## Feature Dependencies

```
Account Renaming (display_name column)
    |
    +--enhances--> Crypto Aggregation (grouped accounts show custom names)
    +--enhances--> Alert Rules (expression builder shows custom account names)
    +--enhances--> Projected Net Worth (account config table shows custom names)

Crypto Aggregation
    +--requires--> Account tagging mechanism (crypto flag per account)
    +--requires--> Account renaming (account settings UI exists to add the crypto tag)

Growth Rate Indicators
    +--standalone-- (only needs existing balance_snapshots data)

Net Worth Drill-Down
    +--standalone-- (only needs existing balance_snapshots data)
    +--enhances--> Dashboard donut (donut becomes clickable link)

Sync Failure Diagnostics
    +--standalone-- (only needs existing sync_log data)

Alert Rules with Email
    +--requires--> SMTP configuration (new settings section)
    +--enhances--> Account Renaming (rules reference display names)
    +--requires--> Sync pipeline hook (evaluate rules after each sync)

Projected Net Worth
    +--requires--> Per-account growth rate configuration (new table)
    +--requires--> Income modeling configuration (new table/settings)
    +--enhances--> Account Renaming (projection config shows display names)
    +--enhances--> Net Worth Drill-Down (historical + projected on one view)
```

### Dependency Notes

- **Account renaming should be built first:** It adds the account settings UI and `display_name` column that crypto aggregation, alerts, and projections all reference. Without it, other features would need to build their own account management UIs.
- **Crypto aggregation depends on account renaming:** The account settings UI from renaming is where users tag accounts as crypto. Building both together avoids a separate "account management" UI.
- **Growth indicators and sync diagnostics are independent:** They have zero dependencies on other v1.1 features and can be built in any order.
- **Net worth drill-down is independent:** It only needs existing data and can be built anytime.
- **Alert rules require SMTP config first:** The email configuration must exist before alerts can be tested or used. The sync pipeline must call the alert evaluator after each sync.
- **Projected net worth is the most dependent feature:** It needs account-level configuration (growth rates, reinvestment) and income modeling -- both new data models. It benefits from having account renaming done so the UI shows human-readable names.

---

## Phase Ordering Recommendation

Based on dependencies and complexity:

### Phase 1: Foundation + Quick Wins
- Account renaming (LOW -- foundation for everything)
- Growth rate indicators (LOW -- high visibility, standalone)
- Sync failure diagnostics (LOW -- standalone, high utility)

### Phase 2: Data Enrichment
- Crypto aggregation (MEDIUM -- depends on account settings from Phase 1)
- Net worth drill-down (MEDIUM -- standalone but pairs well with growth indicators)

### Phase 3: Advanced Features
- Alert rules with email (HIGH -- requires new subsystems: expression evaluator, SMTP, state machine)
- Projected net worth with income modeling (HIGH -- requires new data models, complex frontend)

**Rationale:** Phase 1 delivers immediate visible value with low risk. Phase 2 enriches the data model. Phase 3 tackles the two highest-complexity features when the foundation is solid.

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority | Phase |
|---------|------------|---------------------|----------|-------|
| Account renaming | HIGH | LOW | P1 | 1 |
| Growth rate indicators | HIGH | LOW | P1 | 1 |
| Sync failure diagnostics | MEDIUM | LOW | P1 | 1 |
| Crypto aggregation | MEDIUM | MEDIUM | P1 | 2 |
| Net worth drill-down | HIGH | MEDIUM | P1 | 2 |
| Alert rules with email | MEDIUM | HIGH | P2 | 3 |
| Projected net worth | MEDIUM | HIGH | P2 | 3 |

**Priority key:**
- P1: Should have in v1.1 -- clearly scoped, well-understood
- P2: Should have in v1.1 -- high complexity, may need iteration

---

## Competitor Feature Analysis

| Feature | Empower | Monarch Money | Firefly III | ProjectionLab | Our Approach |
|---------|---------|---------------|-------------|---------------|--------------|
| Crypto aggregation | Groups by institution | Groups by institution | Manual categories | N/A | Auto-group by org_slug for crypto-tagged accounts |
| Account renaming | Yes, inline edit | Yes, inline edit | Yes | N/A | `display_name` column, settings page edit |
| Growth indicators | Portfolio gain/loss ($ and %) | Monthly net worth change % | Manual tagging | Historical performance | Panel-level % change badge, 30-day comparison |
| Net worth drill-down | Dedicated tab with area chart | Net worth page with monthly bars | Net worth report | Full scenario modeling | Dedicated page with line chart + statistics |
| Sync diagnostics | Connection health in settings | Connection status page | Import log viewer | N/A | sync_log table displayed in settings |
| Alert rules | Spending alerts (category-based) | Budget alerts only | Rule-based notifications | N/A | Expression builder: buckets + accounts with math operators |
| Net worth projection | Retirement planner with assumptions | None | None | Core product (Monte Carlo) | Deterministic projection with per-account rates and income modeling |

**Key competitive insight:** No self-hosted finance dashboard currently offers both alert rules with expression building AND net worth projection. Firefly III has basic notifications. ProjectionLab does projections but is a separate SaaS product, not an integrated dashboard. Building both in a single self-hosted tool is genuinely differentiated.

---

## Sources

- Codebase inspection: `internal/db/migrations/000001_init.up.sql`, `internal/sync/sync.go`, `internal/api/handlers/summary.go`, `internal/api/handlers/accounts.go`, `internal/api/handlers/history.go`, `internal/simplefin/client.go`, `frontend/src/pages/Dashboard.tsx`, `frontend/src/components/PanelCard.tsx`, `frontend/src/components/NetWorthDonut.tsx`
- [Fintech design guide with patterns that build trust (Eleken, 2026)](https://www.eleken.co/blog-posts/modern-fintech-design-guide)
- [From Alerts to Intelligence: How Personal Finance Apps Are Evolving (ADD Magazine)](https://addmagazine.co.uk/from-alerts-to-intelligence-how-personal-finance-apps-are-evolving-and-what-it-means-for-software-strategy/)
- [Notification System Design: Architecture and Best Practices (MagicBell)](https://www.magicbell.com/blog/notification-system-design)
- [ProjectionLab -- Modern Net Worth Calculator and Tracking](https://projectionlab.com/net-worth)
- [go-mail (wneessen/go-mail) -- Go email library](https://github.com/wneessen/go-mail)
- [Go Send Email Tutorial (Mailtrap, 2026)](https://mailtrap.io/blog/golang-send-email/)
- [Best Net Worth Calculators in 2026 (Rob Berger)](https://robberger.com/net-worth-calculators/)
- SimpleFIN protocol specification: https://www.simplefin.org/protocol.html
- Training knowledge of Empower, Monarch Money, Firefly III, Actual Budget, YNAB, Copilot (knowledge cutoff May 2025)

---
*Feature research for: Finance Visualizer v1.1 -- 7 new features*
*Researched: 2026-03-15*
