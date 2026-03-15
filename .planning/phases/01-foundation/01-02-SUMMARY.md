---
phase: 01-foundation
plan: 02
subsystem: auth
tags: [go, jwt, bcrypt, chi, httprate, jwtauth, sqlite, httponly-cookie, cors]

# Dependency graph
requires:
  - phase: 01-01
    provides: "db.Open, db.Migrate, config.Load, settings table in schema, all Phase 1 Go deps"
provides:
  - bcrypt password verification via auth.VerifyPassword (internal/auth/auth.go)
  - JWT token creation with 30-day expiry via auth.CreateToken (internal/auth/auth.go)
  - auth.Init + auth.TokenAuth for jwtauth middleware wiring
  - POST /api/auth/login handler with bcrypt verify + HttpOnly jwt cookie response
  - GET /api/health protected endpoint returning {"status":"ok"}
  - Chi router with httplog + CORS middleware, rate-limited login (5/30s), JWT-protected group
  - HTTP server wired in cmd/server/main.go with password_hash seeding
affects: [03-api, 04-frontend, all-phases]

# Tech tracking
tech-stack:
  added:
    - github.com/go-chi/chi/v5 v5.2.5 (HTTP router)
    - github.com/go-chi/jwtauth/v5 v5.4.0 (JWT middleware + TokenFromCookie)
    - github.com/go-chi/httprate v0.15.0 (IP-based rate limiting)
    - github.com/go-chi/cors v1.2.2 (CORS for Vite dev server)
    - github.com/go-chi/httplog/v3 v3.3.0 (structured request logging)
  patterns:
    - HttpOnly JWT cookie named exactly "jwt" (jwtauth.TokenFromCookie requires this name)
    - Login handler reads password_hash from settings table via SQL, not from config at request time
    - jwtauth.Verifier + jwtauth.Authenticator middleware chained in protected route group
    - httprate.LimitByIP applied only to login route, never globally
    - Password hash seeded into settings table via INSERT OR IGNORE at startup

key-files:
  created:
    - internal/auth/auth.go
    - internal/auth/auth_test.go
    - internal/api/handlers/auth.go
    - internal/api/handlers/auth_test.go
    - internal/api/handlers/health.go
    - internal/api/handlers/health_test.go
    - internal/api/router.go
    - internal/api/router_test.go
  modified:
    - cmd/server/main.go
    - go.mod
    - go.sum

key-decisions:
  - "JWT cookie must be named 'jwt' exactly — jwtauth.TokenFromCookie looks for this name specifically"
  - "Login handler queries settings table for password_hash at request time, not from in-memory config — consistent with multi-process/restart scenarios"
  - "Tests use temp file DB (t.TempDir) not :memory: — Migrate() opens its own connection so :memory: would be a different empty database"
  - "httplog.Options field is 'Level' not 'LogLevel' in v3 — discovered during implementation"

patterns-established:
  - "Pattern: Protected route group uses r.Use(jwtauth.Verifier(tokenAuth)) + r.Use(jwtauth.Authenticator(tokenAuth)) in sequence"
  - "Pattern: Login handler uses http.SetCookie with HttpOnly=true, SameSite=http.SameSiteStrictMode, MaxAge=30*24*3600"
  - "Pattern: Auth package is initialized once at startup via auth.Init(secret); all routes use auth.TokenAuth() for middleware"
  - "Pattern: Test DB helper uses db.Open(dbPath) + db.Migrate(dbPath) with same path to ensure migration applies to correct connection"

requirements-completed: [AUTH-01]

# Metrics
duration: 6min
completed: 2026-03-15
---

# Phase 1 Plan 02: Authentication System Summary

**bcrypt + HS256 JWT auth with HttpOnly cookie gate, chi router with IP rate limiting (5/30s) on login, and jwtauth middleware protecting all API routes**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-15T02:02:31Z
- **Completed:** 2026-03-15T02:08:00Z
- **Tasks:** 2
- **Files modified:** 10 (8 created, 2 modified)

## Accomplishments

- Complete auth package: bcrypt verification, JWT encoding with 30-day expiry, TokenAuth accessor for middleware wiring
- Login handler that reads password hash from DB settings table, verifies via bcrypt, and issues HttpOnly SameSite=Strict JWT cookie named "jwt"
- Chi router with httplog request logging, CORS (localhost:5173 for dev), rate-limited login (5 requests/30s per IP), JWT-protected route group
- Server entrypoint wired: seeds password hash to settings, calls auth.Init, creates router, starts http.ListenAndServe
- 16 unit/integration tests across all packages — all green (TDD: RED then GREEN for both tasks)

## Task Commits

Each task was committed atomically:

1. **Task 1: Auth package — bcrypt verification and JWT token helpers** - `ccde66f` (feat)
2. **Task 2: Chi router, login handler, rate limiting, and protected routes** - `f797b57` (feat)

_Note: TDD tasks — tests written first (RED), then implementation (GREEN)_

## Files Created/Modified

- `internal/auth/auth.go` — Init, TokenAuth, VerifyPassword, HashPassword, CreateToken
- `internal/auth/auth_test.go` — 7 tests: VerifyPassword (correct/wrong/empty), CreateToken (valid JWT, 30-day expiry), Init, HashPassword
- `internal/api/handlers/health.go` — Health handler: GET /api/health → {"status":"ok"}
- `internal/api/handlers/health_test.go` — TestHealthHandler_ReturnsOK
- `internal/api/handlers/auth.go` — Login handler: JSON decode, settings table query, bcrypt verify, JWT cookie
- `internal/api/handlers/auth_test.go` — 4 tests: success (cookie attrs), wrong password (no cookie), empty body (400), malformed JSON (400)
- `internal/api/router.go` — NewRouter: chi + httplog + CORS + rate-limited login + jwtauth-protected group
- `internal/api/router_test.go` — 4 tests: no-auth 401, with-auth 200, expired token 401, rate limit 429
- `cmd/server/main.go` — Full wiring: config → db → migrate → seed password_hash → auth.Init → NewRouter → ListenAndServe

## Decisions Made

- Cookie name "jwt" is mandatory — `jwtauth.TokenFromCookie` in go-chi/jwtauth/v5 hardcodes this lookup key. Using any other name silently falls back to header-only auth.
- Login handler reads `password_hash` from the settings table at request time (not from in-memory config) — correct for multi-process environments and consistent with the seeding approach at startup.
- Test helpers use temp file DBs (`t.TempDir()`) because `db.Migrate()` opens its own SQLite connection — calling `Migrate(":memory:")` would migrate a separate empty database, not the one opened by `db.Open(":memory:")`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed incorrect httplog.Options field name**
- **Found during:** Task 2 (router implementation)
- **Issue:** Used `LogLevel` as the field name in `httplog.Options{}`, but httplog/v3 uses `Level` (type `slog.Level`). Build failed.
- **Fix:** Changed `LogLevel: slog.LevelInfo` to `Level: slog.LevelInfo` in router.go
- **Files modified:** `internal/api/router.go`
- **Verification:** `go build ./internal/api/...` succeeds; all router tests pass
- **Committed in:** `f797b57` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — bug in API field name)
**Impact on plan:** One-line fix. No scope creep. All planned functionality delivered exactly as specified.

## Issues Encountered

- `go.mod` did not contain chi/jwtauth/httprate/cors/httplog despite plan 01 summary claiming they were installed. Installed all 5 packages at plan start before beginning TDD cycle. No impact on correctness.

## User Setup Required

None — no external service configuration required. Set `PASSWORD` or `PASSWORD_HASH` and `JWT_SECRET` in `.env` before running.

## Next Phase Readiness

- Auth system complete: POST /api/auth/login + GET /api/health fully functional
- All protected API routes in subsequent phases go in the `jwtauth.Verifier + jwtauth.Authenticator` protected route group in `internal/api/router.go`
- Password hash is seeded into the settings table at startup — consistent access pattern for the login handler
- Ready for Phase 1 Plan 03 (Docker + frontend scaffold) or any plan adding protected API routes

---
*Phase: 01-foundation*
*Completed: 2026-03-15*
