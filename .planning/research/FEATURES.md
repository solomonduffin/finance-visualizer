# Feature Research

**Domain:** Self-hosted personal finance dashboard (read-only, single-user)
**Researched:** 2026-03-15
**Confidence:** MEDIUM — based on training knowledge of the domain (Mint, YNAB, Personal Capital/Empower, Firefly III, Actual Budget, Lunch Money, Copilot, Fina). External search unavailable; knowledge cutoff August 2025.

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Aggregated balance summary (all accounts) | Core promise: "all money in one place." Without this, there's no dashboard. | LOW | The single-number net worth / liquid balance view. Computed from fetched account data. |
| Account list with individual balances | Users need to verify numbers are correct per-account before trusting the total. | LOW | Flat list under each panel (liquid, savings, investments). |
| Balance history charts | Without trend lines, the tool is a point-in-time balance viewer, not a dashboard. Every finance tool ships time-series charts. | MEDIUM | Requires daily snapshot storage. Line chart per account or panel total. |
| Net worth breakdown visualization | Users want to see allocation at a glance. Pie/donut chart is the near-universal pattern. | MEDIUM | Segments: liquid, savings, investments. Optionally drill into investment sub-types. |
| Pending transaction handling | Checking account balances are meaningless without pending debits/credits. Users will immediately notice if the number is "wrong" vs their bank app. | LOW | SimpleFIN provides pending flag per transaction. Subtract pending credit-card charges from liquid. |
| Dark/light mode | Self-hosted tools are used late at night. Omitting dark mode is noticeable friction. | LOW | CSS variables + toggle. |
| Password protection | Without auth, the dashboard is open on the local network. Even single-user self-hosters expect a gate. | LOW | Simple bcrypt-hashed password + session cookie. Not multi-user. |
| Data freshness indicator | Users need to know if the numbers are from "now" or from 3 days ago. Every finance aggregator shows last-sync timestamp. | LOW | "Last updated: 2 hours ago" label on the dashboard. |
| Loading / empty states | On first run (before any sync), the UI must communicate status, not show broken zeroes. | LOW | Skeleton UI or "sync in progress" message. |
| Mobile-responsive layout | Self-hosted dashboards are often checked on a phone. Completely broken mobile = unusable. | MEDIUM | Responsive CSS. Dashboard panels stack vertically on small screens. |

### Differentiators (Competitive Advantage)

Features that set the product apart. Not required, but valuable.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Liquid balance = checking minus credit cards (including pending) | Most tools show accounts in isolation. This computed "true spendable cash" number is the single most useful financial figure and almost no tool surfaces it prominently. | LOW | The formula is the differentiator, not the tech. Defined clearly in PROJECT.md. |
| Investment panel with APY / growth-loss per account | Brokerage/retirement tools require separate apps. Aggregating performance data next to liquid and savings gives a unified picture competitors split across multiple views. | MEDIUM | Growth/loss requires storing cost-basis or prior-value snapshot. SimpleFIN may not provide cost-basis — surface what's available (current value vs. prior snapshot). |
| Panel-level drill-down (not just a flat account list) | Organizing accounts into semantic panels (liquid, savings, investments) and letting users drill down preserves the "one glance" promise while offering detail on demand. | MEDIUM | Routing: `/` (dashboard) → `/liquid`, `/savings`, `/investments`. Each panel gets its own detail page. |
| Savings APY display | Savings accounts have APY attached. Displaying it next to the balance helps users notice rate disparities across banks — a genuine insight. | LOW | SimpleFIN account metadata may include interest rate. Fallback: manual annotation in config. |
| Investment performance over time chart | Balance-over-time for investments is more meaningful than for checking. Shows compounding visually. Most self-hosted tools don't do this well. | MEDIUM | Same daily snapshot infrastructure as balance charts; scope to investment accounts only. |
| SimpleFIN-native (no credential storage) | Tools like Firefly III require you to manually import or use Plaid (which holds credentials). SimpleFIN's read-only token model is meaningfully more privacy-preserving. This is a selling point for self-hosters. | LOW | Surface this in the UI — "read-only, no credentials stored" label in settings/about. |
| Crypto account support | Most personal finance tools ignore crypto or handle it badly. Including it in the investments panel alongside brokerage and retirement normalizes it in the portfolio view. | LOW | If SimpleFIN exposes crypto account data, it's free. If not, manual balance entry is an option. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Budgeting / budget categories | Finance tools "should" have budgets. Every user who opens a finance app expects it. | Budgeting requires transaction categorization, category management UI, budget-vs-actual tracking, and monthly rollover logic. It's a separate product scope that doubles complexity and risks making the dashboard mediocre at both jobs. PROJECT.md explicitly defers this. | Nail the read-only net-worth view first. Add budgeting in a future milestone after validating the core. |
| Transaction categorization | Useful for budgeting; users sometimes request it for "understanding spending." | Categorization requires: category taxonomy (hundreds of merchant names), ML or manual rules engine, UI for editing categories, and ongoing maintenance as merchants change names. None of this is needed for a pure balance/net-worth dashboard. | Show transaction lists in drill-down without categories. Raw data is honest. |
| Real-time / on-demand sync | "Why can't I refresh manually?" — users always ask this. | SimpleFIN rate-limits requests. More importantly, financial data doesn't move fast enough to justify it. On-demand sync adds UI complexity (refresh button, sync status, error handling for partial failures) without proportional value. | Daily cron is sufficient. Show last-sync timestamp clearly so users trust the data. |
| Multi-user / family sharing | Couples want to share the dashboard. | Multi-user requires user management, per-user permissions, possibly separate SimpleFIN tokens per person, and session isolation. It's a significant scope expansion for a tool designed as single-user. | Self-host on a shared network; share the single password for the household. Document this as the supported pattern. |
| Push notifications / alerts | "Alert me when balance drops below X." | Requires a notification delivery mechanism (email SMTP, push API, SMS), threshold management UI, and ongoing maintenance. Adds external service dependencies to a self-contained tool. | Users can set up external monitoring (e.g., Uptime Kuma watching the API endpoint) if they want alerts. |
| CSV / export features | "I want to analyze my data in Excel." | Database is SQLite — users can query it directly with any SQLite client. Building an export UI adds maintenance burden with little benefit over direct DB access. | Document that the SQLite file is at `/data/finance.db` and is directly queryable. |
| Write operations (transfers, payments) | "Can I initiate transfers from here?" | Read-only via SimpleFIN is a feature, not a limitation. Write operations require bank-specific APIs, credential management, and serious security review. The read-only constraint is intentional and correct. | Never. Explicitly document as out of scope. |
| Native mobile app | Mobile-first finance apps are popular. | A native iOS/Android app is a completely separate codebase. The web UI should be mobile-responsive, which covers the use case. | Responsive web UI + PWA capability (add to home screen). |

## Feature Dependencies

```
SimpleFIN Integration (daily cron fetch + snapshot storage)
    └──required by──> All Panels (liquid, savings, investments)
                          └──required by──> Balance History Charts
                          └──required by──> Net Worth Breakdown Chart
                          └──required by──> Panel Drill-Down Views
                                                └──required by──> APY Display
                                                └──required by──> Investment Growth/Loss

Password Authentication
    └──required by──> Dashboard Access (all views)

Pending Transaction Handling
    └──enhances──> Liquid Balance Panel (makes the number accurate)

Investment Performance Charts
    └──requires──> Balance History Charts (same snapshot infrastructure)

Panel Drill-Down Views
    └──enhances──> Net Worth Breakdown Chart (chart segments become clickable)

Data Freshness Indicator
    └──enhances──> All Panels (trust signal on every view)
```

### Dependency Notes

- **SimpleFIN integration required by everything:** The cron fetch and snapshot schema are the foundation. Nothing else can be built until data is flowing and stored.
- **Balance history charts require snapshots:** A daily snapshot row per account must be stored from the start. If this is deferred, historical data is permanently lost — it cannot be backfilled from SimpleFIN.
- **Investment performance requires the same snapshot infrastructure as balance charts:** Build once, apply to both. Don't design snapshot schema for liquid/savings only.
- **Pending transaction handling is a Liquid panel concern only:** It modifies the computed liquid total but doesn't affect savings or investments.
- **APY display is a drill-down enhancement:** It can be added after basic drill-down pages exist.

## MVP Definition

### Launch With (v1)

Minimum viable product — what's needed to validate the concept.

- [ ] SimpleFIN integration with daily cron and snapshot storage — without data, nothing else works
- [ ] Liquid balance panel: checking minus credit cards (including pending) — core value proposition
- [ ] Savings panel: aggregated savings balance — expected on a finance dashboard
- [ ] Investments panel: brokerage + retirement + crypto aggregated — completes the "all money" picture
- [ ] Balance history charts (line chart, per panel or per account) — without this it's a static display not a dashboard
- [ ] Net worth breakdown chart (pie/donut) — table stakes visualization
- [ ] Simple password authentication — required before exposing financial data on a network
- [ ] Data freshness indicator — trust signal; users need to know when data was last fetched
- [ ] Light/dark mode toggle — self-hosters check this at night; noticeably missing without it
- [ ] Docker containerized deployment — required for self-hosted distribution

### Add After Validation (v1.x)

Features to add once the core is working and trusted.

- [ ] Panel drill-down views with individual account detail — once users trust the totals, they want to see per-account breakdown; add when the aggregate view is solid
- [ ] APY display on savings accounts — useful enhancement once drill-down pages exist
- [ ] Investment growth/loss per account — once drill-down exists and snapshot data has accumulated
- [ ] Investment performance over time chart — after baseline snapshots are accumulating

### Future Consideration (v2+)

Features to defer until product-market fit is established.

- [ ] Budgeting module — separate product scope; only if users explicitly request it after using the dashboard
- [ ] Manual account entry (for accounts not supported by SimpleFIN) — useful for crypto or small credit unions; low priority given SimpleFIN coverage
- [ ] PWA manifest (add to home screen) — polish; defer until core is stable

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| SimpleFIN fetch + snapshot storage | HIGH | MEDIUM | P1 |
| Liquid balance panel (with pending) | HIGH | LOW | P1 |
| Savings panel | HIGH | LOW | P1 |
| Investments panel | HIGH | LOW | P1 |
| Password authentication | HIGH | LOW | P1 |
| Balance history charts | HIGH | MEDIUM | P1 |
| Net worth breakdown chart | HIGH | LOW | P1 |
| Data freshness indicator | MEDIUM | LOW | P1 |
| Dark/light mode | MEDIUM | LOW | P1 |
| Docker deployment | HIGH | LOW | P1 |
| Panel drill-down views | MEDIUM | MEDIUM | P2 |
| APY display | MEDIUM | LOW | P2 |
| Investment growth/loss | MEDIUM | MEDIUM | P2 |
| Investment performance chart | MEDIUM | MEDIUM | P2 |
| Mobile-responsive layout | MEDIUM | MEDIUM | P2 |
| Budgeting | HIGH (requested) | HIGH | P3 |
| Transaction categorization | MEDIUM (requested) | HIGH | P3 |
| Push notifications | LOW | HIGH | P3 |

**Priority key:**
- P1: Must have for launch
- P2: Should have, add when possible
- P3: Nice to have, future consideration

## Competitor Feature Analysis

| Feature | Mint (defunct) / Empower | Firefly III | Actual Budget | Our Approach |
|---------|--------------------------|-------------|---------------|--------------|
| Account aggregation | Plaid-based, automatic | Manual import or Plaid addon | Manual import or Plaid addon | SimpleFIN (read-only token, no credential storage) |
| Net worth view | Yes, prominent | Yes, configurable | Yes | Prominent on dashboard home |
| Balance history | Yes | Yes | Yes | Yes — daily snapshots |
| Budget tracking | Central feature | Central feature | Central feature | Explicitly out of scope (v1) |
| Transaction categorization | Yes, ML-powered | Yes, manual rules | Yes, manual rules | Explicitly out of scope (v1) |
| Investment tracking | Empower only; Mint was basic | Basic | Minimal | First-class panel alongside liquid/savings |
| Pending transactions in liquid balance | Empower does; Mint was inconsistent | Not computed this way | Not computed this way | Explicit formula: checking - credit (with pending) |
| Self-hosted | No | Yes | Yes | Yes |
| Multi-user | Yes (Empower) | Yes | Yes | No — single user by design |
| APY on savings | Empower shows rates | Manual entry | Manual entry | Display from SimpleFIN metadata where available |
| Dark mode | Yes (Empower) | Yes | Yes | Yes |

## Sources

- Training knowledge of personal finance products: Mint, Empower (Personal Capital), YNAB, Firefly III, Actual Budget, Lunch Money, Copilot, Fina, Monarch Money (knowledge cutoff August 2025)
- PROJECT.md — explicit scope definitions and constraints used to calibrate table stakes vs. anti-features
- SimpleFIN protocol understanding (https://www.simplefin.org/protocol.html) — informs what data is realistically available
- External web search unavailable (Brave API key not configured, WebSearch tool denied)

**Confidence note:** All findings are from training knowledge, not live research. The personal finance dashboard space is mature and well-documented in training data — table stakes are stable. Differentiators and anti-features are based on known gap analysis of self-hosted tools vs. commercial tools. MEDIUM confidence overall; HIGH confidence on table stakes, MEDIUM on differentiators, MEDIUM on competitor specifics.

---
*Feature research for: self-hosted personal finance dashboard (read-only, single-user, SimpleFIN)*
*Researched: 2026-03-15*
