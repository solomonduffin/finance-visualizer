package handlers_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/solomon/finance-visualizer/internal/api/handlers"
	"github.com/solomon/finance-visualizer/internal/db"
)

// setupFinanceTestDB creates a temp file SQLite DB with schema migrated, no password required.
func setupFinanceTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err := db.Migrate(dbPath); err != nil {
		t.Fatalf("failed to migrate test DB: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// seedAccounts inserts accounts into the test DB.
// accounts is a slice of maps with keys: id, name, account_type, currency, org_name.
// Optional keys: display_name, hidden_at, account_type_override.
func seedAccounts(t *testing.T, database *sql.DB, accounts []map[string]string) {
	t.Helper()
	for _, a := range accounts {
		orgName := a["org_name"]
		_, err := database.Exec(
			`INSERT INTO accounts (id, name, account_type, currency, org_name)
			 VALUES (?, ?, ?, ?, ?)`,
			a["id"], a["name"], a["account_type"], a["currency"], orgName,
		)
		if err != nil {
			t.Fatalf("seedAccounts: %v", err)
		}

		// Update optional metadata columns if provided.
		if dn, ok := a["display_name"]; ok {
			_, err := database.Exec(`UPDATE accounts SET display_name = ? WHERE id = ?`, dn, a["id"])
			if err != nil {
				t.Fatalf("seedAccounts display_name: %v", err)
			}
		}
		if ha, ok := a["hidden_at"]; ok {
			_, err := database.Exec(`UPDATE accounts SET hidden_at = ? WHERE id = ?`, ha, a["id"])
			if err != nil {
				t.Fatalf("seedAccounts hidden_at: %v", err)
			}
		}
		if ato, ok := a["account_type_override"]; ok {
			_, err := database.Exec(`UPDATE accounts SET account_type_override = ? WHERE id = ?`, ato, a["id"])
			if err != nil {
				t.Fatalf("seedAccounts account_type_override: %v", err)
			}
		}
	}
}

// seedSnapshots inserts balance snapshots into the test DB.
// snapshots is a slice of maps with keys: account_id, balance, balance_date.
func seedSnapshots(t *testing.T, database *sql.DB, snapshots []map[string]string) {
	t.Helper()
	for _, s := range snapshots {
		_, err := database.Exec(
			`INSERT INTO balance_snapshots (account_id, balance, balance_date)
			 VALUES (?, ?, ?)`,
			s["account_id"], s["balance"], s["balance_date"],
		)
		if err != nil {
			t.Fatalf("seedSnapshots: %v", err)
		}
	}
}

func TestGetSummary_AllTypes(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "sav1", "name": "Savings", "account_type": "savings", "currency": "USD", "org_name": ""},
		{"id": "crd1", "name": "Credit", "account_type": "credit", "currency": "USD", "org_name": ""},
		{"id": "inv1", "name": "Investment", "account_type": "investment", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "sav1", "balance": "500.00", "balance_date": "2024-01-01"},
		{"account_id": "crd1", "balance": "-200.00", "balance_date": "2024-01-01"},
		{"account_id": "inv1", "balance": "3000.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/summary", nil)
	w := httptest.NewRecorder()
	handlers.GetSummary(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// liquid = checking(1000) + credit(-200) = 800
	if string(resp["liquid"]) != `"800.00"` {
		t.Errorf("liquid: got %s, want \"800.00\"", resp["liquid"])
	}
	if string(resp["savings"]) != `"500.00"` {
		t.Errorf("savings: got %s, want \"500.00\"", resp["savings"])
	}
	if string(resp["investments"]) != `"3000.00"` {
		t.Errorf("investments: got %s, want \"3000.00\"", resp["investments"])
	}
}

func TestGetSummary_LiquidIsSumOfCheckingAndCredit(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking 1", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "chk2", "name": "Checking 2", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "crd1", "name": "Credit 1", "account_type": "credit", "currency": "USD", "org_name": ""},
		{"id": "crd2", "name": "Credit 2", "account_type": "credit", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "chk2", "balance": "500.00", "balance_date": "2024-01-01"},
		{"account_id": "crd1", "balance": "-300.00", "balance_date": "2024-01-01"},
		{"account_id": "crd2", "balance": "-100.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/summary", nil)
	w := httptest.NewRecorder()
	handlers.GetSummary(database)(w, req)

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// liquid = 1000 + 500 + (-300) + (-100) = 1100
	if string(resp["liquid"]) != `"1100.00"` {
		t.Errorf("liquid: got %s, want \"1100.00\"", resp["liquid"])
	}
}

func TestGetSummary_OtherTypeExcluded(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "oth1", "name": "Other", "account_type": "other", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "500.00", "balance_date": "2024-01-01"},
		{"account_id": "oth1", "balance": "999.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/summary", nil)
	w := httptest.NewRecorder()
	handlers.GetSummary(database)(w, req)

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// liquid should only be checking, "other" excluded
	if string(resp["liquid"]) != `"500.00"` {
		t.Errorf("liquid: got %s, want \"500.00\" (other type must be excluded)", resp["liquid"])
	}
	if string(resp["savings"]) != `"0.00"` {
		t.Errorf("savings: got %s, want \"0.00\"", resp["savings"])
	}
	if string(resp["investments"]) != `"0.00"` {
		t.Errorf("investments: got %s, want \"0.00\"", resp["investments"])
	}
}

func TestGetSummary_NoAccounts(t *testing.T) {
	database := setupFinanceTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/summary", nil)
	w := httptest.NewRecorder()
	handlers.GetSummary(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if string(resp["liquid"]) != `"0.00"` {
		t.Errorf("liquid: got %s, want \"0.00\"", resp["liquid"])
	}
	if string(resp["savings"]) != `"0.00"` {
		t.Errorf("savings: got %s, want \"0.00\"", resp["savings"])
	}
	if string(resp["investments"]) != `"0.00"` {
		t.Errorf("investments: got %s, want \"0.00\"", resp["investments"])
	}
}

func TestGetSummary_LastSyncedAt_Success(t *testing.T) {
	database := setupFinanceTestDB(t)

	// Insert a successful sync log entry (no error_text, has finished_at)
	_, err := database.Exec(
		`INSERT INTO sync_log (started_at, finished_at, accounts_fetched)
		 VALUES ('2024-01-01T10:00:00Z', '2024-01-01T10:00:05Z', 3)`,
	)
	if err != nil {
		t.Fatalf("failed to seed sync_log: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/summary", nil)
	w := httptest.NewRecorder()
	handlers.GetSummary(database)(w, req)

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if string(resp["last_synced_at"]) == "null" || resp["last_synced_at"] == nil {
		t.Error("last_synced_at should not be null when successful sync exists")
	}
}

func TestGetSummary_LastSyncedAt_Null(t *testing.T) {
	database := setupFinanceTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/summary", nil)
	w := httptest.NewRecorder()
	handlers.GetSummary(database)(w, req)

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if string(resp["last_synced_at"]) != "null" {
		t.Errorf("last_synced_at: got %s, want null", resp["last_synced_at"])
	}
}

func TestGetSummary_ExcludesHidden(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Visible Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "chk2", "name": "Hidden Checking", "account_type": "checking", "currency": "USD", "org_name": "", "hidden_at": "2024-01-15T10:00:00Z"},
		{"id": "sav1", "name": "Visible Savings", "account_type": "savings", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "chk2", "balance": "500.00", "balance_date": "2024-01-01"},
		{"account_id": "sav1", "balance": "2000.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/summary", nil)
	w := httptest.NewRecorder()
	handlers.GetSummary(database)(w, req)

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// liquid should only include visible checking (1000), not hidden (500)
	if string(resp["liquid"]) != `"1000.00"` {
		t.Errorf("liquid: got %s, want \"1000.00\" (hidden checking excluded)", resp["liquid"])
	}
	if string(resp["savings"]) != `"2000.00"` {
		t.Errorf("savings: got %s, want \"2000.00\"", resp["savings"])
	}
}

func TestGetSummary_TypeOverride(t *testing.T) {
	database := setupFinanceTestDB(t)

	// Account with account_type=checking but overridden to savings
	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Real Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "chk2", "name": "Overridden to Savings", "account_type": "checking", "currency": "USD", "org_name": "", "account_type_override": "savings"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "chk2", "balance": "500.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/summary", nil)
	w := httptest.NewRecorder()
	handlers.GetSummary(database)(w, req)

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// liquid should be 1000 (only real checking), savings should be 500 (overridden)
	if string(resp["liquid"]) != `"1000.00"` {
		t.Errorf("liquid: got %s, want \"1000.00\" (overridden account goes to savings)", resp["liquid"])
	}
	if string(resp["savings"]) != `"500.00"` {
		t.Errorf("savings: got %s, want \"500.00\" (overridden from checking)", resp["savings"])
	}
}

func TestGetSummary_BalancesAreStrings(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1234.56", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/summary", nil)
	w := httptest.NewRecorder()
	handlers.GetSummary(database)(w, req)

	raw := w.Body.Bytes()
	var resp map[string]json.RawMessage
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify liquid is a JSON string (starts with '"'), not a number
	liquid := resp["liquid"]
	if len(liquid) == 0 || liquid[0] != '"' {
		t.Errorf("liquid should be a JSON string, got %s", liquid)
	}
	savings := resp["savings"]
	if len(savings) == 0 || savings[0] != '"' {
		t.Errorf("savings should be a JSON string, got %s", savings)
	}
	investments := resp["investments"]
	if len(investments) == 0 || investments[0] != '"' {
		t.Errorf("investments should be a JSON string, got %s", investments)
	}
}
