package alerts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
)

// NextState implements the 3-state alert machine.
// It returns the new state and whether a notification should be sent.
//
// Transition table:
//
//	current    | conditionMet | notifyRecovery | newState   | shouldNotify
//	normal     | true         | -              | triggered  | true
//	normal     | false        | -              | normal     | false
//	triggered  | true         | -              | triggered  | false
//	triggered  | false        | true           | recovered  | true
//	triggered  | false        | false          | normal     | false
//	recovered  | true         | -              | triggered  | true
//	recovered  | false        | -              | normal     | false
func NextState(current string, conditionMet bool, notifyRecovery bool) (newState string, shouldNotify bool) {
	switch current {
	case "normal":
		if conditionMet {
			return "triggered", true
		}
		return "normal", false
	case "triggered":
		if conditionMet {
			return "triggered", false
		}
		if notifyRecovery {
			return "recovered", true
		}
		return "normal", false
	case "recovered":
		if conditionMet {
			return "triggered", true
		}
		return "normal", false
	}
	return "normal", false
}

// alertRule holds the database representation of a rule during evaluation.
type alertRule struct {
	ID               int
	Name             string
	Operands         string
	Expression       string
	Comparison       string
	Threshold        string
	NotifyOnRecovery bool
	LastState        string
}

// EvaluateAll evaluates all enabled alert rules against current balances.
// It updates rule state and history, and sends email notifications on transitions.
// Individual rule errors are logged but never returned (best-effort).
// Returns an error only if BuildEnvironment fails.
func EvaluateAll(ctx context.Context, db *sql.DB, jwtSecret string) error {
	env, err := BuildEnvironment(ctx, db)
	if err != nil {
		return fmt.Errorf("build environment: %w", err)
	}

	rows, err := db.QueryContext(ctx, `
		SELECT id, name, operands, expression, comparison, threshold,
		       notify_on_recovery, last_state
		FROM alert_rules
		WHERE enabled = 1
	`)
	if err != nil {
		return fmt.Errorf("query rules: %w", err)
	}
	defer rows.Close()

	var rules []alertRule
	for rows.Next() {
		var r alertRule
		var notifyRecovery int
		if err := rows.Scan(&r.ID, &r.Name, &r.Operands, &r.Expression,
			&r.Comparison, &r.Threshold, &notifyRecovery, &r.LastState); err != nil {
			slog.Error("alerts: scan rule", "err", err)
			continue
		}
		r.NotifyOnRecovery = notifyRecovery == 1
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows error: %w", err)
	}

	for _, rule := range rules {
		evaluateRule(ctx, db, rule, env, jwtSecret)
	}

	return nil
}

// evaluateRule processes a single alert rule. Errors are logged, not returned.
func evaluateRule(ctx context.Context, db *sql.DB, rule alertRule, env *Environment, jwtSecret string) {
	conditionMet, err := Evaluate(rule.Expression, *env)
	if err != nil {
		slog.Error("alerts: evaluate expression", "rule_id", rule.ID, "rule", rule.Name, "err", err)
		return
	}

	newState, shouldNotify := NextState(rule.LastState, conditionMet, rule.NotifyOnRecovery)

	// Compute the current value of the expression's LHS for display.
	computedValue := computeLHSValue(rule, env)

	now := time.Now().UTC()

	if newState != rule.LastState {
		// State changed: update rule and optionally record history.
		_, err := db.ExecContext(ctx, `
			UPDATE alert_rules
			SET last_state = ?, last_eval_at = ?, last_value = ?, updated_at = ?
			WHERE id = ?
		`, newState, now, computedValue, now, rule.ID)
		if err != nil {
			slog.Error("alerts: update rule state", "rule_id", rule.ID, "err", err)
			return
		}

		if shouldNotify {
			_, err := db.ExecContext(ctx, `
				INSERT INTO alert_history (rule_id, state, value, notified_at)
				VALUES (?, ?, ?, ?)
			`, rule.ID, newState, computedValue, now)
			if err != nil {
				slog.Error("alerts: insert history", "rule_id", rule.ID, "err", err)
			}

			// Send email notification (best-effort with timeout).
			sendNotification(ctx, db, rule, newState, computedValue, env, jwtSecret, now)
		}
	} else {
		// No state change: just update eval timestamp and value.
		_, err := db.ExecContext(ctx, `
			UPDATE alert_rules
			SET last_eval_at = ?, last_value = ?, updated_at = ?
			WHERE id = ?
		`, now, computedValue, now, rule.ID)
		if err != nil {
			slog.Error("alerts: update eval time", "rule_id", rule.ID, "err", err)
		}
	}
}

// computeLHSValue computes the left-hand side value of the expression for display.
func computeLHSValue(rule alertRule, env *Environment) string {
	var operands []Operand
	if err := json.Unmarshal([]byte(rule.Operands), &operands); err != nil {
		return "0.00"
	}

	total := decimal.Zero
	for _, op := range operands {
		val := operandValue(op, env)
		amount := decimal.NewFromFloat(val)
		if op.Operator == "-" {
			total = total.Sub(amount)
		} else {
			total = total.Add(amount)
		}
	}
	return total.StringFixed(2)
}

// operandValue resolves an operand to its float64 value from the environment.
func operandValue(op Operand, env *Environment) float64 {
	switch op.Type {
	case "bucket":
		switch op.Ref {
		case "liquid":
			return env.Liquid
		case "savings":
			return env.Savings
		case "investments":
			return env.Investments
		case "net_worth":
			return env.NetWorth
		}
	case "account":
		return env.Accounts[op.Ref]
	case "group":
		return env.Groups[op.Ref]
	}
	return 0
}

// sendNotification loads SMTP config and sends an alert email.
// Failures are logged but never propagated.
func sendNotification(ctx context.Context, db *sql.DB, rule alertRule, state, computedValue string, env *Environment, jwtSecret string, ts time.Time) {
	emailCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cfg, err := LoadSMTPConfig(emailCtx, db, jwtSecret)
	if err != nil {
		slog.Error("alerts: load smtp config", "rule_id", rule.ID, "err", err)
		return
	}
	if cfg == nil {
		slog.Debug("alerts: smtp not configured, skipping notification", "rule_id", rule.ID)
		return
	}

	var operands []Operand
	_ = json.Unmarshal([]byte(rule.Operands), &operands)

	opValues := make(map[string]string)
	for _, op := range operands {
		val := operandValue(op, env)
		opValues[op.Ref] = decimal.NewFromFloat(val).StringFixed(2)
	}

	detail := AlertDetail{
		RuleName:      rule.Name,
		Status:        state,
		ComputedValue: computedValue,
		Threshold:     rule.Threshold,
		Comparison:    rule.Comparison,
		Operands:      operands,
		OperandValues: opValues,
		Timestamp:     ts,
	}

	if err := SendAlert(emailCtx, *cfg, detail); err != nil {
		slog.Error("alerts: send email", "rule_id", rule.ID, "err", err)
	}
}
