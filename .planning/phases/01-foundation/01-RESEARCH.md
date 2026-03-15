# Phase 1: Foundation - Research

**Researched:** 2026-03-15
**Domain:** Go backend scaffold, SQLite + golang-migrate, bcrypt + JWT auth, Docker Compose dev/prod, React+Vite scaffold
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Password-only login (no username) — single-user self-hosted app
- 30-day JWT session duration
- Password configured via environment variable (`PASSWORD` or `PASSWORD_HASH` in .env / docker-compose)
- Rate-limited retries: 5 attempts, then 30-second cooldown
- Full schema created upfront in Phase 1 migration — all tables (accounts, balance_snapshots, settings, sync_log) built now
- Settings table (key-value) for password hash and future config
- Balance snapshots: one row per account per day with unique constraint
- Air for Go backend hot-reload during development
- Vite dev server with HMR for React frontend; Nginx proxies to it in dev mode
- Separate Docker Compose files: `docker-compose.yml` (dev) and `docker-compose.prod.yml` (production)
- Named Docker volume for SQLite database persistence
- Go backend: `cmd/internal` pattern (`cmd/server/main.go`, `internal/` for packages)
- Monorepo: Go backend at root, React frontend in `frontend/` directory
- Database migrations in top-level `migrations/` directory (numbered SQL files)
- Tests colocated with source files (`_test.go` beside source, `.test.tsx` beside components)

### Claude's Discretion
- Account type categorization approach (enum column vs other)
- Exact login page styling and layout
- Nginx configuration details
- Internal package organization within `internal/`

### Deferred Ideas (OUT OF SCOPE)
- None — discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| AUTH-01 | App is protected by a simple password gate | bcrypt + go-chi/jwtauth JWT middleware; httprate rate limiting on login endpoint; HttpOnly cookie storage |
| DEPLOY-01 | App runs as Docker containers (Go backend, React frontend, Nginx reverse proxy) | Multi-stage Dockerfile; docker-compose.yml (dev with Air + Vite); docker-compose.prod.yml (production with prebuilt static assets) |
</phase_requirements>

---

## Summary

Phase 1 establishes the complete technical foundation: a running Go backend with chi router, SQLite schema under golang-migrate management, bcrypt+JWT authentication with rate limiting, and a Docker Compose environment with separate dev (Air hot-reload + Vite HMR) and prod (static build + Nginx) configurations. A React+Vite+Tailwind scaffold in `frontend/` completes the frontend baseline, even though it shows only a login page for this phase.

The critical constraints are: (1) use `modernc.org/sqlite` throughout — CGo-free is required for clean Docker multi-stage builds; (2) use `github.com/golang-migrate/migrate/v4/database/sqlite` (not `sqlite3`) which is the modernc-backed driver; (3) set WAL mode via a `RegisterConnectionHook` or DSN pragmas at connection open time — not after — and `busy_timeout=5000` alongside it; (4) store JWT in an HttpOnly cookie (not localStorage) for a financial app.

The Phase 1 output is minimal but correct: visiting the app URL redirects to a login page, entering the correct password issues a 30-day JWT cookie, wrong password is rejected (with rate limiting), `docker compose up` starts the full stack with no manual steps, and SQLite initializes via migrations with WAL mode active.

**Primary recommendation:** Build the DB layer first (connection + migrations + schema), then auth (bcrypt + JWT middleware), then the login handler, then wire Docker Compose. This ordering unblocks all subsequent phases.

---

## Standard Stack

### Core (Phase 1 relevant)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `modernc.org/sqlite` | v1.46.1 | SQLite driver | Pure Go — no CGo; required for Docker multi-arch builds. Confirmed Feb 2026 via pkg.go.dev. |
| `github.com/golang-migrate/migrate/v4` | v4.19.1 | Schema migrations | Explicit modernc support via `database/sqlite` sub-package. `go:embed` + iofs pattern embeds SQL into binary. |
| `github.com/go-chi/chi/v5` | v5.2.5 | HTTP router | Pure stdlib-compatible; zero deps. Confirmed Feb 2026. |
| `github.com/go-chi/jwtauth/v5` | v5.4.0 | JWT middleware | Handles `Authorization: Bearer` header AND cookie named `"jwt"` via `TokenFromCookie`. Confirmed Feb 2026. |
| `golang.org/x/crypto` | v0.49.0 | bcrypt | `bcrypt.GenerateFromPassword` / `bcrypt.CompareHashAndPassword`. Confirmed Mar 2026. |
| `github.com/go-chi/httprate` | v0.15.0 | Rate limiting | `httprate.LimitByIP(5, 30*time.Second)` on the login route. Confirmed Mar 2025. |
| `github.com/go-chi/cors` | v1.2.2 | CORS | Required in dev when Vite (5173) calls Go (8080). |
| `github.com/go-chi/httplog/v3` | v3.3.0 | Structured logging | Built on `log/slog`. Auto-levels by HTTP status. |
| `github.com/joho/godotenv` | v1.5.1 | .env loading | Dev-only. Do not use in production Docker. |
| `github.com/lmittmann/tint` | v1.1.3 | Dev log colors | `log/slog` colorizer for terminal. Zero deps. |

### Frontend (scaffold only in Phase 1)

| Library | Version | Purpose | Why |
|---------|---------|---------|-----|
| React | 18+ | SPA | Decided. |
| TypeScript | 5+ | Type safety | Decided. |
| Vite | 6+ | Build + dev server | Standard React+TS scaffold tool 2025. |
| Tailwind CSS | v4.2 | Styling | CSS-first config (no tailwind.config.js), native Vite plugin. |
| `@tailwindcss/vite` | v4+ | Tailwind Vite integration | Required for v4. `plugins: [tailwindcss()]` in `vite.config.ts`. |

### Installation

```bash
# Go backend (from repo root)
go mod init github.com/youruser/finance-visualizer
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
go get github.com/lmittmann/tint@v1.1.3
go get github.com/joho/godotenv@v1.5.1

# React frontend
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install tailwindcss @tailwindcss/vite
npm install -D @types/react @types/react-dom typescript
```

---

## Architecture Patterns

### Recommended Project Structure

```
finance-visualizer/
├── cmd/
│   └── server/
│       └── main.go              # Entrypoint: wires DB, migrations, router, server
├── internal/
│   ├── db/
│   │   └── db.go                # sql.Open, RegisterConnectionHook (WAL + busy_timeout)
│   ├── migrations/              # OR top-level migrations/ — see decision below
│   │   ├── 000001_init.up.sql
│   │   └── 000001_init.down.sql
│   ├── auth/
│   │   ├── auth.go              # bcrypt verify, JWT encode/decode helpers
│   │   └── auth_test.go
│   ├── api/
│   │   ├── router.go            # chi router, middleware chain, route registration
│   │   └── handlers/
│   │       ├── auth.go          # POST /api/auth/login handler
│   │       └── health.go        # GET /api/health
│   └── config/
│       └── config.go            # Read env vars: PASSWORD_HASH, JWT_SECRET, PORT, etc.
├── migrations/                  # Top-level — numbered SQL migration files
│   ├── 000001_init.up.sql
│   └── 000001_init.down.sql
├── frontend/
│   ├── src/
│   │   ├── pages/
│   │   │   └── Login.tsx        # Login page (Phase 1 only)
│   │   ├── api/
│   │   │   └── client.ts        # Typed fetch wrappers
│   │   └── main.tsx
│   ├── vite.config.ts
│   └── package.json
├── nginx/
│   ├── nginx.dev.conf           # Dev: proxy / to Vite:5173, /api/* to backend:8080
│   └── nginx.prod.conf          # Prod: serve /app/dist static, proxy /api/* to backend:8080
├── docker-compose.yml           # Dev: Air hot-reload backend + Vite frontend + Nginx
├── docker-compose.prod.yml      # Prod: prebuilt Go binary + Nginx serving dist/
├── Dockerfile                   # Multi-stage: node builder + go builder + distroless final
└── .env.example                 # Template: PASSWORD_HASH=, JWT_SECRET=, PORT=8080
```

**Note on migrations directory:** CONTEXT.md specifies top-level `migrations/`. Use `//go:embed` from within `internal/db/db.go` or `cmd/server/main.go` with a relative path. Go `go:embed` requires the directory to be within or below the package declaring the embed — either keep migrations inside `internal/db/migrations/` (simplest for embedding) or embed from `main.go` at the root. Recommend `internal/db/migrations/` for clean embedding.

### Pattern 1: SQLite Connection with WAL Mode via RegisterConnectionHook

**What:** Apply WAL mode and busy_timeout pragmas to every connection via a registered hook. This is safer than DSN parameters alone because it applies to all connections in the pool.

**When to use:** Always, at application startup before any other DB operation.

```go
// Source: https://theitsolutions.io/blog/modernc.org-sqlite-with-go
// internal/db/db.go

package db

import (
    "context"
    "database/sql"
    _ "modernc.org/sqlite"
    "modernc.org/sqlite"
)

const initSQL = `
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA foreign_keys = ON;
`

func Open(dsn string) (*sql.DB, error) {
    sqlite.RegisterConnectionHook(func(conn sqlite.ExecQuerierContext, _ string) error {
        _, err := conn.ExecContext(context.Background(), initSQL, nil)
        return err
    })
    db, err := sql.Open("sqlite", dsn)
    if err != nil {
        return nil, err
    }
    // Single writer — prevents "database is locked" under concurrent writes
    db.SetMaxOpenConns(1)
    return db, nil
}
```

**Alternative (DSN-only):** `"file:finance.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"` — works but RegisterConnectionHook is cleaner for multi-pragma setups.

### Pattern 2: golang-migrate with go:embed + iofs

**What:** Embed migration SQL files into the binary at compile time. Run migrations at startup before serving HTTP.

**Critical:** Use the `database/sqlite` driver (modernc-backed), NOT `database/sqlite3` (CGo-backed).

```go
// Source: pkg.go.dev/github.com/golang-migrate/migrate/v4/source/iofs
// cmd/server/main.go or internal/db/migrations.go

package db

import (
    "embed"
    "errors"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/sqlite"  // modernc driver
    "github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(dsn string) error {
    src, err := iofs.New(migrationsFS, "migrations")
    if err != nil {
        return err
    }
    m, err := migrate.NewWithSourceInstance("iofs", src, "sqlite://"+dsn)
    if err != nil {
        return err
    }
    if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
        return err
    }
    return nil
}
```

**Migration file naming convention:** `000001_init.up.sql` / `000001_init.down.sql`. Migrations must NOT contain explicit `BEGIN`/`COMMIT` — the driver wraps each file in an implicit transaction.

### Pattern 3: bcrypt + JWT Auth with HttpOnly Cookie

**What:** On login, verify bcrypt password, issue JWT in an HttpOnly SameSite=Strict cookie. All `/api/*` routes except login use jwtauth middleware.

```go
// Source: pkg.go.dev/github.com/go-chi/jwtauth/v5
// internal/auth/auth.go

var tokenAuth *jwtauth.JWTAuth

func Init(secret string) {
    tokenAuth = jwtauth.New("HS256", []byte(secret), nil)
}

// Login handler: POST /api/auth/login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
    var req struct{ Password string `json:"password"` }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
        return
    }

    storedHash := getPasswordHashFromSettings() // from DB settings table
    if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password)); err != nil {
        http.Error(w, `{"error":"invalid password"}`, http.StatusUnauthorized)
        return
    }

    _, tokenString, _ := tokenAuth.Encode(map[string]interface{}{
        "exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
    })

    http.SetCookie(w, &http.Cookie{
        Name:     "jwt",          // jwtauth.TokenFromCookie looks for "jwt"
        Value:    tokenString,
        Path:     "/",
        HttpOnly: true,
        SameSite: http.SameSiteStrictMode,
        MaxAge:   30 * 24 * 3600,
    })
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"ok":true}`))
}
```

### Pattern 4: Chi Router with Auth Middleware and Rate Limiting

```go
// Source: github.com/go-chi/chi, github.com/go-chi/httprate, github.com/go-chi/jwtauth/v5
// internal/api/router.go

func NewRouter(tokenAuth *jwtauth.JWTAuth) chi.Router {
    r := chi.NewRouter()
    r.Use(httplog.RequestLogger(logger))
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins: []string{"http://localhost:5173"}, // dev only
        AllowCredentials: true,
    }))

    // Public routes
    r.Group(func(r chi.Router) {
        // 5 attempts per 30-second window per IP
        r.With(httprate.LimitByIP(5, 30*time.Second)).Post("/api/auth/login", LoginHandler)
    })

    // Protected routes
    r.Group(func(r chi.Router) {
        r.Use(jwtauth.Verifier(tokenAuth))      // extract token from header or "jwt" cookie
        r.Use(jwtauth.Authenticator(tokenAuth)) // reject 401 if invalid/missing
        r.Get("/api/health", HealthHandler)
        // Phase 3+ routes added here
    })

    return r
}
```

### Pattern 5: Docker Compose Dev/Prod Split

**Dev (`docker-compose.yml`):**
```yaml
services:
  backend:
    build:
      context: .
      target: dev          # Multi-stage Dockerfile target
    command: air
    volumes:
      - .:/app             # Source mount for Air to watch
    environment:
      - PORT=8080
      - JWT_SECRET=${JWT_SECRET}
      - PASSWORD_HASH=${PASSWORD_HASH}
    ports:
      - "8080:8080"

  frontend:
    image: node:22-alpine
    working_dir: /app
    command: npm run dev -- --host 0.0.0.0
    volumes:
      - ./frontend:/app
    ports:
      - "5173:5173"
    environment:
      - CHOKIDAR_USEPOLLING=true   # Required for file-watching inside Docker

  nginx:
    image: nginx:1.25-alpine
    volumes:
      - ./nginx/nginx.dev.conf:/etc/nginx/conf.d/default.conf
    ports:
      - "80:80"
    depends_on:
      - backend
      - frontend

  db_vol:   # Not a service — use named volume
volumes:
  sqlite_data:
```

**Nginx dev config (`nginx/nginx.dev.conf`):**
```nginx
server {
    listen 80;

    # Vite HMR WebSocket — must come before location /
    location /vite-hmr {
        proxy_pass http://frontend:5173;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
    }

    location /api/ {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location / {
        proxy_pass http://frontend:5173;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
    }
}
```

**Prod (`docker-compose.prod.yml`):**
```yaml
services:
  backend:
    build:
      context: .
      target: prod
    environment:
      - PORT=8080
      - JWT_SECRET=${JWT_SECRET}
      - PASSWORD_HASH=${PASSWORD_HASH}
    volumes:
      - sqlite_data:/data
    restart: unless-stopped

  nginx:
    build:
      context: .
      target: nginx-prod
    ports:
      - "80:80"
    depends_on:
      - backend
    restart: unless-stopped

volumes:
  sqlite_data:
```

### Pattern 6: Upfront Full Schema Migration (Phase 1)

All tables needed by all phases are created in a single migration file. Later phases add code, not schema.

```sql
-- migrations/000001_init.up.sql

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS accounts (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    account_type TEXT NOT NULL CHECK(account_type IN ('checking', 'savings', 'credit', 'investment', 'other')),
    currency     TEXT NOT NULL DEFAULT 'USD',
    org_name     TEXT,
    org_slug     TEXT,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS balance_snapshots (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id   TEXT NOT NULL REFERENCES accounts(id),
    balance      TEXT NOT NULL,         -- stored as string (shopspring/decimal on read)
    balance_date DATE NOT NULL,         -- from SimpleFIN balance-date field
    fetched_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, balance_date)    -- append-only: ON CONFLICT DO NOTHING at write time
);

CREATE INDEX IF NOT EXISTS idx_balance_snapshots_account_date
    ON balance_snapshots(account_id, balance_date DESC);

CREATE TABLE IF NOT EXISTS sync_log (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at       DATETIME NOT NULL,
    finished_at      DATETIME,
    accounts_fetched INTEGER DEFAULT 0,
    accounts_failed  INTEGER DEFAULT 0,
    error_text       TEXT
);
```

**account_type approach (Claude's Discretion):** Use a `CHECK` constraint with an enum-like text column. This is simpler than a separate lookup table and is readable in SQL queries. The values map directly to dashboard panels: `checking` + `savings` = liquid panel sources; `savings` = savings panel; `investment` = investments panel; `credit` = liquid panel subtraction.

### Anti-Patterns to Avoid

- **Using `database/sqlite3` import:** That is the CGo mattn driver. Use `database/sqlite` for modernc.
- **Setting WAL pragma after opening other connections:** WAL must be set before any reads/writes. Use `RegisterConnectionHook`.
- **JWT in localStorage:** XSS risk for a financial app. Use HttpOnly cookie named `"jwt"`.
- **Rate limiting entire router:** Apply `httprate.LimitByIP` only on the login route, not globally.
- **Explicit BEGIN/COMMIT in migration SQL:** golang-migrate wraps each file in a transaction; explicit statements cause errors.
- **Setting `MaxOpenConns > 1` for writes:** SQLite is single-writer. Keep writer pool at 1.
- **Separate reader/writer DB pools in Phase 1:** WAL mode handles concurrent reads fine for this phase's load. Add reader pool only if profiling shows contention (not expected at single-user scale).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Schema migrations | Custom `CREATE TABLE IF NOT EXISTS` in Go init code | `golang-migrate` with `go:embed` | Schema drift between versions, no rollback, no migration history |
| JWT issuance and verification | Custom HMAC JWT implementation | `go-chi/jwtauth/v5` | Correct `exp` handling, cookie + header extraction, `lestrrat-go/jwx` internals |
| Login rate limiting | In-memory counter with mutexes | `go-chi/httprate` | Sliding window, IP key extraction, correct 429 handling, reset logic |
| Password hashing | SHA-256 or MD5 | `golang.org/x/crypto/bcrypt` cost 12 | Timing-safe, salted, computationally expensive by design |
| SQLite WAL setup | Custom pragma executor at startup | `sqlite.RegisterConnectionHook` | Applies to every connection including pooled connections, not just the first |
| .env loading | Manual `os.ReadFile` + `strings.Split` | `github.com/joho/godotenv` | Handles quotes, comments, multiline values, returns clear errors |

**Key insight:** For a financial app, the auth and DB migration layers are correctness-critical. Using maintained libraries for these eliminates entire categories of bugs (timing attacks, schema drift, lock contention).

---

## Common Pitfalls

### Pitfall 1: Wrong golang-migrate Driver Import
**What goes wrong:** Importing `_ "github.com/golang-migrate/migrate/v4/database/sqlite3"` pulls in mattn/go-sqlite3 (CGo), breaking Docker builds (`CGO_ENABLED=0` will fail to compile).
**Why it happens:** Documentation examples often show `sqlite3`; the `sqlite` (modernc) driver is a separate sub-package.
**How to avoid:** Import `_ "github.com/golang-migrate/migrate/v4/database/sqlite"` (no trailing `3`). URL scheme is `sqlite://` (not `sqlite3://`).
**Warning signs:** Build fails with `undefined: Ctypes` or similar CGo errors.

### Pitfall 2: WAL Mode Not Set Before First Connection
**What goes wrong:** WAL pragma applied after connections are already open has no effect on existing connections. Default rollback journal causes write-lock to block all HTTP reads during cron sync (Phase 2+).
**Why it happens:** Pragma run as a one-off SQL statement, not via `RegisterConnectionHook`.
**How to avoid:** Use `sqlite.RegisterConnectionHook` before calling `sql.Open`. Set `journal_mode=WAL` AND `busy_timeout=5000` in the hook.
**Warning signs:** No `-wal` file in the data directory after the first write.

### Pitfall 3: JWT Cookie Named Wrongly
**What goes wrong:** `jwtauth.TokenFromCookie` specifically looks for a cookie named `"jwt"`. A cookie named `"token"`, `"auth"`, `"session"`, etc. is silently ignored — the middleware falls back to header-only auth.
**Why it happens:** Developers name cookies intuitively without reading the jwtauth source.
**How to avoid:** Set cookie `Name: "jwt"` exactly. Confirmed in pkg.go.dev documentation.

### Pitfall 4: Vite HMR Broken Through Nginx in Dev
**What goes wrong:** Vite uses WebSocket for HMR. Without `proxy_set_header Upgrade $http_upgrade; proxy_set_header Connection "Upgrade";` in Nginx, the WebSocket upgrade is stripped and HMR silently fails — file changes do not hot-reload.
**Why it happens:** Nginx HTTP proxying does not forward WebSocket upgrade headers by default.
**How to avoid:** Add both Upgrade headers to both the general `/` proxy block and any HMR-specific location. Also set `CHOKIDAR_USEPOLLING=true` in docker-compose for file-watching inside containers.

### Pitfall 5: Password Hash Not Set on First Run
**What goes wrong:** If `PASSWORD_HASH` env var is empty or malformed at startup, the bcrypt compare on login always fails. No clear error surfaced to the operator.
**Why it happens:** Env var not documented, not validated at startup.
**How to avoid:** Validate `PASSWORD_HASH` at startup: if empty, log a fatal error with instructions on how to generate it (`htpasswd -bnBC 12 "" mypassword | tr -d ':\n'`). Also support `PASSWORD` (plaintext) env var for first-run convenience: hash it at startup, store hash in settings table, warn in logs.

### Pitfall 6: Migration SQL Contains BEGIN/COMMIT
**What goes wrong:** golang-migrate wraps each `.sql` file in an implicit transaction. If the file also contains `BEGIN` or `COMMIT`, the migration fails with a transaction nesting error.
**How to avoid:** Write migration SQL without explicit transaction control statements. Use `x-no-tx-wrap=true` in the DSN only if you genuinely need multi-statement transactions across DDL (rare).

### Pitfall 7: Named Volume Not Declared in Compose
**What goes wrong:** `docker-compose.yml` references `sqlite_data:/data` but the `volumes:` top-level key is missing. Docker silently creates an anonymous volume that is lost on `docker compose down -v`.
**How to avoid:** Always declare named volumes at the top level of the compose file. Verify with `docker volume ls` after first `docker compose up`.

---

## Code Examples

### Database Connection Setup (Verified Pattern)
```go
// Source: https://theitsolutions.io/blog/modernc.org-sqlite-with-go
// internal/db/db.go
package db

import (
    "context"
    "database/sql"
    "modernc.org/sqlite"
    _ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
    sqlite.RegisterConnectionHook(func(conn sqlite.ExecQuerierContext, _ string) error {
        _, err := conn.ExecContext(context.Background(), `
            PRAGMA journal_mode = WAL;
            PRAGMA busy_timeout = 5000;
            PRAGMA foreign_keys = ON;
        `, nil)
        return err
    })
    db, err := sql.Open("sqlite", "file:"+path)
    if err != nil {
        return nil, err
    }
    db.SetMaxOpenConns(1)
    return db, nil
}
```

### Migration Runner with Embedded SQL (Verified Pattern)
```go
// Source: pkg.go.dev/github.com/golang-migrate/migrate/v4/source/iofs
package db

import (
    "embed"
    "errors"
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/sqlite" // modernc, NOT sqlite3
    "github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Migrate(dsn string) error {
    src, err := iofs.New(migrationsFS, "migrations")
    if err != nil {
        return err
    }
    m, err := migrate.NewWithSourceInstance("iofs", src, "sqlite://"+dsn)
    if err != nil {
        return err
    }
    if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
        return err
    }
    return nil
}
```

### Rate-Limited Login Route
```go
// Source: github.com/go-chi/httprate README
r.With(httprate.LimitByIP(5, 30*time.Second)).Post("/api/auth/login", handlers.Login)
```

### In-Memory SQLite for Tests
```go
// Source: modernc.org/sqlite docs — ":memory:" DSN
func TestDB(t *testing.T) *sql.DB {
    db, err := Open(":memory:")
    if err != nil {
        t.Fatal(err)
    }
    if err := Migrate(":memory:"); err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() { db.Close() })
    return db
}
```

### Vite Config for Docker HMR
```typescript
// frontend/vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: '0.0.0.0',         // bind to all interfaces inside Docker
    port: 5173,
    watch: {
      usePolling: true,        // required for Docker volume file watching
    },
  },
})
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| mattn/go-sqlite3 (CGo) | modernc.org/sqlite (pure Go) | ~2021, stable by 2023 | Clean Docker builds without gcc; cross-compilation works |
| tailwind.config.js | CSS-first config in CSS file | Tailwind v4, Jan 2025 | No config file needed; Vite plugin replaces PostCSS setup |
| postcss for Tailwind | `@tailwindcss/vite` plugin | Tailwind v4, Jan 2025 | Simpler Vite config; `plugins: [tailwindcss()]` only |
| golang-jwt/jwt (manual) | go-chi/jwtauth/v5 (middleware) | Ongoing | Middleware handles extraction from header + cookie; less boilerplate |
| localStorage for JWT | HttpOnly cookie | Best practice (ongoing) | XSS-resistant token storage for financial apps |

**Deprecated/outdated:**
- `database/sqlite3` in golang-migrate: The mattn/CGo backed driver. Do not use in this project.
- `pressly/goose` with modernc: goose's sqlite3 driver resolves to mattn by default; not recommended (see STACK.md).
- `CHOKIDAR_USEPOLLING` env var: Still works but `server.watch.usePolling: true` in `vite.config.ts` is the canonical Vite v6 approach.

---

## Open Questions

1. **migrations/ directory location for go:embed**
   - What we know: `go:embed` requires the embedded directory to be within or below the package containing the directive. CONTEXT.md specifies top-level `migrations/`.
   - What's unclear: If migrations live at repo root and `go:embed` is declared in `cmd/server/main.go`, that works. If declared in `internal/db/db.go`, migrations must be under `internal/db/migrations/`.
   - Recommendation: Place migrations at `internal/db/migrations/` (embed from `db` package) OR at root and embed from `main.go`. Pick one and document it. Either works; `internal/db/migrations/` keeps migration code co-located with DB code.

2. **PASSWORD vs PASSWORD_HASH env var bootstrap**
   - What we know: CONTEXT.md says password configured via `PASSWORD` or `PASSWORD_HASH` env var.
   - What's unclear: First-run UX — if operator sets `PASSWORD=mypassword`, should the app hash it at startup and store in settings table? Or require pre-hashed `PASSWORD_HASH`?
   - Recommendation: Support both. If `PASSWORD` is set, bcrypt-hash it and write to settings table on startup (with a startup log warning). If `PASSWORD_HASH` is set, use directly. Fail fast with a clear error if neither is set.

---

## Validation Architecture

> `nyquist_validation: true` in config.json — section included.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go built-in `testing` package (no external framework needed for unit tests) |
| Config file | None — `go test ./...` discovers `*_test.go` files automatically |
| Quick run command | `go test ./internal/... -count=1 -timeout 30s` |
| Full suite command | `go test ./... -count=1 -race -timeout 60s` |
| Frontend tests | `cd frontend && npm test` (vitest, to be configured in Wave 0) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| AUTH-01 | Correct password returns 200 + JWT cookie | unit | `go test ./internal/auth/... -run TestLogin_Success` | ❌ Wave 0 |
| AUTH-01 | Wrong password returns 401 | unit | `go test ./internal/auth/... -run TestLogin_WrongPassword` | ❌ Wave 0 |
| AUTH-01 | 6th login attempt within 30s returns 429 | integration | `go test ./internal/api/... -run TestLogin_RateLimited` | ❌ Wave 0 |
| AUTH-01 | Protected route returns 401 without cookie | unit | `go test ./internal/api/... -run TestProtectedRoute_NoAuth` | ❌ Wave 0 |
| AUTH-01 | Protected route returns 200 with valid JWT cookie | unit | `go test ./internal/api/... -run TestProtectedRoute_WithAuth` | ❌ Wave 0 |
| DEPLOY-01 | `docker compose up` starts without error | smoke | `docker compose up -d && docker compose ps` (manual check) | ❌ Wave 0 |
| DEPLOY-01 | Migrations run on startup (schema_migrations table exists) | integration | `go test ./internal/db/... -run TestMigrations` | ❌ Wave 0 |
| DEPLOY-01 | WAL mode active after DB open | unit | `go test ./internal/db/... -run TestWALMode` | ❌ Wave 0 |
| DEPLOY-01 | Named volume persists data across container restart | smoke | Manual: `docker compose restart backend && query DB` | manual-only |

### Sampling Rate
- **Per task commit:** `go test ./internal/... -count=1 -timeout 30s`
- **Per wave merge:** `go test ./... -count=1 -race -timeout 60s`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

All test files are missing — greenfield project.

- [ ] `internal/auth/auth_test.go` — covers AUTH-01 (bcrypt verify, JWT encode/decode, cookie name)
- [ ] `internal/api/handlers/auth_test.go` — covers AUTH-01 (login handler HTTP tests with httptest)
- [ ] `internal/db/db_test.go` — covers DEPLOY-01 (migration run, WAL mode pragma verification)
- [ ] `internal/api/router_test.go` — covers AUTH-01 (rate limiting integration, protected route 401/200)
- [ ] Framework install: `go mod init` + `go get` commands above — if `go.mod` does not yet exist
- [ ] Frontend: `cd frontend && npm install vitest @vitest/ui @testing-library/react @testing-library/user-event` — if vitest not yet configured

---

## Sources

### Primary (HIGH confidence)
- `pkg.go.dev/modernc.org/sqlite` v1.46.1 — WAL mode, driver name `"sqlite"`, RegisterConnectionHook
- `pkg.go.dev/github.com/golang-migrate/migrate/v4/database/sqlite` — modernc driver, URL scheme `sqlite://`, import path
- `pkg.go.dev/github.com/golang-migrate/migrate/v4/source/iofs` — go:embed + iofs pattern, migration file requirements
- `pkg.go.dev/github.com/go-chi/jwtauth/v5` v5.4.0 — TokenFromCookie looks for cookie named `"jwt"`, Verifier/Authenticator pattern
- `pkg.go.dev/github.com/go-chi/httprate` v0.15.0 — LimitByIP signature and usage
- `pkg.go.dev/golang.org/x/crypto` v0.49.0 — bcrypt functions
- `sqlite.org/pragma.html` — WAL mode, busy_timeout semantics
- `theitsolutions.io/blog/modernc.org-sqlite-with-go` — RegisterConnectionHook pattern, reader/writer pool setup
- `aronschueler.de/blog/2024/07/29/enabling-hot-module-replacement-hmr-in-vite-with-nginx-reverse-proxy/` — Nginx WebSocket headers for Vite HMR
- `vitejs.dev/config/server-options` — Vite `server.watch.usePolling` config option
- `tailwindcss.com/docs` — v4.2 CSS-first config, @tailwindcss/vite plugin

### Secondary (MEDIUM confidence)
- `github.com/golang-migrate/migrate/tree/master/database/sqlite` — driver details confirmed via WebFetch
- `github.com/go-chi/jwtauth/_example/main.go` (rate limited; fetched via pkg.go.dev instead)
- `horsfallnathan.com/blog/golang-docker-setup...` — Air + multi-stage Dockerfile dev/prod pattern
- WebSearch: docker-compose dev/prod split pattern with Air and Vite — multiple corroborating sources

### Tertiary (LOW confidence)
- None — all critical claims verified with primary sources.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all library versions verified via pkg.go.dev (March 2026)
- Architecture: HIGH — patterns derived from official docs + prior project research (ARCHITECTURE.md)
- golang-migrate modernc driver: HIGH — confirmed import path `database/sqlite` via official GitHub and pkg.go.dev
- JWT cookie name: HIGH — confirmed `"jwt"` in pkg.go.dev documentation
- Vite HMR Nginx headers: HIGH — confirmed via official Vite docs + dedicated blog post with verified config
- Pitfalls: HIGH — built on PITFALLS.md research + verified details from primary sources
- WAL connection hook: HIGH — confirmed via theitsolutions.io article corroborating modernc docs

**Research date:** 2026-03-15
**Valid until:** 2026-06-15 (stable ecosystem; golang-migrate, jwtauth, modernc.org/sqlite all have low churn)
