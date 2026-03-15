---
phase: 2
slug: data-pipeline
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-15
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | go test (stdlib) |
| **Framework (Frontend)** | Vitest 3.2.1 |
| **Go config file** | none — stdlib test runner |
| **Go quick run command** | `go test ./internal/... -count=1 -timeout 30s` |
| **Go full suite command** | `go test ./... -count=1 -timeout 60s` |
| **Frontend quick run** | `cd frontend && npx vitest run --reporter=verbose` |
| **Frontend full suite** | `cd frontend && npx vitest run` |
| **Estimated runtime** | ~15 seconds (Go) + ~10 seconds (Frontend) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/... -count=1 -timeout 30s`
- **After every plan wave:** Run `go test ./... -count=1 && cd frontend && npx vitest run`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 0 | DATA-01 | unit | `go test ./internal/simplefin/... -run TestFetchAccounts -v` | ❌ W0 | ⬜ pending |
| 02-01-02 | 01 | 0 | DATA-01 | unit | `go test ./internal/simplefin/... -run TestFetchAccounts_Error -v` | ❌ W0 | ⬜ pending |
| 02-01-03 | 01 | 0 | DATA-01 | unit | `go test ./internal/api/handlers/... -run TestGetSettings -v` | ❌ W0 | ⬜ pending |
| 02-01-04 | 01 | 0 | DATA-01 | unit | `go test ./internal/api/handlers/... -run TestSaveSettings -v` | ❌ W0 | ⬜ pending |
| 02-01-05 | 01 | 0 | DATA-02 | unit | `go test ./internal/sync/... -run TestNextRunTime -v` | ❌ W0 | ⬜ pending |
| 02-01-06 | 01 | 0 | DATA-02 | unit | `go test ./internal/api/handlers/... -run TestSyncNow -v` | ❌ W0 | ⬜ pending |
| 02-01-07 | 01 | 0 | DATA-03 | unit | `go test ./internal/sync/... -run TestSyncOnce_FirstSync -v` | ❌ W0 | ⬜ pending |
| 02-01-08 | 01 | 0 | DATA-04 | unit | `go test ./internal/sync/... -run TestInsertSnapshot_Duplicate -v` | ❌ W0 | ⬜ pending |
| 02-01-09 | 01 | 0 | DATA-04 | unit | `go test ./internal/sync/... -run TestUpsertAccount -v` | ❌ W0 | ⬜ pending |
| 02-01-10 | 01 | 0 | DATA-01 | unit | `cd frontend && npx vitest run src/pages/Settings.test.tsx` | ❌ W0 | ⬜ pending |
| 02-01-11 | 01 | 0 | DATA-01 | unit | `cd frontend && npx vitest run src/pages/Settings.test.tsx` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/simplefin/client_test.go` — stubs for DATA-01 (mock httptest.Server for /accounts)
- [ ] `internal/sync/sync_test.go` — stubs for DATA-02, DATA-03, DATA-04 (temp file DB)
- [ ] `internal/api/handlers/settings_test.go` — stubs for DATA-01 settings API
- [ ] `frontend/src/pages/Settings.test.tsx` — stubs for DATA-01 settings UI

*Existing infrastructure covers framework install — Go stdlib test runner and Vitest are already configured.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Daily cron fires at correct hour | DATA-02 | Requires waiting for real clock | Set SYNC_HOUR to 1 minute ahead, watch logs |
| SimpleFIN live API returns real data | DATA-01 | Requires real access URL | Provide test access URL, trigger sync, verify DB |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
