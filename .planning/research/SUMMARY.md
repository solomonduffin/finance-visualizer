# Project Research Summary

**Project:** Finance Visualizer v1.1
**Domain:** Self-hosted personal finance dashboard — feature expansion of an existing v1.0 product
**Researched:** 2026-03-15
**Confidence:** HIGH

## Executive Summary

Finance Visualizer v1.1 adds 7 new capabilities to an existing, production Go/React/SQLite dashboard. The existing stack is validated and stable; only two net-new dependencies are required (go-mail for SMTP email, react-querybuilder for the alert rule builder UI). The project is well-positioned because the v1.0 codebase is clean, the existing data model already contains most of what the new features need, and the feature scope maps directly onto well-understood domain patterns from Empower, Monarch Money, and ProjectionLab. All 7 features were analyzed against the actual codebase, not just theoretical requirements.

The recommended build order starts with account renaming and soft-delete safety (low-risk, high-value, establishes the data foundation), moves through sync diagnostics and growth indicators (quick wins on existing data), then crypto aggregation and net worth drill-down (analytics expansion), and finishes with the alert system and projection engine (complex new subsystems that benefit from the earlier groundwork). No feature requires a full architectural overhaul — each integrates as an extension of the existing handler and sync patterns.

The dominant risks are data integrity and security. The existing hard-delete of stale accounts will destroy user-configured data (display names, APY settings, alert rules) on any SimpleFIN outage; this must be converted to a soft-delete before any user-owned per-account metadata is introduced. Expression injection in alert rules is a critical security risk (reference: CVE-2025-68613, CVSS 9.9) that is fully prevented by storing alert conditions as structured JSON rather than free-text expressions. SMTP credentials must never be returned in API responses. All three risks are preventable by design decisions made at schema time.

---

## Key Findings

### Recommended Stack

The existing stack (Go 1.25, go-chi, React 19, TypeScript 5.9, Tailwind v4, SQLite via modernc.org/sqlite, recharts 3.x, shopspring/decimal, JWT auth, Docker, Nginx) requires exactly two additions. `github.com/wneessen/go-mail v0.7.1` handles SMTP email for alert notifications — it is the only actively maintained Go SMTP library with proper STARTTLS support, required for Protonmail Bridge integration. `react-querybuilder v8.14.0` provides the alert rule expression builder UI; it explicitly supports React 19, ships unstyled (Tailwind-compatible), and eliminates 2-4 weeks of custom UI work. All projection math uses the existing `shopspring/decimal` with iterative compounding (no `math.Pow`). All new charts use the existing `recharts` (dashed lines via `strokeDasharray`). The `expr-lang/expr` library handles safe alert expression evaluation on the Go side.

**Core new technologies:**
- `github.com/wneessen/go-mail v0.7.1`: SMTP sending — only maintained Go SMTP library with STARTTLS; required for Protonmail Bridge
- `react-querybuilder v8.14.0`: Alert rule builder UI — React 19 native, Tailwind-compatible, eliminates custom operator/group-nesting parser work
- `github.com/expr-lang/expr`: Alert expression evaluation — sandboxed, prevents injection, used by Google Cloud/Uber in production

**No new library needed for:** financial projections (use existing shopspring/decimal with iterative monthly compounding), new chart types (recharts ComposedChart + `strokeDasharray` covers all cases), SimpleFIN holdings (existing net/http client, just add struct fields and remove `balances-only=1`).

### Expected Features

**Must have (table stakes):**
- Account renaming (`display_name` column) — every finance app supports this; institutional names like "SAVINGS PLUS ACCOUNT" are cryptic
- Growth rate indicators (+2.3% this month badges on panel cards) — a dashboard without trend signals feels static
- Sync failure diagnostics — the `sync_log` table already captures everything; purely a new endpoint and frontend display
- Crypto aggregation by institution — multiple Coinbase wallets should appear as one grouped line in the Investments panel

**Should have (competitive differentiators):**
- Net worth drill-down page — turns a single number into a historical insight view; no self-hosted tool does this well
- Alert rules with email notifications — expression-based threshold alerts; no self-hosted finance dashboard currently offers this
- Projected net worth with income modeling — deterministic compound interest projection; ProjectionLab charges $100/year for equivalent functionality

**Explicitly deferred (anti-features):**
- Real-time crypto price feeds — conflicts with SimpleFIN daily sync cadence; adds external API dependency
- Monte Carlo simulation — dedicated-product scope (ProjectionLab territory); deterministic projections are sufficient and honest
- Holdings-level investment detail — SimpleFIN `balances-only=1` suppresses holdings; availability varies by institution; building UI around inconsistent data creates a half-broken feature
- SMS notifications — requires paid third-party service; email + email-to-SMS gateway is the right answer for self-hosted

**Feature dependency order matters:** Account renaming must come first. It introduces `display_name` which crypto aggregation, alert rule displays, and projection account config all reference from day one. Building it last means retrofitting display names into three already-shipped systems.

### Architecture Approach

All 7 features integrate into the existing handler-per-resource architecture without requiring a service layer refactor. Features with real business logic (alerts, projections) get dedicated packages (`internal/alerts/`, `internal/projections/`) with pure functions — testable without HTTP. The sync flow gains a post-sync hook that calls `alerts.EvaluateAll()` after each successful sync, outside the sync mutex, so SMTP latency never blocks sync completion. Three new database migrations (000002 through 000004) introduce `display_name` and `hidden_at`, alert tables, and projection tables respectively. Thirteen new API endpoints are added; three existing endpoints are extended.

**Major new components:**
1. `internal/alerts/` (NEW) — evaluator (expr-lang sandboxed expressions), engine (NORMAL/TRIGGERED state machine, state transitions), notifier (go-mail SMTP with context deadline)
2. `internal/projections/` (NEW) — compound interest engine using shopspring/decimal iterative compounding (360 multiplications for 30-year monthly — microseconds, not a concern), income allocation
3. Migrations 000002-000004 (NEW) — `display_name`/`hidden_at` on accounts, `alert_rules`/`alert_history` tables, `projection_config`/`income_allocations` tables
4. Four new frontend pages — NetWorthDrillDown, Alerts, Projections, plus Settings extensions for sync diagnostics, SMTP config, and account names
5. Sync hook (MODIFY `sync.go`) — calls `alerts.EvaluateAll()` after sync, outside mutex; SMTP failure is logged but never fails the sync

**Key patterns to follow:**
- Soft-delete accounts with `hidden_at DATETIME` rather than hard-deleting — prerequisite for all user-owned per-account data
- Alert conditions stored as structured JSON, not free-text — validated with `expr.Compile()` at write time, evaluated at sync time
- `COALESCE(display_name, name)` in every query that returns account names
- `shopspring/decimal` for all financial arithmetic — no `float64` anywhere in the financial pipeline
- Aggregation keyed on `org_slug` (stable domain-like identifier), not `org_name` (human-readable, can change between syncs)

### Critical Pitfalls

1. **Stale account hard-delete destroys user data** — The existing `removeStaleAccounts()` permanently deletes accounts that disappear from SimpleFIN, including all `balance_snapshots`. With v1.1 user-owned metadata, a temporary SimpleFIN outage erases display names, alert rule references, and APY settings permanently. Fix before any user-owned data is introduced: add `hidden_at DATETIME` column and convert to soft-delete. Auto-restore when account reappears on subsequent sync. Manual "permanently delete" in settings.

2. **Expression injection in alert rules** — Using a general-purpose expression engine for alert evaluation exposes `JWT_SECRET`, `PASSWORD_HASH`, and SMTP credentials via `os.Getenv()`. CVE-2025-68613 in n8n (CVSS 9.9) is exactly this attack pattern. Fix: store conditions as structured JSON `{"metric": "liquid", "operator": "<", "value": 5000}` and evaluate with a Go switch statement or the sandboxed `expr-lang/expr`. Never accept free-text expressions from users.

3. **Alert flooding from threshold oscillation** — A balance hovering at the alert threshold sends an email on every sync cycle where the condition is true. Fix: implement a per-rule state machine (NORMAL → TRIGGERED, TRIGGERED → NORMAL) that fires exactly once per threshold crossing and once per recovery. Establish baseline on rule creation so a rule already in a triggered state does not fire immediately when saved.

4. **SMTP credentials exposed via API or logs** — The `settings` table pattern makes it tempting to add `smtp_pass` as a key-value pair and return it in `GET /api/settings`. Fix: never return the SMTP password in API responses (return `smtp_configured: true/false` only). Store credentials as environment variables in `docker-compose.prod.yml` matching the existing `JWT_SECRET`/`PASSWORD_HASH` pattern. Never pass password to `slog` fields.

5. **SQLite migrations failing on populated tables** — New `NOT NULL` columns without a `DEFAULT` fail against the v1.0 production database. A dirty migration state prevents the application from starting. Fix: every `ALTER TABLE ADD COLUMN` must include a `DEFAULT`. Test all migrations against a database seeded with v1.0 realistic data, not just empty `:memory:` test databases.

---

## Implications for Roadmap

Based on combined research, the natural phase structure is driven by dependency chains, not arbitrary grouping. Account renaming is the keystone — every other feature that stores per-account user configuration is blocked until `display_name` exists and soft-delete is in place to protect it.

### Phase 1: Data Foundation
**Rationale:** Two schema changes must land before any user-owned data is introduced. Soft-delete must exist before `display_name` (otherwise a SimpleFIN outage on day two destroys the user's renamed accounts). `display_name` must exist before alerts and projections reference it.
**Delivers:** Accounts survive SimpleFIN outages with all config intact; users can rename accounts to human-readable names; all new features have a stable `display_name` to reference from day one.
**Addresses:** Account Renaming (table stakes)
**Avoids:** Stale account hard-delete destroying user data (Pitfall 1); SQLite migration failures on populated tables (Pitfall 5)
**Research flag:** Standard patterns — SQLite `ALTER TABLE ADD COLUMN` and soft-delete are well-established. No phase research needed.

### Phase 2: Operational Quick Wins
**Rationale:** Sync diagnostics and growth indicators are independent features with zero external dependencies. Both read from tables that already exist and already have the necessary data. High visible value for minimal risk — good early-phase momentum.
**Delivers:** Users can diagnose expired SimpleFIN tokens from the Settings UI; panel cards show "+2.3% this month" trend badges.
**Addresses:** Sync Failure Diagnostics (table stakes), Growth Rate Indicators (table stakes)
**Avoids:** SimpleFIN credential leakage in sync error text — sanitize `error_text` before storing (Pitfall 4); growth indicator division-by-zero and misleading percentages on new/credit-card accounts (Pitfall 8)
**Research flag:** Standard patterns. No phase research needed.

### Phase 3: Analytics Expansion
**Rationale:** Crypto aggregation and net worth drill-down both operate exclusively on existing snapshot data with no new background processing or external integrations. Lower risk profile than Phases 4-5.
**Delivers:** Coinbase wallets grouped into one combined investment line with expand-to-detail; dedicated `/net-worth` page with historical stacked area chart and time range picker.
**Addresses:** Crypto Aggregation (table stakes), Net Worth Drill-Down (differentiator)
**Avoids:** Aggregation merging accounts of different types at the same institution by keying on `(org_slug, account_type)` (Pitfall 5); `org_name` instability breaking grouping by using `org_slug` as the stable key (Pitfall 6); drill-down performance with 1000+ data points via server-side resolution parameter
**Research flag:** Validate `org_slug` stability across institutions in practice — the SimpleFIN spec guarantees it is a domain-like identifier but real-world behavior may vary.

### Phase 4: Alert System
**Rationale:** The alert system is the most architecturally novel addition — new package, new background hook, external SMTP dependency, state machine, expression evaluator. Isolated as its own phase to focus testing effort. Account renaming (Phase 1) must be complete so alert expressions can reference display names.
**Delivers:** User-defined threshold alerts that fire once on crossing and once on recovery, delivered via email to any SMTP-configured address (Protonmail Bridge or standard SMTP).
**Addresses:** Alert Rules with Email Notifications (differentiator)
**Avoids:** Expression injection via structured JSON storage and expr-lang sandboxing (Pitfall 2); alert flooding via state machine (Pitfall 3); SMTP credential exposure via environment variable pattern and masked API responses (Pitfall 10)
**Research flag:** Phase research recommended — Protonmail Bridge Docker service-to-service networking; `expr-lang/expr` sandboxing scope vs. `react-querybuilder` JSON output format alignment.

### Phase 5: Projection Engine
**Rationale:** The projection engine is the most standalone feature — it reads current balances and its own config tables with no dependency on alerts or other v1.1 features. Saved for last because it requires the most complex frontend configuration UI. Account renaming (Phase 1) ensures the projection config table shows human-readable names.
**Delivers:** Forward-looking net worth projection page with per-account APY settings, reinvestment toggles (compound vs. simple), income allocation modeling, and time horizon selector (1y/5y/10y/30y).
**Addresses:** Projected Net Worth with Income Modeling (differentiator)
**Avoids:** float64 precision errors in 30-year projections via iterative shopspring/decimal compounding — never use `math.Pow` (Pitfall 7); misleading projections presented as guarantees via explicit "Estimates" disclaimers and dashed chart lines
**Research flag:** Standard patterns — iterative compound interest math is unambiguous. No phase research needed.

### Phase Ordering Rationale

- **Soft-delete before display_name before everything else:** The dependency chain is `hidden_at` → `display_name` → alerts/projections. Reversing this order means retrofitting data safety into features already live with real user data.
- **Quick wins in Phase 2:** Sync diagnostics and growth indicators deliver immediate user value with zero external dependencies. Building them early establishes momentum before the high-complexity phases.
- **Analytics before alerts:** Crypto aggregation and drill-down are read-only analytics with no background processing or external integrations. Lower risk profile — ship these before introducing the SMTP subsystem.
- **Alerts before projections:** The alert system introduces the most new moving parts (background hook, state machine, email). Completing it in isolation keeps the risk contained. Projections in Phase 5 are computation-intensive but architecturally simpler.
- **No feature blocked on another after Phase 1:** Once `display_name` and soft-delete are in place, Phases 2-5 are theoretically order-independent from a data perspective — the phase ordering above is risk-driven, not dependency-driven beyond Phase 1.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 4 (Alert System):** Protonmail Bridge Docker networking configuration for service-to-service SMTP; `expr-lang/expr` sandboxing scope and whether it exposes environment variables; `react-querybuilder` JSON output format and whether it maps cleanly to the Go evaluator's expected input.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Data Foundation):** SQLite `ALTER TABLE ADD COLUMN` with DEFAULT and soft-delete are universally documented patterns.
- **Phase 2 (Quick Wins):** Both features are read-only over existing tables with established query patterns.
- **Phase 3 (Analytics):** Recharts stacked area charts and server-side resolution control are well-documented.
- **Phase 5 (Projections):** Iterative compound interest with shopspring/decimal is mathematically unambiguous; the reference answer (30-year $100k at 7% = $761,225.50) is a reliable verification test.

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All libraries verified against official sources; version compatibility confirmed; only 2 net-new dependencies identified |
| Features | HIGH | Codebase fully inspected; existing schema documented; competitor analysis cross-referenced; feature dependency graph validated against code |
| Architecture | HIGH | All 7 features analyzed file-by-file; exact handler/package/migration mapping provided; no speculative components |
| Pitfalls | HIGH | Based on direct codebase analysis plus CVE research (CVE-2025-68613) and real-world precedent (Actual Budget PR #2836 for null org_slug) |

**Overall confidence:** HIGH

### Gaps to Address

- **Protonmail Bridge Docker networking:** The Bridge Docker container (`shenxn/protonmail-bridge`) is community-maintained, not officially by Proton. The exact Docker Compose service-to-service SMTP hostname and whether the container requires interactive setup must be validated during Phase 4 planning before committing to the docker-compose pattern.
- **expr-lang vs. react-querybuilder JSON format alignment:** ARCHITECTURE.md recommends `expr-lang/expr` for Go-side evaluation and `react-querybuilder` for the frontend builder. The exact JSON format react-querybuilder exports needs confirmation that it maps cleanly to what expr-lang expects, or a translation layer must be planned in Phase 4.
- **SimpleFIN holdings data availability by institution:** The protocol supports holdings (confirmed via spec), but `balances-only=1` in the existing client suppresses them. Real-world availability varies by institution. Do not build the investment drill-down around holdings data for v1.1; account-level balances are reliable.
- **Credit card balance sign semantics in growth indicators:** Going from -$500 to -$200 is a $300 improvement, but naive `(current-previous)/previous*100` gives -60%. Phase 2 must explicitly address sign handling for negative-balance accounts.

---

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis — all Go handlers, sync logic, schema, frontend components (read from repository during research)
- [wneessen/go-mail GitHub releases](https://github.com/wneessen/go-mail/releases) — v0.7.1, Go 1.24+ requirement, CVE-2025-59937 security fix
- [go-mail pkg.go.dev](https://pkg.go.dev/github.com/wneessen/go-mail) — API reference, auth methods, STARTTLS support
- [react-querybuilder npm](https://www.npmjs.com/package/react-querybuilder) — v8.14.0, React 19 support confirmed
- [react-querybuilder docs](https://react-querybuilder.js.org/) — TypeScript reference, custom fields/operators, JSON export format
- [SimpleFIN Protocol Specification](https://www.simplefin.org/protocol.html) — holdings schema, account fields, org.id semantics
- [SimpleFIN GitHub protocol.md](https://github.com/simplefin/simplefin.github.com/blob/master/protocol.md) — authoritative spec source
- [shopspring/decimal pkg.go.dev](https://pkg.go.dev/github.com/shopspring/decimal) — Pow fractional exponent limitation, PowWithPrecision workaround
- [expr-lang/expr](https://github.com/expr-lang/expr) — sandboxed expression evaluation for Go, used by Google Cloud/Uber/ByteDance

### Secondary (MEDIUM confidence)
- [CVE-2025-68613: RCE via Expression Injection in n8n](https://nvd.nist.gov/vuln/detail/CVE-2025-68613) — CVSS 9.9; informs alert rule design decision to use structured JSON
- [Actual Budget PR #2836](https://github.com/actualbudget/actual/pull/2836) — real-world evidence that SimpleFIN `org_slug` can be null; informs fallback handling
- [Protonmail Bridge Docker (shenxn)](https://github.com/shenxn/protonmail-bridge-docker) — community-maintained Docker container; setup pattern needs Phase 4 validation
- [recharts DashedLineChart example](https://recharts.github.io/en-US/examples/DashedLineChart/) — `strokeDasharray` support confirmed for projection visualization

### Tertiary (LOW confidence / training knowledge)
- Empower, Monarch Money, Firefly III, ProjectionLab feature comparison — training knowledge as of May 2025; specific UI details may have changed but broad feature characterization is reliable

---
*Research completed: 2026-03-15*
*Ready for roadmap: yes*
