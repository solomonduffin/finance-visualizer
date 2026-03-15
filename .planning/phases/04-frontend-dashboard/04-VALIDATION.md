---
phase: 4
slug: frontend-dashboard
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-15
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest 3.2.1 + @testing-library/react 16.3.2 |
| **Config file** | `frontend/vitest.config.ts` |
| **Quick run command** | `cd frontend && npx vitest run --reporter=verbose` |
| **Full suite command** | `cd frontend && npx vitest run` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && npx vitest run --reporter=verbose`
- **After every plan wave:** Run `cd frontend && npx vitest run`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 0 | VIZ-01 | unit | `cd frontend && npx vitest run src/components/BalanceLineChart.test.tsx` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 0 | VIZ-02 | unit | `cd frontend && npx vitest run src/components/NetWorthDonut.test.tsx` | ❌ W0 | ⬜ pending |
| 04-01-03 | 01 | 0 | UX-03 | unit | `cd frontend && npx vitest run src/hooks/useDarkMode.test.ts` | ❌ W0 | ⬜ pending |
| 04-01-04 | 01 | 0 | UX-04 | unit | `cd frontend && npx vitest run src/components/PanelCard.test.tsx` | ❌ W0 | ⬜ pending |
| 04-01-05 | 01 | 0 | UX-01, UX-02 | unit | `cd frontend && npx vitest run src/pages/Dashboard.test.tsx` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/hooks/useDarkMode.test.ts` — stubs for UX-03
- [ ] `frontend/src/components/PanelCard.test.tsx` — stubs for UX-04
- [ ] `frontend/src/components/BalanceLineChart.test.tsx` — stubs for VIZ-01
- [ ] `frontend/src/components/NetWorthDonut.test.tsx` — stubs for VIZ-02
- [ ] `frontend/src/pages/Dashboard.test.tsx` — stubs for UX-01, UX-02
- [ ] ResizeObserver stub addition to `frontend/src/test-setup.ts`
- [ ] `recharts` package install: `cd frontend && npm install recharts`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Responsive layout on mobile | UX-04 | Visual/layout verification | Resize browser to 375px width, verify panels stack vertically |
| Dark/light mode toggle visual | UX-03 | Color scheme is visual | Toggle dark mode, verify background/text contrast |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
