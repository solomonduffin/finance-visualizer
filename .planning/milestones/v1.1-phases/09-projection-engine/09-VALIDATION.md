---
phase: 9
slug: projection-engine
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-17
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (backend) / vitest (frontend) |
| **Config file** | `vitest.config.ts` (frontend), `go test ./...` (backend) |
| **Quick run command** | `cd frontend && npx vitest run --reporter=verbose` |
| **Full suite command** | `cd backend && go test ./... && cd ../frontend && npx vitest run` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && npx vitest run --reporter=verbose`
- **After every plan wave:** Run `cd backend && go test ./... && cd ../frontend && npx vitest run`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | PROJ-01 | unit | `go test ./internal/projections/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PROJ-02 | unit | `go test ./internal/projections/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PROJ-03 | unit | `npx vitest run src/components/projections` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PROJ-04 | unit | `go test ./internal/projections/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PROJ-05 | integration | `go test ./internal/handlers/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PROJ-06 | unit | `npx vitest run src/components/projections` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PROJ-07 | unit | `npx vitest run src/components/projections` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PROJ-08 | integration | `go test ./internal/simplefin/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `backend/internal/projections/` — new package for projection calculations and handlers
- [ ] `frontend/src/components/projections/` — new component directory
- [ ] Migration file for projection settings tables

*Existing test infrastructure (go test, vitest) covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Chart renders dashed line for projections | PROJ-03 | Visual rendering verification | Open /projections, verify dashed line appears after "Now" marker |
| Holdings display from SimpleFIN | PROJ-08 | Depends on live SimpleFIN data | Check investment accounts show holdings when available |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
