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
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	OriginalName        string  `json:"original_name"`
	Type                string  `json:"type"`
	Balance             string  `json:"balance"`
	Currency            string  `json:"currency"`
	OrgName             string  `json:"org_name"`
	DisplayName         *string `json:"display_name"`
	HiddenAt            *string `json:"hidden_at"`
	AccountTypeOverride *string `json:"account_type_override"`
}

// groupItemJSON mirrors the groupItem struct in the accounts response.
type groupItemJSON struct {
	ID           int                   `json:"id"`
	Name         string                `json:"name"`
	PanelType    string                `json:"panel_type"`
	TotalBalance string                `json:"total_balance"`
	Members      []groupMemberItemJSON `json:"members"`
}

type groupMemberItemJSON struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	OriginalName        string  `json:"original_name"`
	Balance             string  `json:"balance"`
	Currency            string  `json:"currency"`
	OrgName             string  `json:"org_name"`
	DisplayName         *string `json:"display_name"`
	AccountTypeOverride *string `json:"account_type_override"`
}

// accountsResponseJSON mirrors the grouped response from GetAccounts.
type accountsResponseJSON struct {
	Liquid      []accountItemJSON `json:"liquid"`
	Savings     []accountItemJSON `json:"savings"`
	Investments []accountItemJSON `json:"investments"`
	Other       []accountItemJSON `json:"other"`
	Groups      []groupItemJSON   `json:"groups"`
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

func TestGetAccounts_DisplayName(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Chase Checking 1234", "account_type": "checking", "currency": "USD", "org_name": "Chase", "display_name": "Main Checking"},
		{"id": "chk2", "name": "Chase Checking 5678", "account_type": "checking", "currency": "USD", "org_name": "Chase"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "chk2", "balance": "500.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	var resp accountsResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 2 {
		t.Fatalf("expected 2 liquid accounts, got %d", len(resp.Liquid))
	}

	// Account with display_name set should return it as name
	var withDisplayName, withoutDisplayName *accountItemJSON
	for i := range resp.Liquid {
		if resp.Liquid[i].ID == "chk1" {
			withDisplayName = &resp.Liquid[i]
		}
		if resp.Liquid[i].ID == "chk2" {
			withoutDisplayName = &resp.Liquid[i]
		}
	}

	if withDisplayName == nil || withoutDisplayName == nil {
		t.Fatal("expected both accounts in response")
	}

	// COALESCE(display_name, name) should be "Main Checking"
	if withDisplayName.Name != "Main Checking" {
		t.Errorf("name with display_name set: got %q, want %q", withDisplayName.Name, "Main Checking")
	}
	// original_name should always be the raw SimpleFIN name
	if withDisplayName.OriginalName != "Chase Checking 1234" {
		t.Errorf("original_name: got %q, want %q", withDisplayName.OriginalName, "Chase Checking 1234")
	}
	// display_name field should be non-nil
	if withDisplayName.DisplayName == nil || *withDisplayName.DisplayName != "Main Checking" {
		t.Errorf("display_name field: got %v, want \"Main Checking\"", withDisplayName.DisplayName)
	}

	// Account without display_name: name should be original name
	if withoutDisplayName.Name != "Chase Checking 5678" {
		t.Errorf("name without display_name: got %q, want %q", withoutDisplayName.Name, "Chase Checking 5678")
	}
	// original_name should match name
	if withoutDisplayName.OriginalName != "Chase Checking 5678" {
		t.Errorf("original_name without display_name: got %q, want %q", withoutDisplayName.OriginalName, "Chase Checking 5678")
	}
	// display_name should be nil
	if withoutDisplayName.DisplayName != nil {
		t.Errorf("display_name field should be null, got %v", withoutDisplayName.DisplayName)
	}
}

func TestGetAccounts_HiddenExcluded(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Visible Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "chk2", "name": "Hidden Checking", "account_type": "checking", "currency": "USD", "org_name": "", "hidden_at": "2024-01-15T10:00:00Z"},
		{"id": "sav1", "name": "Visible Savings", "account_type": "savings", "currency": "USD", "org_name": ""},
		{"id": "sav2", "name": "Hidden Savings", "account_type": "savings", "currency": "USD", "org_name": "", "hidden_at": "2024-01-15T10:00:00Z"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "chk2", "balance": "500.00", "balance_date": "2024-01-01"},
		{"account_id": "sav1", "balance": "2000.00", "balance_date": "2024-01-01"},
		{"account_id": "sav2", "balance": "3000.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	var resp accountsResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Only visible accounts should appear
	if len(resp.Liquid) != 1 {
		t.Errorf("expected 1 visible liquid account, got %d", len(resp.Liquid))
	}
	if len(resp.Savings) != 1 {
		t.Errorf("expected 1 visible savings account, got %d", len(resp.Savings))
	}
	if len(resp.Liquid) == 1 && resp.Liquid[0].ID != "chk1" {
		t.Errorf("expected visible checking (chk1), got %q", resp.Liquid[0].ID)
	}
	if len(resp.Savings) == 1 && resp.Savings[0].ID != "sav1" {
		t.Errorf("expected visible savings (sav1), got %q", resp.Savings[0].ID)
	}
}

func TestGetAccounts_TypeOverride(t *testing.T) {
	database := setupFinanceTestDB(t)

	// An account with account_type=checking but override=savings should appear in savings group
	seedAccounts(t, database, []map[string]string{
		{"id": "acct1", "name": "Misclassified Account", "account_type": "checking", "currency": "USD", "org_name": "", "account_type_override": "savings"},
		{"id": "acct2", "name": "Normal Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "acct1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "acct2", "balance": "500.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	var resp accountsResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// acct1 should appear in savings (override), acct2 in liquid
	if len(resp.Savings) != 1 {
		t.Errorf("expected 1 savings account (overridden), got %d", len(resp.Savings))
	}
	if len(resp.Liquid) != 1 {
		t.Errorf("expected 1 liquid account, got %d", len(resp.Liquid))
	}
	if len(resp.Savings) == 1 && resp.Savings[0].ID != "acct1" {
		t.Errorf("expected overridden account in savings, got %q", resp.Savings[0].ID)
	}
	if len(resp.Savings) == 1 {
		// account_type_override should be set
		if resp.Savings[0].AccountTypeOverride == nil || *resp.Savings[0].AccountTypeOverride != "savings" {
			t.Errorf("account_type_override should be \"savings\", got %v", resp.Savings[0].AccountTypeOverride)
		}
	}
}

func TestGetAccounts_IncludeHidden(t *testing.T) {
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

	// Without include_hidden: hidden accounts excluded
	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	var resp accountsResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Liquid) != 1 {
		t.Errorf("without include_hidden: expected 1 liquid, got %d", len(resp.Liquid))
	}

	// With include_hidden=true: all accounts included
	req2 := httptest.NewRequest(http.MethodGet, "/api/accounts?include_hidden=true", nil)
	w2 := httptest.NewRecorder()
	handlers.GetAccounts(database)(w2, req2)

	var resp2 accountsResponseJSON
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp2.Liquid) != 2 {
		t.Errorf("with include_hidden: expected 2 liquid, got %d", len(resp2.Liquid))
	}
	// Hidden account should have hidden_at set
	var hiddenFound bool
	for _, a := range resp2.Liquid {
		if a.ID == "chk2" && a.HiddenAt != nil {
			hiddenFound = true
		}
	}
	if !hiddenFound {
		t.Error("hidden account chk2 not found or missing hidden_at")
	}
}

func TestGetAccounts_GroupsArrayPresent(t *testing.T) {
	database := setupFinanceTestDB(t)

	// No groups, no accounts — groups should be empty array not null
	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &rawMap); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if string(rawMap["groups"]) != "[]" {
		t.Errorf("groups: got %s, want [] (empty array, not null)", rawMap["groups"])
	}
}

func TestGetAccounts_GroupedAccountExcludedFromPanels(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "acct1", "name": "Coinbase BTC", "account_type": "investment", "currency": "USD", "org_name": "Coinbase"},
		{"id": "acct2", "name": "Fidelity 401k", "account_type": "investment", "currency": "USD", "org_name": "Fidelity"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "acct1", "balance": "5000.00", "balance_date": "2024-01-01"},
		{"account_id": "acct2", "balance": "3000.00", "balance_date": "2024-01-01"},
	})

	// Create group and add acct1
	_, err := database.Exec(`INSERT INTO account_groups (name, panel_type) VALUES ('Coinbase', 'investment')`)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	_, err = database.Exec(`INSERT INTO group_members (group_id, account_id) VALUES (1, 'acct1')`)
	if err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	var resp accountsResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// acct1 should NOT be in investments panel (it's in a group)
	for _, inv := range resp.Investments {
		if inv.ID == "acct1" {
			t.Error("grouped account acct1 should NOT appear in investments panel")
		}
	}

	// acct2 should still be in investments panel
	found := false
	for _, inv := range resp.Investments {
		if inv.ID == "acct2" {
			found = true
		}
	}
	if !found {
		t.Error("standalone account acct2 should appear in investments panel")
	}

	// acct1 should be inside a group in the groups array
	if len(resp.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(resp.Groups))
	}
	if resp.Groups[0].Name != "Coinbase" {
		t.Errorf("group name: got %q, want %q", resp.Groups[0].Name, "Coinbase")
	}
	if len(resp.Groups[0].Members) != 1 {
		t.Fatalf("expected 1 member in group, got %d", len(resp.Groups[0].Members))
	}
	if resp.Groups[0].Members[0].ID != "acct1" {
		t.Errorf("member id: got %q, want %q", resp.Groups[0].Members[0].ID, "acct1")
	}
}

func TestGetAccounts_GroupTotalBalance(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "cb1", "name": "Coinbase BTC", "account_type": "investment", "currency": "USD", "org_name": "Coinbase"},
		{"id": "cb2", "name": "Coinbase ETH", "account_type": "investment", "currency": "USD", "org_name": "Coinbase"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "cb1", "balance": "5000.00", "balance_date": "2024-01-01"},
		{"account_id": "cb2", "balance": "3000.00", "balance_date": "2024-01-01"},
	})

	_, err := database.Exec(`INSERT INTO account_groups (name, panel_type) VALUES ('Coinbase', 'investment')`)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	database.Exec(`INSERT INTO group_members (group_id, account_id) VALUES (1, 'cb1')`)
	database.Exec(`INSERT INTO group_members (group_id, account_id) VALUES (1, 'cb2')`)

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	var resp accountsResponseJSON
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(resp.Groups))
	}
	// 5000 + 3000 = 8000
	if resp.Groups[0].TotalBalance != "8000.00" {
		t.Errorf("total_balance: got %q, want %q", resp.Groups[0].TotalBalance, "8000.00")
	}
}

func TestGetAccounts_GroupPanelTypeCheckingMapsToLiquid(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "BofA Checking", "account_type": "checking", "currency": "USD", "org_name": "BofA"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
	})

	_, err := database.Exec(`INSERT INTO account_groups (name, panel_type) VALUES ('BofA Bundle', 'checking')`)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	database.Exec(`INSERT INTO group_members (group_id, account_id) VALUES (1, 'chk1')`)

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	handlers.GetAccounts(database)(w, req)

	var resp accountsResponseJSON
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(resp.Groups))
	}
	if resp.Groups[0].PanelType != "checking" {
		t.Errorf("group panel_type: got %q, want %q", resp.Groups[0].PanelType, "checking")
	}
	// chk1 should NOT be in liquid panel since it's grouped
	if len(resp.Liquid) != 0 {
		t.Errorf("expected 0 liquid accounts (grouped), got %d", len(resp.Liquid))
	}
}
