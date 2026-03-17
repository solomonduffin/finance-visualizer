# Requirements: Finance Visualizer

**Defined:** 2026-03-15
**Core Value:** Show the user exactly where all their money is right now, with one glance at a single dashboard.

## v1.0 Requirements (Complete)

All 16 requirements shipped and verified in v1.0.

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

- [x] **VIZ-01**: Balance-over-time line chart for each panel (liquid, savings, investments)
- [x] **VIZ-02**: Net worth breakdown pie/donut chart (liquid vs savings vs investments)

### Auth & UX

- [x] **AUTH-01**: App is protected by a simple password gate
- [x] **UX-01**: Dashboard shows data freshness indicator ("Last updated: X ago")
- [x] **UX-02**: App shows appropriate loading/empty states on first run before data exists
- [x] **UX-03**: Dark/light mode toggle
- [x] **UX-04**: Mobile-responsive layout

### Deployment

- [x] **DEPLOY-01**: App runs as Docker containers (Go backend, React frontend, Nginx reverse proxy)

## v1.1 Requirements

Requirements for the enhancements milestone. Each maps to roadmap phases.

### Account Management

- [x] **ACCT-01**: User can set a custom display name for any connected account in settings
- [x] **ACCT-02**: Custom display names appear everywhere the account is referenced (panels, charts, dropdowns, alerts, projections)
- [x] **ACCT-03**: User can create named account groups in Settings and assign accounts to them
- [x] **ACCT-04**: Account groups appear as a single combined line in their panel with summed balance
- [ ] **ACCT-05**: User can expand an account group to see individual account balances beneath it

### Dashboard Insights

- [x] **INSIGHT-01**: Each panel card shows percentage change over the last 30 days with green/red color coding
- [x] **INSIGHT-02**: User can click net worth donut to navigate to a dedicated net worth page
- [x] **INSIGHT-03**: Net worth page shows historical net worth line chart with per-panel breakdown
- [x] **INSIGHT-04**: Net worth page shows summary statistics (current net worth, period change in $ and %, all-time high)
- [x] **INSIGHT-05**: Net worth page has a time range selector (30d, 90d, 6m, 1y, all)
- [x] **INSIGHT-06**: User can toggle the 30-day growth rate badge on/off from settings

### Operational

- [x] **OPS-01**: Settings page shows log of recent sync attempts with timestamps, success/failure status, and account counts
- [x] **OPS-02**: Failed syncs show expandable error details (with sensitive data sanitized)
- [x] **OPS-03**: Stale accounts are soft-deleted to preserve user-owned metadata (display names, alert rules, projection rates)

### Alerts & Notifications

- [x] **ALERT-01**: User can create alert rules using an expression builder combining buckets and/or accounts with +/- operators
- [x] **ALERT-02**: Alert rules compare computed value against a threshold using <, <=, >, >=, == operators
- [x] **ALERT-03**: Alerts fire once on threshold crossing and once on recovery (3-state machine)
- [x] **ALERT-04**: Alert email includes rule name, computed value, threshold, and crossing direction with account context
- [ ] **ALERT-05**: User can configure email provider in settings (SMTP or API service with provider-specific fields)
- [ ] **ALERT-06**: User can send a test email to verify configuration
- [x] **ALERT-07**: User can create, edit, enable/disable, and delete alert rules

### Financial Projections

- [ ] **PROJ-01**: User can set APY per savings account and expected growth rate per investment account
- [ ] **PROJ-02**: User can toggle reinvestment (compound vs simple) per account
- [ ] **PROJ-03**: User can enable/disable which accounts are included in the projection
- [ ] **PROJ-04**: User can model income: annual amount, monthly savings %, and per-account allocation
- [x] **PROJ-05**: Projection chart shows projected net worth over a custom time horizon
- [x] **PROJ-06**: All projection settings persist in the database across sessions
- [ ] **PROJ-07**: Projections page accessible from main navigation
- [x] **PROJ-08**: Investment accounts display available holdings detail from SimpleFIN where supported (e.g., Vanguard funds)

## Future Requirements

Deferred to future release. Tracked but not in current roadmap.

### Data Export

- **EXPORT-01**: User can download balance history as CSV or JSON
- **EXPORT-02**: Export supports date range filtering

### Drill-Down Views

- **DRILL-01**: User can drill into each panel to see per-account detail with APY and growth/loss
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
| Real-time crypto price feeds | SimpleFIN provides daily snapshots; second data source creates sync conflicts |
| Monte Carlo simulation | Dedicated product scope (ProjectionLab); deterministic projection sufficient |
| SMS notifications | Requires paid service (Twilio); email-to-SMS gateways available as workaround |
| Year-over-year chart comparison | Declined for v1.1 |
| Native mobile app | Responsive web UI covers the use case |
| Write operations to financial accounts | Read-only via SimpleFIN is a feature, not a limitation |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

### v1.0 (Complete)

| Requirement | Phase | Status |
|-------------|-------|--------|
| DATA-01 | Phase 2 | Complete |
| DATA-02 | Phase 2 | Complete |
| DATA-03 | Phase 2 | Complete |
| DATA-04 | Phase 2 | Complete |
| DASH-01 | Phase 3 | Complete |
| DASH-02 | Phase 3 | Complete |
| DASH-03 | Phase 3 | Complete |
| DASH-04 | Phase 3 | Complete |
| VIZ-01 | Phase 4 | Complete |
| VIZ-02 | Phase 4 | Complete |
| AUTH-01 | Phase 1 | Complete |
| UX-01 | Phase 4 | Complete |
| UX-02 | Phase 4 | Complete |
| UX-03 | Phase 4 | Complete |
| UX-04 | Phase 4 | Complete |
| DEPLOY-01 | Phase 1 | Complete |

### v1.1 (Active)

| Requirement | Phase | Status |
|-------------|-------|--------|
| ACCT-01 | Phase 5 | Complete |
| ACCT-02 | Phase 5 | Complete |
| ACCT-03 | Phase 7 | Complete |
| ACCT-04 | Phase 7 | Complete |
| ACCT-05 | Phase 7 | Pending |
| INSIGHT-01 | Phase 6 | Complete |
| INSIGHT-02 | Phase 7 | Complete |
| INSIGHT-03 | Phase 7 | Complete |
| INSIGHT-04 | Phase 7 | Complete |
| INSIGHT-05 | Phase 7 | Complete |
| INSIGHT-06 | Phase 6 | Complete |
| OPS-01 | Phase 6 | Complete |
| OPS-02 | Phase 6 | Complete |
| OPS-03 | Phase 5 | Complete |
| ALERT-01 | Phase 8 | Complete |
| ALERT-02 | Phase 8 | Complete |
| ALERT-03 | Phase 8 | Complete |
| ALERT-04 | Phase 8 | Complete |
| ALERT-05 | Phase 8 | Pending |
| ALERT-06 | Phase 8 | Pending |
| ALERT-07 | Phase 8 | Complete |
| PROJ-01 | Phase 9 | Pending |
| PROJ-02 | Phase 9 | Pending |
| PROJ-03 | Phase 9 | Pending |
| PROJ-04 | Phase 9 | Pending |
| PROJ-05 | Phase 9 | Complete |
| PROJ-06 | Phase 9 | Complete |
| PROJ-07 | Phase 9 | Pending |
| PROJ-08 | Phase 9 | Complete |

**Coverage:**
- v1.0 requirements: 16 total, 16 mapped (Complete)
- v1.1 requirements: 29 total, 29 mapped
- Unmapped: 0

---
*Requirements defined: 2026-03-15*
*Last updated: 2026-03-15 after v1.1 roadmap creation*
