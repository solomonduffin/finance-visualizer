# Finance Visualizer

## What This Is

A self-hosted personal finance dashboard that aggregates data from financial institutions via SimpleFIN and presents it in a modern, polished interface. Built for a single user to see all their money in one place — liquid cash, savings, and investments — with drill-down detail, historical charts, threshold-based email alerts, and forward-looking projections.

## Core Value

Show the user exactly where all their money is right now, with one glance at a single dashboard.

## Requirements

### Validated

- ✓ Liquid balance panel: checking minus credit card balances — v1.0
- ✓ Savings panel: all savings accounts aggregated — v1.0
- ✓ Investments panel: brokerage, retirement, and crypto accounts — v1.0
- ✓ Balance-over-time line charts — v1.0
- ✓ Net worth breakdown chart (pie/donut) — v1.0
- ✓ SimpleFIN integration for read-only financial data — v1.0
- ✓ Daily cron job (goroutine) to fetch and store data — v1.0
- ✓ Full history pull on first sync, daily snapshots thereafter — v1.0
- ✓ Simple password protection — v1.0
- ✓ Modern fintech UI with light/dark toggle — v1.0
- ✓ Docker containerized deployment — v1.0
- ✓ Account custom display names propagated globally — v1.1
- ✓ Soft-delete: accounts survive SimpleFIN outages with metadata preserved — v1.1
- ✓ Custom account groups with drag-and-drop, summed in panel — v1.1
- ✓ 30-day growth rate badges on panel cards with settings toggle — v1.1
- ✓ Sync failure diagnostics log in settings with sanitized error expansion — v1.1
- ✓ Dedicated net worth page: stacked area chart, stats, time range selector — v1.1
- ✓ Expression-based alert rules with 3-state machine (fire once on cross, once on recovery) — v1.1
- ✓ AES-256-GCM encrypted SMTP config, test email button — v1.1
- ✓ Projection engine: per-account APY, compound/simple interest, income modeling — v1.1
- ✓ SimpleFIN holdings detail for investment accounts (e.g., Vanguard funds) — v1.1

### Active

_(Define in `/gsd:new-milestone` for v1.2)_

### Out of Scope

- Budgeting — deferred to future milestone
- Transaction categorization — not needed for pure balance/net-worth dashboard
- Multi-user support — single user, self-hosted
- Mobile app — responsive web UI covers the use case
- Write operations to financial accounts — read-only via SimpleFIN
- Year-over-year chart comparison — declined for v1.1
- Monte Carlo simulation — deterministic projection sufficient (ProjectionLab for that)
- SMS notifications — email-to-SMS gateways available as workaround
- CSV/data export — deferred (EXPORT-01, EXPORT-02 in future requirements)

## Context

- Shipped v1.0 (2026-03-15) and v1.1 (2026-03-17) in rapid succession
- ~12,625 Go LOC + ~10,348 TypeScript LOC across 9 phases
- Tech stack: Go (chi router), React (TypeScript, Recharts, Tailwind), SQLite, Nginx, Docker
- SimpleFIN provides a simple protocol for read-only JSON data from financial institutions
- Self-hosted on the user's own infrastructure; small data volume (single user, daily snapshots)
- This is both a learning/practice project and genuinely useful personal tooling

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
| SQLite over Postgres | Small data volume, single user, no network DB needed | ✓ Good — no operational overhead, WAL mode handles concurrent reads |
| SimpleFIN as sole data source | Simple protocol, read-only, covers major institutions | ✓ Good — holdings detail worked for Vanguard |
| Go backend | Practice + performance, good fit for background cron jobs | ✓ Good — goroutine scheduler clean, expr-lang/expr for alert evaluation |
| Daily fetch cadence | Financial data doesn't change fast enough to justify more | ✓ Good — no complaints |
| Include pending transactions in liquid balance | More accurate real-time picture of spendable cash | ✓ Good — correct behavior |
| COALESCE(display_name, name) pattern | Null-safe display name override without breaking existing data | ✓ Good — propagated to all handlers cleanly |
| Soft-delete (hidden_at) over hard delete | Preserve user metadata (display names, alert rules, projection rates) across outages | ✓ Good — auto-restore on sync worked as designed |
| Client-side projection calculation | Avoid server round-trips for interactive rate adjustments | ✓ Good — debounced auto-save + useMemo kept it responsive |
| AES-256-GCM for SMTP credentials | Credentials stored encrypted at rest in SQLite | ✓ Good — key derived from JWT secret |
| expr-lang/expr for alert expression evaluation | Sandboxed, typed expression evaluation with operand compilation | ✓ Good — TDD green-phase approach proved the engine correct |

---
*Last updated: 2026-03-17 after v1.1 milestone*
