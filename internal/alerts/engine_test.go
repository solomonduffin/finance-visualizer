package alerts

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/solomon/finance-visualizer/internal/db"
)

func TestNextState_NormalToTriggered(t *testing.T) {
	newState, shouldNotify := NextState("normal", true, true)
	if newState != "triggered" {
		t.Errorf("newState: got %q, want %q", newState, "triggered")
	}
	if !shouldNotify {
		t.Error("shouldNotify: got false, want true")
	}
}

func TestNextState_NormalStaysNormal(t *testing.T) {
	newState, shouldNotify := NextState("normal", false, true)
	if newState != "normal" {
		t.Errorf("newState: got %q, want %q", newState, "normal")
	}
	if shouldNotify {
		t.Error("shouldNotify: got true, want false")
	}
}

func TestNextState_TriggeredStaysTriggered(t *testing.T) {
	newState, shouldNotify := NextState("triggered", true, true)
	if newState != "triggered" {
		t.Errorf("newState: got %q, want %q", newState, "triggered")
	}
	if shouldNotify {
		t.Error("shouldNotify: got true, want false")
	}
}

func TestNextState_TriggeredToRecovered(t *testing.T) {
	newState, shouldNotify := NextState("triggered", false, true)
	if newState != "recovered" {
		t.Errorf("newState: got %q, want %q", newState, "recovered")
	}
	if !shouldNotify {
		t.Error("shouldNotify: got false, want true")
	}
}

func TestNextState_TriggeredToNormalSilent(t *testing.T) {
	newState, shouldNotify := NextState("triggered", false, false)
	if newState != "normal" {
		t.Errorf("newState: got %q, want %q", newState, "normal")
	}
	if shouldNotify {
		t.Error("shouldNotify: got true, want false")
	}
}

func TestNextState_RecoveredToTriggered(t *testing.T) {
	newState, shouldNotify := NextState("recovered", true, true)
	if newState != "triggered" {
		t.Errorf("newState: got %q, want %q", newState, "triggered")
	}
	if !shouldNotify {
		t.Error("shouldNotify: got false, want true")
	}
}

func TestNextState_RecoveredToNormal(t *testing.T) {
	newState, shouldNotify := NextState("recovered", false, true)
	if newState != "normal" {
		t.Errorf("newState: got %q, want %q", newState, "normal")
	}
	if shouldNotify {
		t.Error("shouldNotify: got true, want false")
	}
}

// setupAlertTestDB creates a migrated test database.
func setupAlertTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test DB: %v", err)
	}
	if err := db.Migrate(dbPath); err != nil {
		t.Fatalf("migrate test DB: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestBuildEnvironment(t *testing.T) {
	database := setupAlertTestDB(t)

	// Seed accounts.
	for _, a := range []struct {
		id, name, acctType string
	}{
		{"chk1", "Checking", "checking"},
		{"sav1", "Savings", "savings"},
		{"inv1", "Investment", "investment"},
		{"crd1", "Credit Card", "credit"},
	} {
		_, err := database.Exec(
			`INSERT INTO accounts (id, name, account_type, currency, org_name) VALUES (?, ?, ?, 'USD', '')`,
			a.id, a.name, a.acctType,
		)
		if err != nil {
			t.Fatalf("seed account %s: %v", a.id, err)
		}
	}

	// Seed snapshots.
	for _, s := range []struct {
		accountID, balance string
	}{
		{"chk1", "1000.00"},
		{"sav1", "2000.00"},
		{"inv1", "5000.00"},
		{"crd1", "-300.00"},
	} {
		_, err := database.Exec(
			`INSERT INTO balance_snapshots (account_id, balance, balance_date) VALUES (?, ?, '2024-01-01')`,
			s.accountID, s.balance,
		)
		if err != nil {
			t.Fatalf("seed snapshot %s: %v", s.accountID, err)
		}
	}

	// Create a group with one member.
	_, err := database.Exec(`INSERT INTO account_groups (name, panel_type) VALUES ('My Group', 'savings')`)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	_, err = database.Exec(`INSERT INTO group_members (group_id, account_id) VALUES (1, 'sav1')`)
	if err != nil {
		t.Fatalf("add member: %v", err)
	}

	env, err := BuildEnvironment(context.Background(), database)
	if err != nil {
		t.Fatalf("BuildEnvironment: %v", err)
	}

	// Liquid = checking(1000) + credit(-300) = 700
	if env.Liquid != 700 {
		t.Errorf("Liquid: got %v, want 700", env.Liquid)
	}
	// Savings = 2000 (sav1 is in a savings group, group panel_type takes precedence)
	if env.Savings != 2000 {
		t.Errorf("Savings: got %v, want 2000", env.Savings)
	}
	// Investments = 5000
	if env.Investments != 5000 {
		t.Errorf("Investments: got %v, want 5000", env.Investments)
	}
	// NetWorth = 700 + 2000 + 5000 = 7700
	if env.NetWorth != 7700 {
		t.Errorf("NetWorth: got %v, want 7700", env.NetWorth)
	}

	// Accounts map should contain all 4 accounts.
	if len(env.Accounts) != 4 {
		t.Errorf("Accounts map length: got %d, want 4", len(env.Accounts))
	}
	if env.Accounts["chk1"] != 1000 {
		t.Errorf("Accounts[chk1]: got %v, want 1000", env.Accounts["chk1"])
	}
	if env.Accounts["crd1"] != -300 {
		t.Errorf("Accounts[crd1]: got %v, want -300", env.Accounts["crd1"])
	}

	// Groups map should contain group "1" with value 2000 (sav1's balance).
	if env.Groups["1"] != 2000 {
		t.Errorf("Groups[1]: got %v, want 2000", env.Groups["1"])
	}
}

func TestEvaluateAll(t *testing.T) {
	database := setupAlertTestDB(t)

	// Seed accounts and snapshots.
	_, err := database.Exec(
		`INSERT INTO accounts (id, name, account_type, currency, org_name) VALUES ('chk1', 'Checking', 'checking', 'USD', '')`,
	)
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}
	_, err = database.Exec(
		`INSERT INTO balance_snapshots (account_id, balance, balance_date) VALUES ('chk1', '4000.00', '2024-01-01')`,
	)
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}

	// Create an alert rule: liquid < 5000 (should trigger since liquid=4000).
	operands, _ := json.Marshal([]Operand{{Type: "bucket", Ref: "liquid", Label: "Liquid", Operator: "+"}})
	_, err = database.Exec(
		`INSERT INTO alert_rules (name, operands, expression, comparison, threshold, notify_on_recovery, enabled)
		 VALUES (?, ?, ?, ?, ?, 1, 1)`,
		"Low Liquid", string(operands), "liquid < 5000", "<", "5000",
	)
	if err != nil {
		t.Fatalf("insert rule: %v", err)
	}

	// Run EvaluateAll (no SMTP configured, so no email will be sent).
	err = EvaluateAll(context.Background(), database, "test-jwt-secret")
	if err != nil {
		t.Fatalf("EvaluateAll: %v", err)
	}

	// Check that the rule state changed to "triggered".
	var lastState, lastValue string
	err = database.QueryRow(`SELECT last_state, last_value FROM alert_rules WHERE id = 1`).Scan(&lastState, &lastValue)
	if err != nil {
		t.Fatalf("query rule state: %v", err)
	}
	if lastState != "triggered" {
		t.Errorf("last_state: got %q, want %q", lastState, "triggered")
	}
	if lastValue != "4000.00" {
		t.Errorf("last_value: got %q, want %q", lastValue, "4000.00")
	}

	// Check alert_history has a "triggered" entry.
	var histState string
	err = database.QueryRow(`SELECT state FROM alert_history WHERE rule_id = 1`).Scan(&histState)
	if err != nil {
		t.Fatalf("query history: %v", err)
	}
	if histState != "triggered" {
		t.Errorf("history state: got %q, want %q", histState, "triggered")
	}

	// Run EvaluateAll again -- should stay triggered (no new notification).
	err = EvaluateAll(context.Background(), database, "test-jwt-secret")
	if err != nil {
		t.Fatalf("EvaluateAll second run: %v", err)
	}

	var histCount int
	err = database.QueryRow(`SELECT COUNT(*) FROM alert_history WHERE rule_id = 1`).Scan(&histCount)
	if err != nil {
		t.Fatalf("count history: %v", err)
	}
	if histCount != 1 {
		t.Errorf("history count after second run: got %d, want 1 (no duplicate)", histCount)
	}
}
