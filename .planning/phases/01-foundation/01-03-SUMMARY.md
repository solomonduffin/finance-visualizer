---
phase: 01-foundation
plan: 03
subsystem: infra
tags: [react, vite, tailwind, typescript, vitest, testing-library, docker, nginx, air, sqlite, compose]

# Dependency graph
requires:
  - phase: 01-01
    provides: "db.Open, db.Migrate, config.Load, cmd/server/main.go entrypoint"
  - phase: 01-02
    provides: "POST /api/auth/login, GET /api/health, JWT cookie auth, chi router"
provides:
  - React+Vite+Tailwind v4 frontend scaffold (frontend/)
  - Password-only login page with 401/429 error handling (frontend/src/pages/Login.tsx)
  - Typed API client with credentials:include cookie auth (frontend/src/api/client.ts)
  - 5 passing frontend tests (vitest+jsdom+@testing-library/react)
  - Multi-stage Dockerfile: dev(Air), frontend-build, backend-build, prod(distroless), nginx-prod
  - Dev compose: Air backend + Vite frontend + Nginx with HMR WebSocket support
  - Prod compose: distroless backend + nginx-prod serving static + proxying /api/
  - Nginx dev config: /api/ -> backend:8080, / -> frontend:5173 with WebSocket upgrade
  - Nginx prod config: /api/ proxy + SPA try_files /index.html fallback
affects: [phase-02, phase-03, phase-04, all-phases]

# Tech tracking
tech-stack:
  added:
    - react 19 (UI library)
    - vite 7 (build tool + dev server; downgraded from 8 for @tailwindcss/vite compatibility)
    - tailwindcss v4 (CSS-first, no config file needed)
    - "@tailwindcss/vite" (Vite plugin for Tailwind v4)
    - vitest 4 (test runner)
    - "@testing-library/react, @testing-library/user-event, @testing-library/jest-dom, @testing-library/dom" (test utilities)
    - jsdom (browser environment for tests)
    - typescript (strict + verbatimModuleSyntax)
    - golang:1.25-alpine (dev/build image; 1.23 tag unavailable)
  patterns:
    - Tailwind v4 CSS-first: @import "tailwindcss" in index.css, no tailwind.config.js needed
    - All fetch calls use credentials:'include' for HttpOnly cookie-based auth
    - vitest environment:jsdom + setupFiles for @testing-library/jest-dom matchers
    - vite server.watch.usePolling:true for Docker file-watching across bind mounts
    - Docker multi-stage: dev uses golang:1.25-alpine with Air -buildvcs=false, prod uses distroless/static-debian12
    - Air config watches Go+SQL only, excludes frontend/node_modules/tmp/data
    - docker-compose.yml frontend command runs npm install before npm run dev (bind mount overwrites image node_modules)

key-files:
  created:
    - frontend/package.json
    - frontend/vite.config.ts
    - frontend/vitest.config.ts
    - frontend/src/index.css
    - frontend/src/main.tsx
    - frontend/src/App.tsx
    - frontend/src/test-setup.ts
    - frontend/src/api/client.ts
    - frontend/src/pages/Login.tsx
    - frontend/src/pages/Login.test.tsx
    - Dockerfile
    - docker-compose.yml
    - docker-compose.prod.yml
    - nginx/nginx.dev.conf
    - nginx/nginx.prod.conf
    - .air.toml
    - .gitignore
    - .dockerignore
  modified: []

key-decisions:
  - "Tailwind v4 CSS-first config (@import 'tailwindcss') — no tailwind.config.js required"
  - "login() returns {ok:true} | {error:string} — rate_limited error string distinguishes 429 from 401 without exposing HTTP status to UI logic"
  - "vitest globals:true not used — explicit imports avoid ambiguity in TypeScript strict mode"
  - "nginx/nginx.dev.conf includes proxy_set_header Upgrade + Connection:Upgrade for Vite HMR WebSocket"
  - "prod compose uses PASSWORD_HASH not PASSWORD — operators pre-hash in prod for security"
  - "Vite 7 used instead of 8 — Vite 8 broke @tailwindcss/vite compatibility at time of execution"
  - "Air -buildvcs=false required — Go 1.25 enforces VCS stamping which fails in Docker build context without .git"
  - "golang:1.25-alpine used — golang:1.23-alpine tag unavailable on Docker Hub at execution time"
  - "Frontend compose command runs npm install before npm run dev — bind mount overwrites node_modules from image"

patterns-established:
  - "Pattern: Frontend fetch calls always include credentials:'include' — required for HttpOnly cookie session"
  - "Pattern: Login error handling maps 429 -> rate_limited string before setting state, not raw HTTP status"
  - "Pattern: Nginx WebSocket upgrade headers required on / location for Vite HMR through Docker network"

requirements-completed: [AUTH-01, DEPLOY-01]

# Metrics
duration: 40min
completed: 2026-03-15
---

# Phase 1 Plan 03: React Frontend + Docker Environment Summary

**React+Vite 7+Tailwind v4 login page with 5 passing tests, multi-stage Dockerfile (Air dev + distroless prod), Nginx proxying Vite HMR and API — full stack verified end-to-end in browser: login, auth error, and dashboard redirect all functional**

## Performance

- **Duration:** ~40 min (including Docker build debugging and user verification)
- **Started:** 2026-03-15T02:09:46Z
- **Completed:** 2026-03-15T02:49:51Z (fix commit after checkpoint approval)
- **Tasks:** 3 (2 automated + 1 human-verify checkpoint, approved)
- **Files modified:** 18 created + 3 modified in fix commit

## Accomplishments

- Complete React+TypeScript frontend scaffold with Vite 7, Tailwind v4 CSS-first configuration; 5 unit tests passing
- Password-only login page that calls `POST /api/auth/login`, handles 200/401/429 responses, and transitions to placeholder dashboard on success
- Typed API client (`login`, `checkAuth`) using `credentials: 'include'` for HttpOnly cookie auth
- Multi-stage Dockerfile with 5 targets: dev (Air hot reload on golang:1.25), frontend-build, backend-build, prod (distroless), nginx-prod
- Dev compose: Air-watched Go backend + Vite dev server + Nginx with WebSocket upgrade for HMR, all starting with `docker compose up --build`
- User verified end-to-end: login page at http://localhost, wrong password shows error, correct password shows dashboard

## Task Commits

Each task was committed atomically:

1. **Task 1: React+Vite+Tailwind scaffold with login page** - `088d06c` (feat)
2. **Task 2: Docker environment -- Dockerfile, Compose files, Nginx** - `e258f13` (feat)
3. **Task 3: Docker build fixes (post-checkpoint)** - `777ec33` (fix — Go 1.25 image, Air -buildvcs=false, Vite 7 downgrade)

**Plan metadata:** `(this commit)` (docs: complete plan)

## Files Created/Modified

- `frontend/package.json` — React 19, Vite 7, Tailwind v4, vitest, @testing-library/*
- `frontend/vite.config.ts` — react+tailwindcss plugins, host 0.0.0.0, polling, /api proxy
- `frontend/vitest.config.ts` — jsdom environment, test-setup.ts for jest-dom matchers
- `frontend/src/index.css` — `@import "tailwindcss"` (Tailwind v4 CSS-first)
- `frontend/src/main.tsx` — React 18 createRoot, StrictMode
- `frontend/src/App.tsx` — checkAuth() on mount, conditional Login or placeholder dashboard
- `frontend/src/test-setup.ts` — imports @testing-library/jest-dom
- `frontend/src/api/client.ts` — login() + checkAuth() with credentials:include
- `frontend/src/pages/Login.tsx` — password form, error/rate-limit handling, Tailwind centered card
- `frontend/src/pages/Login.test.tsx` — 5 tests with vi.mock(client)
- `Dockerfile` — 5-stage multi-stage build
- `docker-compose.yml` — dev stack: backend+frontend+nginx, sqlite_data volume
- `docker-compose.prod.yml` — prod stack: backend+nginx-prod, restart unless-stopped
- `nginx/nginx.dev.conf` — /api/ -> backend:8080, / -> frontend:5173 + WebSocket upgrade
- `nginx/nginx.prod.conf` — /api/ proxy + try_files /index.html SPA fallback
- `.air.toml` — Go+SQL watch, excludes frontend/tmp/node_modules/data
- `.gitignore` — data/, *.db, tmp/, node_modules/, dist/, .env
- `.dockerignore` — .git, node_modules, data/, tmp/, .env

## Decisions Made

- Tailwind v4 CSS-first approach: `@import "tailwindcss"` in index.css, no separate config file needed. The `@tailwindcss/vite` plugin handles everything.
- `login()` returns `{error: 'rate_limited'}` for 429 rather than exposing HTTP status — UI logic stays clean, test assertions are readable.
- Prod compose uses `PASSWORD_HASH` (not `PASSWORD`) so operators pre-hash their password before deploying rather than sending plaintext through env vars.
- Nginx WebSocket upgrade headers (`Upgrade`, `Connection: Upgrade`, `proxy_http_version 1.1`) required on the `/` location for Vite HMR to function through the Docker network proxy.
- **Vite 7 instead of 8:** `@tailwindcss/vite` was incompatible with Vite 8 at execution time; Vite 7 is the supported version for Tailwind v4 plugin.
- **Air `-buildvcs=false`:** Go 1.25 enforces VCS stamping that fails in Docker build contexts (no `.git` in build context). The flag disables it.
- **golang:1.25-alpine:** Plan specified 1.23 but that image tag was unavailable on Docker Hub; 1.25 is the current stable Alpine image.
- **npm install in compose command:** The `./frontend:/app` bind mount overwrites node_modules from the image layer; the compose command runs `npm install && npm run dev` to ensure dependencies are present.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Installed missing @testing-library/dom peer dependency**
- **Found during:** Task 1 (vitest test run)
- **Issue:** `@testing-library/react` requires `@testing-library/dom` as a peer dependency; it was not listed in the plan's install command, causing `require()` to fail at test runtime.
- **Fix:** `npm install @testing-library/dom --legacy-peer-deps`
- **Files modified:** `frontend/package.json`, `frontend/package-lock.json`
- **Verification:** All 5 vitest tests pass after installation
- **Committed in:** `088d06c` (Task 1 commit)

**2. [Rule 1 - Bug] Fixed type-only import for FormEvent under verbatimModuleSyntax**
- **Found during:** Task 1 (TypeScript build)
- **Issue:** `import { FormEvent } from 'react'` causes TS1484 error when `verbatimModuleSyntax` is enabled in tsconfig.app.json — type imports must use `import type` syntax.
- **Fix:** Changed to `import { useState, type FormEvent } from 'react'`
- **Files modified:** `frontend/src/pages/Login.tsx`
- **Verification:** `tsc --noEmit` passes; `npm run build` succeeds
- **Committed in:** `088d06c` (Task 1 commit)

---

**3. [Rule 3 - Blocking] Resolved Docker build failures blocking end-to-end verification**
- **Found during:** Task 3 (full-stack end-to-end verification at checkpoint)
- **Issue:** Three separate blocking issues: (a) `golang:1.23-alpine` image unavailable on Docker Hub — manifest unknown; (b) Air build failed with "VCS stamping requires either a Git repository or -buildvcs=false" (Go 1.25 enforcement); (c) `@tailwindcss/vite` incompatible with Vite 8, frontend build failed
- **Fix:** (a) Changed Dockerfile FROM to `golang:1.25-alpine` for dev and backend-build stages; (b) Added `-buildvcs=false` to Air build command in `.air.toml`; (c) Downgraded Vite from 8 to 7 in `frontend/package.json`; (d) Added `npm install &&` prefix to frontend docker-compose command
- **Files modified:** `Dockerfile`, `.air.toml`, `frontend/package.json`
- **Verification:** `docker compose up --build` succeeded; user verified login page functional at http://localhost
- **Committed in:** `777ec33` (fix commit after checkpoint)

---

**Total deviations:** 3 auto-fixed (1 blocking dependency, 1 TypeScript strict mode compliance, 1 blocking Docker build cluster)
**Impact on plan:** All fixes were version/compatibility issues with upstream tooling changes. No scope creep, no plan logic altered.

## Issues Encountered

- `@tailwindcss/vite` peer dependency conflict with Vite 8; resolved with `--legacy-peer-deps` and ultimately Vite 7 downgrade.
- `golang:1.23-alpine` image tag no longer available on Docker Hub — Docker Hub image lifecycle removed older patch tags. Required upgrade to 1.25.
- Go 1.25 stricter VCS stamping enforcement in Air hot reload path — `-buildvcs=false` is the standard workaround for containerized builds.

## User Setup Required

Before running `docker compose up`, users must:
1. Copy `.env.example` to `.env`: `cp .env.example .env`
2. Set `PASSWORD=yourpassword` (any string you choose)
3. Set `JWT_SECRET` to a random 32+ char string: `openssl rand -hex 32`

For production (`docker-compose.prod.yml`), set `PASSWORD_HASH` instead of `PASSWORD`.

## Next Phase Readiness

- Phase 1 complete: Go backend (SQLite + migrations + auth + JWT) + React frontend (login page) + Docker dev/prod environment all verified working
- `docker compose up --build` starts all 3 services with no manual steps; user confirmed end-to-end login flow
- All Go backend tests green (16 tests), all frontend tests green (5 tests)
- Ready for Phase 2: SimpleFIN integration (cron worker, account sync, balance display)
- Carried forward: SimpleFIN `balance-date` field semantics need verification against current protocol spec before cron worker implementation

---
*Phase: 01-foundation*
*Completed: 2026-03-15*
