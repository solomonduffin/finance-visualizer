package alerts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/shopspring/decimal"
)

// Operand represents a single term in an alert expression.
type Operand struct {
	Type     string `json:"type"`     // "bucket", "account", or "group"
	Ref      string `json:"ref"`      // bucket name, account ID, or group ID
	Label    string `json:"label"`    // human-readable display label
	Operator string `json:"operator"` // "+" or "-"
}

// Environment holds the evaluation context for alert expressions.
type Environment struct {
	Liquid      float64            `expr:"liquid"`
	Savings     float64            `expr:"savings"`
	Investments float64            `expr:"investments"`
	NetWorth    float64            `expr:"net_worth"`
	Accounts    map[string]float64 `expr:"accounts"`
	Groups      map[string]float64 `expr:"groups"`
}

// validComparisons lists the allowed comparison operators.
var validComparisons = map[string]bool{
	"<":  true,
	"<=": true,
	">":  true,
	">=": true,
	"==": true,
}

// CompileOperands compiles a JSON operand array, comparison, and threshold into
// a valid expr-lang expression string. The expression is validated against
// the Environment type to ensure it compiles correctly.
func CompileOperands(operandsJSON []byte, comparison string, threshold string) (string, error) {
	var operands []Operand
	if err := json.Unmarshal(operandsJSON, &operands); err != nil {
		return "", fmt.Errorf("invalid operands JSON: %w", err)
	}
	if len(operands) == 0 {
		return "", fmt.Errorf("at least one operand is required")
	}
	if !validComparisons[comparison] {
		return "", fmt.Errorf("invalid comparison operator: %q", comparison)
	}

	// Build the left-hand side of the expression.
	var terms []string
	for _, op := range operands {
		var ref string
		switch op.Type {
		case "bucket":
			ref = op.Ref // e.g., "liquid", "savings", "investments", "net_worth"
		case "account":
			ref = fmt.Sprintf(`accounts["%s"]`, op.Ref)
		case "group":
			ref = fmt.Sprintf(`groups["%s"]`, op.Ref)
		default:
			return "", fmt.Errorf("unknown operand type: %q", op.Type)
		}
		terms = append(terms, fmt.Sprintf("%s %s", op.Operator, ref))
	}

	// Join terms and clean up the leading operator for the first term.
	lhs := strings.Join(terms, " ")
	// Remove leading "+ " from the expression.
	lhs = strings.TrimPrefix(lhs, "+ ")
	lhs = strings.TrimPrefix(lhs, "- ")
	if operands[0].Operator == "-" {
		// If the first operand is subtracted, prefix with "-".
		lhs = "-" + strings.TrimPrefix(terms[0], "- ")
		if len(terms) > 1 {
			lhs = lhs + " " + strings.Join(terms[1:], " ")
		}
	}

	// Wrap in parentheses if multiple terms.
	if len(operands) > 1 {
		lhs = "(" + lhs + ")"
	}

	expression := fmt.Sprintf("%s %s %s", lhs, comparison, threshold)

	// Validate by compiling against the Environment type.
	_, err := expr.Compile(expression, expr.Env(Environment{}), expr.AsBool())
	if err != nil {
		return "", fmt.Errorf("expression compilation failed: %w", err)
	}

	return expression, nil
}

// Validate checks that an expression string compiles against the Environment type.
func Validate(expression string) error {
	_, err := expr.Compile(expression, expr.Env(Environment{}), expr.AsBool())
	return err
}

// Evaluate compiles and runs an expression against the given environment,
// returning a boolean result.
func Evaluate(expression string, env Environment) (bool, error) {
	program, err := expr.Compile(expression, expr.Env(Environment{}), expr.AsBool())
	if err != nil {
		return false, fmt.Errorf("compile: %w", err)
	}
	output, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("run: %w", err)
	}
	result, ok := output.(bool)
	if !ok {
		return false, fmt.Errorf("expression did not return bool, got %T", output)
	}
	return result, nil
}

// BuildEnvironment queries the database for current balances and constructs
// an evaluation Environment. It uses the same panel aggregation logic as the
// summary endpoint: group panel_type > account_type_override > account_type.
func BuildEnvironment(ctx context.Context, db *sql.DB) (*Environment, error) {
	env := &Environment{
		Accounts: make(map[string]float64),
		Groups:   make(map[string]float64),
	}

	// Query latest balance per visible account with effective panel type.
	rows, err := db.QueryContext(ctx, `
		SELECT a.id,
		       COALESCE(ag.panel_type, a.account_type_override, a.account_type) AS effective_type,
		       COALESCE(bs.balance, '0') AS balance
		FROM accounts a
		LEFT JOIN balance_snapshots bs ON bs.account_id = a.id
		  AND bs.balance_date = (
		      SELECT MAX(bs2.balance_date)
		      FROM balance_snapshots bs2
		      WHERE bs2.account_id = a.id
		  )
		LEFT JOIN group_members gm ON gm.account_id = a.id
		LEFT JOIN account_groups ag ON ag.id = gm.group_id
		WHERE a.hidden_at IS NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("query accounts: %w", err)
	}
	defer rows.Close()

	liquidDec := decimal.Zero
	savingsDec := decimal.Zero
	investmentsDec := decimal.Zero

	for rows.Next() {
		var accountID, effectiveType, balance string
		if err := rows.Scan(&accountID, &effectiveType, &balance); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}

		amount, err := decimal.NewFromString(balance)
		if err != nil {
			continue
		}

		// Populate per-account map.
		f, _ := amount.Float64()
		env.Accounts[accountID] = f

		// Sum into panel buckets.
		switch effectiveType {
		case "checking", "credit":
			liquidDec = liquidDec.Add(amount)
		case "savings":
			savingsDec = savingsDec.Add(amount)
		case "investment":
			investmentsDec = investmentsDec.Add(amount)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	// Query group totals (sum of member balances per group).
	groupRows, err := db.QueryContext(ctx, `
		SELECT ag.id,
		       COALESCE(SUM(
		           CAST(COALESCE(bs.balance, '0') AS REAL)
		       ), 0) AS total
		FROM account_groups ag
		LEFT JOIN group_members gm ON gm.group_id = ag.id
		LEFT JOIN accounts a ON a.id = gm.account_id
		LEFT JOIN balance_snapshots bs ON bs.account_id = a.id
		  AND bs.balance_date = (
		      SELECT MAX(bs2.balance_date)
		      FROM balance_snapshots bs2
		      WHERE bs2.account_id = a.id
		  )
		GROUP BY ag.id
	`)
	if err != nil {
		return nil, fmt.Errorf("query groups: %w", err)
	}
	defer groupRows.Close()

	for groupRows.Next() {
		var groupID int
		var total float64
		if err := groupRows.Scan(&groupID, &total); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		env.Groups[fmt.Sprintf("%d", groupID)] = total
	}
	if err := groupRows.Err(); err != nil {
		return nil, fmt.Errorf("group rows error: %w", err)
	}

	// Populate float64 bucket fields.
	env.Liquid, _ = liquidDec.Float64()
	env.Savings, _ = savingsDec.Float64()
	env.Investments, _ = investmentsDec.Float64()
	env.NetWorth = env.Liquid + env.Savings + env.Investments

	return env, nil
}
