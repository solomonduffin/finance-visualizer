# Phase 8: Alert System - Context

**Gathered:** 2026-03-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Expression-based alert rules with a 3-state machine (normal/triggered/recovered) and email notifications via SMTP. Users define threshold-based rules using a visual expression builder, receive email on threshold crossing and recovery, and manage rules from a dedicated Alerts page. SMTP configuration lives in Settings.

</domain>

<decisions>
## Implementation Decisions

### Expression Builder UX
- Visual query builder only — no raw expression text field
- Modular operand model: "When" section has a dynamic list of operand terms with [+] and [-] buttons to add/subtract terms
- Each term is a dropdown selecting from: buckets (Liquid Balance, Savings Balance, Investments Balance, Net Worth), account groups, or individual accounts (by display name)
- Terms connected by +/- arithmetic operators
- Minimum 1 operand, no maximum
- Comparison section: operator dropdown (<, <=, >, >=, ==) and numeric threshold input
- "Notify on recovery" checkbox per rule
- Builder form appears inline on the Alerts page — expands when clicking "+ New Alert", no modal

### Rule Identity
- Required name field — users give each rule a descriptive name (e.g., "Low cash warning")
- Name appears in alert emails, management list, and history

### Alert Management Page
- New route: /alerts
- Top nav link "Alerts" alongside Dashboard, Net Worth, Settings (consistent with Phase 7 nav pattern)
- Card-row layout for rules: each card shows rule name, expression summary, current status badge (Normal/Triggered), last checked timestamp, toggle switch (enable/disable), edit and delete actions
- Expandable history per rule: chevron reveals recent trigger/recovery events with timestamps and computed values
- Edit mode: clicking edit re-opens the inline builder pre-filled with rule values
- Empty state: illustration + "No alert rules yet" message + "Create your first alert" CTA button (uses EmptyState component pattern)

### Alert Notification Content
- Plain text emails only
- Subject format: `[Finance Alert] {rule name}` (append "— recovered" for recovery)
- Body includes: rule name, status (TRIGGERED/RECOVERED with directional indicator), computed value, threshold with operator, timestamp
- Account breakdown section: lists each operand account/bucket with its individual value
- Recovery emails use identical format with RECOVERED status
- Footer: "Finance Visualizer" signature line

### Email Provider Configuration
- SMTP only — no API provider support
- Configuration lives in Settings page as a new "Email Configuration" section (below Accounts)
- Fields: SMTP Host, Port, Username, Password, From address, To address
- SMTP password encrypted at rest using key derived from app auth secret (AES-256), decrypted on-the-fly when sending
- "Test Email" button sends using current form values (even unsaved) with inline success/error message below the button
- "Save" button persists configuration to settings table

### 3-State Machine
- States: normal → triggered → recovered → normal (cycle)
- Alert fires exactly once on threshold crossing (normal → triggered)
- Recovery fires exactly once when value crosses back (triggered → recovered)
- No repeated emails on subsequent syncs while condition holds
- Evaluation triggered after each successful SyncOnce completion

### Claude's Discretion
- Database schema design for alert_rules and alert_history tables
- Expression compilation/evaluation implementation details (expr-lang/expr recommended from prior research)
- SMTP library choice (go-mail recommended from prior research)
- Alert card styling, spacing, responsive breakpoints
- Loading states for the alerts page
- Error handling for expression evaluation failures
- How to render complex multi-term expressions as readable summary text in rule cards

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Alert requirements
- `.planning/REQUIREMENTS.md` — ALERT-01 through ALERT-07: expression builder, threshold comparison, 3-state machine, email content, SMTP config, test email, CRUD operations

### Architecture research
- `.planning/research/ARCHITECTURE.md` — Feature 6 section: proposed schema (alert_rules, alert_history), file locations (internal/alerts/), API endpoints, frontend components, expr-lang/expr evaluation pattern

### Prior phase context
- `.planning/phases/05-data-foundation/05-CONTEXT.md` — COALESCE(display_name, name) pattern, NullableString for PATCH semantics
- `.planning/phases/07-analytics-expansion/07-CONTEXT.md` — Account groups model (groups as operands in alert expressions), nav bar pattern

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `frontend/src/components/EmptyState.tsx`: Reusable empty state component — use for alerts page empty state
- `frontend/src/components/Toast.tsx`: Toast notification component — may complement inline test email feedback
- `frontend/src/api/client.ts`: Single API client — extend with alert CRUD functions
- `internal/api/router.go`: Chi router with existing protected route group — add alert routes here
- `internal/sync/sync.go`: SyncOnce function — hook point for triggering alert evaluation after sync

### Established Patterns
- Settings stored as key-value pairs in `settings` table — extend for SMTP config keys
- `COALESCE(display_name, name)` for account display names — use in alert operand dropdowns and email breakdowns
- Protected routes use JWT cookie auth via `jwtauth` — alert routes follow same pattern
- Account groups CRUD at `/api/groups/*` — similar pattern for `/api/alerts/*`

### Integration Points
- `internal/sync/sync.go` after `SyncOnce` — call alert evaluation engine
- `internal/api/router.go` protected group — register alert CRUD routes
- `frontend/src/App.tsx` — add `/alerts` route and nav link
- `frontend/src/pages/Settings.tsx` — add Email Configuration section

</code_context>

<specifics>
## Specific Ideas

- Expression builder must feel modular — user adds/removes operand rows with +/- buttons, not typing expressions
- Alert cards should show enough info at a glance (name, expression summary, status, last checked) without needing to expand
- Recovery emails should mirror trigger emails exactly (same layout, different status) for consistency
- Test email should work with unsaved form values for quick iteration on SMTP config

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 08-alert-system*
*Context gathered: 2026-03-16*
