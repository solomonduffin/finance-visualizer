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
    - vite 8 (build tool + dev server)
    - tailwindcss v4 (CSS-first, no config file needed)
    - "@tailwindcss/vite" (Vite plugin for Tailwind v4)
    - vitest 4 (test runner)
    - "@testing-library/react, @testing-library/user-event, @testing-library/jest-dom, @testing-library/dom" (test utilities)
    - jsdom (browser environment for tests)
    - typescript (strict + verbatimModuleSyntax)
  patterns:
    - Tailwind v4 CSS-first: @import "tailwindcss" in index.css, no tailwind.config.js needed
    - All fetch calls use credentials:'include' for HttpOnly cookie-based auth
    - vitest environment:jsdom + setupFiles for @testing-library/jest-dom matchers
    - vite server.watch.usePolling:true for Docker file-watching across bind mounts
    - Docker multi-stage: dev stays in golang:1.23-alpine, prod uses distroless/static-debian12
    - Air config watches Go+SQL only, excludes frontend/node_modules/tmp/data

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

patterns-established:
  - "Pattern: Frontend fetch calls always include credentials:'include' — required for HttpOnly cookie session"
  - "Pattern: Login error handling maps 429 -> rate_limited string before setting state, not raw HTTP status"
  - "Pattern: Nginx WebSocket upgrade headers required on / location for Vite HMR through Docker network"

requirements-completed: [AUTH-01, DEPLOY-01]

# Metrics
duration: 4min
completed: 2026-03-15
---

# Phase 1 Plan 03: React Frontend + Docker Environment Summary

**React+Vite+Tailwind v4 login page with 5 passing tests, multi-stage Dockerfile (Air dev + distroless prod), and Nginx proxying Vite HMR and API — full stack starts with `docker compose up`**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-15T02:09:46Z
- **Completed:** 2026-03-15T02:13:57Z
- **Tasks:** 2 automated (Task 3 is human verification checkpoint)
- **Files modified:** 18 created

## Accomplishments

- Complete React+TypeScript frontend scaffold with Vite 8, Tailwind v4 CSS-first configuration
- Password-only login page that calls `POST /api/auth/login`, handles 200/401/429 responses, and transitions to placeholder dashboard on success
- Typed API client (`login`, `checkAuth`) using `credentials: 'include'` for HttpOnly cookie auth
- 5 unit tests passing: render, submit fires API, success -> onSuccess, 401 -> error, 429 -> rate limit message
- Multi-stage Dockerfile with 5 targets: dev (Air hot reload), frontend-build, backend-build, prod (distroless), nginx-prod
- Dev compose: Air-watched Go backend + Vite dev server + Nginx with WebSocket upgrade for HMR
- Prod compose: distroless backend + Nginx serving static assets with SPA fallback

## Task Commits

Each task was committed atomically:

1. **Task 1: React+Vite+Tailwind scaffold with login page** - `088d06c` (feat)
2. **Task 2: Docker environment -- Dockerfile, Compose files, Nginx** - `e258f13` (feat)

_Task 3 is a human-verify checkpoint — no commit until user approves._

## Files Created/Modified

- `frontend/package.json` — React 19, Vite 8, Tailwind v4, vitest, @testing-library/*
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

**Total deviations:** 2 auto-fixed (1 blocking dependency, 1 TypeScript strict mode compliance)
**Impact on plan:** Both fixes necessary for build/test to work. No scope creep.

## Issues Encountered

- `@tailwindcss/vite` had a peer dependency conflict with Vite 8; resolved with `--legacy-peer-deps`. This is a known transient issue with newly released Vite 8.

## User Setup Required

Before running `docker compose up`, users must:
1. Copy `.env.example` to `.env`: `cp .env.example .env`
2. Set `PASSWORD=yourpassword` (any string you choose)
3. Set `JWT_SECRET` to a random 32+ char string: `openssl rand -hex 32`

For production (`docker-compose.prod.yml`), set `PASSWORD_HASH` instead of `PASSWORD`.

## Next Phase Readiness

- Full dev stack ready: `docker compose up --build` starts all 3 services
- Human verification (Task 3) confirms end-to-end login flow in browser
- After Task 3 approval: Phase 1 is complete, Phase 2 (SimpleFIN sync) can begin
- All Go backend tests green (16 tests), all frontend tests green (5 tests)

---
*Phase: 01-foundation*
*Completed: 2026-03-15*
