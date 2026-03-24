package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/solomon/finance-visualizer/internal/api/handlers"
)

// netWorthPointJSON mirrors the JSON structure for a single net worth time-series point.
type netWorthPointJSON struct {
	Date        string `json:"date"`
	Liquid      string `json:"liquid"`
	Savings     string `json:"savings"`
	Investments string `json:"investments"`
}

// netWorthStatsJSON mirrors the JSON structure for net worth statistics.
type netWorthStatsJSON struct {
	CurrentNetWorth    string  `json:"current_net_worth"`
	PeriodChangeDollar string  `json:"period_change_dollars"`
	PeriodChangePct    *string `json:"period_change_pct"`
	AllTimeHigh        string  `json:"all_time_high"`
	AllTimeHighDate    string  `json:"all_time_high_date"`
}

// netWorthResponseJSON mirrors the JSON structure of the net worth endpoint response.
type netWorthResponseJSON struct {
	Points []netWorthPointJSON `json:"points"`
	Stats  *netWorthStatsJSON  `json:"stats"`
}

func TestNetWorth_BasicResponse(t *testing.T) {
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
		{"account_id": "chk1", "balance": "1200.00", "balance_date": "2024-01-02"},
		{"account_id": "sav1", "balance": "600.00", "balance_date": "2024-01-02"},
		{"account_id": "inv1", "balance": "3200.00", "balance_date": "2024-01-02"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/net-worth?days=0", nil)
	w := httptest.NewRecorder()
	handlers.GetNetWorth(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp netWorthResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should have 2 points
	if len(resp.Points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(resp.Points))
	}

	// Each point has date, liquid, savings, investments
	p := resp.Points[0]
	if p.Date != "2024-01-01" {
		t.Errorf("point[0].date: got %q, want %q", p.Date, "2024-01-01")
	}
	if p.Liquid != "1000.00" {
		t.Errorf("point[0].liquid: got %q, want %q", p.Liquid, "1000.00")
	}
	if p.Savings != "500.00" {
		t.Errorf("point[0].savings: got %q, want %q", p.Savings, "500.00")
	}
	if p.Investments != "3000.00" {
		t.Errorf("point[0].investments: got %q, want %q", p.Investments, "3000.00")
	}

	// Stats should be computed
	if resp.Stats == nil {
		t.Fatal("expected stats to be non-nil")
	}
	// Current net worth = 1200 + 600 + 3200 = 5000
	if resp.Stats.CurrentNetWorth != "5000.00" {
		t.Errorf("current_net_worth: got %q, want %q", resp.Stats.CurrentNetWorth, "5000.00")
	}
}

func TestNetWorth_PointsChronologicalOrder(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "300.00", "balance_date": "2024-01-03"},
		{"account_id": "chk1", "balance": "100.00", "balance_date": "2024-01-01"},
		{"account_id": "chk1", "balance": "200.00", "balance_date": "2024-01-02"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/net-worth?days=0", nil)
	w := httptest.NewRecorder()
	handlers.GetNetWorth(database)(w, req)

	var resp netWorthResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Points) != 3 {
		t.Fatalf("expected 3 points, got %d", len(resp.Points))
	}
	for i := 0; i < len(resp.Points)-1; i++ {
		if resp.Points[i].Date >= resp.Points[i+1].Date {
			t.Errorf("points not chronological: %q >= %q at index %d", resp.Points[i].Date, resp.Points[i+1].Date, i)
		}
	}
}

func TestNetWorth_StatsFields(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "sav1", "name": "Savings", "account_type": "savings", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "sav1", "balance": "500.00", "balance_date": "2024-01-01"},
		{"account_id": "chk1", "balance": "1500.00", "balance_date": "2024-01-02"},
		{"account_id": "sav1", "balance": "700.00", "balance_date": "2024-01-02"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/net-worth?days=0", nil)
	w := httptest.NewRecorder()
	handlers.GetNetWorth(database)(w, req)

	var resp netWorthResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Stats == nil {
		t.Fatal("expected stats to be non-nil")
	}

	// Current = 1500 + 700 = 2200
	if resp.Stats.CurrentNetWorth != "2200.00" {
		t.Errorf("current_net_worth: got %q, want %q", resp.Stats.CurrentNetWorth, "2200.00")
	}
	// Period change = 2200 - 1500 = 700
	if resp.Stats.PeriodChangeDollar != "700.00" {
		t.Errorf("period_change_dollars: got %q, want %q", resp.Stats.PeriodChangeDollar, "700.00")
	}
	// Pct = 700/1500 * 100 = 46.67
	if resp.Stats.PeriodChangePct == nil {
		t.Fatal("period_change_pct should not be nil")
	}
	if *resp.Stats.PeriodChangePct != "46.67" {
		t.Errorf("period_change_pct: got %q, want %q", *resp.Stats.PeriodChangePct, "46.67")
	}
	// All-time high = 2200 on 2024-01-02
	if resp.Stats.AllTimeHigh != "2200.00" {
		t.Errorf("all_time_high: got %q, want %q", resp.Stats.AllTimeHigh, "2200.00")
	}
	if resp.Stats.AllTimeHighDate != "2024-01-02" {
		t.Errorf("all_time_high_date: got %q, want %q", resp.Stats.AllTimeHighDate, "2024-01-02")
	}
}

func TestNetWorth_DaysZeroReturnsAll(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "100.00", "balance_date": "2000-01-01"},
		{"account_id": "chk1", "balance": "200.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/net-worth?days=0", nil)
	w := httptest.NewRecorder()
	handlers.GetNetWorth(database)(w, req)

	var resp netWorthResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Points) != 2 {
		t.Errorf("days=0 should return all data, got %d points", len(resp.Points))
	}
}

func TestNetWorth_CarryForwardMissingPanelData(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "sav1", "name": "Savings", "account_type": "savings", "currency": "USD", "org_name": ""},
	})
	// Day 1: both panels have data. Day 2: only checking has data (savings missing)
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "sav1", "balance": "500.00", "balance_date": "2024-01-01"},
		{"account_id": "chk1", "balance": "1200.00", "balance_date": "2024-01-02"},
		// No savings snapshot on 2024-01-02
	})

	req := httptest.NewRequest(http.MethodGet, "/api/net-worth?days=0", nil)
	w := httptest.NewRecorder()
	handlers.GetNetWorth(database)(w, req)

	var resp netWorthResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(resp.Points))
	}

	// Day 2 savings should carry forward from Day 1
	if resp.Points[1].Savings != "500.00" {
		t.Errorf("day 2 savings should carry forward: got %q, want %q", resp.Points[1].Savings, "500.00")
	}
}

func TestNetWorth_PeriodChangePctNilWhenFirstTotalZero(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "crd1", "name": "Credit", "account_type": "credit", "currency": "USD", "org_name": ""},
	})
	// Day 1: checking 200 + credit -200 = 0 net worth total
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "200.00", "balance_date": "2024-01-01"},
		{"account_id": "crd1", "balance": "-200.00", "balance_date": "2024-01-01"},
		{"account_id": "chk1", "balance": "500.00", "balance_date": "2024-01-02"},
		{"account_id": "crd1", "balance": "-100.00", "balance_date": "2024-01-02"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/net-worth?days=0", nil)
	w := httptest.NewRecorder()
	handlers.GetNetWorth(database)(w, req)

	var resp netWorthResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Stats == nil {
		t.Fatal("expected stats to be non-nil")
	}
	if resp.Stats.PeriodChangePct != nil {
		t.Errorf("period_change_pct should be nil when first total is zero, got %q", *resp.Stats.PeriodChangePct)
	}
}

func TestNetWorth_AllTimeHighAcrossRange(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})
	// Peak on day 2, then decline
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "chk1", "balance": "5000.00", "balance_date": "2024-01-02"},
		{"account_id": "chk1", "balance": "3000.00", "balance_date": "2024-01-03"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/net-worth?days=0", nil)
	w := httptest.NewRecorder()
	handlers.GetNetWorth(database)(w, req)

	var resp netWorthResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Stats == nil {
		t.Fatal("expected stats")
	}
	if resp.Stats.AllTimeHigh != "5000.00" {
		t.Errorf("all_time_high: got %q, want %q", resp.Stats.AllTimeHigh, "5000.00")
	}
	if resp.Stats.AllTimeHighDate != "2024-01-02" {
		t.Errorf("all_time_high_date: got %q, want %q", resp.Stats.AllTimeHighDate, "2024-01-02")
	}
}

func TestNetWorth_NoSnapshotsReturnsEmptyResponse(t *testing.T) {
	database := setupFinanceTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/net-worth?days=90", nil)
	w := httptest.NewRecorder()
	handlers.GetNetWorth(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp netWorthResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Points should be empty array, not null
	if resp.Points == nil {
		t.Error("points should be empty array, not null")
	}
	if len(resp.Points) != 0 {
		t.Errorf("expected 0 points, got %d", len(resp.Points))
	}
	if resp.Stats != nil {
		t.Errorf("stats should be nil when no data, got %+v", resp.Stats)
	}

	// Verify JSON structure: points should be []
	raw := w.Body.Bytes()
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		t.Fatalf("failed to parse raw: %v", err)
	}
	if string(rawMap["points"]) != "[]" {
		t.Errorf("points JSON: got %s, want []", rawMap["points"])
	}
}

func TestNetWorth_CoalesceAccountTypeOverride(t *testing.T) {
	database := setupFinanceTestDB(t)

	// Account is type=checking but override=savings, should appear in savings panel
	seedAccounts(t, database, []map[string]string{
		{"id": "acct1", "name": "Override Account", "account_type": "checking", "currency": "USD", "org_name": "", "account_type_override": "savings"},
		{"id": "acct2", "name": "Normal Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "acct1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "acct2", "balance": "500.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/net-worth?days=0", nil)
	w := httptest.NewRecorder()
	handlers.GetNetWorth(database)(w, req)

	var resp netWorthResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(resp.Points))
	}

	// acct1 overridden to savings, acct2 is checking -> liquid
	if resp.Points[0].Liquid != "500.00" {
		t.Errorf("liquid: got %q, want %q (only normal checking)", resp.Points[0].Liquid, "500.00")
	}
	if resp.Points[0].Savings != "1000.00" {
		t.Errorf("savings: got %q, want %q (overridden account)", resp.Points[0].Savings, "1000.00")
	}
}

func TestNetWorth_HiddenAccountsExcluded(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Visible", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "chk2", "name": "Hidden", "account_type": "checking", "currency": "USD", "org_name": "", "hidden_at": "2024-01-15T10:00:00Z"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": "2024-01-01"},
		{"account_id": "chk2", "balance": "9999.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/net-worth?days=0", nil)
	w := httptest.NewRecorder()
	handlers.GetNetWorth(database)(w, req)

	var resp netWorthResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(resp.Points))
	}
	// Only visible account's balance should appear
	if resp.Points[0].Liquid != "1000.00" {
		t.Errorf("liquid: got %q, want %q (hidden excluded)", resp.Points[0].Liquid, "1000.00")
	}
}

func TestNetWorth_LiquidCombinesCheckingAndCredit(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "crd1", "name": "Credit Card", "account_type": "credit", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "2000.00", "balance_date": "2024-01-01"},
		{"account_id": "crd1", "balance": "-500.00", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/net-worth?days=0", nil)
	w := httptest.NewRecorder()
	handlers.GetNetWorth(database)(w, req)

	var resp netWorthResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(resp.Points))
	}
	// Liquid = checking(2000) + credit(-500) = 1500
	if resp.Points[0].Liquid != "1500.00" {
		t.Errorf("liquid: got %q, want %q", resp.Points[0].Liquid, "1500.00")
	}
}

func TestNetWorth_DefaultDays90(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})
	// One very old snapshot and one recent
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "100.00", "balance_date": "2000-01-01"},
		{"account_id": "chk1", "balance": "200.00", "balance_date": "2026-03-15"},
	})

	// No days param = default 90
	req := httptest.NewRequest(http.MethodGet, "/api/net-worth", nil)
	w := httptest.NewRecorder()
	handlers.GetNetWorth(database)(w, req)

	var resp netWorthResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should only include recent data, not the 2000-01-01 snapshot
	if len(resp.Points) != 1 {
		t.Errorf("default days=90 should filter old data, got %d points", len(resp.Points))
	}
}

// TestNetWorth_LOCFCreditCardMissingToday verifies that LOCF (last observation
// carried forward) is applied at the per-account level in the SQL query.
//
// Scenario: checking has a snapshot on day 2, but credit card only has a
// snapshot on day 1. Day 2's liquid total must include the credit card's
// day-1 balance carried forward, not just the checking balance.
//
// This is the regression test for the bug where the Net Worth tab showed
// $11,824 instead of $8,394 — because the credit card (which adds a negative
// balance to liquid) was absent from the day's SQL results when it had no
// same-day snapshot.
func TestNetWorth_LOCFCreditCardMissingToday(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "crd1", "name": "Credit Card", "account_type": "credit", "currency": "USD", "org_name": ""},
	})
	// Day 1: both checking and credit have snapshots.
	// Day 2: only checking has a snapshot; credit card has no same-day snapshot.
	// LOCF must carry credit's day-1 balance into day-2 liquid calculation.
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "4700.00", "balance_date": "2024-01-01"},
		{"account_id": "crd1", "balance": "-3946.00", "balance_date": "2024-01-01"},
		{"account_id": "chk1", "balance": "4700.00", "balance_date": "2024-01-02"},
		// No credit card snapshot on 2024-01-02
	})

	req := httptest.NewRequest(http.MethodGet, "/api/net-worth?days=0", nil)
	w := httptest.NewRecorder()
	handlers.GetNetWorth(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp netWorthResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(resp.Points))
	}

	// Day 1: liquid = 4700 + (-3946) = 754
	if resp.Points[0].Date != "2024-01-01" {
		t.Errorf("point[0].date: got %q, want 2024-01-01", resp.Points[0].Date)
	}
	if resp.Points[0].Liquid != "754.00" {
		t.Errorf("point[0].liquid: got %q, want 754.00 (checking + credit)", resp.Points[0].Liquid)
	}

	// Day 2: checking = 4700, credit LOCF carries forward -3946 → liquid = 754
	// Without LOCF fix, this would return "4700.00" (credit excluded).
	if resp.Points[1].Date != "2024-01-02" {
		t.Errorf("point[1].date: got %q, want 2024-01-02", resp.Points[1].Date)
	}
	if resp.Points[1].Liquid != "754.00" {
		t.Errorf("point[1].liquid: got %q, want 754.00 (credit carried forward from day 1); LOCF not working", resp.Points[1].Liquid)
	}

	// Stats current net worth should also use carried-forward credit
	if resp.Stats == nil {
		t.Fatal("expected stats to be non-nil")
	}
	if resp.Stats.CurrentNetWorth != "754.00" {
		t.Errorf("current_net_worth: got %q, want 754.00", resp.Stats.CurrentNetWorth)
	}
}
