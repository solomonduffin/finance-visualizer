# Phase 8: Alert System - Research

**Researched:** 2026-03-16
**Domain:** Threshold-based alert rules with expression evaluation, 3-state machine, SMTP email notifications
**Confidence:** HIGH

## Summary

Phase 8 implements a complete alert system: users create threshold-based rules via a visual expression builder, the system evaluates rules after each sync using a 3-state machine (normal/triggered/recovered), and sends email notifications on state transitions via SMTP. This phase introduces two new Go dependencies (expr-lang/expr for expression evaluation, wneessen/go-mail for SMTP), a new `internal/alerts/` package, new database tables, CRUD API endpoints, a new frontend Alerts page, and SMTP configuration in Settings.

The architecture follows the established project patterns: thin handlers delegating to a dedicated `internal/alerts/` package, settings stored as key-value pairs in the existing `settings` table, migrations as separate numbered files, and the standard chi router + JWT auth middleware. The CONTEXT.md specifies a custom visual expression builder (not react-querybuilder) with modular operand rows using +/- buttons, which is simpler and more tailored than a generic query builder library.

**Primary recommendation:** Build in 3 sub-phases: (1) database schema + CRUD API + frontend Alerts page with expression builder, (2) expression evaluation engine + sync hook + 3-state machine, (3) SMTP configuration + email notifier + test email.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Visual query builder only -- no raw expression text field
- Modular operand model: "When" section with dynamic list of operand terms, [+] and [-] buttons
- Each term is a dropdown: buckets (Liquid Balance, Savings Balance, Investments Balance, Net Worth), account groups, or individual accounts (by display name)
- Terms connected by +/- arithmetic operators; minimum 1 operand, no maximum
- Comparison section: operator dropdown (<, <=, >, >=, ==) and numeric threshold input
- "Notify on recovery" checkbox per rule
- Builder form appears inline on the Alerts page (no modal)
- Required name field per rule
- New route: /alerts with top nav link
- Card-row layout for rules with: name, expression summary, status badge (Normal/Triggered), last checked timestamp, toggle switch, edit/delete actions
- Expandable history per rule with chevron
- Plain text emails only
- Subject format: `[Finance Alert] {rule name}` (append "-- recovered" for recovery)
- Body includes: rule name, status, computed value, threshold, operator, timestamp, account breakdown
- SMTP only -- no API provider support
- Settings page "Email Configuration" section below Accounts
- Fields: SMTP Host, Port, Username, Password, From address, To address
- SMTP password encrypted at rest using key derived from app auth secret (AES-256)
- "Test Email" button with inline success/error feedback
- 3-state machine: normal -> triggered -> recovered -> normal
- Evaluation triggered after each successful SyncOnce completion

### Claude's Discretion
- Database schema design for alert_rules and alert_history tables
- Expression compilation/evaluation implementation details (expr-lang/expr recommended)
- SMTP library choice (go-mail recommended)
- Alert card styling, spacing, responsive breakpoints
- Loading states for the alerts page
- Error handling for expression evaluation failures
- How to render complex multi-term expressions as readable summary text in rule cards

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| ALERT-01 | User can create alert rules using an expression builder combining buckets and/or accounts with +/- operators | Custom visual builder with operand rows, stored as JSON operand array in DB, compiled to expr-lang expression at evaluation time |
| ALERT-02 | Alert rules compare computed value against a threshold using <, <=, >, >=, == operators | Comparison operator stored per rule, applied during expr-lang evaluation against threshold |
| ALERT-03 | Alerts fire once on threshold crossing and once on recovery (3-state machine) | last_state column tracks normal/triggered, state transitions trigger notifications, notify_on_recovery flag controls recovery emails |
| ALERT-04 | Alert email includes rule name, computed value, threshold, and crossing direction with account context | Plain text email with account breakdown section, go-mail for SMTP delivery |
| ALERT-05 | User can configure email provider in settings (SMTP with provider-specific fields) | SMTP config as key-value pairs in settings table, password encrypted with AES-256-GCM |
| ALERT-06 | User can send a test email to verify configuration | Dedicated POST /api/email/test endpoint, uses current form values (even unsaved) |
| ALERT-07 | User can create, edit, enable/disable, and delete alert rules | Standard CRUD endpoints at /api/alerts/*, toggle enable via PATCH |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| expr-lang/expr | v1.17.8 | Safe, sandboxed expression evaluation | Used by Google, Uber; compile-time type checking, no side effects, always terminates. Perfect for evaluating user-defined threshold expressions. |
| wneessen/go-mail | v0.7.2 | SMTP email delivery | Only actively maintained Go SMTP library with STARTTLS, auto-discovery auth, and proper TLS handling. Replaces deprecated net/smtp. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| crypto/aes + crypto/cipher (stdlib) | Go 1.25 | AES-256-GCM encryption for SMTP password | Encrypting SMTP password at rest, deriving key from JWT_SECRET via SHA-256 |
| crypto/sha256 (stdlib) | Go 1.25 | Key derivation from JWT_SECRET | Derive 32-byte AES key from variable-length JWT_SECRET |
| shopspring/decimal | v1.4.0 (already installed) | Financial arithmetic in expression evaluation | Computing operand values with monetary precision |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| expr-lang/expr | Manual arithmetic | expr provides safe sandboxing, type checking, and handles edge cases; hand-rolling is error-prone |
| wneessen/go-mail | net/smtp (stdlib) | net/smtp deprecated since Go 1.24; go-mail handles TLS, auth discovery, connection pooling |
| react-querybuilder | Custom operand builder | User decision: custom builder with +/- operand rows is simpler and more tailored than react-querybuilder's generic SQL-style UI |
| AES-256-GCM | Plain text storage | Security requirement: SMTP passwords must be encrypted at rest |

**Installation:**
```bash
# Go dependencies
go get github.com/expr-lang/expr@v1.17.8
go get github.com/wneessen/go-mail@v0.7.2
```

No new frontend npm packages required. The expression builder is a custom component.

## Architecture Patterns

### Recommended Project Structure
```
internal/
  alerts/
    evaluator.go       # Expression compilation and evaluation
    engine.go          # Alert evaluation orchestrator (post-sync)
    notifier.go        # SMTP email sender
    crypto.go          # AES-256-GCM encrypt/decrypt for SMTP password
    evaluator_test.go  # Unit tests for expression evaluation
    engine_test.go     # Unit tests for state machine logic
    notifier_test.go   # Unit tests for email formatting (not sending)
    crypto_test.go     # Unit tests for encryption round-trip
  api/handlers/
    alerts.go          # CRUD handlers for alert rules
    email.go           # SMTP config save + test email endpoint
    alerts_test.go     # HTTP handler tests
    email_test.go      # HTTP handler tests
  db/migrations/
    000004_alert_rules.up.sql
    000004_alert_rules.down.sql

frontend/src/
  pages/
    Alerts.tsx         # Alert management page
    Alerts.test.tsx    # Page tests
  components/
    AlertRuleForm.tsx       # Expression builder form
    AlertRuleCard.tsx       # Rule card with status, toggle, expand
    AlertRuleForm.test.tsx
    AlertRuleCard.test.tsx
  api/
    client.ts          # Extended with alert CRUD + email config functions
```

### Pattern 1: Operand-Based Expression Model
**What:** Instead of storing raw expression strings, store a structured JSON operand array. The frontend builds operand rows (each with a type, reference ID, and +/- operator). The backend compiles this structure into an expr-lang expression at evaluation time.
**When to use:** Alert rule creation, editing, and evaluation.
**Example:**
```typescript
// Frontend operand model
interface Operand {
  type: 'bucket' | 'group' | 'account'
  ref: string        // "liquid" | "savings" | "investments" | "net_worth" | group ID | account ID
  label: string      // Display label for the UI
  operator: '+' | '-' // How this operand combines (first operand always '+')
}

interface AlertRule {
  name: string
  operands: Operand[]
  comparison: '<' | '<=' | '>' | '>=' | '=='
  threshold: string   // Decimal string
  notify_on_recovery: boolean
}
```

```go
// Backend: stored in alert_rules.expression as JSON
// [{"type":"bucket","ref":"liquid","operator":"+"},{"type":"account","ref":"acct-123","operator":"-"}]
//
// At evaluation time, compiled to: env.Liquid + env.Accounts["acct-123"]
// Then compared: result < threshold
```

**Rationale:** Storing structured operands (not raw expression text) makes the visual builder bidirectional -- rules can be loaded back into the builder for editing. It also prevents expression injection since the backend controls compilation.

### Pattern 2: 3-State Machine
**What:** Each alert rule maintains a `last_state` column: `normal`, `triggered`, or `recovered`. State transitions trigger notifications.
**When to use:** After each sync evaluation.
**Example:**
```go
// State transition table:
// current_state  | condition_met | new_state   | action
// normal         | true          | triggered   | send trigger email
// normal         | false         | normal      | no-op
// triggered      | true          | triggered   | no-op (suppress duplicate)
// triggered      | false         | recovered   | send recovery email (if enabled)
// recovered      | true          | triggered   | send trigger email
// recovered      | false         | normal      | no-op (silent reset)

func nextState(current string, conditionMet bool, notifyRecovery bool) (string, bool) {
    switch current {
    case "normal":
        if conditionMet {
            return "triggered", true // send trigger email
        }
        return "normal", false
    case "triggered":
        if !conditionMet {
            if notifyRecovery {
                return "recovered", true // send recovery email
            }
            return "normal", false // silent reset
        }
        return "triggered", false
    case "recovered":
        if conditionMet {
            return "triggered", true // send trigger email
        }
        return "normal", false
    }
    return "normal", false
}
```

### Pattern 3: Post-Sync Hook (Established)
**What:** After SyncOnce completes successfully, call `alerts.EvaluateAll(ctx, db)`. Alert failures never fail the sync.
**When to use:** Wiring alert evaluation into the sync flow.
**Example:**
```go
// In sync.go, after finalize(fetched, failed, nil):
finalize(fetched, failed, nil)

// Evaluate alert rules (best-effort, never fails sync)
if fetched > 0 {
    if err := alerts.EvaluateAll(ctx, db); err != nil {
        slog.Error("sync: alert evaluation failed", "err", err)
    }
}

slog.Info("sync: complete", "fetched", fetched, "failed", failed)
return restored, nil
```

### Pattern 4: AES-256-GCM Encryption for SMTP Password
**What:** SMTP password is encrypted at rest in the settings table using AES-256-GCM. The encryption key is derived from JWT_SECRET via SHA-256 hash (producing exactly 32 bytes for AES-256).
**When to use:** Saving and loading SMTP configuration.
**Example:**
```go
// internal/alerts/crypto.go
package alerts

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "io"
)

func DeriveKey(secret string) []byte {
    h := sha256.Sum256([]byte(secret))
    return h[:]
}

func Encrypt(plaintext string, key []byte) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(encoded string, key []byte) (string, error) {
    data, err := base64.StdEncoding.DecodeString(encoded)
    if err != nil {
        return "", err
    }
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    if len(data) < gcm.NonceSize() {
        return "", fmt.Errorf("ciphertext too short")
    }
    nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }
    return string(plaintext), nil
}
```

### Pattern 5: Expression Environment from Current Balances
**What:** Build an expr-lang environment struct from current account balances and bucket totals. Use the same aggregation logic as the summary endpoint.
**When to use:** Alert evaluation after sync.
**Example:**
```go
// Source: architecture research + expr-lang docs
type Environment struct {
    Liquid      float64            `expr:"liquid"`
    Savings     float64            `expr:"savings"`
    Investments float64            `expr:"investments"`
    NetWorth    float64            `expr:"net_worth"`
    Accounts    map[string]float64 `expr:"accounts"` // account ID -> balance
    Groups      map[string]float64 `expr:"groups"`   // group name -> summed balance
}

// BuildEnvironment queries current balances and constructs the evaluation environment.
func BuildEnvironment(ctx context.Context, db *sql.DB) (*Environment, error) {
    // Query latest balance per visible account, with panel type assignment
    // (same logic as summary endpoint: group panel_type > override > inferred)
    // Sum into bucket totals and populate account/group maps
}
```

### Anti-Patterns to Avoid
- **Storing raw expression strings from user input:** The user never types expressions. The frontend sends structured operand JSON, and the backend compiles it. This prevents injection and ensures editability.
- **Embedding alert logic in sync.go:** Keep sync.go thin. It calls `alerts.EvaluateAll()` as a single function call after sync completes.
- **Failing sync on alert errors:** Alert evaluation and email sending are best-effort. Log errors but never return them from the sync flow.
- **Sending emails synchronously in the evaluation loop:** Use a context deadline (10s) on email sending to prevent slow SMTP servers from blocking the sync scheduler.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Expression evaluation | Custom parser/evaluator for arithmetic + comparison | expr-lang/expr with typed environment | Handles operator precedence, type safety, sandboxing, and is battle-tested at Google/Uber scale |
| SMTP delivery | Raw net/smtp client | wneessen/go-mail | Handles TLS negotiation, auth discovery, connection pooling, and proper error handling |
| Encryption | Custom XOR or simple cipher | crypto/aes + crypto/cipher (AES-256-GCM) | Authenticated encryption prevents tampering; GCM mode handles nonce/padding correctly |
| State machine | Ad-hoc if/else chains in evaluation loop | Explicit state transition function | Prevents missed transitions, makes behavior testable and verifiable |

**Key insight:** The expression evaluation and email delivery are deceptively complex. expr-lang handles edge cases (division by zero, type coercion, sandboxing) that would take hundreds of lines to hand-roll. go-mail handles SMTP quirks (STARTTLS negotiation, auth method selection, connection timeouts) that net/smtp cannot.

## Common Pitfalls

### Pitfall 1: Expression-Builder-to-Evaluation Mismatch
**What goes wrong:** The frontend operand model does not map cleanly to expr-lang expressions, causing evaluation failures at runtime.
**Why it happens:** The frontend stores structured operands but the backend needs to compile them into arithmetic expressions with proper field references.
**How to avoid:** Define a single `CompileOperands(operands []Operand, comparison string, threshold string) (string, error)` function that deterministically converts operand JSON to an expr string. Validate at write time with `expr.Compile()`. Store the compiled expression alongside the operands so evaluation never re-compiles from operands.
**Warning signs:** Rules that save successfully but fail during post-sync evaluation.

### Pitfall 2: Duplicate Emails on Rapid Syncs
**What goes wrong:** Two syncs fire in quick succession, both see the same `last_state`, both send trigger emails.
**Why it happens:** The sync mutex prevents concurrent SyncOnce, but if alert evaluation is async or the state update is not atomic, a race can occur.
**How to avoid:** Run `EvaluateAll` synchronously within the sync mutex (it's already held). Update `last_state` in the same transaction as inserting `alert_history`. The existing `SetMaxOpenConns(1)` on SQLite ensures serialization.
**Warning signs:** Users receiving 2+ identical trigger emails.

### Pitfall 3: SMTP Password Encryption Key Loss
**What goes wrong:** JWT_SECRET changes (user rotates it), and all encrypted SMTP passwords become undecryptable.
**Why it happens:** The AES key is derived from JWT_SECRET. If the secret changes, the derived key changes.
**How to avoid:** Document that changing JWT_SECRET invalidates stored SMTP passwords. On decryption failure, log a warning and prompt the user to re-enter SMTP credentials. Consider displaying a clear error in the Settings UI.
**Warning signs:** "Test Email" suddenly fails after env var changes.

### Pitfall 4: Account Reference Staleness
**What goes wrong:** An alert rule references an account by ID, but the account is later soft-deleted. The expression evaluation cannot find the account balance.
**Why it happens:** Accounts can disappear from SimpleFIN and get soft-deleted.
**How to avoid:** When building the evaluation environment, include all accounts (even hidden ones) in the account map. This ensures existing rules continue to evaluate. Alternatively, treat missing accounts as having a $0 balance and log a warning.
**Warning signs:** Alert rules suddenly trigger/recover after an account disappears.

### Pitfall 5: Float Precision in Comparisons
**What goes wrong:** Expression evaluates `liquid < 5000` but liquid is `4999.9999999` due to float64 arithmetic, causing unexpected triggers.
**Why it happens:** The project uses shopspring/decimal for storage, but expr-lang operates on float64.
**How to avoid:** Convert decimal strings to float64 for the expr environment (acceptable for threshold comparison -- exact penny precision is not critical for alerts). Round to 2 decimal places when displaying in emails. Document that alert thresholds are approximate.
**Warning signs:** Rules triggering at seemingly incorrect values.

### Pitfall 6: SMTP Timeout Blocking Sync
**What goes wrong:** Slow or unreachable SMTP server causes email send to hang, blocking the sync scheduler.
**Why it happens:** go-mail's `DialAndSend` blocks until complete or error.
**How to avoid:** Wrap email sending with a context timeout (10 seconds). Use `go-mail`'s context-aware send methods.
**Warning signs:** Sync taking minutes instead of seconds when SMTP server is down.

## Code Examples

### Alert Rule CRUD Handler Pattern
```go
// Source: established project pattern (groups_test.go, handlers/groups.go)
// internal/api/handlers/alerts.go

type createAlertRequest struct {
    Name             string    `json:"name"`
    Operands         json.RawMessage `json:"operands"`      // JSON array of operand objects
    Comparison       string    `json:"comparison"`           // "<", "<=", ">", ">=", "=="
    Threshold        string    `json:"threshold"`            // Decimal string
    NotifyOnRecovery bool      `json:"notify_on_recovery"`
}

func CreateAlert(database *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req createAlertRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
            return
        }
        // Validate name
        if strings.TrimSpace(req.Name) == "" {
            http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
            return
        }
        // Validate operands and compile expression
        expression, err := alerts.CompileOperands(req.Operands, req.Comparison, req.Threshold)
        if err != nil {
            http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
            return
        }
        // Insert into database
        res, err := database.ExecContext(r.Context(), `
            INSERT INTO alert_rules(name, operands, expression, comparison, threshold, notify_on_recovery)
            VALUES(?, ?, ?, ?, ?, ?)
        `, req.Name, string(req.Operands), expression, req.Comparison, req.Threshold,
           boolToInt(req.NotifyOnRecovery))
        // ... return created rule as JSON
    }
}
```

### go-mail SMTP Send Pattern
```go
// Source: go-mail wiki (https://github.com/wneessen/go-mail/wiki/Getting-started)
// internal/alerts/notifier.go

func SendAlert(ctx context.Context, cfg SMTPConfig, rule AlertRule, state string, value string) error {
    msg := mail.NewMsg()
    if err := msg.From(cfg.FromAddress); err != nil {
        return fmt.Errorf("invalid from address: %w", err)
    }
    if err := msg.To(cfg.ToAddress); err != nil {
        return fmt.Errorf("invalid to address: %w", err)
    }

    // Subject format per CONTEXT.md
    subject := fmt.Sprintf("[Finance Alert] %s", rule.Name)
    if state == "recovered" {
        subject += " -- recovered"
    }
    msg.Subject(subject)

    // Plain text body with account breakdown
    body := formatAlertBody(rule, state, value)
    msg.SetBodyString(mail.TypeTextPlain, body)

    // Create client with timeout context
    client, err := mail.NewClient(cfg.Host,
        mail.WithPort(cfg.Port),
        mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
        mail.WithUsername(cfg.Username),
        mail.WithPassword(cfg.Password),
    )
    if err != nil {
        return fmt.Errorf("create SMTP client: %w", err)
    }

    sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    if err := client.DialAndSendWithContext(sendCtx, msg); err != nil {
        return fmt.Errorf("send email: %w", err)
    }
    return nil
}
```

### expr-lang Evaluation Pattern
```go
// Source: expr-lang.org/docs/getting-started, expr-lang.org/docs/environment
// internal/alerts/evaluator.go

type Environment struct {
    Liquid      float64            `expr:"liquid"`
    Savings     float64            `expr:"savings"`
    Investments float64            `expr:"investments"`
    NetWorth    float64            `expr:"net_worth"`
    Accounts    map[string]float64 `expr:"accounts"`
    Groups      map[string]float64 `expr:"groups"`
}

// Validate compiles an expression against the environment to check syntax.
func Validate(expression string) error {
    _, err := expr.Compile(expression, expr.Env(Environment{}), expr.AsBool())
    return err
}

// Evaluate runs a compiled expression against a populated environment.
func Evaluate(expression string, env Environment) (bool, error) {
    program, err := expr.Compile(expression, expr.Env(env), expr.AsBool())
    if err != nil {
        return false, fmt.Errorf("compile: %w", err)
    }
    result, err := expr.Run(program, env)
    if err != nil {
        return false, fmt.Errorf("run: %w", err)
    }
    return result.(bool), nil
}
```

### Database Schema
```sql
-- 000004_alert_rules.up.sql
CREATE TABLE IF NOT EXISTS alert_rules (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    name                TEXT NOT NULL,
    operands            TEXT NOT NULL,         -- JSON array of operand objects
    expression          TEXT NOT NULL,         -- Compiled expr-lang expression string
    comparison          TEXT NOT NULL CHECK(comparison IN ('<', '<=', '>', '>=', '==')),
    threshold           TEXT NOT NULL,         -- Decimal string
    notify_on_recovery  INTEGER NOT NULL DEFAULT 1,
    enabled             INTEGER NOT NULL DEFAULT 1,
    last_state          TEXT NOT NULL DEFAULT 'normal' CHECK(last_state IN ('normal', 'triggered', 'recovered')),
    last_eval_at        DATETIME,
    last_value          TEXT,                  -- Last computed value for display
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS alert_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id     INTEGER NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    state       TEXT NOT NULL CHECK(state IN ('triggered', 'recovered')),
    value       TEXT,                          -- Computed value at event time
    notified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_alert_history_rule
    ON alert_history(rule_id, notified_at DESC);

-- 000004_alert_rules.down.sql
DROP INDEX IF EXISTS idx_alert_history_rule;
DROP TABLE IF EXISTS alert_history;
DROP TABLE IF EXISTS alert_rules;
```

### Frontend Expression Builder Model
```typescript
// frontend/src/components/AlertRuleForm.tsx

interface Operand {
  id: string            // unique key for React list rendering
  type: 'bucket' | 'group' | 'account'
  ref: string           // bucket key, group ID, or account ID
  label: string         // Display name for the operand
  operator: '+' | '-'   // First operand always '+'
}

interface AlertRuleFormData {
  name: string
  operands: Operand[]
  comparison: '<' | '<=' | '>' | '>=' | '=='
  threshold: string
  notify_on_recovery: boolean
}

// Operand dropdown options derived from:
// 1. Buckets: { type: 'bucket', ref: 'liquid', label: 'Liquid Balance' }
// 2. Groups: fetched from GET /api/groups -> { type: 'group', ref: group.id, label: group.name }
// 3. Accounts: fetched from GET /api/accounts -> { type: 'account', ref: account.id, label: account.name }
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| net/smtp for SMTP | wneessen/go-mail | Go 1.24 (net/smtp deprecated) | go-mail handles TLS/auth automatically |
| antonmedv/expr | expr-lang/expr | 2024 (namespace change) | Same library, new import path |
| Raw expression strings | Structured operand JSON | This phase design | Enables bidirectional builder, prevents injection |

**Deprecated/outdated:**
- `net/smtp`: Deprecated since Go 1.24. Use wneessen/go-mail instead.
- `github.com/antonmedv/expr`: Old import path. Use `github.com/expr-lang/expr` instead.

## Open Questions

1. **Account group references in expressions**
   - What we know: Groups exist in `account_groups` table with `panel_type` and members. The expression builder should allow selecting groups as operands.
   - What's unclear: Should a group operand reference the group by its integer ID or name? IDs are stable across renames; names are human-readable in the expression.
   - Recommendation: Reference by integer ID internally (stable), display by name in the UI and emails. Store `ref` as string-serialized ID.

2. **Expression recompilation on account changes**
   - What we know: If an account is renamed or a group's membership changes, stored expressions referencing account IDs remain valid (IDs don't change).
   - What's unclear: What if a group is deleted? Rules referencing it would fail evaluation.
   - Recommendation: On group deletion, log a warning. During evaluation, treat missing group/account references as $0 and mark the rule as having a warning state. Do not prevent group deletion.

3. **Recovered state transition without recovery email**
   - What we know: `notify_on_recovery` is a per-rule flag. When false and value recovers, we skip the email.
   - What's unclear: Should the state go directly to "normal" (skipping "recovered") when recovery notification is disabled?
   - Recommendation: Yes -- if `notify_on_recovery` is false, transition directly from "triggered" to "normal" on recovery, skipping the "recovered" state entirely. This keeps the state machine clean.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | Go testing + httptest (standard) |
| Framework (Frontend) | Vitest 3.2.1 + @testing-library/react 16.3.2 |
| Config file (Go) | None needed -- `go test ./...` |
| Config file (Frontend) | `frontend/vitest.config.ts` |
| Quick run command (Go) | `go test ./internal/alerts/... ./internal/api/handlers/... -run Alert -count=1` |
| Quick run command (Frontend) | `cd frontend && npx vitest run --reporter=verbose src/pages/Alerts.test.tsx src/components/AlertRule*.test.tsx` |
| Full suite command | `go test ./... -count=1 && cd frontend && npx vitest run` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ALERT-01 | Expression builder creates valid operand JSON | unit (frontend) | `cd frontend && npx vitest run src/components/AlertRuleForm.test.tsx` | Wave 0 |
| ALERT-01 | CompileOperands produces valid expr-lang expression | unit (Go) | `go test ./internal/alerts/... -run TestCompileOperands -count=1` | Wave 0 |
| ALERT-02 | Comparison operators produce correct boolean results | unit (Go) | `go test ./internal/alerts/... -run TestEvaluate -count=1` | Wave 0 |
| ALERT-03 | State machine transitions correctly | unit (Go) | `go test ./internal/alerts/... -run TestStateMachine -count=1` | Wave 0 |
| ALERT-03 | EvaluateAll fires email on transition, suppresses duplicates | unit (Go) | `go test ./internal/alerts/... -run TestEvaluateAll -count=1` | Wave 0 |
| ALERT-04 | Email body contains required fields | unit (Go) | `go test ./internal/alerts/... -run TestFormatAlertBody -count=1` | Wave 0 |
| ALERT-05 | SMTP config saved/loaded with encrypted password | unit (Go) | `go test ./internal/alerts/... -run TestCrypto -count=1` | Wave 0 |
| ALERT-05 | Settings UI renders SMTP fields | unit (frontend) | `cd frontend && npx vitest run src/pages/Settings.test.tsx` | Exists (extend) |
| ALERT-06 | Test email endpoint sends using provided config | unit (Go) | `go test ./internal/api/handlers/... -run TestTestEmail -count=1` | Wave 0 |
| ALERT-07 | CRUD operations for alert rules | unit (Go) | `go test ./internal/api/handlers/... -run TestAlert -count=1` | Wave 0 |
| ALERT-07 | Alerts page renders rule list, empty state | unit (frontend) | `cd frontend && npx vitest run src/pages/Alerts.test.tsx` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/alerts/... -count=1` (Go) + relevant frontend test
- **Per wave merge:** `go test ./... -count=1 && cd frontend && npx vitest run`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/alerts/evaluator_test.go` -- covers ALERT-01, ALERT-02 (expression compilation and evaluation)
- [ ] `internal/alerts/engine_test.go` -- covers ALERT-03 (state machine transitions)
- [ ] `internal/alerts/notifier_test.go` -- covers ALERT-04 (email body formatting)
- [ ] `internal/alerts/crypto_test.go` -- covers ALERT-05 (encryption round-trip)
- [ ] `internal/api/handlers/alerts_test.go` -- covers ALERT-07 (CRUD endpoints)
- [ ] `internal/api/handlers/email_test.go` -- covers ALERT-06 (test email endpoint)
- [ ] `frontend/src/pages/Alerts.test.tsx` -- covers ALERT-07 (alerts page rendering)
- [ ] `frontend/src/components/AlertRuleForm.test.tsx` -- covers ALERT-01 (builder form)
- [ ] `frontend/src/components/AlertRuleCard.test.tsx` -- covers ALERT-07 (rule card display)
- [ ] Migration `000004_alert_rules.up.sql` and `000004_alert_rules.down.sql`

## Sources

### Primary (HIGH confidence)
- [expr-lang/expr GitHub](https://github.com/expr-lang/expr) -- expression language API, environment struct pattern, AsBool() option
- [expr-lang.org/docs/getting-started](https://expr-lang.org/docs/getting-started) -- Compile/Run pattern, Env() option
- [expr-lang.org/docs/environment](https://expr-lang.org/docs/environment) -- Map and struct environment configuration, nested field access
- [wneessen/go-mail GitHub](https://github.com/wneessen/go-mail) -- SMTP client API, DialAndSend pattern
- [go-mail wiki: Getting started](https://github.com/wneessen/go-mail/wiki/Getting-started) -- NewMsg, NewClient, WithSMTPAuth, DialAndSendWithContext
- [pkg.go.dev/crypto/cipher](https://pkg.go.dev/crypto/cipher) -- AES-256-GCM authenticated encryption
- `.planning/research/ARCHITECTURE.md` -- Feature 6 schema, file structure, patterns, dependency graph
- `.planning/phases/08-alert-system/08-CONTEXT.md` -- All user decisions, UX specifications, email format

### Secondary (MEDIUM confidence)
- `npm view react-querybuilder version` -- v8.14.0 (confirmed NOT needed per user decision for custom builder)
- `go list -m -versions github.com/expr-lang/expr` -- v1.17.8 confirmed as latest
- `go list -m -versions github.com/wneessen/go-mail` -- v0.7.2 confirmed as latest

### Tertiary (LOW confidence)
- None -- all critical claims verified against primary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- expr-lang/expr and go-mail are well-documented, actively maintained, and verified against official sources
- Architecture: HIGH -- follows established project patterns (thin handlers, dedicated packages, settings key-value, chi router), schema design from architecture research
- Pitfalls: HIGH -- state machine race conditions, SMTP timeout, float precision are well-understood problems with clear solutions
- Expression builder UX: MEDIUM -- custom builder implementation details not verified against prior art; straightforward but needs careful frontend work

**Research date:** 2026-03-16
**Valid until:** 2026-04-16 (30 days -- stable libraries, no fast-moving dependencies)
