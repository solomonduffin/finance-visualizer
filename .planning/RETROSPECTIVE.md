# Retrospective: Finance Visualizer

---

## Milestone: v1.1 — Enhancements

**Shipped:** 2026-03-17
**Phases:** 5 (Phases 5-9) | **Plans:** 18

### What Was Built

- **Phase 5:** Soft-delete + COALESCE display name system — account metadata survives SimpleFIN outages
- **Phase 6:** Sync diagnostics timeline in settings + 30-day growth rate badges on panel cards
- **Phase 7:** Custom account groups with drag-and-drop + dedicated net worth drill-down page
- **Phase 8:** Expression-based alert rules with 3-state machine + AES-256-GCM encrypted SMTP notifications
- **Phase 9:** Projection engine — per-account APY/growth rates, compound/simple interest, income modeling, SimpleFIN holdings detail

### What Worked

- **Phase independence (5-fan-out pattern):** Phases 6, 7, 8, 9 all depended on Phase 5 but were fully independent of each other. Clear dependency documentation meant no integration surprises.
- **TDD approach on core algorithms:** Alert expression evaluator and projection calculation engine were both built test-first. This caught a compound interest bug (contributions also compound) before it shipped.
- **COALESCE pattern propagation:** Established once in Phase 5 and propagated consistently to all handlers — zero divergence discovered in later phases.
- **Component decomposition:** Large phases (8, 9) decomposed into clean backend/API/frontend/integration splits that executed cleanly.
- **Debounced auto-save:** Page-level state orchestration with debounced persistence kept the projection UX responsive without hammering the API.

### What Was Inefficient

- **ROADMAP.md progress table fell stale:** Phases 6, 7, 8 checkboxes and plan counts were not updated as phases completed. Required manual correction at milestone archive time.
- **Snapshot re-checking at archive time:** Had to re-verify disk status via `roadmap analyze` because the ROADMAP.md text diverged from actual disk state. A ROADMAP auto-update after plan completion would prevent this.
- **No v1.1 milestone audit:** Completed milestone without running `/gsd:audit-milestone` first. Would have caught the ROADMAP stale state earlier.

### Patterns Established

- **Invisible placeholder pattern:** GrowthBadge uses a same-sized invisible element when hidden to prevent layout shift — reusable for any conditional badge/indicator.
- **Panel contribution pattern:** Standalone accounts use `effective_type`, grouped accounts use `group.panel_type` — consistent across growth, accounts, and net worth endpoints.
- **Operand encoding:** Alert expression operands encode as `type:ref:label` strings for round-trip fidelity between frontend builder and backend evaluator.
- **hasHoldings flag:** Prevents double-counting in projection engine when holdings are present.

### Key Lessons

1. Update ROADMAP.md plan checkboxes immediately when a plan completes — don't rely on end-of-milestone cleanup.
2. Run `/gsd:audit-milestone` before `/gsd:complete-milestone` — catches stale docs and verifiable gaps.
3. Phase 5 as a dedicated "schema prerequisites" phase (soft-delete, display_name, type_override) before any feature that stores per-account user config was the right architectural decision — zero schema conflicts in Phases 6-9.
4. Client-side projection calculation with debounced persistence is the right pattern for interactive financial modeling — avoids server round-trips for UX responsiveness.

### Cost Observations

- Timeline: 3 days for 18 plans across 5 phases
- Phases 6 and 7 executed fast (sync log and growth endpoints were straightforward); Phases 8 and 9 required more back-and-forth on state machine and projection math

---

## Cross-Milestone Trends

| Milestone | Phases | Plans | Timeline | Avg Plans/Day |
|-----------|--------|-------|----------|---------------|
| v1.0 MVP | 4 | 11 | 1 day (2026-03-15) | 11 |
| v1.1 Enhancements | 5 | 18 | 3 days (2026-03-15 → 2026-03-17) | 6 |

**Observations:**
- Both milestones shipped quickly, benefiting from GSD's atomic commit + state tracking discipline
- v1.1 took longer due to higher algorithmic complexity (alert state machine, projection math) vs v1.0's infrastructure work
- Phase fan-out pattern (independent phases off a single prerequisite) worked well for v1.1 and should be carried forward
