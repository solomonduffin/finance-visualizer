package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/solomon/finance-visualizer/internal/api/handlers"
)

// accountItemJSON mirrors the JSON structure returned by GetAccounts.
type accountItemJSON struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Balance  string `json:"balance"`
	Currency string `json:"currency"`
	OrgName  string `json:"org_name"`
}

// accountsResponseJSON mirrors the grouped response from GetAccounts.
type accountsResponseJSON struct {
	Liquid      []accountItemJSON `json:"liquid"`
	Savings     []accountItemJSON `json:"savings"`
	Investments []accountItemJSON `json:"investments"`
	Other       []accountItemJSON `json:"other"`
}

func TestGetAccounts_GroupedByType(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Chase Checking", "account_type": "checking", "currency": "USD", "org_name": "Chase"},
		{"id": "sav1", "name": "Ally Savings", "account_type": "savings", "currency": "USD", "org_name": "Ally"},
		{"id": "crd1", "name": "Amex Credit", "account_type": "credit", "currency": "USD", "org_name": "Amex"},
		{"id": "inv1", "name": "Fidelity 401k", "account_type": "investment", "currency": "USD", "org_name": "Fidelity"},
		{"id": "oth1", "name": "Other Account", "account_type": "other", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "sav1", "balance": "500.00", "balance_date": "2024-01-01"},
		{"account_id": "crd1", "balance": "-200.00", "balance_date": "2024-01-01"},
		{"account_id": "inv1", "balance": "3000.00", "balance_date": "2024-01-01"},
		{"account_id": "oth1", "balance": "100.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp accountsResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 2 {
		t.Errorf("expected 2 liquid accounts (checking + credit), got %d", len(resp.Liquid))
	}
	if len(resp.Savings) != 1 {
		t.Errorf("expected 1 savings account, got %d", len(resp.Savings))
	}
	if len(resp.Investments) != 1 {
		t.Errorf("expected 1 investment account, got %d", len(resp.Investments))
	}
	if len(resp.Other) != 1 {
		t.Errorf("expected 1 other account, got %d", len(resp.Other))
	}
}

func TestGetAccounts_AccountFields(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Chase Checking", "account_type": "checking", "currency": "USD", "org_name": "Chase Bank"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1234.56", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	var resp accountsResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 1 {
		t.Fatalf("expected 1 liquid account, got %d", len(resp.Liquid))
	}
	a := resp.Liquid[0]
	if a.ID != "chk1" {
		t.Errorf("id: got %q, want %q", a.ID, "chk1")
	}
	if a.Name != "Chase Checking" {
		t.Errorf("name: got %q, want %q", a.Name, "Chase Checking")
	}
	if a.Type != "checking" {
		t.Errorf("type: got %q, want %q", a.Type, "checking")
	}
	if a.Balance != "1234.56" {
		t.Errorf("balance: got %q, want %q", a.Balance, "1234.56")
	}
	if a.Currency != "USD" {
		t.Errorf("currency: got %q, want %q", a.Currency, "USD")
	}
	if a.OrgName != "Chase Bank" {
		t.Errorf("org_name: got %q, want %q", a.OrgName, "Chase Bank")
	}
}

func TestGetAccounts_LatestBalance(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})
	// Insert multiple snapshots; latest date should win
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "100.00", "balance_date": "2024-01-01"},
		{"account_id": "chk1", "balance": "200.00", "balance_date": "2024-01-02"},
		{"account_id": "chk1", "balance": "999.99", "balance_date": "2024-01-03"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	var resp accountsResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 1 {
		t.Fatalf("expected 1 liquid account, got %d", len(resp.Liquid))
	}
	if resp.Liquid[0].Balance != "999.99" {
		t.Errorf("balance: got %q, want \"999.99\" (latest snapshot)", resp.Liquid[0].Balance)
	}
}

func TestGetAccounts_NoSnapshotDefaultsToZero(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})
	// No snapshots seeded for this account

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	var resp accountsResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 1 {
		t.Fatalf("expected 1 liquid account, got %d", len(resp.Liquid))
	}
	if resp.Liquid[0].Balance != "0" {
		t.Errorf("balance: got %q, want \"0\" for account with no snapshots", resp.Liquid[0].Balance)
	}
}

func TestGetAccounts_EmptyGroupsAreArraysNotNull(t *testing.T) {
	database := setupFinanceTestDB(t)

	// Seed only a checking account; other groups should be empty arrays
	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	raw := w.Body.Bytes()
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Empty groups must be "[]", not "null"
	for _, key := range []string{"savings", "investments", "other"} {
		if string(rawMap[key]) != "[]" {
			t.Errorf("%s: got %s, want [] (empty array, not null)", key, rawMap[key])
		}
	}
}

func TestGetAccounts_NoAccounts(t *testing.T) {
	database := setupFinanceTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp accountsResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 0 {
		t.Errorf("expected empty liquid, got %d accounts", len(resp.Liquid))
	}
	if len(resp.Savings) != 0 {
		t.Errorf("expected empty savings, got %d accounts", len(resp.Savings))
	}
	if len(resp.Investments) != 0 {
		t.Errorf("expected empty investments, got %d accounts", len(resp.Investments))
	}
	if len(resp.Other) != 0 {
		t.Errorf("expected empty other, got %d accounts", len(resp.Other))
	}
}

func TestGetAccounts_OrderedByNameWithinGroup(t *testing.T) {
	database := setupFinanceTestDB(t)

	// Seed checking accounts in reverse alphabetical order
	seedAccounts(t, database, []map[string]string{
		{"id": "chk3", "name": "Zeta Bank", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "chk1", "name": "Alpha Credit Union", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "chk2", "name": "Mid State Bank", "account_type": "checking", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "100.00", "balance_date": "2024-01-01"},
		{"account_id": "chk2", "balance": "200.00", "balance_date": "2024-01-01"},
		{"account_id": "chk3", "balance": "300.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	var resp accountsResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 3 {
		t.Fatalf("expected 3 liquid accounts, got %d", len(resp.Liquid))
	}
	names := []string{resp.Liquid[0].Name, resp.Liquid[1].Name, resp.Liquid[2].Name}
	expected := []string{"Alpha Credit Union", "Mid State Bank", "Zeta Bank"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("position %d: got %q, want %q", i, names[i], want)
		}
	}
}
