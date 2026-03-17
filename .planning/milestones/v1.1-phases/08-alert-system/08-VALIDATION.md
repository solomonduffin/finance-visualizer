---
phase: 8
slug: alert-system
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-16
---

# Phase 8 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | Go testing + httptest (standard) |
| **Framework (Frontend)** | Vitest 3.2.1 + @testing-library/react 16.3.2 |
| **Config file (Go)** | None — `go test ./...` |
| **Config file (Frontend)** | `frontend/vitest.config.ts` |
| **Quick run command (Go)** | `go test ./internal/alerts/... ./internal/api/handlers/... -run Alert -count=1` |
| **Quick run command (Frontend)** | `cd frontend && npx vitest run --reporter=verbose src/pages/Alerts.test.tsx src/components/AlertRule*.test.tsx` |
| **Full suite command** | `go test ./... -count=1 && cd frontend && npx vitest run` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** `go test ./internal/alerts/... -count=1` (Go) + relevant frontend test
- **After every plan wave:** `go test ./... -count=1 && cd frontend && npx vitest run`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30s

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 08-01-01 | 01 | 1 | ALERT-01 | unit (frontend) | `cd frontend && npx vitest run src/components/AlertRuleForm.test.tsx` | ❌ W0 | ⬜ pending |
| 08-01-02 | 01 | 1 | ALERT-01 | unit (Go) | `go test ./internal/alerts/... -run TestCompileOperands -count=1` | ❌ W0 | ⬜ pending |
| 08-01-03 | 01 | 1 | ALERT-02 | unit (Go) | `go test ./internal/alerts/... -run TestEvaluate -count=1` | ❌ W0 | ⬜ pending |
| 08-02-01 | 02 | 2 | ALERT-03 | unit (Go) | `go test ./internal/alerts/... -run TestStateMachine -count=1` | ❌ W0 | ⬜ pending |
| 08-02-02 | 02 | 2 | ALERT-03 | unit (Go) | `go test ./internal/alerts/... -run TestEvaluateAll -count=1` | ❌ W0 | ⬜ pending |
| 08-02-03 | 02 | 2 | ALERT-04 | unit (Go) | `go test ./internal/alerts/... -run TestFormatAlertBody -count=1` | ❌ W0 | ⬜ pending |
| 08-03-01 | 03 | 3 | ALERT-05 | unit (Go) | `go test ./internal/alerts/... -run TestCrypto -count=1` | ❌ W0 | ⬜ pending |
| 08-03-02 | 03 | 3 | ALERT-05 | unit (frontend) | `cd frontend && npx vitest run src/pages/Settings.test.tsx` | ✅ (extend) | ⬜ pending |
| 08-03-03 | 03 | 3 | ALERT-06 | unit (Go) | `go test ./internal/api/handlers/... -run TestTestEmail -count=1` | ❌ W0 | ⬜ pending |
| 08-03-04 | 03 | 3 | ALERT-07 | unit (Go) | `go test ./internal/api/handlers/... -run TestAlert -count=1` | ❌ W0 | ⬜ pending |
| 08-03-05 | 03 | 3 | ALERT-07 | unit (frontend) | `cd frontend && npx vitest run src/pages/Alerts.test.tsx` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/alerts/evaluator_test.go` — stubs for ALERT-01, ALERT-02 (expression compilation and evaluation)
- [ ] `internal/alerts/engine_test.go` — stubs for ALERT-03 (state machine transitions)
- [ ] `internal/alerts/notifier_test.go` — stubs for ALERT-04 (email body formatting)
- [ ] `internal/alerts/crypto_test.go` — stubs for ALERT-05 (encryption round-trip)
- [ ] `internal/api/handlers/alerts_test.go` — stubs for ALERT-07 (CRUD endpoints)
- [ ] `internal/api/handlers/email_test.go` — stubs for ALERT-06 (test email endpoint)
- [ ] `frontend/src/pages/Alerts.test.tsx` — stubs for ALERT-07 (alerts page rendering)
- [ ] `frontend/src/components/AlertRuleForm.test.tsx` — stubs for ALERT-01 (builder form)
- [ ] `frontend/src/components/AlertRuleCard.test.tsx` — stubs for ALERT-07 (rule card display)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| SMTP delivery to real mailbox | ALERT-06 | Requires external SMTP server | Configure SMTP in settings, click "Send Test Email", verify arrival in inbox |
| Visual expression builder UX | ALERT-01 | Visual layout and interaction quality | Open Alerts page, create rule with 3+ operands, verify +/- buttons work, verify operator dropdown |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
