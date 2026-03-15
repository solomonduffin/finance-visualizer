---
phase: 3
slug: backend-api
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-15
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` stdlib + `net/http/httptest` |
| **Config file** | none — `go test ./internal/...` is the runner |
| **Quick run command** | `go test ./internal/api/handlers/... -run TestSummary\|TestAccounts\|TestHistory -v` |
| **Full suite command** | `go test ./internal/...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/api/handlers/... -v`
- **After every plan wave:** Run `go test ./internal/...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | DASH-01 | unit | `go test ./internal/api/handlers/... -run TestGetSummary -v` | ❌ W0 | ⬜ pending |
| 03-01-02 | 01 | 1 | DASH-01 | unit | `go test ./internal/api/... -run TestSummaryRoute_NoAuth -v` | ❌ W0 | ⬜ pending |
| 03-01-03 | 01 | 1 | DASH-02 | unit | `go test ./internal/api/handlers/... -run TestGetSummary -v` | ❌ W0 | ⬜ pending |
| 03-01-04 | 01 | 1 | DASH-03 | unit | `go test ./internal/api/handlers/... -run TestGetSummary -v` | ❌ W0 | ⬜ pending |
| 03-02-01 | 02 | 1 | DASH-04 | unit | `go test ./internal/api/handlers/... -run TestGetAccounts -v` | ❌ W0 | ⬜ pending |
| 03-02-02 | 02 | 1 | DASH-04 | unit | `go test ./internal/api/... -run TestAccountsRoute_NoAuth -v` | ❌ W0 | ⬜ pending |
| 03-03-01 | 03 | 1 | DASH-01,DASH-02,DASH-03 | unit | `go test ./internal/api/handlers/... -run TestGetBalanceHistory -v` | ❌ W0 | ⬜ pending |
| 03-03-02 | 03 | 1 | DASH-01,DASH-02,DASH-03 | unit | `go test ./internal/api/... -run TestHistoryRoute_NoAuth -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/api/handlers/summary_test.go` — stubs for DASH-01, DASH-02, DASH-03
- [ ] `internal/api/handlers/accounts_test.go` — stubs for DASH-04
- [ ] `internal/api/handlers/history_test.go` — stubs for balance history
- [ ] Route-level auth tests in existing `internal/api/handlers/auth_test.go` or new file

*Existing infrastructure covers framework needs — no new framework install required.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
