package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/solomon/finance-visualizer/internal/api/handlers"
)

// balancePointJSON mirrors the JSON structure for a single time-series data point.
type balancePointJSON struct {
	Date    string `json:"date"`
	Balance string `json:"balance"`
}

// historyResponseJSON mirrors the response from GetBalanceHistory.
type historyResponseJSON struct {
	Liquid      []balancePointJSON `json:"liquid"`
	Savings     []balancePointJSON `json:"savings"`
	Investments []balancePointJSON `json:"investments"`
}

func TestGetBalanceHistory_EmptyHistory(t *testing.T) {
	database := setupFinanceTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/balance-history", nil)
	w := httptest.NewRecorder()
	handlers.GetBalanceHistory(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp historyResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Empty history must return empty arrays, not null
	raw := w.Body.Bytes()
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		t.Fatalf("failed to parse raw response: %v", err)
	}
	for _, key := range []string{"liquid", "savings", "investments"} {
		if string(rawMap[key]) != "[]" {
			t.Errorf("%s: got %s, want [] (empty array, not null)", key, rawMap[key])
		}
	}
}

func TestGetBalanceHistory_ThreeSeries(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "sav1", "name": "Savings", "account_type": "savings", "currency": "USD", "org_name": ""},
		{"id": "inv1", "name": "Investment", "account_type": "investment", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "sav1", "balance": "500.00", "balance_date": "2024-01-01"},
		{"account_id": "inv1", "balance": "3000.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/balance-history", nil)
	w := httptest.NewRecorder()
	handlers.GetBalanceHistory(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp historyResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 1 {
		t.Errorf("liquid: expected 1 point, got %d", len(resp.Liquid))
	}
	if len(resp.Savings) != 1 {
		t.Errorf("savings: expected 1 point, got %d", len(resp.Savings))
	}
	if len(resp.Investments) != 1 {
		t.Errorf("investments: expected 1 point, got %d", len(resp.Investments))
	}
}

func TestGetBalanceHistory_LiquidIsSumOfCheckingAndCredit(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "crd1", "name": "Credit", "account_type": "credit", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "crd1", "balance": "-200.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/balance-history", nil)
	w := httptest.NewRecorder()
	handlers.GetBalanceHistory(database)(w, req)

	var resp historyResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 1 {
		t.Fatalf("liquid: expected 1 point, got %d", len(resp.Liquid))
	}
	// liquid = 1000 + (-200) = 800
	if resp.Liquid[0].Balance != "800.00" {
		t.Errorf("liquid balance: got %q, want \"800.00\"", resp.Liquid[0].Balance)
	}
	if resp.Liquid[0].Date != "2024-01-01" {
		t.Errorf("liquid date: got %q, want \"2024-01-01\"", resp.Liquid[0].Date)
	}
}

func TestGetBalanceHistory_CheckingOnlyStillProducesLiquidEntry(t *testing.T) {
	database := setupFinanceTestDB(t)

	// Only checking, no credit accounts — liquid should still work (credit defaults to zero)
	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "750.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/balance-history", nil)
	w := httptest.NewRecorder()
	handlers.GetBalanceHistory(database)(w, req)

	var resp historyResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 1 {
		t.Fatalf("liquid: expected 1 point, got %d", len(resp.Liquid))
	}
	if resp.Liquid[0].Balance != "750.00" {
		t.Errorf("liquid balance: got %q, want \"750.00\"", resp.Liquid[0].Balance)
	}
}

func TestGetBalanceHistory_SavingsSeriesSumsPerDay(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "sav1", "name": "Savings 1", "account_type": "savings", "currency": "USD", "org_name": ""},
		{"id": "sav2", "name": "Savings 2", "account_type": "savings", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "sav1", "balance": "300.00", "balance_date": "2024-01-01"},
		{"account_id": "sav2", "balance": "200.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/balance-history", nil)
	w := httptest.NewRecorder()
	handlers.GetBalanceHistory(database)(w, req)

	var resp historyResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Savings) != 1 {
		t.Fatalf("savings: expected 1 point, got %d", len(resp.Savings))
	}
	// savings = 300 + 200 = 500
	if resp.Savings[0].Balance != "500.00" {
		t.Errorf("savings balance: got %q, want \"500.00\"", resp.Savings[0].Balance)
	}
}

func TestGetBalanceHistory_InvestmentsSeriesSumsPerDay(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "inv1", "name": "401k", "account_type": "investment", "currency": "USD", "org_name": ""},
		{"id": "inv2", "name": "IRA", "account_type": "investment", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "inv1", "balance": "10000.00", "balance_date": "2024-01-01"},
		{"account_id": "inv2", "balance": "5000.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/balance-history", nil)
	w := httptest.NewRecorder()
	handlers.GetBalanceHistory(database)(w, req)

	var resp historyResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Investments) != 1 {
		t.Fatalf("investments: expected 1 point, got %d", len(resp.Investments))
	}
	// investments = 10000 + 5000 = 15000
	if resp.Investments[0].Balance != "15000.00" {
		t.Errorf("investments balance: got %q, want \"15000.00\"", resp.Investments[0].Balance)
	}
}

func TestGetBalanceHistory_OtherTypeExcluded(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "oth1", "name": "Other", "account_type": "other", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "500.00", "balance_date": "2024-01-01"},
		{"account_id": "oth1", "balance": "999.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/balance-history", nil)
	w := httptest.NewRecorder()
	handlers.GetBalanceHistory(database)(w, req)

	var resp historyResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 1 {
		t.Fatalf("liquid: expected 1 point, got %d", len(resp.Liquid))
	}
	// Only checking (500), other (999) must be excluded
	if resp.Liquid[0].Balance != "500.00" {
		t.Errorf("liquid balance: got %q, want \"500.00\" (other excluded)", resp.Liquid[0].Balance)
	}
}

func TestGetBalanceHistory_DaysParameterFilters(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})

	// Seed data across a range: today - 30 days (old), today - 3 days (recent), today - 1 day (recent)
	now := time.Now()
	oldDate := now.AddDate(0, 0, -30).Format("2006-01-02")
	recentDate1 := now.AddDate(0, 0, -3).Format("2006-01-02")
	recentDate2 := now.AddDate(0, 0, -1).Format("2006-01-02")

	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "100.00", "balance_date": oldDate},
		{"account_id": "chk1", "balance": "200.00", "balance_date": recentDate1},
		{"account_id": "chk1", "balance": "300.00", "balance_date": recentDate2},
	})

	// ?days=7 should return only the 2 recent snapshots (within last 7 days)
	req := httptest.NewRequest(http.MethodGet, "/api/balance-history?days=7", nil)
	w := httptest.NewRecorder()
	handlers.GetBalanceHistory(database)(w, req)

	var resp historyResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 2 {
		t.Errorf("liquid: expected 2 points with days=7, got %d (dates: %v)", len(resp.Liquid), resp.Liquid)
	}
}

func TestGetBalanceHistory_InvalidDaysIgnored(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "100.00", "balance_date": "2024-01-01"},
		{"account_id": "chk1", "balance": "200.00", "balance_date": "2024-01-02"},
	})

	// Test invalid days values — all should return all data
	for _, daysParam := range []string{"abc", "-5", "0"} {
		t.Run(fmt.Sprintf("days=%s", daysParam), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/balance-history?days="+daysParam, nil)
			w := httptest.NewRecorder()
			handlers.GetBalanceHistory(database)(w, req)

			var resp historyResponseJSON
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			// All data should be returned (2 snapshots = 2 liquid points since unique dates)
			if len(resp.Liquid) != 2 {
				t.Errorf("days=%s: liquid expected 2 points, got %d", daysParam, len(resp.Liquid))
			}
		})
	}
}

func TestGetBalanceHistory_MultipleAccountsSameDaySummed(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking 1", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "chk2", "name": "Checking 2", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "crd1", "name": "Credit 1", "account_type": "credit", "currency": "USD", "org_name": ""},
		{"id": "sav1", "name": "Savings 1", "account_type": "savings", "currency": "USD", "org_name": ""},
		{"id": "sav2", "name": "Savings 2", "account_type": "savings", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "500.00", "balance_date": "2024-01-01"},
		{"account_id": "chk2", "balance": "300.00", "balance_date": "2024-01-01"},
		{"account_id": "crd1", "balance": "-100.00", "balance_date": "2024-01-01"},
		{"account_id": "sav1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "sav2", "balance": "500.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/balance-history", nil)
	w := httptest.NewRecorder()
	handlers.GetBalanceHistory(database)(w, req)

	var resp historyResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 1 {
		t.Fatalf("liquid: expected 1 point, got %d", len(resp.Liquid))
	}
	// liquid = 500 + 300 + (-100) = 700
	if resp.Liquid[0].Balance != "700.00" {
		t.Errorf("liquid balance: got %q, want \"700.00\"", resp.Liquid[0].Balance)
	}

	if len(resp.Savings) != 1 {
		t.Fatalf("savings: expected 1 point, got %d", len(resp.Savings))
	}
	// savings = 1000 + 500 = 1500
	if resp.Savings[0].Balance != "1500.00" {
		t.Errorf("savings balance: got %q, want \"1500.00\"", resp.Savings[0].Balance)
	}
}

func TestGetBalanceHistory_ExcludesHidden(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Visible Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "chk2", "name": "Hidden Checking", "account_type": "checking", "currency": "USD", "org_name": "", "hidden_at": "2024-01-15T10:00:00Z"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "chk2", "balance": "500.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/balance-history", nil)
	w := httptest.NewRecorder()
	handlers.GetBalanceHistory(database)(w, req)

	var resp historyResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 1 {
		t.Fatalf("liquid: expected 1 point, got %d", len(resp.Liquid))
	}
	// Only visible checking (1000), hidden (500) excluded
	if resp.Liquid[0].Balance != "1000.00" {
		t.Errorf("liquid balance: got %q, want \"1000.00\" (hidden excluded)", resp.Liquid[0].Balance)
	}
}

func TestGetBalanceHistory_TypeOverride(t *testing.T) {
	database := setupFinanceTestDB(t)

	// Account with account_type=checking but overridden to savings
	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Real Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "chk2", "name": "Override to Savings", "account_type": "checking", "currency": "USD", "org_name": "", "account_type_override": "savings"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "chk2", "balance": "500.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/balance-history", nil)
	w := httptest.NewRecorder()
	handlers.GetBalanceHistory(database)(w, req)

	var resp historyResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// liquid should be 1000 (only real checking), savings should be 500 (overridden)
	if len(resp.Liquid) != 1 {
		t.Fatalf("liquid: expected 1 point, got %d", len(resp.Liquid))
	}
	if resp.Liquid[0].Balance != "1000.00" {
		t.Errorf("liquid balance: got %q, want \"1000.00\"", resp.Liquid[0].Balance)
	}
	if len(resp.Savings) != 1 {
		t.Fatalf("savings: expected 1 point, got %d", len(resp.Savings))
	}
	if resp.Savings[0].Balance != "500.00" {
		t.Errorf("savings balance: got %q, want \"500.00\" (overridden from checking)", resp.Savings[0].Balance)
	}
}

func TestGetBalanceHistory_TimeSeriesOrdering(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})
	// Seed in non-chronological order to verify ASC ordering
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "300.00", "balance_date": "2024-01-03"},
		{"account_id": "chk1", "balance": "100.00", "balance_date": "2024-01-01"},
		{"account_id": "chk1", "balance": "200.00", "balance_date": "2024-01-02"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/balance-history", nil)
	w := httptest.NewRecorder()
	handlers.GetBalanceHistory(database)(w, req)

	var resp historyResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Liquid) != 3 {
		t.Fatalf("liquid: expected 3 points, got %d", len(resp.Liquid))
	}

	expectedDates := []string{"2024-01-01", "2024-01-02", "2024-01-03"}
	expectedBalances := []string{"100.00", "200.00", "300.00"}
	for i, pt := range resp.Liquid {
		if pt.Date != expectedDates[i] {
			t.Errorf("liquid[%d].Date: got %q, want %q", i, pt.Date, expectedDates[i])
		}
		if pt.Balance != expectedBalances[i] {
			t.Errorf("liquid[%d].Balance: got %q, want %q", i, pt.Balance, expectedBalances[i])
		}
	}
}
