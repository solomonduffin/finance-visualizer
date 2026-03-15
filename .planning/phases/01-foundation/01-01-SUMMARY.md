---
phase: 01-foundation
plan: 01
subsystem: database
tags: [go, sqlite, golang-migrate, modernc, bcrypt, config, wal-mode]

# Dependency graph
requires: []
provides:
  - SQLite connection with WAL mode via RegisterConnectionHook (internal/db/db.go)
  - Migration runner with embedded SQL via go:embed + iofs (internal/db/migrations.go)
  - Full schema: settings, accounts, balance_snapshots, sync_log (000001_init.up.sql)
  - Config loading from env vars with validation and defaults (internal/config/config.go)
  - Application entrypoint stub (cmd/server/main.go)
  - All Phase 1 Go dependencies installed in go.mod
affects: [02-auth, 03-api, 04-frontend, all-phases]

# Tech tracking
tech-stack:
  added:
    - modernc.org/sqlite v1.46.1 (pure Go SQLite, no CGo)
    - github.com/golang-migrate/migrate/v4 v4.19.1
    - github.com/go-chi/chi/v5 v5.2.5
    - github.com/go-chi/jwtauth/v5 v5.4.0
    - github.com/go-chi/httprate v0.15.0
    - github.com/go-chi/cors v1.2.2
    - github.com/go-chi/httplog/v3 v3.3.0
    - golang.org/x/crypto v0.49.0
    - github.com/joho/godotenv v1.5.1
    - github.com/lmittmann/tint v1.1.3
  patterns:
    - SQLite WAL mode via sqlite.RegisterConnectionHook (applies to every pooled connection)
    - Single-writer pool: db.SetMaxOpenConns(1) for SQLite write safety
    - Migration embedding: //go:embed migrations/*.sql + iofs.New + migrate.NewWithSourceInstance
    - Config fail-fast: Load() returns descriptive error if required env vars are missing
    - Dual password env var: PASSWORD (plaintext hashed at load time) or PASSWORD_HASH (pre-hashed)

key-files:
  created:
    - go.mod
    - go.sum
    - internal/db/db.go
    - internal/db/db_test.go
    - internal/db/migrations.go
    - internal/db/migrations_test.go
    - internal/db/migrations/000001_init.up.sql
    - internal/db/migrations/000001_init.down.sql
    - internal/config/config.go
    - internal/config/config_test.go
    - cmd/server/main.go
    - .env.example
  modified: []

key-decisions:
  - "Use sqlite.RegisterConnectionHook (not DSN pragmas) to set WAL+busy_timeout+foreign_keys — applies to all pooled connections"
  - "Migrations placed in internal/db/migrations/ for clean go:embed from db package"
  - "Support both PASSWORD (hash at startup) and PASSWORD_HASH (use directly) for operator flexibility"
  - "WAL mode test uses temp file DB (not :memory:) because in-memory SQLite always uses memory journal mode"
  - "Migration SQL uses no explicit BEGIN/COMMIT — golang-migrate wraps each file in an implicit transaction"

patterns-established:
  - "Pattern: SQLite open via db.Open(path) always sets WAL mode via RegisterConnectionHook before first connection"
  - "Pattern: Migrations run via db.Migrate(dbPath) at startup before serving HTTP"
  - "Pattern: Config loaded via config.Load() at startup; app exits on validation error"

requirements-completed: [DEPLOY-01]

# Metrics
duration: 4min
completed: 2026-03-15
---

# Phase 1 Plan 01: Go Module, SQLite Foundation, and Config Summary

**Pure-Go SQLite connection with WAL mode via RegisterConnectionHook, golang-migrate embedded schema (settings/accounts/balance_snapshots/sync_log), and env-var config with bcrypt password hashing**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-15T01:55:52Z
- **Completed:** 2026-03-15T01:59:53Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments

- SQLite connection layer with WAL mode, busy_timeout=5000, foreign_keys=ON set via RegisterConnectionHook on every pooled connection
- Full upfront schema migration (4 tables + 1 index) embedded in binary via go:embed, runs idempotently at startup
- Config loader that reads PASSWORD/PASSWORD_HASH, JWT_SECRET, PORT, DB_PATH from env vars with validation and defaults
- All Phase 1 Go dependencies installed (chi, jwtauth, httprate, cors, httplog, crypto, godotenv, tint, golang-migrate)
- Application entrypoint stub that chains config → mkdir → db.Open → db.Migrate → log

## Task Commits

Each task was committed atomically:

1. **Task 1: Go module, SQLite connection, and config** - `6c5a48b` (feat)
2. **Task 2: Migration runner, full schema, server entrypoint** - `e763c4b` (feat)

_Note: TDD tasks — tests written first (RED), then implementation (GREEN)_

## Files Created/Modified

- `go.mod` / `go.sum` — Go module with all Phase 1 dependencies
- `internal/db/db.go` — Open() with RegisterConnectionHook (WAL+busy_timeout+foreign_keys), SetMaxOpenConns(1)
- `internal/db/db_test.go` — WAL mode, busy_timeout, foreign_keys, max open conns tests (uses temp file for WAL test)
- `internal/db/migrations.go` — Migrate() with go:embed + iofs, modernc sqlite driver
- `internal/db/migrations_test.go` — 8 migration tests covering table creation, constraints, idempotency
- `internal/db/migrations/000001_init.up.sql` — Full schema: settings, accounts, balance_snapshots, sync_log, index
- `internal/db/migrations/000001_init.down.sql` — Rollback in reverse dependency order
- `internal/config/config.go` — Load() with fail-fast validation, dual password support, defaults
- `internal/config/config_test.go` — 6 config tests covering success, missing vars, defaults, pre-hashed password
- `cmd/server/main.go` — Entrypoint stub with slog+tint logging, config → db → migrate sequence
- `.env.example` — Documents all required environment variables with generation instructions

## Decisions Made

- Used `sqlite.RegisterConnectionHook` instead of DSN pragmas: ensures WAL mode is set on every connection in the pool, not just the first one opened.
- Placed migrations in `internal/db/migrations/` (not top-level) so `go:embed` works from the `db` package without needing to embed from `main.go`.
- WAL mode test uses a temp file DB (not `:memory:`) because SQLite in-memory databases always use `memory` journal mode, making WAL verification impossible with `:memory:`.
- Support both `PASSWORD` (hash at startup) and `PASSWORD_HASH` (use directly) so operators can choose between first-run convenience and pre-hashing.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test uses temp file DB for WAL mode verification**
- **Found during:** Task 1 (TestOpen_WALMode)
- **Issue:** Plan specified `Open(":memory:")` in tests, but SQLite in-memory databases always return `memory` for `PRAGMA journal_mode`, not `wal`. Test would always fail.
- **Fix:** Added `openFileTestDB()` helper using `t.TempDir()` for WAL-sensitive tests; `openTestDB()` with `:memory:` for all other tests.
- **Files modified:** `internal/db/db_test.go`
- **Verification:** `TestOpen_WALMode` passes and correctly verifies `journal_mode=wal` on a file-based DB
- **Committed in:** `6c5a48b` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — bug in test expectation vs. SQLite in-memory limitation)
**Impact on plan:** Necessary correctness fix. WAL mode is still fully verified, just against a file-based DB as required by SQLite semantics. No scope creep.

## Issues Encountered

- `go mod tidy` was required twice during implementation as new source files using different imports were created after the initial `go get` run. Each `tidy` resolved correctly.

## User Setup Required

None — no external service configuration required. Copy `.env.example` to `.env` and fill in values before running.

## Next Phase Readiness

- DB layer complete: `db.Open()` and `db.Migrate()` are ready for use by all subsequent plans
- Schema complete: all 4 tables exist, Plans 02+ add code against existing schema
- Config pattern established: `config.Load()` is the entry point for all env var reading
- Ready for Plan 02: bcrypt auth handler + JWT middleware + chi router wiring

---
*Phase: 01-foundation*
*Completed: 2026-03-15*
