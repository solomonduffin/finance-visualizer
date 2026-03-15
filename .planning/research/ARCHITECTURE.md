# Architecture Research

**Domain:** Self-hosted personal finance dashboard (single user, read-only data aggregation)
**Researched:** 2026-03-15
**Confidence:** HIGH — stack is fully decided; patterns are well-established for this class of app

## Standard Architecture

### System Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                          External                                   │
│  ┌─────────────────────────────────────────────────┐               │
│  │          SimpleFIN Bridge (HTTPS, 3rd party)    │               │
│  └─────────────────────────┬───────────────────────┘               │
└────────────────────────────┼───────────────────────────────────────┘
                             │ HTTPS (daily cron pull)
┌────────────────────────────┼───────────────────────────────────────┐
│                     Docker Network                                  │
│                             │                                       │
│  ┌──────────────────────────▼──────────────────────────────────┐   │
│  │                    Nginx (port 80/443)                       │   │
│  │           Reverse proxy + static file serving                │   │
│  └──────┬──────────────────────────────────────────────────┘   │   │
│         │ /api/* → proxy_pass                 /* → static       │   │
│         │                                                        │   │
│  ┌──────▼──────────────────────────────────────────────────┐   │   │
│  │                  Go Backend (go-chi)                     │   │   │
│  │  ┌────────────┐  ┌──────────────┐  ┌─────────────────┐  │   │   │
│  │  │ HTTP Layer │  │ Service Layer │  │   Cron Worker   │  │   │   │
│  │  │ (chi router│  │ (business    │  │ (goroutine,     │  │   │   │
│  │  │  handlers) │  │  logic)      │  │  daily fetch)   │  │   │   │
│  │  └─────┬──────┘  └──────┬───────┘  └────────┬────────┘  │   │   │
│  │        │                │                    │            │   │   │
│  │        └────────────────┼────────────────────┘            │   │   │
│  │                         │                                  │   │   │
│  │               ┌─────────▼──────────┐                      │   │   │
│  │               │   Repository Layer  │                      │   │   │
│  │               │  (SQLite queries)   │                      │   │   │
│  │               └─────────┬──────────┘                      │   │   │
│  └─────────────────────────┼───────────────────────────────┘   │   │
│                             │                                    │   │
│  ┌──────────────────────────▼──────────────────────────────┐   │   │
│  │              SQLite (data/finance.db)                    │   │   │
│  │    accounts | balances | transactions | snapshots        │   │   │
│  └──────────────────────────────────────────────────────────┘   │   │
│                                                                    │   │
│  ┌──────────────────────────────────────────────────────────┐   │   │
│  │          React/TS SPA (built static assets)              │   │   │
│  │    Dashboard | Panels | Charts | Drill-down views        │   │   │
│  └──────────────────────────────────────────────────────────┘   │   │
└────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Communicates With |
|-----------|----------------|-------------------|
| Nginx | TLS termination, reverse proxy `/api/*` to Go, serve `/` static React build | Go backend, static files |
| Go HTTP Layer (chi) | Auth middleware, request routing, response serialization, error handling | Service layer |
| Go Service Layer | Business logic: aggregate balances, compute net worth, classify accounts by type | Repository layer, Cron worker |
| Go Cron Worker | Background goroutine; daily SimpleFIN pull, snapshot insertion, full history on first run | SimpleFIN HTTP client, Repository layer |
| SimpleFIN HTTP Client | Fetch accounts/transactions JSON from SimpleFIN bridge URL, handle auth token | SimpleFIN bridge (external) |
| Repository Layer | SQL queries against SQLite: reads, writes, migrations | SQLite file |
| SQLite | Persistent storage: accounts, balance snapshots, transactions | Repository layer only |
| React SPA | Render dashboard panels, charts, drill-down views; call `/api/*` endpoints | Go HTTP layer (REST) |

## Recommended Project Structure

```
finance-visualizer/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go          # Entrypoint: wires everything, starts server + cron
│   ├── internal/
│   │   ├── api/
│   │   │   ├── router.go        # chi router setup, middleware chain
│   │   │   ├── handlers/
│   │   │   │   ├── accounts.go  # GET /api/accounts
│   │   │   │   ├── balances.go  # GET /api/balances (snapshots, history)
│   │   │   │   ├── networth.go  # GET /api/networth
│   │   │   │   └── auth.go      # POST /api/login, session check
│   │   │   └── middleware/
│   │   │       └── auth.go      # Session/password middleware
│   │   ├── service/
│   │   │   ├── accounts.go      # Account aggregation, type classification
│   │   │   ├── balances.go      # Balance history, snapshot logic
│   │   │   └── networth.go      # Net worth computation across account types
│   │   ├── sync/
│   │   │   ├── cron.go          # Goroutine scheduler (daily trigger)
│   │   │   ├── simplefin.go     # SimpleFIN HTTP client, auth token handling
│   │   │   └── ingest.go        # Parse SimpleFIN response, write to DB
│   │   ├── repository/
│   │   │   ├── accounts.go      # SQL: account CRUD
│   │   │   ├── balances.go      # SQL: balance snapshot insert/query
│   │   │   └── transactions.go  # SQL: transaction insert/query
│   │   └── db/
│   │       ├── db.go            # SQLite connection, WAL mode setup
│   │       └── migrations/      # SQL migration files (numbered)
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── panels/          # LiquidPanel, SavingsPanel, InvestmentsPanel
│   │   │   ├── charts/          # LineChart, DonutChart, BarChart wrappers
│   │   │   └── ui/              # Shared: Card, Spinner, Toggle, etc.
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx    # Main overview page
│   │   │   └── DrillDown.tsx    # Account-level detail page
│   │   ├── hooks/
│   │   │   ├── useBalances.ts   # Fetch + cache balance history
│   │   │   └── useAccounts.ts   # Fetch + cache account list
│   │   ├── api/
│   │   │   └── client.ts        # Typed fetch wrappers for all /api/* endpoints
│   │   ├── types/
│   │   │   └── finance.ts       # Shared TypeScript interfaces (Account, Snapshot, etc.)
│   │   └── main.tsx
│   ├── public/
│   └── package.json
├── nginx/
│   └── nginx.conf               # Proxy config: /api/* → backend:8080, /* → /static
├── docker-compose.yml           # Orchestrates: nginx, backend, (frontend build as stage)
└── Dockerfile                   # Multi-stage: node build → go build → final image
```

### Structure Rationale

- **internal/**: Go convention enforcing package privacy; nothing outside the module can import these packages. Prevents accidental coupling.
- **sync/ vs service/**: Cron/ingest logic is separated from business logic. Ingest writes raw data; service layer interprets it. Swapping the data source later only touches `sync/`.
- **repository/ separate from service/**: Clean boundary for testing — services can be tested with mock repositories without a real SQLite file.
- **frontend/api/client.ts**: Centralizing all API calls in one typed file means TypeScript catches backend contract changes at compile time.
- **Multi-stage Dockerfile**: Node build produces static assets; Go build produces binary; final image contains only the binary + static files + nginx config. Keeps image small.

## Architectural Patterns

### Pattern 1: Layered Backend (Handler → Service → Repository)

**What:** Each layer has a single responsibility. Handlers parse HTTP; services compute business logic; repositories execute SQL.

**When to use:** Always — this is the standard for Go HTTP services.

**Trade-offs:** Slight verbosity (more files, more interfaces), but enables testing each layer independently.

**Example:**
```go
// handler calls service, not SQL directly
func (h *BalanceHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
    history, err := h.svc.GetBalanceHistory(r.Context(), accountType)
    // ...
}

// service calls repository, not SQL directly
func (s *BalanceService) GetBalanceHistory(ctx context.Context, t AccountType) ([]Snapshot, error) {
    return s.repo.ListSnapshots(ctx, t, 90) // last 90 days
}
```

### Pattern 2: Snapshot-Based History

**What:** Rather than deriving historical balances from transaction sums (fragile), store a daily balance snapshot per account at ingest time. Charts read from snapshots directly.

**When to use:** Any time you need a time-series chart of balances. This is the standard pattern for read-only aggregators (Copilot, Monarch Money all do this).

**Trade-offs:** Slight storage overhead (one row per account per day), but queries are trivially fast and don't depend on complete transaction history.

**Example schema:**
```sql
CREATE TABLE balance_snapshots (
    id          INTEGER PRIMARY KEY,
    account_id  TEXT NOT NULL,
    balance     REAL NOT NULL,
    fetched_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, date(fetched_at))  -- one snapshot per account per day
);
```

### Pattern 3: Singleton Cron Goroutine with Startup Sync

**What:** On startup, launch one goroutine that: (1) checks if full history has been pulled (flag or empty DB), (2) runs a full historical fetch if needed, (3) then schedules daily ticks via `time.Ticker` or a library like `robfig/cron`.

**When to use:** Small single-user services with a daily polling requirement. Avoids external job scheduler (no cron daemon required inside Docker).

**Trade-offs:** The goroutine dies with the process (acceptable — Docker restart handles this). State is in SQLite so restarts are safe.

**Example:**
```go
func StartCron(ctx context.Context, svc *SyncService) {
    go func() {
        if !svc.HasFullHistory(ctx) {
            svc.FetchFullHistory(ctx)
        }
        ticker := time.NewTicker(24 * time.Hour)
        for {
            select {
            case <-ticker.C:
                svc.FetchDaily(ctx)
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

## Data Flow

### Request Flow (Dashboard Load)

```
Browser loads /
    ↓
Nginx serves static React bundle (index.html + JS)
    ↓
React mounts, fires API calls:
  GET /api/accounts
  GET /api/balances/history?days=90
  GET /api/networth
    ↓
Nginx proxies /api/* → Go backend:8080
    ↓
Auth middleware checks session cookie
    ↓
chi handler → Service → Repository → SQLite query
    ↓
JSON response ← Service transforms rows into API types
    ↓
React renders panels and charts
```

### Sync Flow (Daily Cron)

```
time.Ticker fires (or startup)
    ↓
sync.FetchDaily()
    ↓
SimpleFIN HTTP client
  → GET https://[bridge-url]/accounts
  ← JSON: { accounts: [{ id, name, balance, transactions: [...] }] }
    ↓
ingest.Process()
  → Upsert accounts table (id, name, type, org)
  → Insert balance_snapshots (account_id, balance, fetched_at)
     (UNIQUE constraint deduplicates same-day re-runs)
  → Upsert transactions (id, account_id, amount, posted_date, description)
    ↓
SQLite updated — next API request sees fresh data
```

### Key Data Flows

1. **Balance history for charts:** Cron inserts one snapshot row per account per day. Chart endpoint queries `balance_snapshots` grouped by date, aggregated by account type (liquid, savings, investments). No computation from transactions needed.

2. **Liquid balance (net spendable):** Service layer sums checking account balances minus credit card balances (including pending), reading the most recent snapshot per account. Business logic lives in service, not SQL.

3. **Auth flow:** Simple: POST /api/login with password, server sets an HttpOnly session cookie (signed token or server-side session map). All other `/api/*` routes check for valid cookie via middleware. No JWT complexity needed for single-user.

## Scaling Considerations

This is a single-user self-hosted tool. Scaling is not a real concern. Document only what breaks first if misused.

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 1 user (target) | Current architecture is correct. SQLite with WAL mode handles concurrent reads fine. |
| 5-10 users | SQLite write contention becomes possible. Use connection pool with single writer. Still fine. |
| Multi-user | Would require Postgres + proper auth (out of scope per PROJECT.md) |

### Scaling Priorities

1. **First bottleneck:** SQLite write lock during cron sync while HTTP reads are happening. Mitigation: enable WAL mode (`PRAGMA journal_mode=WAL`) — allows concurrent reads during writes.
2. **Non-issue:** API throughput. A single user will never generate enough requests to stress a Go HTTP server.

## Anti-Patterns

### Anti-Pattern 1: Computing Balances from Transactions

**What people do:** Store only transactions, derive current and historical balances by summing transactions.

**Why it's wrong:** SimpleFIN does not guarantee complete transaction history (institutions limit history). Derived balance totals will drift from reality. Account balances provided directly by SimpleFIN are authoritative.

**Do this instead:** Store the balance field provided directly in the SimpleFIN account response as a daily snapshot. Use transaction data only for drill-down display, never for balance computation.

### Anti-Pattern 2: Fetching Live from SimpleFIN on Every Request

**What people do:** Call SimpleFIN from the API handler on every dashboard load.

**Why it's wrong:** SimpleFIN bridges are rate-limited and add ~500ms-2s latency. The dashboard becomes slow and fragile to external availability.

**Do this instead:** Cron fetches and stores; API always reads from local SQLite. The dashboard is fast and works even if SimpleFIN is temporarily down.

### Anti-Pattern 3: Storing SimpleFIN Auth Token in Plain Environment Variable Without Rotation Plan

**What people do:** Hard-code the SimpleFIN access URL (which contains the auth token) in docker-compose.yml committed to git.

**Why it's wrong:** The access URL is a bearer credential. Committing it leaks it to anyone with repo access.

**Do this instead:** Pass via a `.env` file excluded by `.gitignore`, or Docker secrets. Document this in setup instructions as a required step.

### Anti-Pattern 4: Fat Handlers (Business Logic in HTTP Layer)

**What people do:** Write SQL queries or balance aggregation logic directly inside chi handler functions.

**Why it's wrong:** Can't test business logic without spinning up HTTP. Tightly couples transport to logic.

**Do this instead:** Handlers parse request, call service method, serialize response. All logic lives in service layer.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| SimpleFIN Bridge | HTTPS GET with Basic Auth (token embedded in URL). Pull model — we call them, they don't call us. | The "access URL" from SimpleFIN setup is a full authenticated URL. Store as secret. One-shot claim URL must be consumed once to get access URL. |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| Nginx ↔ Go backend | HTTP reverse proxy on Docker network | No TLS between them (same host); TLS terminates at Nginx |
| Go HTTP ↔ Go Service | Direct Go function calls (same process) | Pass `context.Context` for cancellation |
| Go Service ↔ Repository | Direct Go function calls with interface types | Interface allows mock in tests |
| Go Cron ↔ Repository | Direct Go function calls (same process) | Cron and HTTP share the same DB connection pool |
| React ↔ Go backend | REST JSON over HTTP (`/api/*`) | Nginx proxies; no WebSocket needed (data is not real-time) |
| Frontend build ↔ Nginx | Static file volume / COPY in Docker image | React `dist/` output served directly by Nginx |

## Build Order Implications

Dependencies drive the order. Each phase should be independently runnable/testable:

1. **DB schema + repository layer** — Everything else reads/writes data. Schema must be stable before services are built on it. Define `balance_snapshots`, `accounts`, `transactions` tables with migrations.

2. **SimpleFIN client + ingest** — Can be built and tested standalone (returns parsed structs) before any HTTP layer exists. Use a test access URL from SimpleFIN's sandbox or a recorded fixture.

3. **Service layer** — Build aggregation logic (liquid balance, net worth) against the repository interface. Unit-testable with mock repository.

4. **Cron worker** — Wires the SimpleFIN client to the ingest/repository. Requires steps 1-3.

5. **Go HTTP API** — Build endpoints once services exist. Auth middleware, JSON handlers, chi routing.

6. **React frontend** — Can begin with mock API responses (MSW or hardcoded fixtures). Wire to real API once step 5 is ready.

7. **Nginx + Docker Compose** — Integration layer. Wire all components together last, after each runs in isolation.

This ordering means backend and frontend can be built in parallel (steps 3-5 and step 6 respectively) once the DB schema is settled.

## Sources

- SimpleFIN protocol design (read-only pull, JSON accounts/transactions structure): https://www.simplefin.org/protocol.html
- Go project layout conventions: https://github.com/golang-standards/project-layout
- SQLite WAL mode for concurrent read/write: https://www.sqlite.org/wal.html
- chi router documentation: https://github.com/go-chi/chi
- Snapshot-based balance history: standard pattern used by Monarch Money, Copilot, YNAB (observed in community discussions)
- Docker multi-stage build for Go + Node: https://docs.docker.com/build/building/multi-stage/

---
*Architecture research for: self-hosted personal finance dashboard (Go/chi + React/TS + SQLite + Nginx + Docker)*
*Researched: 2026-03-15*
