# Stack Research

**Domain:** Self-hosted personal finance dashboard (Go + React + SQLite + Docker)
**Researched:** 2026-03-15
**Confidence:** HIGH (all core library versions verified via pkg.go.dev and official docs)

---

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.24+ | Backend API + background jobs | Decided. Single binary, low resource overhead, excellent stdlib HTTP server. |
| github.com/go-chi/chi/v5 | v5.2.5 | HTTP router | Decided. Pure stdlib-compatible, zero external deps. Latest confirmed via pkg.go.dev. |
| modernc.org/sqlite | v1.46.1 | SQLite driver | Pure Go — no CGo. Critical for Docker multi-arch builds. mattn/go-sqlite3 requires a C compiler in every build stage, bloating images and breaking cross-compilation. |
| React | 18+ | Frontend SPA | Decided. |
| TypeScript | 5+ | Type safety on frontend | Decided. |
| Vite | 6+ | Frontend build tool | Standard React+TS scaffolding tool in 2025. First-class Tailwind v4 plugin (`@tailwindcss/vite`). |
| Tailwind CSS | v4.2 | UI styling | CSS-first config (no tailwind.config.js), 3.7x faster builds than v3, native Vite plugin, zero runtime overhead. v4 released Jan 2025. |
| Docker | 20.10+ | Containerization | Decided. Multi-stage build: Node builder → Go builder → minimal `gcr.io/distroless/static` final image. |
| Nginx | 1.25+ | Reverse proxy | Decided. Serves pre-built React static files, proxies `/api/*` to Go backend. |

### Supporting Libraries — Go Backend

| Library | Version | Purpose | Why / When to Use |
|---------|---------|---------|-------------------|
| github.com/golang-migrate/migrate/v4 | v4.19.1 | DB schema migrations | Explicitly uses modernc.org/sqlite (no CGo). Embed migration SQL files with `go:embed`. Use `database/sqlite` driver name — NOT `sqlite3`. |
| github.com/go-chi/cors | v1.2.2 | CORS middleware | Required during development when React dev server (port 5173) calls Go backend (port 8080). Restrict to `localhost` origins in prod. |
| github.com/go-chi/httplog/v3 | v3.3.0 | Structured HTTP request logging | Built on stdlib `log/slog`. Auto-levels by HTTP status (5xx=error, 4xx=warn). Zero extra dependencies. |
| github.com/go-chi/httprate | v0.15.0 | Rate limiting middleware | Protect the login endpoint from brute-force. Sliding window, in-memory, no Redis needed for single-user. |
| github.com/go-chi/jwtauth/v5 | v5.4.0 | JWT auth middleware | Handles token extraction from `Authorization: Bearer` header and cookies. Protects all API routes with one `r.Use()`. Uses `lestrrat-go/jwx` internally — do NOT also add golang-jwt as a separate dep. |
| golang.org/x/crypto | v0.49.0 | bcrypt password hashing | Hash the single admin password at startup/config time. Use `bcrypt.GenerateFromPassword` for initial hash, `bcrypt.CompareHashAndPassword` on login. |
| github.com/shopspring/decimal | v1.4.0 | Financial arithmetic | Never use `float64` for money. SimpleFIN returns balance strings — parse to `decimal.Decimal`, compute net worth, store as TEXT in SQLite, marshal to string in JSON. |
| github.com/lmittmann/tint | v1.1.3 | Colorized slog output | Dev-only: tinted terminal output for `log/slog`. Zero dependencies. Swap for plain JSON handler in production (`PRETTY_LOGS=false`). |
| github.com/robfig/cron/v3 | v3.0.1 | Cron job scheduler | Powers the daily SimpleFIN sync goroutine. Last released 2020 but stable and used by 5,400+ packages. Alternative: use `time.AfterFunc` + simple ticker for a single daily job — simpler for a single cron entry. |
| github.com/joho/godotenv | v1.5.1 | `.env` file loading | Local dev secret management. Load before `os.Getenv` calls. Do NOT use in production Docker — pass env vars via `docker-compose.yml`. |

### Supporting Libraries — React Frontend

| Library | Version | Purpose | Why / When to Use |
|---------|---------|---------|-------------------|
| recharts | 2.x | Chart components | The dominant React charting library. Built on D3 + SVG. Native TypeScript types. Composable API (LineChart, AreaChart, PieChart, ResponsiveContainer) maps directly to the required charts: balance-over-time line, net worth donut, investment performance area. |
| @tanstack/react-query | v5+ | Server state / data fetching | Replaces manual `useEffect` + `fetch`. Handles caching, stale-while-revalidate, background refetch. Essential for a dashboard that polls `/api/accounts`, `/api/net-worth`, etc. |
| react-router-dom | v6+ | Client-side routing | SPA routing between dashboard, drill-down views, settings. Go backend serves `index.html` on all non-`/api/*` routes. |
| zustand | 4+ | Global UI state | Lightweight store (auth token, dark mode preference, selected account). Do NOT use Redux — overkill for a single-user app. |
| @tailwindcss/vite | v4+ | Tailwind Vite integration | Required for Tailwind v4 — replaces postcss config. `plugins: [tailwindcss()]` in `vite.config.ts`. |
| lucide-react | latest | Icons | Tree-shakable, TypeScript-native icon set. Pairs well with Tailwind. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `sqlc` | Generate type-safe Go from SQL queries | v1.30.0. Write SQL in `.sql` files, run `sqlc generate`, get fully typed `Queries` struct. Eliminates `rows.Scan` boilerplate. Configure `engine: sqlite` in `sqlc.yaml`. |
| golangci-lint | Go linter aggregator | Run in CI. Catches shadowed variables, error ignores, ineffective assignments. |
| `vite build` | Frontend production build | Outputs to `dist/`. Embed into Go binary or copy into Docker image for Nginx to serve. |
| air | Live reload for Go | `cosmtrek/air` — watches `.go` files, rebuilds binary. Only for local dev; not in Docker. |
| Docker Compose | Local dev orchestration | Single `docker-compose.yml` wiring Go API + Nginx. Use `build: .` with multi-stage Dockerfile. |

---

## Installation

```bash
# Go backend (go.mod)
go get github.com/go-chi/chi/v5@v5.2.5
go get github.com/go-chi/cors@v1.2.2
go get github.com/go-chi/httplog/v3@v3.3.0
go get github.com/go-chi/httprate@v0.15.0
go get github.com/go-chi/jwtauth/v5@v5.4.0
go get golang.org/x/crypto@v0.49.0
go get modernc.org/sqlite@v1.46.1
go get github.com/golang-migrate/migrate/v4@v4.19.1
go get github.com/golang-migrate/migrate/v4/database/sqlite
go get github.com/golang-migrate/migrate/v4/source/iofs
go get github.com/shopspring/decimal@v1.4.0
go get github.com/robfig/cron/v3@v3.0.1
go get github.com/lmittmann/tint@v1.1.3
go get github.com/joho/godotenv@v1.5.1

# React frontend
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install recharts @tanstack/react-query react-router-dom zustand lucide-react
npm install tailwindcss @tailwindcss/vite
npm install -D @types/react @types/react-dom typescript
```

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| modernc.org/sqlite | mattn/go-sqlite3 | Only if you need FTS5 extensions or WAL mode features that modernc doesn't support. Requires CGo, complicates Docker builds. |
| golang-migrate | pressly/goose | Goose has a nicer CLI UX but doesn't explicitly list modernc support. golang-migrate v4.19.1 explicitly uses modernc. Use goose if you prefer declarative migration status tracking over linear versioning. |
| go-chi/jwtauth | chi middleware.BasicAuth | BasicAuth stores plaintext passwords in code/env. Only acceptable if password stored in env var directly. JWT approach is more conventional for an SPA: login endpoint exchanges password for token, token sent in header. |
| recharts | Nivo | Nivo is more powerful and beautiful but ~3x larger bundle. Overkill for 3-4 chart types. Recharts' composable SVG API is sufficient and well-documented. |
| recharts | Chart.js / react-chartjs-2 | Chart.js renders to Canvas (recharts uses SVG). Canvas is faster at high data density (10k+ points). For daily snapshots of ~10 accounts over a few years (3,650 points max), SVG is fine. |
| @tanstack/react-query | SWR | Both are excellent. TanStack Query has more features (mutations, optimistic updates, devtools). SWR is simpler. Choose SWR if the app stays read-only forever. |
| zustand | Redux Toolkit | Redux adds unnecessary boilerplate for a single-user app with minimal shared state. |
| Tailwind v4 | Tailwind v3 | v3 is more battle-tested but v4 is stable (released Jan 2025), has official Vite plugin, and eliminates the config file. For a greenfield project, v4 is correct. |
| robfig/cron | time.Ticker | For a single daily job, a simple `time.AfterFunc` or ticker loop is simpler than a cron library. Use robfig/cron if you anticipate multiple schedules or need standard cron expression syntax. |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| mattn/go-sqlite3 | Requires CGo. In Docker, the build stage must have `gcc` installed, adding ~100MB to the builder image and preventing cross-compilation. | modernc.org/sqlite (pure Go, same database/sql interface) |
| float64 for money | Binary floating point cannot represent `0.1` exactly. `0.1 + 0.2 = 0.30000000000000004`. Financial calculations will accumulate errors. | shopspring/decimal — parse SimpleFIN balance strings directly to Decimal |
| pressly/goose with modernc | goose v3 sqlite3 driver name resolves to mattn/go-sqlite3 by default; using it with modernc requires custom wiring that isn't documented. | golang-migrate v4.19.1 which explicitly uses modernc |
| Redux / Redux Toolkit | Significant boilerplate overhead for a single-user app with ~3 pieces of shared state (auth, dark mode, selected account). | zustand (tiny, hook-based, no reducers) |
| axios | Adds 40KB for a fetch wrapper. TanStack Query already wraps fetch elegantly; native fetch API is sufficient. | fetch + @tanstack/react-query |
| GORM / sqlx ORM | GORM adds hidden N+1 queries and magic that makes SQL debugging harder. sqlx is better but still less safe than generated code. | sqlc for type-safe generated queries + raw database/sql for complex cases |
| Ent (Go ORM) | Heavy framework with code generation patterns that obscure the SQL layer. Overkill for a small, well-understood schema. | sqlc |
| Next.js / Remix | Server-side rendering adds complexity with no benefit for a self-hosted, single-user, already-authenticated dashboard. | Vite + React SPA |
| go-chi/jwtauth with golang-jwt added separately | jwtauth v5 uses lestrrat-go/jwx internally — adding golang-jwt as a second dep creates confusion and two JWT libraries in the dependency tree. | Pick jwtauth (it handles everything) OR golang-jwt + custom middleware |

---

## Stack Patterns by Variant

**For SimpleFIN client implementation:**
- Do NOT use `github.com/jazzboME/simplefin` (v0.1.2, pre-stable, limited error handling, missing token setup flow)
- Implement a small internal `internal/simplefin/client.go` (~80 lines):
  - `ClaimAccessURL(setupToken string) (accessURL string, error)` — POST to setup token URL to get permanent access URL
  - `FetchAccounts(accessURL string) ([]Account, error)` — GET access URL with HTTP Basic Auth using access token
  - SimpleFIN returns JSON: `{"errors": [], "accounts": [{"id": "...", "name": "...", "balance": "123.45", "currency": "USD", "transactions": [...]}]}`
  - Store the access URL (not setup token) in the database after initial claim

**For Docker multi-stage build:**
```dockerfile
# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.24-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o finance-app ./cmd/server

# Stage 3: Minimal final image
FROM gcr.io/distroless/static:nonroot
COPY --from=go-builder /app/finance-app /
EXPOSE 8080
ENTRYPOINT ["/finance-app"]
```
- `CGO_ENABLED=0` is required. modernc.org/sqlite is pure Go so this works.
- Embed `frontend/dist` into Go binary using `go:embed` or serve via Nginx separately.
- Prefer Nginx serving static files (better caching headers) with Go only handling `/api/*`.

**For password protection (simple single-user auth):**
- Store hashed password (bcrypt cost 12) in SQLite or env var
- `POST /api/auth/login` — verify password, issue JWT (24h expiry, HS256)
- All `/api/*` routes except `/api/auth/login` protected by `jwtauth.Verifier` + `jwtauth.Authenticator`
- Frontend stores JWT in `localStorage` (acceptable for single-user self-hosted; no XSS risk from third parties)
- `go-chi/httprate.LimitByIP(10, time.Minute)` on the login endpoint only

**For the daily cron job:**
- Use `robfig/cron/v3` with `cron.New(cron.WithSeconds())` if second-precision needed, or plain cron expression `"0 6 * * *"` for 6 AM daily
- Alternatively for a single job: start a goroutine in `main()` that sleeps until next 6 AM then loops — avoids the cron dep entirely
- Always run cron in same process as API server (no separate container needed for this scale)

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| golang-migrate/migrate v4.19.1 | modernc.org/sqlite v1.46.1 | Use `database/sqlite` (NOT `database/sqlite3`) as the driver import. URL format: `sqlite://./finance.db` |
| go-chi/jwtauth v5.4.0 | go-chi/chi v5.2.5 | jwtauth is framework-agnostic; the `v5` module label refers to JWT spec version, not chi version |
| Tailwind v4.2 | Vite 6+ | Use `@tailwindcss/vite` plugin. Do NOT use `@tailwindcss/postcss` — the Vite plugin is simpler and faster |
| recharts 2.x | React 18+ | Peer dep is React >= 16.8; React 18 fully supported |
| @tanstack/react-query v5 | React 18+ | v5 dropped React 16 support. Requires React 18+ |
| sqlc v1.30.0 | modernc.org/sqlite | Set `engine: sqlite` in `sqlc.yaml`. Generated code uses `database/sql` so works with any driver |

---

## Sources

- `pkg.go.dev/modernc.org/sqlite` — v1.46.1 confirmed, pure Go, SQLite 3.51.2, Feb 2026
- `pkg.go.dev/github.com/go-chi/chi/v5` — v5.2.5 confirmed, Feb 2026
- `pkg.go.dev/github.com/go-chi/cors` — v1.2.2 confirmed, Jul 2025
- `pkg.go.dev/github.com/go-chi/httplog/v3` — v3.3.0 confirmed, slog-based, Sep 2025
- `pkg.go.dev/github.com/go-chi/httprate` — v0.15.0 confirmed, Mar 2025
- `pkg.go.dev/github.com/go-chi/jwtauth/v5` — v5.4.0 confirmed, uses lestrrat-go/jwx, Feb 2026
- `pkg.go.dev/github.com/golang-migrate/migrate/v4/database/sqlite` — v4.19.1, explicitly uses modernc.org/sqlite, Nov 2025
- `pkg.go.dev/github.com/golang-jwt/jwt/v5` — v5.3.1 confirmed, Jan 2026
- `pkg.go.dev/golang.org/x/crypto` — v0.49.0 confirmed, bcrypt available, Mar 2026
- `pkg.go.dev/github.com/shopspring/decimal` — v1.4.0 confirmed, Apr 2024
- `pkg.go.dev/github.com/lmittmann/tint` — v1.1.3 confirmed, Feb 2026
- `pkg.go.dev/github.com/robfig/cron/v3` — v3.0.1 (stable, last release Jan 2020, 5400+ dependents)
- `pkg.go.dev/github.com/pressly/goose/v3` — v3.27.0, but NOT recommended (modernc compat unclear)
- `pkg.go.dev/github.com/sqlc-dev/sqlc` — v1.30.0 confirmed, Sep 2025
- `tailwindcss.com/docs/installation` — v4.2 confirmed, Vite plugin documented, Mar 2026
- tailwindcss.com blog — v4.0 released Jan 22, 2025
- `pkg.go.dev/search?q=simplefin` — No mature Go SimpleFIN client found; custom implementation recommended
- Training data (MEDIUM confidence): recharts 2.x, TanStack Query v5, Vite 6, React Router v6, zustand v4 — all widely used, versions consistent with ecosystem knowledge

---

*Stack research for: self-hosted personal finance dashboard (Go + React + SQLite + Docker)*
*Researched: 2026-03-15*
