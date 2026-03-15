# Project Research Summary

**Project:** Finance Visualizer — Self-Hosted Personal Finance Dashboard
**Domain:** Read-only personal finance aggregator (single-user, self-hosted, SimpleFIN-based)
**Researched:** 2026-03-15
**Confidence:** HIGH (stack), MEDIUM (features, pitfalls), HIGH (architecture)

## Executive Summary

This project is a self-hosted, read-only personal finance dashboard that pulls data from SimpleFIN and visualizes net worth across liquid, savings, and investment account panels. The stack is fully decided: Go (chi router) for the backend, React/TypeScript with Vite and Tailwind v4 for the frontend, SQLite with WAL mode for persistence, and Nginx + Docker for deployment. The architecture is a classic layered backend (HTTP handler → service → repository) with a background cron goroutine for daily SimpleFIN polling and a React SPA that reads exclusively from local SQLite via REST API. All library versions are confirmed current (Go 1.24+, modernc.org/sqlite v1.46.1, chi v5.2.5, Tailwind v4.2).

The most important architectural commitment is the snapshot-based balance history model: the cron job must store one `balance_snapshots` row per account per day from the very first sync. This data cannot be backfilled retroactively if omitted early. The "liquid balance" differentiator — checking minus credit card balances including pending — is the core value proposition and is straightforward to compute using the `balance` field SimpleFIN provides directly. Budgeting, transaction categorization, and multi-user are explicitly out of scope for v1.

The key risks are data integrity (append-only snapshot constraint must be enforced from day one, not retrofitted), credential security (SimpleFIN access URL is a bearer token and must never be committed to git), and auth correctness (bcrypt + rate-limited login must be in place before network exposure). All three risks are preventable with correct initial design and are not recoverable cheaply after the fact.

## Key Findings

### Recommended Stack

The backend is Go with the chi router ecosystem (chi, cors, httplog, httprate, jwtauth), modernc.org/sqlite (pure Go — no CGo, critical for Docker multi-arch), golang-migrate for schema migrations, shopspring/decimal for financial arithmetic, and robfig/cron for the daily sync goroutine. The frontend is React 18 + TypeScript 5 + Vite 6 + Tailwind v4 with recharts for charts, TanStack Query v5 for server state, react-router-dom v6 for routing, and zustand for global UI state. The build pipeline is a 3-stage Dockerfile: Node builder → Go builder → distroless final image. sqlc is recommended for type-safe SQL generation.

Two non-obvious but critical choices: (1) use `modernc.org/sqlite` not `mattn/go-sqlite3` — the mattn driver requires CGo, which breaks cross-compilation and bloats the Docker builder stage; (2) use `shopspring/decimal` not `float64` for all financial values — binary floating point cannot represent `0.1` exactly and balance errors will accumulate.

**Core technologies:**
- Go 1.24 + chi v5.2.5: backend API and background jobs — single binary, low overhead, excellent stdlib HTTP
- modernc.org/sqlite v1.46.1: persistence — pure Go, no CGo, required for clean Docker multi-arch builds
- golang-migrate v4.19.1: schema migrations — explicitly supports modernc.org/sqlite (use `database/sqlite` driver name, not `sqlite3`)
- React 18 + Vite 6 + Tailwind v4.2: frontend SPA — CSS-first Tailwind config, native Vite plugin, no tailwind.config.js
- recharts 2.x: charts — composable SVG API covering all 3 required chart types (line, donut, area)
- shopspring/decimal v1.4.0: financial arithmetic — never use float64 for money
- go-chi/jwtauth v5.4.0: JWT auth middleware — do NOT also add golang-jwt as a separate dep

### Expected Features

**Must have (table stakes — v1 launch):**
- SimpleFIN integration with daily cron and append-only snapshot storage — everything depends on this
- Liquid balance panel: checking minus credit cards (the core differentiator)
- Savings and investments panels: aggregated balances by account type
- Balance history line charts — daily snapshots enable these; missing from launch = permanently missing historical data
- Net worth breakdown donut chart — table stakes visualization for any finance dashboard
- Password authentication (bcrypt + rate-limited login) — required before network exposure
- Data freshness indicator ("Last synced: X hours ago") — trust signal; users notice when it's missing
- Dark/light mode toggle — self-hosters check dashboards at night
- Docker containerized deployment — required for self-hosted distribution

**Should have (v1.x — add after core is validated):**
- Panel drill-down views with per-account detail
- APY display on savings accounts
- Investment growth/loss per account (with correct "includes contributions" labeling)
- Investment performance over time chart
- Mobile-responsive layout

**Defer (v2+):**
- Budgeting module — separate product scope, doubles complexity
- Transaction categorization — needs ML/rules engine; not needed for balance/net-worth dashboard
- Push notifications — external service dependencies
- Multi-user/family sharing — significant scope expansion
- PWA manifest

### Architecture Approach

The system follows a strict layered pattern: Nginx serves static React assets and reverse-proxies `/api/*` to a Go backend; the Go backend separates HTTP handlers, service layer (business logic), repository layer (SQL), and a cron worker (SimpleFIN sync) all within the same process. SQLite is the sole datastore. The React SPA communicates exclusively via REST JSON; there is no WebSocket requirement since data is not real-time. Backend and frontend can be built in parallel once the DB schema is finalized.

**Major components:**
1. Nginx — TLS termination, static file serving (`/*`), reverse proxy (`/api/*`)
2. Go HTTP layer (chi) — auth middleware, request routing, JSON serialization
3. Go Service layer — balance aggregation, net worth computation, account classification
4. Go Cron worker — daily SimpleFIN pull, snapshot insertion, full history on first run
5. SimpleFIN HTTP client (`internal/simplefin/client.go`) — custom ~80 line client; no third-party Go SimpleFIN library is mature enough
6. Repository layer — SQL queries via sqlc-generated code against SQLite
7. SQLite — append-only balance snapshots, accounts, transactions
8. React SPA — Dashboard, panel drill-downs, recharts visualizations

### Critical Pitfalls

1. **Snapshot clobbering with upsert logic** — Use `INSERT ... ON CONFLICT DO NOTHING` with a `UNIQUE(account_id, snapshot_date)` constraint on `balance_snapshots`. Never use `INSERT OR REPLACE` for snapshots. If this is wrong from day one, historical chart data is permanently lossy. Address in Phase 1 (schema).

2. **SimpleFIN access URL in version control** — The access URL is a long-lived bearer token for all connected financial accounts. `.gitignore` the `.env` file from the first commit; never store the URL in the database in plaintext. Address in Phase 2 before writing any fetch logic.

3. **No cron error isolation or sync log** — A single failing account can abort the entire daily sync silently. Wrap each fetch in an error boundary, write a `sync_log` table from the start, and surface sync status in the UI. Address in Phase 2.

4. **Pending transaction double-counting in liquid balance** — SimpleFIN's `balance` field typically already includes pending transactions. Do not add pending transaction amounts on top. Validate the computed liquid balance against the actual bank app balance after first real fetch.

5. **SQLite WAL mode not set** — Without `PRAGMA journal_mode=WAL`, the daily cron write lock blocks all HTTP reads. Set WAL mode and `busy_timeout` at connection open time. Address in Phase 1 (DB setup). One line of code; painful to debug if missed.

## Implications for Roadmap

Based on research, the dependency chain is clear: schema must be correct before services, services before cron, cron before there is real data to display, and the frontend can proceed in parallel with the backend service/cron work once the API contract is defined.

### Phase 1: Foundation — Database, Auth, and Docker Skeleton

**Rationale:** Schema correctness cannot be retrofitted cheaply. The append-only snapshot model, WAL mode, and auth must be right before any other code is written on top of them. Docker setup here establishes the build/run pattern all subsequent phases use.
**Delivers:** Working SQLite schema with migrations, bcrypt auth with rate-limited login, JWT middleware protecting all `/api/*` routes, WAL mode enabled, Docker Compose dev environment running.
**Addresses:** Password authentication, Docker deployment (table stakes)
**Avoids:** Snapshot clobbering (UNIQUE constraint from day one), SQLite read-blocking (WAL pragma from day one), brute-force on login (httprate middleware from day one), Docker root user issue (non-root USER in Dockerfile)

### Phase 2: SimpleFIN Integration and Data Pipeline

**Rationale:** Without real data flowing, nothing else can be built or validated. This phase is the highest-risk phase (external API, credential handling, cron reliability) and must be addressed before any UI work invests in a data model.
**Delivers:** Custom SimpleFIN HTTP client, daily cron goroutine with error isolation, `sync_log` table and observability, full history pull on first run, accounts and balance snapshots populating the DB.
**Uses:** modernc.org/sqlite, golang-migrate, shopspring/decimal, robfig/cron, golang.org/x/crypto (not for passwords here — for potential future use), internal/simplefin/client.go
**Implements:** Cron worker, SimpleFIN HTTP client, ingest layer
**Avoids:** Access URL in git (establish .env pattern), pending transaction double-counting (use balance field directly), silent cron failures (sync_log from start), setup token vs. access URL confusion

### Phase 3: Backend API Layer

**Rationale:** Once data is in SQLite, the service and HTTP layers can be built and tested against real data. This phase delivers the full REST API that the frontend depends on.
**Delivers:** All `/api/*` endpoints: accounts list, balance history (snapshots), net worth aggregation, account type classification, sync status. Full Handler → Service → Repository layering.
**Uses:** go-chi/chi, go-chi/httplog, go-chi/jwtauth, sqlc-generated repository code
**Implements:** HTTP layer, Service layer, Repository layer
**Avoids:** Fat handlers (business logic in service layer, not handlers), N+1 queries (sqlc + explicit queries)

### Phase 4: React Frontend — Dashboard and Core Charts

**Rationale:** Frontend can begin with MSW mock responses while Phase 3 is in progress, then wire to the real API. Core charts (balance history line, net worth donut) are table stakes; their underlying snapshot data is already flowing from Phase 2.
**Delivers:** Dashboard page with liquid/savings/investments panels, balance history line charts, net worth donut chart, data freshness indicator, dark/light mode toggle, loading/empty states.
**Uses:** React 18, Vite 6, Tailwind v4, recharts, TanStack Query v5, zustand, react-router-dom
**Implements:** React SPA, frontend/api/client.ts typed wrappers, panel components, chart wrappers
**Avoids:** JWT in localStorage (use HttpOnly cookies), investment charts labeled as "performance" (label as "Account Value" with contributions caveat), missing empty states (design for "awaiting first sync"), UTC date offset on chart axes

### Phase 5: Drill-Down Views and Polish

**Rationale:** Once the aggregate dashboard is validated and trusted, add per-account detail. This phase adds the v1.x differentiators without blocking the core launch.
**Delivers:** Panel drill-down pages (per-account balances, transaction lists), APY display on savings, investment growth/loss with correct labeling, mobile-responsive layout.
**Addresses:** Panel drill-down views, APY display, investment growth/loss (v1.x features)
**Avoids:** Investment chart mislabeling (cost basis unavailable via SimpleFIN — label correctly)

### Phase 6: Production Hardening and Deployment

**Rationale:** Security and operational concerns deferred from earlier phases — non-root Docker user, Nginx TLS, CSRF protection, structured error responses — are addressed here before any networked self-hosting.
**Delivers:** Non-root Docker user, Nginx TLS configuration, rate limiting verified, error messages sanitized (no stack traces to client), docker-compose production configuration with named volumes.
**Avoids:** Docker root container, plaintext network exposure, detailed errors leaking to client

### Phase Ordering Rationale

- Schema first because snapshot append-only constraint and WAL mode cannot be retrofitted without data loss or tricky migrations
- SimpleFIN before API because real data is needed to validate service-layer business logic (especially the liquid balance calculation)
- Backend API before (or in parallel with) frontend because the frontend API client types are derived from the backend contract
- Drill-downs deferred to Phase 5 because they depend on the aggregate view being trusted, and they share the same snapshot infrastructure already built
- Production hardening last because most security concerns are one-line additions (rate limiting, WAL, non-root user) that are lower friction once the functional system is working

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2 (SimpleFIN Integration):** The SimpleFIN protocol `balance-date` field semantics, per-account `errors` array structure, and rate-limit behavior should be verified against the current spec at `simplefin.org/protocol.html` before implementation. All pitfalls research on this integration was from training knowledge (MEDIUM confidence).
- **Phase 2 (SimpleFIN client):** No mature Go SimpleFIN library exists; custom client implementation is required. Estimate ~80 lines but validate the claim + setup flow against current docs.

Phases with standard patterns (research-phase not required):
- **Phase 1 (Foundation):** Go + SQLite + chi auth is extremely well-documented. WAL mode, bcrypt, httprate are standard patterns.
- **Phase 3 (Backend API):** Layered Go HTTP service is the canonical pattern. sqlc codegen is well-documented.
- **Phase 4 (React Frontend):** React + Vite + Tailwind v4 + recharts + TanStack Query is a well-trodden combination with abundant documentation.
- **Phase 6 (Deployment):** Docker non-root user, Nginx TLS, multi-stage builds are standard DevOps patterns.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All library versions verified via pkg.go.dev and official docs. Version compatibility matrix explicitly checked. |
| Features | MEDIUM | Based on training knowledge of the personal finance domain (Mint, Empower, Firefly III, Actual Budget). Table stakes are HIGH confidence; differentiators and competitor specifics are MEDIUM. |
| Architecture | HIGH | Stack is fully decided; patterns (layered backend, snapshot model, cron goroutine) are well-established for this class of app. |
| Pitfalls | MEDIUM | WebSearch was unavailable; findings from training knowledge. Security and SQLite pitfalls are HIGH confidence (well-established patterns). SimpleFIN-specific pitfalls (pending semantics, balance-date) are MEDIUM — should be verified against current SimpleFIN spec. |

**Overall confidence:** HIGH for the technical approach; MEDIUM for SimpleFIN protocol specifics.

### Gaps to Address

- **SimpleFIN `balance` field semantics for pending transactions:** The research conclusion is to use `balance` directly (not add pending on top), but this varies by institution bridge. Validate against the current SimpleFIN protocol spec and cross-check with a real account during Phase 2 integration testing.
- **SimpleFIN `balance-date` field:** Store `balance_date` separately from `fetched_at` in the schema. The protocol may report balances dated to the previous business day. Verify behavior during Phase 2 and ensure the freshness indicator reflects `balance_date` not `fetched_at`.
- **SimpleFIN rate limits:** The daily cron schedule assumes SimpleFIN rate limits allow at least one fetch per day. Verify before committing to the cron schedule; the cron interval should be configurable via environment variable from Phase 1.
- **Crypto account support via SimpleFIN:** FEATURES.md identifies this as a potential low-cost differentiator "if SimpleFIN exposes crypto account data." Verify during Phase 2 whether any connected crypto accounts appear in the SimpleFIN response.
- **Tailwind v4 production stability:** v4 was released January 2025 and is recommended for greenfield projects. Confirm no known issues with the `@tailwindcss/vite` plugin and Vite 6 at project start.

## Sources

### Primary (HIGH confidence)
- `pkg.go.dev/modernc.org/sqlite` — v1.46.1 confirmed, pure Go, SQLite 3.51.2 (Feb 2026)
- `pkg.go.dev/github.com/go-chi/chi/v5` — v5.2.5 confirmed (Feb 2026)
- `pkg.go.dev/github.com/go-chi/jwtauth/v5` — v5.4.0 confirmed, uses lestrrat-go/jwx (Feb 2026)
- `pkg.go.dev/github.com/golang-migrate/migrate/v4/database/sqlite` — v4.19.1, explicitly uses modernc.org/sqlite (Nov 2025)
- `pkg.go.dev/golang.org/x/crypto` — v0.49.0, bcrypt available (Mar 2026)
- `pkg.go.dev/github.com/shopspring/decimal` — v1.4.0 confirmed (Apr 2024)
- `tailwindcss.com/docs/installation` — v4.2 confirmed, Vite plugin documented (Mar 2026)
- SQLite WAL mode: `sqlite.org/wal.html` — concurrent read/write behavior

### Secondary (MEDIUM confidence)
- SimpleFIN protocol specification (`simplefin.org/protocol.html`) — training knowledge; balance-date and per-account errors array should be verified against current spec
- Personal finance product feature analysis — training knowledge of Mint, Empower, Firefly III, Actual Budget, Lunch Money, Monarch Money (knowledge cutoff August 2025)
- Snapshot-based balance history pattern — observed in Monarch Money, Copilot, YNAB community discussions
- go-chi/cors v1.2.2, httplog v3.3.0, httprate v0.15.0, sqlc v1.30.0 — verified via pkg.go.dev

### Tertiary (LOW confidence)
- SimpleFIN rate limit behavior — inferred from protocol design; needs empirical verification
- SimpleFIN crypto account data availability — inferred; needs testing with real crypto-connected accounts

---
*Research completed: 2026-03-15*
*Ready for roadmap: yes*
