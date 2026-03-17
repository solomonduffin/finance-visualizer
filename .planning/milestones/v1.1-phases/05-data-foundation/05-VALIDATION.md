---
phase: 5
slug: data-foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-15
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | `go test` (stdlib) |
| **Framework (Frontend)** | Vitest 3.2.1 + jsdom + @testing-library/react |
| **Config file (Go)** | None needed (stdlib) |
| **Config file (Frontend)** | `frontend/vitest.config.ts` |
| **Quick run command (Go)** | `go test ./internal/... -count=1 -run TestPhase5` |
| **Quick run command (Frontend)** | `cd frontend && npx vitest run --reporter=verbose src/pages/Settings.test.tsx src/components/PanelCard.test.tsx` |
| **Full suite command** | `go test ./... -count=1 && cd frontend && npx vitest run` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/... -count=1 && cd frontend && npx vitest run`
- **After every plan wave:** Run `go test ./... -count=1 && cd frontend && npx vitest run`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | 0 | ACCT-01 | unit (Go) | `go test ./internal/api/handlers/ -run TestUpdateAccount_DisplayName -count=1` | ❌ W0 | ⬜ pending |
| 05-01-02 | 01 | 0 | ACCT-01 | unit (Frontend) | `cd frontend && npx vitest run src/pages/Settings.test.tsx` | ❌ W0 | ⬜ pending |
| 05-01-03 | 01 | 0 | ACCT-02 | unit (Go) | `go test ./internal/api/handlers/ -run TestGetAccounts_DisplayName -count=1` | ❌ W0 | ⬜ pending |
| 05-01-04 | 01 | 0 | ACCT-02 | unit (Frontend) | `cd frontend && npx vitest run src/components/PanelCard.test.tsx` | ❌ W0 | ⬜ pending |
| 05-01-05 | 01 | 0 | ACCT-02 | unit (Go) | `go test ./internal/api/handlers/ -run TestGetSummary_ExcludesHidden -count=1` | ❌ W0 | ⬜ pending |
| 05-01-06 | 01 | 0 | ACCT-02 | unit (Go) | `go test ./internal/api/handlers/ -run TestGetBalanceHistory_TypeOverride -count=1` | ❌ W0 | ⬜ pending |
| 05-01-07 | 01 | 0 | OPS-03 | unit (Go) | `go test ./internal/sync/ -run TestSoftDelete -count=1` | ❌ W0 | ⬜ pending |
| 05-01-08 | 01 | 0 | OPS-03 | unit (Go) | `go test ./internal/sync/ -run TestSoftDelete_PreservesSnapshots -count=1` | ❌ W0 | ⬜ pending |
| 05-01-09 | 01 | 0 | OPS-03 | unit (Go) | `go test ./internal/sync/ -run TestRestore -count=1` | ❌ W0 | ⬜ pending |
| 05-01-10 | 01 | 0 | OPS-03 | unit (Go) | `go test ./internal/api/handlers/ -run TestUpdateAccount_HideUnhide -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/api/handlers/update_account_test.go` — stubs for ACCT-01, OPS-03 (display_name, hide/unhide)
- [ ] `internal/api/handlers/accounts_test.go` — extend with display_name and hidden_at test cases (ACCT-02)
- [ ] `internal/api/handlers/summary_test.go` — extend with hidden account exclusion tests (ACCT-02)
- [ ] `internal/api/handlers/history_test.go` — extend with type override and hidden exclusion tests (ACCT-02)
- [ ] `internal/sync/sync_test.go` — extend with soft-delete and restore tests (OPS-03)
- [ ] `frontend/src/pages/Settings.test.tsx` — extend with accounts section tests (ACCT-01)
- [ ] `frontend/src/components/PanelCard.test.tsx` — extend with display_name rendering test (ACCT-02)
- [ ] Migration file `000002_account_metadata.up.sql` and `.down.sql` — needed before any code changes
- [ ] `@dnd-kit/react` and `@dnd-kit/dom` npm install — needed for account type reassignment UI

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Drag-and-drop visual feedback | ACCT-02 | Visual animation/layout | Drag account between panel type groups, verify visual indicators |
| Toast notification auto-dismiss | ACCT-01 | Timing/visual behavior | Save display name, verify toast appears and auto-dismisses after 3s |
| Hidden account visual treatment | OPS-03 | Visual layout change | Trigger sync with missing account, verify it disappears from dashboard |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
