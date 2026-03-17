# Feature Landscape: v1.2

**Domain:** Personal finance dashboard -- next-step features after balance/net-worth/alerts/projections
**Researched:** 2026-03-17
**Confidence:** MEDIUM-HIGH -- feature patterns well-understood from competitor analysis; SimpleFIN transaction availability confirmed via protocol spec

## Table Stakes

Features that users of a finance dashboard expect once balance tracking is established. Missing these means the product feels like a "view-only" tool.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Transaction list view | Users see balances change but cannot see WHY. Every competitor (Mint, YNAB, Monarch, Actual Budget) shows transactions. | Medium | SimpleFIN provides transaction data (id, posted, amount, description, pending). Currently not fetched. Requires new table, sync changes, API endpoint, and frontend page. |
| Data export (CSV) | Users expect to get their data out. This was explicitly deferred from v1.1 (EXPORT-01, EXPORT-02 in PROJECT.md). | Low | Backend streaming CSV via encoding/csv. Two endpoints: balance history export and transaction export. |
| Transaction search | Once transactions exist, finding specific ones is essential. | Low | FTS5 virtual table on transaction descriptions. Already compiled into modernc.org/sqlite. |
| Toast/notification feedback | Operations like sync, settings save, export complete, alert test currently have no visual feedback or use ad-hoc patterns. | Low | Sonner provides this with minimal setup. |

## Differentiators

Features that add value beyond what bare-bones finance dashboards provide, aligned with the project's strengths.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Transaction categorization | Turns raw transaction list into actionable spending insights. Manual rule-based categorization (merchant name matching) is sufficient for single user. | Medium | Categories table + user-defined rules. AI/ML is overkill for one person -- pattern matching on merchant descriptions is enough. |
| Spending analytics (monthly breakdown) | Shows WHERE money goes, not just how much. Bar charts by category, month-over-month trends. | Medium | Depends on transaction categorization. SQL aggregation + Recharts bar/composed charts (already in stack). |
| Recurring transaction detection | Automatically identifies subscriptions, bills, recurring charges. Surfaces "you pay $X/month in subscriptions." | Medium | Group transactions by normalized merchant + similar amounts at regular intervals. Pure SQL + Go logic. |
| Budget tracking | Set spending limits per category per month, see progress. | Medium-High | Requires categorization first. Budgets table, monthly rollup queries, progress bars/gauges on frontend. |
| Goal tracking | Save toward specific targets (emergency fund, vacation, house). Link to accounts, track progress over time. | Medium | Goals table, progress calculation from linked account balances. Good visual impact with Recharts. |
| Dashboard summary enhancements | Spending-this-month badge, upcoming bills widget, recent transactions snippet on dashboard. | Low-Medium | Depends on transactions being stored. Lightweight queries, small UI additions to existing dashboard. |

## Anti-Features

Features to explicitly NOT build in v1.2.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| AI-powered auto-categorization | Massive complexity for a single-user app. Training data requirements, model hosting, accuracy issues. | Manual rule-based categorization with keyword matching. User defines rules like "NETFLIX -> Entertainment". One person's transactions are manageable. |
| Multi-currency with real-time rates | Touches every balance calculation, chart, and aggregation. Large blast radius for a feature that may only apply to 0-2 accounts. | Defer to v1.3+. If user has foreign accounts, show them in original currency for now. |
| Mobile app | Responsive web UI is already sufficient. Native app development is a separate project. | Ensure responsive design works well on mobile browsers. |
| Multi-user support | Architectural rework (auth per user, data isolation). Not needed for self-hosted single-user. | Keep single-user design. |
| Bank write operations | SimpleFIN is read-only. Any write capability requires a different API (Plaid, etc.) with different security implications. | Stay read-only. |
| PDF report generation | Over-engineered for personal use. CSV export covers data portability. | Use browser print for one-off reports. Export CSV for spreadsheet analysis. |
| Monte Carlo simulation | Deterministic projections are already implemented. Monte Carlo was explicitly declined in v1.1. ProjectionLab exists for that. | Keep deterministic projection engine. |
| SMS notifications | Email-to-SMS gateways work. Building SMS infra is disproportionate effort. | Document email-to-SMS gateway workaround. |

## Feature Dependencies

```
Transaction Sync (fetch + store) --> Transaction List View
Transaction List View --> Transaction Search (FTS5)
Transaction List View --> Transaction Categorization
Transaction Categorization --> Spending Analytics
Transaction Categorization --> Budget Tracking
Transaction Categorization --> Recurring Transaction Detection
Transaction Sync --> Data Export (transactions)
Balance History (existing) --> Data Export (balances)
Goal Tracking (standalone -- depends only on existing account balances)
Toast Notifications (standalone -- improves all existing + new features)
```

## MVP Recommendation

**Phase 1 -- Foundation (must-haves that unlock everything else):**
1. Transaction sync and storage -- the enabler for all transaction-based features
2. Transaction list page with search
3. CSV data export (balance history + transactions) -- deferred from v1.1
4. Toast notification system -- improves UX across all features

**Phase 2 -- Intelligence (what makes the data useful):**
1. Transaction categorization (rule-based)
2. Spending analytics (monthly category breakdown)
3. Recurring transaction detection

**Phase 3 -- Planning (forward-looking features):**
1. Budget tracking per category
2. Goal tracking with progress visualization
3. Dashboard summary widgets (spending badge, upcoming bills, recent transactions)

**Defer to v1.3+:** Multi-currency, year-over-year comparison, investment performance tracking (beyond holdings display)

## Sources

- [SimpleFIN Protocol spec](https://www.simplefin.org/protocol.html) -- transaction field availability confirmed
- [Personal Finance Apps: What Users Expect in 2025](https://www.wildnetedge.com/blogs/personal-finance-apps-what-users-expect-in-2025) -- feature expectations
- [The State of Personal Finance Apps in 2025](https://bountisphere.com/blog/personal-finance-apps-2025-review) -- competitor feature landscape
- [Actual Budget SimpleFIN docs](https://actualbudget.org/docs/advanced/bank-sync/simplefin/) -- SimpleFIN integration patterns
- [SQL Habit: How to detect recurring payments](https://www.sqlhabit.com/blog/how-to-detect-recurring-payments-with-sql) -- recurring detection approach

---
*Feature research for: Finance Visualizer v1.2*
*Researched: 2026-03-17*
