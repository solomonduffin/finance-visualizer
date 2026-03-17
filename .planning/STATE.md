---
gsd_state_version: 1.0
milestone: v1.1
milestone_name: Enhancements
status: executing
stopped_at: Completed 09-02-PLAN.md
last_updated: "2026-03-17T03:04:06Z"
last_activity: 2026-03-17 — Completed Plan 09-02 (Projection Settings API & Frontend Client)
progress:
  total_phases: 5
  completed_phases: 4
  total_plans: 18
  completed_plans: 15
  percent: 83
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-15)

**Core value:** Show the user exactly where all their money is right now, with one glance at a single dashboard.
**Current focus:** Milestone v1.1 — Phase 9 (Projection Engine) in progress

## Current Position

Phase: 9 of 9 (Projection Engine)
Plan: 4 of 5 (09-02 complete)
Status: In Progress
Last activity: 2026-03-17 — Completed Plan 09-02 (Projection Settings API & Frontend Client)

Progress: [████████░░] 83% (15/18 plans complete through Phase 9.4)

## Performance Metrics

**Velocity (v1.0):**
- Total plans completed: 11
- Total execution time: ~10.5 hours
- Average duration: ~57 min/plan

**By Phase (v1.0):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 Foundation | 3 | ~6.7h | ~134min |
| 02 Data Pipeline | 3 | ~36min | ~12min |
| 03 Backend API | 2 | ~7min | ~3.5min |
| 04 Frontend Dashboard | 3 | ~3.7h | ~74min |

**By Phase (v1.1):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 05 Data Foundation | 3/3 | 32min | 10.7min |
| 06 Operational Quick Wins | 3/3 | 12min | 4min |
| 07 Analytics Expansion | 1/3 | 8min | 8min |

*Updated after each plan completion*
| Phase 07 P01 | 9min | 2 tasks | 9 files |
| Phase 08 P01 | 5min | 2 tasks | 12 files |
| Phase 08 P03 | 5min | 2 tasks | 7 files |
| Phase 09 P01 | 6min | 2 tasks | 6 files |
| Phase 09 P03 | 4min | 2 tasks | 6 files |
| Phase 09 P02 | 4min | 2 tasks | 4 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [v1.0->v1.1]: SyncOnce now soft-deletes (UPDATE hidden_at) instead of hard-deleting stale accounts (completed in 05-01)
- [v1.1]: COALESCE(display_name, name) pattern implemented in all account queries (GetAccounts, GetSummary, GetBalanceHistory)
- [v1.1]: SyncOnce returns ([]string, error) with restored account display names
- [v1.1]: SyncNow handler runs synchronously and returns {ok:true, restored:[...]}
- [v1.1]: System-owned vs user-owned column separation enforced in processAccount upsert
- [v1.1]: NullableString custom JSON unmarshal for PATCH null/absent/string distinction
- [v1.1]: getAccountDisplayName utility as single source of truth for account name rendering
- [v1.1]: Optimistic state update for drag-and-drop type reassignment (prevents fly-off animation)
- [v1.1]: include_hidden=true query param on GET /api/accounts for Settings page
- [v1.1]: Custom account groups (Coinbase aggregation) deferred to Phase 7
- [v1.0]: Use shopspring/decimal for all financial arithmetic -- applies to growth indicators and projections
- [v1.1 research]: expr-lang/expr for alert expression evaluation, react-querybuilder for alert rule builder UI
- [v1.1 research]: go-mail v0.7.1 for SMTP email -- only maintained Go SMTP library with STARTTLS
- [v1.1]: SanitizeErrorText exported for testability; strips user:pass@host and base64 tokens from sync error text
- [v1.1]: Growth endpoint returns nil (JSON null) for panels with zero prior total -- avoids division by zero
- [v1.1]: queryPanelTotals helper reuses panel aggregation logic between current and prior snapshot queries
- [v1.1]: Custom CSS toggle switch (role=switch, aria-checked) rather than toggle library -- matches hand-rolled convention
- [v1.1]: DashboardPreferences receives props from Settings parent; parent owns state and error toast
- [v1.1]: SyncHistory accordion uses expandedId state with max-h CSS transition, motion-reduce support
- [v1.1]: Invisible placeholder pattern for GrowthBadge prevents layout shift when badge hidden
- [v1.1]: PanelCard font-bold migrated to font-semibold per UI-SPEC typography contract
- [v1.1]: Net worth endpoint carry-forward: missing panel data filled with last known value (not zero) for continuous stacked chart
- [v1.1]: period_change_pct null when first-point total is zero (division-by-zero protection, consistent with growth endpoint)
- [v1.1]: NetWorthDonut wraps in clickable div with role=link for drill-down navigation to /net-worth
- [v1.1]: TimeRangeSelector uses role=radiogroup/radio pattern for accessibility compliance
- [Phase 07]: fetchGroupResponse helper reused for consistent group response shape
- [Phase 07]: Panel contribution pattern: standalone accounts use effective_type, grouped accounts use group.panel_type
- [Phase 08]: float64 for expr-lang Environment values; shopspring/decimal for computation display
- [Phase 08]: Group totals queried separately from per-account balances in BuildEnvironment
- [Phase 08]: Operand select uses encoded value strings (type:ref:label) for round-trip fidelity
- [Phase 08]: formatExpressionSummary exported from AlertRuleCard for testability and reuse
- [Phase 08]: AlertRuleForm loads accounts/groups on mount via getAccounts() for dropdown population
- [Phase 09]: fetchAccountData private helper extracts shared logic between FetchAccounts and FetchAccountsWithHoldings
- [Phase 09]: Holdings persistence uses transactional DELETE+INSERT rather than individual UPSERTs
- [Phase 09]: persistHoldings only called for InferAccountType == "investment" with non-empty Holdings
- [Phase 09]: Compound interest test expectation corrected: plan ~$16386 vs actual $16651.05 (contributions also compound)
- [Phase 09]: hasHoldings flag prevents double-counting: accounts with holdings skipped in account loop
- [Phase 09]: Allocation validation suppresses income when sum != 100% (growth-only fallback)

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 8]: Protonmail Bridge Docker networking needs validation during planning
- [Phase 8]: react-querybuilder JSON output to expr-lang input alignment needs confirmation
- [Phase 6]: Credit card balance sign semantics in growth indicators (negative to less-negative is improvement)

## Session Continuity

Last session: 2026-03-17T03:04:06Z
Stopped at: Completed 09-02-PLAN.md
Resume file: None
