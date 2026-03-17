---
phase: 7
slug: analytics-expansion
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-16
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (frontend)** | Vitest + @testing-library/react + jsdom |
| **Framework (backend)** | Go testing with httptest |
| **Config file (frontend)** | `frontend/vitest.config.ts` |
| **Config file (backend)** | Built-in Go test runner |
| **Quick run command (frontend)** | `cd frontend && npx vitest run --reporter=verbose` |
| **Quick run command (backend)** | `cd internal && go test ./...` |
| **Full suite command** | `cd frontend && npx vitest run && cd ../internal && go test ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick command for affected area (frontend or backend)
- **After every plan wave:** Run `cd frontend && npx vitest run && cd ../internal && go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 07-01-01 | 01 | 1 | ACCT-03 | unit (backend) | `go test ./internal/api/handlers/ -run TestGroup -v` | ❌ W0 | ⬜ pending |
| 07-01-02 | 01 | 1 | ACCT-03 | unit (frontend) | `cd frontend && npx vitest run src/components/AccountsSection.test.tsx` | ✅ extend | ⬜ pending |
| 07-02-01 | 02 | 2 | ACCT-04 | unit (frontend) | `cd frontend && npx vitest run src/components/PanelCard.test.tsx` | ✅ extend | ⬜ pending |
| 07-02-02 | 02 | 2 | ACCT-05 | unit (frontend) | `cd frontend && npx vitest run src/components/GroupRow.test.tsx` | ❌ W0 | ⬜ pending |
| 07-03-01 | 03 | 2 | INSIGHT-02 | unit (frontend) | `cd frontend && npx vitest run src/components/NetWorthDonut.test.tsx` | ✅ extend | ⬜ pending |
| 07-04-01 | 04 | 3 | INSIGHT-03 | unit (frontend) | `cd frontend && npx vitest run src/components/StackedAreaChart.test.tsx` | ❌ W0 | ⬜ pending |
| 07-04-02 | 04 | 3 | INSIGHT-04 | unit (frontend) | `cd frontend && npx vitest run src/components/NetWorthStats.test.tsx` | ❌ W0 | ⬜ pending |
| 07-04-03 | 04 | 3 | INSIGHT-04 | unit (backend) | `go test ./internal/api/handlers/ -run TestNetWorth -v` | ❌ W0 | ⬜ pending |
| 07-04-04 | 04 | 3 | INSIGHT-05 | unit (frontend) | `cd frontend && npx vitest run src/components/TimeRangeSelector.test.tsx` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/api/handlers/groups_test.go` — stubs for ACCT-03 (CRUD operations, auto-delete empty group, panel type validation)
- [ ] `internal/api/handlers/networth_test.go` — stubs for INSIGHT-03/04/05 (time-series, stats, time range filtering)
- [ ] `frontend/src/components/GroupRow.test.tsx` — stubs for ACCT-04/05 (collapse/expand, member display)
- [ ] `frontend/src/components/StackedAreaChart.test.tsx` — stubs for INSIGHT-03 (renders 3 stacked areas)
- [ ] `frontend/src/components/NetWorthStats.test.tsx` — stubs for INSIGHT-04 (stats display)
- [ ] `frontend/src/components/TimeRangeSelector.test.tsx` — stubs for INSIGHT-05 (selection, callback)
- [ ] `frontend/src/pages/NetWorth.test.tsx` — stubs for INSIGHT-02/03/04/05 (page integration)
- Existing test files to extend: `PanelCard.test.tsx`, `NetWorthDonut.test.tsx`, `AccountsSection.test.tsx`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Donut chart click navigation | INSIGHT-02 | Recharts onClick + router navigation hard to test in jsdom | Click donut center → verify URL changes to /net-worth |
| Drag-drop account into group | ACCT-03 | dnd-kit pointer events not reliably simulatable | Drag account row onto group → verify membership updates |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
