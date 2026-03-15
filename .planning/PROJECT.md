# Finance Visualizer

## What This Is

A self-hosted personal finance dashboard that aggregates data from financial institutions via SimpleFIN and presents it in a modern, polished interface. Built for a single user to see all their money in one place — liquid cash, savings, and investments — with drill-down detail and historical charts.

## Core Value

Show the user exactly where all their money is right now, with one glance at a single dashboard.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Liquid balance panel: checking minus credit card balances (including pending transactions)
- [ ] Savings panel: all savings accounts aggregated
- [ ] Investments panel: brokerage, retirement (401k/IRA), and crypto accounts
- [ ] Drill-down views for each panel (individual accounts, APY, growth/loss)
- [ ] Balance-over-time line charts
- [ ] Net worth breakdown chart (pie/donut)
- [ ] Investment performance charts (growth/loss over time)
- [ ] SimpleFIN integration for read-only financial data
- [ ] Daily cron job (goroutine) to fetch and store data
- [ ] Full history pull on first sync, daily snapshots thereafter
- [ ] Simple password protection
- [ ] Modern fintech UI with light/dark toggle
- [ ] Docker containerized deployment

### Out of Scope

- Budgeting — deferred to future milestone
- Transaction categorization — not needed for pure information display
- Multi-user support — single user, self-hosted
- Mobile app — web-first
- Write operations to financial accounts — read-only via SimpleFIN

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
*Last updated: 2026-03-15 after initialization*
