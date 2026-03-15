# Finance Visualizer

## What This Is

A self-hosted personal finance dashboard that aggregates data from financial institutions via SimpleFIN and presents it in a modern, polished interface. Built for a single user to see all their money in one place — liquid cash, savings, and investments — with drill-down detail and historical charts.

## Core Value

Show the user exactly where all their money is right now, with one glance at a single dashboard.

## Current Milestone: v1.1 Enhancements

**Goal:** Add crypto aggregation, account management, growth insights, net worth drill-down, alert notifications, financial projections, and operational improvements.

**Target features:**
- Crypto account aggregation by institution (e.g., all Coinbase wallets → one line)
- Account renaming (custom display names used globally)
- Growth rate indicators on panel cards
- Net worth drill-down page with detailed graphs and patterns
- Sync failure diagnostics in settings
- Alert rules with email notifications (expression builder, threshold crossing)
- Projected net worth tab (APY, growth rates, reinvestment toggle, income modeling)

## Requirements

### Validated

- ✓ Liquid balance panel: checking minus credit card balances — v1.0 Phase 3
- ✓ Savings panel: all savings accounts aggregated — v1.0 Phase 3
- ✓ Investments panel: brokerage, retirement, and crypto accounts — v1.0 Phase 3
- ✓ Balance-over-time line charts — v1.0 Phase 4
- ✓ Net worth breakdown chart (pie/donut) — v1.0 Phase 4
- ✓ SimpleFIN integration for read-only financial data — v1.0 Phase 2
- ✓ Daily cron job (goroutine) to fetch and store data — v1.0 Phase 2
- ✓ Full history pull on first sync, daily snapshots thereafter — v1.0 Phase 2
- ✓ Simple password protection — v1.0 Phase 1
- ✓ Modern fintech UI with light/dark toggle — v1.0 Phase 4
- ✓ Docker containerized deployment — v1.0 Phase 1

### Active

- [ ] Crypto accounts aggregated by institution (e.g., all Coinbase wallets summed into one entry)
- [ ] Account renaming in settings with global display name override
- [ ] Growth rate indicators (% change) on panel cards
- [ ] Net worth drill-down page with detailed historical graphs and patterns
- [ ] Sync failure diagnostics shown in settings
- [ ] Alert rules with expression builder (bucket/account math, comparison operators)
- [ ] Email notifications on threshold crossing (fire once on cross, once on recovery)
- [ ] SMTP/API email provider configuration in settings
- [ ] Projected net worth tab with per-account APY and growth rates
- [ ] Reinvestment toggle per account in projections
- [ ] Income modeling with savings allocation across accounts
- [ ] Custom projection time horizon

### Out of Scope

- Budgeting — deferred to future milestone
- Transaction categorization — not needed for pure balance/net-worth dashboard
- Multi-user support — single user, self-hosted
- Mobile app — responsive web UI covers the use case
- Write operations to financial accounts — read-only via SimpleFIN
- Year-over-year chart comparison — declined for v1.1
- CSV/data export — deferred (backburner)

## Context

- SimpleFIN provides a simple protocol for read-only JSON data from financial institutions (https://www.simplefin.org/protocol.html)
- This is also a learning/practice project alongside being genuinely useful
- Self-hosted on the user's own infrastructure
- Small data volume — single user, daily snapshots

## Constraints

- **Backend**: Go with go-chi router — decided
- **Frontend**: React with TypeScript — decided
- **Database**: SQLite — small data, single user, no network separation needed
- **Web Server**: Nginx as reverse proxy — decided
- **Deployment**: Docker containers — required for self-hosted distribution
- **Data Source**: SimpleFIN protocol — sole data provider

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| SQLite over Postgres | Small data volume, single user, no network DB needed | — Pending |
| SimpleFIN as sole data source | Simple protocol, read-only, covers major institutions | — Pending |
| Go backend | Practice + performance, good fit for background cron jobs | — Pending |
| Daily fetch cadence | Financial data doesn't change fast enough to justify more | — Pending |
| Include pending transactions in liquid balance | More accurate real-time picture of spendable cash | — Pending |

---
*Last updated: 2026-03-15 after v1.1 milestone initialization*
