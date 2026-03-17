---
phase: 08-alert-system
plan: 01
subsystem: alerts
tags: [expr-lang, go-mail, aes-256-gcm, sqlite, state-machine, smtp]

# Dependency graph
requires:
  - phase: 05-data-foundation
    provides: account_type_override, hidden_at, group_members tables
  - phase: 07-analytics-expansion
    provides: account_groups, panel contribution pattern
provides:
  - internal/alerts package with crypto, evaluator, engine, notifier
  - alert_rules and alert_history database tables (migration 000004)
  - expr-lang/expr and wneessen/go-mail Go dependencies
affects: [08-02 (API handlers), 08-03 (frontend), 08-04 (sync hook + SMTP settings)]

# Tech tracking
tech-stack:
  added: [github.com/expr-lang/expr@v1.17.8, github.com/wneessen/go-mail@v0.7.2]
  patterns: [operand-based expression compilation, 3-state alert machine, AES-256-GCM encryption with SHA-256 key derivation]

key-files:
  created:
    - internal/db/migrations/000004_alert_rules.up.sql
    - internal/db/migrations/000004_alert_rules.down.sql
    - internal/alerts/crypto.go
    - internal/alerts/crypto_test.go
    - internal/alerts/evaluator.go
    - internal/alerts/evaluator_test.go
    - internal/alerts/engine.go
    - internal/alerts/engine_test.go
    - internal/alerts/notifier.go
    - internal/alerts/notifier_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "float64 for expression environment values (monetary precision via shopspring/decimal for computation, float64 for expr-lang evaluation)"
  - "Group totals queried separately from per-account balances for clean separation in BuildEnvironment"
  - "computeLHSValue helper recomputes operand arithmetic using decimal for accurate display values"

patterns-established:
  - "Operand compilation: JSON operand array -> expr-lang expression string via CompileOperands"
  - "3-state machine: NextState pure function with (current, conditionMet, notifyRecovery) -> (newState, shouldNotify)"
  - "Best-effort notification: sendNotification with 10s context timeout, errors logged not returned"
  - "Settings-based SMTP config: LoadSMTPConfig reads key-value pairs from settings table with encrypted password"

requirements-completed: [ALERT-01, ALERT-02, ALERT-03, ALERT-04]

# Metrics
duration: 5min
completed: 2026-03-16
---

# Phase 8 Plan 1: Core Alert Engine Summary

**expr-lang expression evaluator with 3-state machine, AES-256-GCM crypto, and SMTP notifier -- 30 tests passing**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-16T20:39:09Z
- **Completed:** 2026-03-16T20:44:22Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments
- Database migration with alert_rules and alert_history tables, including foreign key cascade and index
- Expression evaluator that compiles structured JSON operands into validated expr-lang expressions supporting bucket, account, and group references
- 3-state alert machine (normal/triggered/recovered) with correct transition table and notification suppression
- AES-256-GCM encryption for SMTP password storage with SHA-256 key derivation
- Plain text email formatter with subject format, account breakdown, and go-mail SMTP delivery
- BuildEnvironment queries current balances using same panel aggregation logic as summary endpoint
- EvaluateAll orchestrator that evaluates all enabled rules, updates state, records history, and sends notifications best-effort
- 30 unit tests covering all core functions with zero regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Database migration and Go dependencies** - `e349df6` (feat)
2. **Task 2 RED: Failing tests for alerts package** - `a9fbb14` (test)
3. **Task 2 GREEN: Implement core alerts package** - `5bbe229` (feat)

## Files Created/Modified
- `internal/db/migrations/000004_alert_rules.up.sql` - alert_rules and alert_history tables with CHECK constraints
- `internal/db/migrations/000004_alert_rules.down.sql` - Drop tables and index
- `internal/alerts/crypto.go` - DeriveKey, Encrypt, Decrypt (AES-256-GCM)
- `internal/alerts/crypto_test.go` - 4 tests: key derivation, round-trip, wrong key, nonce uniqueness
- `internal/alerts/evaluator.go` - CompileOperands, Validate, Evaluate, BuildEnvironment with Environment/Operand types
- `internal/alerts/evaluator_test.go` - 13 tests: operand compilation variants, evaluation, validation
- `internal/alerts/engine.go` - NextState, EvaluateAll, evaluateRule, computeLHSValue helpers
- `internal/alerts/engine_test.go` - 9 tests: all state transitions, BuildEnvironment, EvaluateAll integration
- `internal/alerts/notifier.go` - FormatSubject, FormatAlertBody, SendAlert, LoadSMTPConfig with SMTPConfig/AlertDetail types
- `internal/alerts/notifier_test.go` - 4 tests: body format triggered/recovered, subject format
- `go.mod` - Added expr-lang/expr v1.17.8 and wneessen/go-mail v0.7.2
- `go.sum` - Updated dependency checksums

## Decisions Made
- Used float64 for expr-lang Environment fields since expr-lang requires numeric types for comparisons; shopspring/decimal is used in computeLHSValue for accurate display formatting
- Group totals are queried in a separate SQL query from per-account balances for cleaner separation and to avoid double-counting in BuildEnvironment
- computeLHSValue helper re-evaluates operand arithmetic using decimal to produce accurate formatted values independent of the boolean expression result

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Core alerts package ready for Plan 08-02 (CRUD API handlers) to build on
- alert_rules and alert_history tables available via migration 000004
- CompileOperands, Evaluate, EvaluateAll, SendAlert, LoadSMTPConfig all exported and tested
- Plan 08-04 can wire EvaluateAll into SyncOnce and build SMTP settings endpoints

## Self-Check: PASSED

- All 11 created files verified present on disk
- All 3 commits verified in git log (e349df6, a9fbb14, 5bbe229)
- Full test suite passes with zero regressions

---
*Phase: 08-alert-system*
*Completed: 2026-03-16*
