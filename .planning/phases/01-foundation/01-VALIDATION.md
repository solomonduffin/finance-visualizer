---
phase: 1
slug: foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-15
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go built-in `testing` package + vitest (frontend) |
| **Config file** | None — `go test` discovers `*_test.go`; vitest configured in Wave 0 |
| **Quick run command** | `go test ./internal/... -count=1 -timeout 30s` |
| **Full suite command** | `go test ./... -count=1 -race -timeout 60s && cd frontend && npm test` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/... -count=1 -timeout 30s`
- **After every plan wave:** Run `go test ./... -count=1 -race -timeout 60s && cd frontend && npm test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 | 1 | AUTH-01 | unit | `go test ./internal/auth/... -run TestLogin_Success` | ❌ W0 | ⬜ pending |
| 01-01-02 | 01 | 1 | AUTH-01 | unit | `go test ./internal/auth/... -run TestLogin_WrongPassword` | ❌ W0 | ⬜ pending |
| 01-01-03 | 01 | 1 | AUTH-01 | integration | `go test ./internal/api/... -run TestLogin_RateLimited` | ❌ W0 | ⬜ pending |
| 01-01-04 | 01 | 1 | AUTH-01 | unit | `go test ./internal/api/... -run TestProtectedRoute_NoAuth` | ❌ W0 | ⬜ pending |
| 01-01-05 | 01 | 1 | AUTH-01 | unit | `go test ./internal/api/... -run TestProtectedRoute_WithAuth` | ❌ W0 | ⬜ pending |
| 01-02-01 | 02 | 1 | DEPLOY-01 | smoke | `docker compose up -d && docker compose ps` | ❌ W0 | ⬜ pending |
| 01-02-02 | 02 | 1 | DEPLOY-01 | integration | `go test ./internal/db/... -run TestMigrations` | ❌ W0 | ⬜ pending |
| 01-02-03 | 02 | 1 | DEPLOY-01 | unit | `go test ./internal/db/... -run TestWALMode` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/auth/auth_test.go` — stubs for AUTH-01 (bcrypt verify, JWT encode/decode, cookie name)
- [ ] `internal/api/handlers/auth_test.go` — covers AUTH-01 (login handler HTTP tests)
- [ ] `internal/db/db_test.go` — covers DEPLOY-01 (migration run, WAL mode pragma)
- [ ] `internal/api/router_test.go` — covers AUTH-01 (rate limiting, protected route 401/200)
- [ ] Frontend: `cd frontend && npm install vitest @testing-library/react @testing-library/user-event` — if vitest not configured

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `docker compose up` starts without error | DEPLOY-01 | Requires Docker daemon | Run `docker compose up -d`, verify all containers healthy via `docker compose ps` |
| Named volume persists data across restart | DEPLOY-01 | Requires container lifecycle | `docker compose restart backend`, query DB to confirm data persists |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
