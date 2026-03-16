---
phase: 6
slug: operational-quick-wins
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-16
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | `go test` with `testing` package |
| **Framework (Frontend)** | Vitest 3.2.1 + @testing-library/react 16.3.2 |
| **Config file (Go)** | None needed (standard `go test`) |
| **Config file (Frontend)** | `frontend/vitest.config.ts` (jsdom environment) |
| **Quick run command (Go)** | `cd /home/solomon/finance-visualizer && go test ./internal/api/handlers/ -run TestSyncLog -v` |
| **Quick run command (Frontend)** | `cd /home/solomon/finance-visualizer/frontend && npx vitest run --reporter=verbose src/components/GrowthBadge.test.tsx` |
| **Full suite command (Go)** | `cd /home/solomon/finance-visualizer && go test ./...` |
| **Full suite command (Frontend)** | `cd /home/solomon/finance-visualizer/frontend && npx vitest run` |
| **Estimated runtime** | ~15 seconds (Go) + ~10 seconds (Frontend) |

---

## Sampling Rate

- **After every task commit:** Run relevant Go handler tests + relevant frontend component tests
- **After every plan wave:** Run `go test ./...` + `npx vitest run`
- **Before `/gsd:verify-work`:** Both full suites must be green
- **Max feedback latency:** 25 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 06-01-01 | 01 | 1 | OPS-01 | unit (Go) | `go test ./internal/api/handlers/ -run TestGetSyncLog -v` | ❌ W0 | ⬜ pending |
| 06-01-02 | 01 | 1 | OPS-02 | unit (Go) | `go test ./internal/api/handlers/ -run TestSanitize -v` | ❌ W0 | ⬜ pending |
| 06-02-01 | 02 | 1 | INSIGHT-01 | unit (Go) | `go test ./internal/api/handlers/ -run TestGetGrowth -v` | ❌ W0 | ⬜ pending |
| 06-02-02 | 02 | 1 | INSIGHT-01 | unit (Go) | `go test ./internal/api/handlers/ -run TestGetGrowthZero -v` | ❌ W0 | ⬜ pending |
| 06-03-01 | 03 | 2 | OPS-01 | unit (FE) | `npx vitest run src/components/SyncHistory.test.tsx` | ❌ W0 | ⬜ pending |
| 06-03-02 | 03 | 2 | OPS-02 | unit (FE) | `npx vitest run src/components/SyncHistory.test.tsx` | ❌ W0 | ⬜ pending |
| 06-03-03 | 03 | 2 | INSIGHT-01 | unit (FE) | `npx vitest run src/components/GrowthBadge.test.tsx` | ❌ W0 | ⬜ pending |
| 06-03-04 | 03 | 2 | INSIGHT-06 | unit (FE) | `npx vitest run src/components/DashboardPreferences.test.tsx` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/api/handlers/synclog_test.go` — stubs for OPS-01, OPS-02 (sync log endpoint + sanitization)
- [ ] `internal/api/handlers/growth_test.go` — stubs for INSIGHT-01 (growth calculation with edge cases)
- [ ] `frontend/src/components/GrowthBadge.test.tsx` — stubs for INSIGHT-01 frontend
- [ ] `frontend/src/components/SyncHistory.test.tsx` — stubs for OPS-01, OPS-02 frontend
- [ ] `frontend/src/components/DashboardPreferences.test.tsx` — stubs for INSIGHT-06 frontend
- [ ] `internal/api/handlers/settings_test.go` — extend existing file with growth toggle tests

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Layout shift when badge appears | INSIGHT-01 | Visual rendering check | 1. Load dashboard 2. Verify badge doesn't shift card layout 3. Toggle badge off/on |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 25s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
