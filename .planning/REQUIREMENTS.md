# Requirements: Finance Visualizer

**Defined:** 2026-03-15
**Core Value:** Show the user exactly where all their money is right now, with one glance at a single dashboard.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Data Pipeline

- [x] **DATA-01**: App connects to SimpleFIN and fetches account data via read-only token
- [x] **DATA-02**: Daily cron goroutine fetches data automatically and stores snapshots in SQLite
- [x] **DATA-03**: First sync pulls up to one month of historical data from SimpleFIN
- [x] **DATA-04**: Each daily fetch creates append-only balance snapshots (one row per account per day)

### Dashboard

- [x] **DASH-01**: User sees liquid balance (checking minus credit card balances including pending)
- [x] **DASH-02**: User sees total savings across all savings accounts
- [x] **DASH-03**: User sees total investments (brokerage + retirement + crypto)
- [x] **DASH-04**: User sees individual account list with balances under each panel

### Visualizations

- [ ] **VIZ-01**: Balance-over-time line chart for each panel (liquid, savings, investments)
- [ ] **VIZ-02**: Net worth breakdown pie/donut chart (liquid vs savings vs investments)

### Auth & UX

- [x] **AUTH-01**: App is protected by a simple password gate
- [x] **UX-01**: Dashboard shows data freshness indicator ("Last updated: X ago")
- [x] **UX-02**: App shows appropriate loading/empty states on first run before data exists
- [x] **UX-03**: Dark/light mode toggle
- [x] **UX-04**: Mobile-responsive layout

### Deployment

- [x] **DEPLOY-01**: App runs as Docker containers (Go backend, React frontend, Nginx reverse proxy)

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Drill-Down Views

- **DRILL-01**: User can drill into each panel to see per-account detail
- **DRILL-02**: Savings accounts show APY where available
- **DRILL-03**: Investment accounts show growth/loss per account
- **DRILL-04**: Investment performance over time chart per account

### Future

- **FUT-01**: Budgeting module with spending categories
- **FUT-02**: Manual account entry for accounts not on SimpleFIN
- **FUT-03**: PWA manifest (add to home screen)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Transaction categorization | Not needed for pure balance/net-worth dashboard; doubles complexity |
| Multi-user support | Single user, self-hosted by design |
| Real-time / on-demand sync | SimpleFIN rate limits; daily cron is sufficient |
| Push notifications / alerts | Adds external service dependencies to a self-contained tool |
| Write operations (transfers, payments) | Read-only via SimpleFIN is a feature, not a limitation |
| Native mobile app | Responsive web UI covers the use case |
| CSV / export | SQLite file is directly queryable |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| AUTH-01 | Phase 1 | Complete |
| DEPLOY-01 | Phase 1 | Complete |
| DATA-01 | Phase 2 | Complete |
| DATA-02 | Phase 2 | Complete |
| DATA-03 | Phase 2 | Complete |
| DATA-04 | Phase 2 | Complete |
| DASH-01 | Phase 3 | Complete |
| DASH-02 | Phase 3 | Complete |
| DASH-03 | Phase 3 | Complete |
| DASH-04 | Phase 3 | Complete |
| VIZ-01 | Phase 4 | Pending |
| VIZ-02 | Phase 4 | Pending |
| UX-01 | Phase 4 | Complete |
| UX-02 | Phase 4 | Complete |
| UX-03 | Phase 4 | Complete |
| UX-04 | Phase 4 | Complete |

**Coverage:**
- v1 requirements: 16 total
- Mapped to phases: 16
- Unmapped: 0

---
*Requirements defined: 2026-03-15*
*Last updated: 2026-03-15 after roadmap creation*
