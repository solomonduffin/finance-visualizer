package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/solomon/finance-visualizer/internal/api/handlers"
)

// projectionSettingsJSON mirrors the JSON structure returned by GET /api/projections/settings.
type projectionSettingsJSON struct {
	Accounts []projectionAccountJSON `json:"accounts"`
	Income   projectionIncomeJSON    `json:"income"`
}

type projectionAccountJSON struct {
	AccountID   string                   `json:"account_id"`
	AccountName string                   `json:"account_name"`
	AccountType string                   `json:"account_type"`
	Balance     string                   `json:"balance"`
	APY         string                   `json:"apy"`
	Compound    bool                     `json:"compound"`
	Included    bool                     `json:"included"`
	Holdings    []projectionHoldingJSON  `json:"holdings"`
}

type projectionHoldingJSON struct {
	HoldingID   string `json:"holding_id"`
	Symbol      string `json:"symbol"`
	Description string `json:"description"`
	MarketValue string `json:"market_value"`
	APY         string `json:"apy"`
	Compound    bool   `json:"compound"`
	Included    bool   `json:"included"`
	Allocation  string `json:"allocation"`
}

type projectionIncomeJSON struct {
	Enabled          bool   `json:"enabled"`
	AnnualIncome     string `json:"annual_income"`
	MonthlySavingsPct string `json:"monthly_savings_pct"`
	AllocationJSON   string `json:"allocation_json"`
}

// TestGetProjectionSettings_Defaults verifies that accounts are returned with default projection settings
// when no projection_account_settings rows exist.
func TestGetProjectionSettings_Defaults(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": "Bank"},
		{"id": "sav1", "name": "Savings", "account_type": "savings", "currency": "USD", "org_name": "Bank"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-15"},
		{"account_id": "sav1", "balance": "5000.00", "balance_date": "2024-01-15"},
	})

	r := chi.NewRouter()
	r.Get("/api/projections/settings", handlers.GetProjectionSettings(database))

	req := httptest.NewRequest(http.MethodGet, "/api/projections/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp projectionSettingsJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(resp.Accounts))
	}

	// Check defaults: apy="0", compound=true, included=false
	for _, acct := range resp.Accounts {
		if acct.APY != "0" {
			t.Errorf("account %s: expected apy='0', got %q", acct.AccountID, acct.APY)
		}
		if !acct.Compound {
			t.Errorf("account %s: expected compound=true", acct.AccountID)
		}
		if acct.Included {
			t.Errorf("account %s: expected included=false", acct.AccountID)
		}
	}
}

// TestGetProjectionSettings_HoldingsNested verifies that investment accounts
// have their holdings nested in the response.
func TestGetProjectionSettings_HoldingsNested(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "inv1", "name": "Brokerage", "account_type": "investment", "currency": "USD", "org_name": "Vanguard"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "inv1", "balance": "50000.00", "balance_date": "2024-01-15"},
	})

	// Seed holdings for the investment account.
	_, err := database.Exec(
		`INSERT INTO holdings (id, account_id, symbol, description, market_value)
		 VALUES ('h1', 'inv1', 'VTSAX', 'Vanguard Total Stock', '30000.00'),
		        ('h2', 'inv1', 'VBTLX', 'Vanguard Total Bond', '20000.00')`,
	)
	if err != nil {
		t.Fatalf("seed holdings: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/api/projections/settings", handlers.GetProjectionSettings(database))

	req := httptest.NewRequest(http.MethodGet, "/api/projections/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp projectionSettingsJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(resp.Accounts))
	}

	inv := resp.Accounts[0]
	if len(inv.Holdings) != 2 {
		t.Fatalf("expected 2 holdings for investment account, got %d", len(inv.Holdings))
	}

	// Holdings should have default settings
	for _, h := range inv.Holdings {
		if h.APY != "0" {
			t.Errorf("holding %s: expected apy='0', got %q", h.HoldingID, h.APY)
		}
		if !h.Compound {
			t.Errorf("holding %s: expected compound=true", h.HoldingID)
		}
		if h.Included {
			t.Errorf("holding %s: expected included=false", h.HoldingID)
		}
		if h.Allocation != "0" {
			t.Errorf("holding %s: expected allocation='0', got %q", h.HoldingID, h.Allocation)
		}
	}
}

// TestGetProjectionSettings_InvestmentNoHoldings verifies investment accounts
// with no holdings data return an empty holdings array.
func TestGetProjectionSettings_InvestmentNoHoldings(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "inv1", "name": "Empty Brokerage", "account_type": "investment", "currency": "USD", "org_name": "Fidelity"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "inv1", "balance": "10000.00", "balance_date": "2024-01-15"},
	})

	r := chi.NewRouter()
	r.Get("/api/projections/settings", handlers.GetProjectionSettings(database))

	req := httptest.NewRequest(http.MethodGet, "/api/projections/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp projectionSettingsJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(resp.Accounts))
	}

	if resp.Accounts[0].Holdings == nil {
		t.Error("expected empty holdings array, got nil")
	}
	if len(resp.Accounts[0].Holdings) != 0 {
		t.Errorf("expected 0 holdings, got %d", len(resp.Accounts[0].Holdings))
	}
}

// TestGetProjectionSettings_IncomeDefaults verifies income settings defaults
// when no projection_income_settings row exists.
func TestGetProjectionSettings_IncomeDefaults(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Get("/api/projections/settings", handlers.GetProjectionSettings(database))

	req := httptest.NewRequest(http.MethodGet, "/api/projections/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp projectionSettingsJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Income.Enabled {
		t.Error("expected income enabled=false by default")
	}
	if resp.Income.AnnualIncome != "0" {
		t.Errorf("expected annual_income='0', got %q", resp.Income.AnnualIncome)
	}
	if resp.Income.MonthlySavingsPct != "0" {
		t.Errorf("expected monthly_savings_pct='0', got %q", resp.Income.MonthlySavingsPct)
	}
	if resp.Income.AllocationJSON != "{}" {
		t.Errorf("expected allocation_json='{}', got %q", resp.Income.AllocationJSON)
	}
}

// TestSaveProjectionSettings verifies PUT /api/projections/settings upserts
// account and holding settings.
func TestSaveProjectionSettings(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "sav1", "name": "Savings", "account_type": "savings", "currency": "USD", "org_name": "Bank"},
	})

	r := chi.NewRouter()
	r.Put("/api/projections/settings", handlers.SaveProjectionSettings(database))
	r.Get("/api/projections/settings", handlers.GetProjectionSettings(database))

	body := strings.NewReader(`{
		"accounts": [
			{"account_id": "sav1", "apy": "5.0", "compound": true, "included": true}
		],
		"holdings": []
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/projections/settings", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var okResp map[string]bool
	if err := json.Unmarshal(w.Body.Bytes(), &okResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !okResp["ok"] {
		t.Error("expected ok=true")
	}

	// Verify the settings were persisted by reading them back.
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "sav1", "balance": "5000.00", "balance_date": "2024-01-15"},
	})

	getReq := httptest.NewRequest(http.MethodGet, "/api/projections/settings", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	var settings projectionSettingsJSON
	if err := json.Unmarshal(getW.Body.Bytes(), &settings); err != nil {
		t.Fatalf("failed to parse GET response: %v", err)
	}

	if len(settings.Accounts) == 0 {
		t.Fatal("expected at least 1 account")
	}

	found := false
	for _, acct := range settings.Accounts {
		if acct.AccountID == "sav1" {
			found = true
			if acct.APY != "5.0" {
				t.Errorf("expected apy='5.0', got %q", acct.APY)
			}
			if !acct.Compound {
				t.Error("expected compound=true")
			}
			if !acct.Included {
				t.Error("expected included=true")
			}
		}
	}
	if !found {
		t.Error("sav1 not found in settings response")
	}
}

// TestSaveIncomeSettings verifies PUT /api/projections/income upserts
// the income settings singleton row.
func TestSaveIncomeSettings(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Put("/api/projections/income", handlers.SaveIncomeSettings(database))
	r.Get("/api/projections/settings", handlers.GetProjectionSettings(database))

	body := strings.NewReader(`{
		"enabled": true,
		"annual_income": "75000",
		"monthly_savings_pct": "20",
		"allocation_json": "{\"sav1\": 60, \"inv1\": 40}"
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/projections/income", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var okResp map[string]bool
	if err := json.Unmarshal(w.Body.Bytes(), &okResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !okResp["ok"] {
		t.Error("expected ok=true")
	}

	// Verify persisted by reading back
	getReq := httptest.NewRequest(http.MethodGet, "/api/projections/settings", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	var settings projectionSettingsJSON
	if err := json.Unmarshal(getW.Body.Bytes(), &settings); err != nil {
		t.Fatalf("failed to parse GET response: %v", err)
	}

	if !settings.Income.Enabled {
		t.Error("expected income enabled=true")
	}
	if settings.Income.AnnualIncome != "75000" {
		t.Errorf("expected annual_income='75000', got %q", settings.Income.AnnualIncome)
	}
	if settings.Income.MonthlySavingsPct != "20" {
		t.Errorf("expected monthly_savings_pct='20', got %q", settings.Income.MonthlySavingsPct)
	}
	if settings.Income.AllocationJSON != `{"sav1": 60, "inv1": 40}` {
		t.Errorf("expected allocation_json, got %q", settings.Income.AllocationJSON)
	}
}

// TestGetProjectionSettings_ExcludesHidden verifies hidden accounts
// are excluded from projection settings.
func TestGetProjectionSettings_ExcludesHidden(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Visible", "account_type": "checking", "currency": "USD", "org_name": "Bank"},
		{"id": "chk2", "name": "Hidden", "account_type": "checking", "currency": "USD", "org_name": "Bank", "hidden_at": "2024-01-01T00:00:00Z"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-15"},
		{"account_id": "chk2", "balance": "500.00", "balance_date": "2024-01-15"},
	})

	r := chi.NewRouter()
	r.Get("/api/projections/settings", handlers.GetProjectionSettings(database))

	req := httptest.NewRequest(http.MethodGet, "/api/projections/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp projectionSettingsJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Accounts) != 1 {
		t.Fatalf("expected 1 account (hidden excluded), got %d", len(resp.Accounts))
	}
	if resp.Accounts[0].AccountID != "chk1" {
		t.Errorf("expected chk1, got %s", resp.Accounts[0].AccountID)
	}
}

// ─── GetProjectionHistory tests ──────────────────────────────────────────────

type projectionHistoryJSON struct {
	Points []struct {
		Date  string `json:"date"`
		Value string `json:"value"`
	} `json:"points"`
}

// TestGetProjectionHistory_SumsOnlyIncludedAccounts verifies that only the requested
// account IDs are summed and accounts not in account_ids are excluded.
func TestGetProjectionHistory_SumsOnlyIncludedAccounts(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": "Bank"},
		{"id": "sav1", "name": "Savings", "account_type": "savings", "currency": "USD", "org_name": "Bank"},
		{"id": "crd1", "name": "Credit", "account_type": "credit", "currency": "USD", "org_name": "Bank"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "sav1", "balance": "5000.00", "balance_date": "2024-01-01"},
		{"account_id": "crd1", "balance": "-200.00", "balance_date": "2024-01-01"},
	})

	r := chi.NewRouter()
	r.Get("/api/projections/history", handlers.GetProjectionHistory(database))

	// Request only chk1 and sav1 — crd1 should be excluded.
	req := httptest.NewRequest(http.MethodGet, "/api/projections/history?days=0&account_ids=chk1,sav1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp projectionHistoryJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(resp.Points))
	}
	if resp.Points[0].Date != "2024-01-01" {
		t.Errorf("expected date 2024-01-01, got %s", resp.Points[0].Date)
	}
	// 1000.00 + 5000.00 = 6000.00 (not 5800.00 which would include credit)
	if resp.Points[0].Value != "6000.00" {
		t.Errorf("expected value 6000.00, got %s", resp.Points[0].Value)
	}
}

// TestGetProjectionHistory_LOCFCarryForward verifies that an account with no snapshot
// on a later date carries forward its most recent balance.
func TestGetProjectionHistory_LOCFCarryForward(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": "Bank"},
		{"id": "sav1", "name": "Savings", "account_type": "savings", "currency": "USD", "org_name": "Bank"},
	})
	// chk1 has snapshots on both days; sav1 only on day 1.
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "sav1", "balance": "5000.00", "balance_date": "2024-01-01"},
		{"account_id": "chk1", "balance": "1200.00", "balance_date": "2024-01-02"},
	})

	r := chi.NewRouter()
	r.Get("/api/projections/history", handlers.GetProjectionHistory(database))

	req := httptest.NewRequest(http.MethodGet, "/api/projections/history?days=0&account_ids=chk1,sav1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp projectionHistoryJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Points) != 2 {
		t.Fatalf("expected 2 points, got %d: %+v", len(resp.Points), resp.Points)
	}
	// Day 1: 1000 + 5000 = 6000
	if resp.Points[0].Date != "2024-01-01" || resp.Points[0].Value != "6000.00" {
		t.Errorf("day1: expected {2024-01-01, 6000.00}, got {%s, %s}", resp.Points[0].Date, resp.Points[0].Value)
	}
	// Day 2: chk1=1200 (new snapshot) + sav1=5000 (LOCF from day1) = 6200
	if resp.Points[1].Date != "2024-01-02" || resp.Points[1].Value != "6200.00" {
		t.Errorf("day2: expected {2024-01-02, 6200.00}, got {%s, %s}", resp.Points[1].Date, resp.Points[1].Value)
	}
}

// TestGetProjectionHistory_EmptyAccountIDs verifies that an empty account_ids param
// returns an empty points array.
func TestGetProjectionHistory_EmptyAccountIDs(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Get("/api/projections/history", handlers.GetProjectionHistory(database))

	req := httptest.NewRequest(http.MethodGet, "/api/projections/history?days=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp projectionHistoryJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Points) != 0 {
		t.Errorf("expected 0 points for empty account_ids, got %d", len(resp.Points))
	}
}

// TestGetProjectionHistory_ExcludesHiddenAccounts verifies that hidden accounts
// are excluded even if their ID is listed in account_ids.
func TestGetProjectionHistory_ExcludesHiddenAccounts(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Visible", "account_type": "checking", "currency": "USD", "org_name": "Bank"},
		{"id": "chk2", "name": "Hidden", "account_type": "checking", "currency": "USD", "org_name": "Bank", "hidden_at": "2024-01-01T00:00:00Z"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "chk2", "balance": "9999.00", "balance_date": "2024-01-01"},
	})

	r := chi.NewRouter()
	r.Get("/api/projections/history", handlers.GetProjectionHistory(database))

	// Request both accounts — chk2 is hidden and should be excluded.
	req := httptest.NewRequest(http.MethodGet, "/api/projections/history?days=0&account_ids=chk1,chk2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp projectionHistoryJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(resp.Points))
	}
	// Only chk1's balance — hidden chk2 excluded.
	if resp.Points[0].Value != "1000.00" {
		t.Errorf("expected 1000.00 (hidden account excluded), got %s", resp.Points[0].Value)
	}
}
