package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/solomon/finance-visualizer/internal/api/handlers"
)

func TestGetGrowth_CorrectPctChange(t *testing.T) {
	database := setupFinanceTestDB(t)

	today := time.Now().Format("2006-01-02")
	prior := time.Now().AddDate(0, 0, -31).Format("2006-01-02")

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "sav1", "name": "Savings", "account_type": "savings", "currency": "USD", "org_name": ""},
		{"id": "inv1", "name": "Investment", "account_type": "investment", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		// Today snapshots
		{"account_id": "chk1", "balance": "1100.00", "balance_date": today},
		{"account_id": "sav1", "balance": "550.00", "balance_date": today},
		{"account_id": "inv1", "balance": "3300.00", "balance_date": today},
		// 31-day-ago snapshots
		{"account_id": "chk1", "balance": "1000.00", "balance_date": prior},
		{"account_id": "sav1", "balance": "500.00", "balance_date": prior},
		{"account_id": "inv1", "balance": "3000.00", "balance_date": prior},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/growth", nil)
	w := httptest.NewRecorder()
	handlers.GetGrowth(database).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Liquid      *struct{ PctChange string `json:"pct_change"` } `json:"liquid"`
		Savings     *struct{ PctChange string `json:"pct_change"` } `json:"savings"`
		Investments *struct{ PctChange string `json:"pct_change"` } `json:"investments"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.Liquid == nil {
		t.Fatal("expected liquid growth data, got nil")
	}
	// (1100-1000)/1000 * 100 = 10%
	if resp.Liquid.PctChange != "10.00" {
		t.Errorf("liquid pct_change: expected 10.00, got %s", resp.Liquid.PctChange)
	}

	if resp.Savings == nil {
		t.Fatal("expected savings growth data, got nil")
	}
	// (550-500)/500 * 100 = 10%
	if resp.Savings.PctChange != "10.00" {
		t.Errorf("savings pct_change: expected 10.00, got %s", resp.Savings.PctChange)
	}

	if resp.Investments == nil {
		t.Fatal("expected investments growth data, got nil")
	}
	// (3300-3000)/3000 * 100 = 10%
	if resp.Investments.PctChange != "10.00" {
		t.Errorf("investments pct_change: expected 10.00, got %s", resp.Investments.PctChange)
	}
}

func TestGetGrowth_LiquidIncludesCheckingAndCredit(t *testing.T) {
	database := setupFinanceTestDB(t)

	today := time.Now().Format("2006-01-02")
	prior := time.Now().AddDate(0, 0, -31).Format("2006-01-02")

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "crd1", "name": "Credit", "account_type": "credit", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		// Today: checking=1000, credit=-200 => liquid=800
		{"account_id": "chk1", "balance": "1000.00", "balance_date": today},
		{"account_id": "crd1", "balance": "-200.00", "balance_date": today},
		// Prior: checking=900, credit=-100 => liquid=800
		{"account_id": "chk1", "balance": "900.00", "balance_date": prior},
		{"account_id": "crd1", "balance": "-100.00", "balance_date": prior},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/growth", nil)
	w := httptest.NewRecorder()
	handlers.GetGrowth(database).ServeHTTP(w, req)

	var resp struct {
		Liquid *struct {
			CurrentTotal string `json:"current_total"`
			PriorTotal   string `json:"prior_total"`
		} `json:"liquid"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.Liquid == nil {
		t.Fatal("expected liquid growth data, got nil")
	}
	// Current: 1000 + (-200) = 800
	if resp.Liquid.CurrentTotal != "800.00" {
		t.Errorf("liquid current_total: expected 800.00, got %s", resp.Liquid.CurrentTotal)
	}
	// Prior: 900 + (-100) = 800
	if resp.Liquid.PriorTotal != "800.00" {
		t.Errorf("liquid prior_total: expected 800.00, got %s", resp.Liquid.PriorTotal)
	}
}

func TestGetGrowth_NullWhenPriorIsZero(t *testing.T) {
	database := setupFinanceTestDB(t)

	today := time.Now().Format("2006-01-02")
	prior := time.Now().AddDate(0, 0, -31).Format("2006-01-02")

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1000.00", "balance_date": today},
		{"account_id": "chk1", "balance": "0.00", "balance_date": prior},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/growth", nil)
	w := httptest.NewRecorder()
	handlers.GetGrowth(database).ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp["liquid"] != nil {
		t.Errorf("expected liquid=null when prior is zero, got %v", resp["liquid"])
	}
}

func TestGetGrowth_NullWhenNoSnapshots(t *testing.T) {
	database := setupFinanceTestDB(t)

	// No accounts, no snapshots
	req := httptest.NewRequest(http.MethodGet, "/api/growth", nil)
	w := httptest.NewRecorder()
	handlers.GetGrowth(database).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp["liquid"] != nil {
		t.Errorf("expected liquid=null with no snapshots, got %v", resp["liquid"])
	}
	if resp["savings"] != nil {
		t.Errorf("expected savings=null with no snapshots, got %v", resp["savings"])
	}
	if resp["investments"] != nil {
		t.Errorf("expected investments=null with no snapshots, got %v", resp["investments"])
	}
}

func TestGetGrowth_UsesTypeOverride(t *testing.T) {
	database := setupFinanceTestDB(t)

	today := time.Now().Format("2006-01-02")
	prior := time.Now().AddDate(0, 0, -31).Format("2006-01-02")

	seedAccounts(t, database, []map[string]string{
		// account_type=checking but overridden to savings
		{"id": "chk1", "name": "Overridden", "account_type": "checking", "currency": "USD", "org_name": "", "account_type_override": "savings"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1100.00", "balance_date": today},
		{"account_id": "chk1", "balance": "1000.00", "balance_date": prior},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/growth", nil)
	w := httptest.NewRecorder()
	handlers.GetGrowth(database).ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// Should appear in savings, not liquid
	if resp["liquid"] != nil {
		t.Errorf("expected liquid=null (account overridden to savings), got %v", resp["liquid"])
	}
	if resp["savings"] == nil {
		t.Error("expected savings to have growth data from overridden account")
	}
}

func TestGetGrowth_ExcludesHiddenAccounts(t *testing.T) {
	database := setupFinanceTestDB(t)

	today := time.Now().Format("2006-01-02")
	prior := time.Now().AddDate(0, 0, -31).Format("2006-01-02")

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Visible", "account_type": "checking", "currency": "USD", "org_name": ""},
		{"id": "chk2", "name": "Hidden", "account_type": "checking", "currency": "USD", "org_name": "", "hidden_at": "2024-01-15T10:00:00Z"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1100.00", "balance_date": today},
		{"account_id": "chk1", "balance": "1000.00", "balance_date": prior},
		{"account_id": "chk2", "balance": "5000.00", "balance_date": today},
		{"account_id": "chk2", "balance": "4000.00", "balance_date": prior},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/growth", nil)
	w := httptest.NewRecorder()
	handlers.GetGrowth(database).ServeHTTP(w, req)

	var resp struct {
		Liquid *struct {
			CurrentTotal string `json:"current_total"`
		} `json:"liquid"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.Liquid == nil {
		t.Fatal("expected liquid growth data")
	}
	// Only visible account's 1100
	if resp.Liquid.CurrentTotal != "1100.00" {
		t.Errorf("expected current_total=1100.00 (hidden excluded), got %s", resp.Liquid.CurrentTotal)
	}
}

func TestGetGrowth_BadgeEnabledDefault(t *testing.T) {
	database := setupFinanceTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/growth", nil)
	w := httptest.NewRecorder()
	handlers.GetGrowth(database).ServeHTTP(w, req)

	var resp struct {
		GrowthBadgeEnabled bool `json:"growth_badge_enabled"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if !resp.GrowthBadgeEnabled {
		t.Error("expected growth_badge_enabled=true by default")
	}
}

func TestGetGrowth_BadgeDisabled(t *testing.T) {
	database := setupFinanceTestDB(t)

	_, err := database.Exec(
		`INSERT INTO settings (key, value) VALUES ('growth_badge_enabled', 'false')`,
	)
	if err != nil {
		t.Fatalf("failed to insert setting: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/growth", nil)
	w := httptest.NewRecorder()
	handlers.GetGrowth(database).ServeHTTP(w, req)

	var resp struct {
		GrowthBadgeEnabled bool `json:"growth_badge_enabled"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.GrowthBadgeEnabled {
		t.Error("expected growth_badge_enabled=false when setting is 'false'")
	}
}

func TestSaveGrowthBadge_SetFalse(t *testing.T) {
	database := setupFinanceTestDB(t)

	body := strings.NewReader(`{"value":"false"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/settings/growth-badge", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.SaveGrowthBadge(database).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify persisted
	var val string
	err := database.QueryRow(`SELECT value FROM settings WHERE key='growth_badge_enabled'`).Scan(&val)
	if err != nil {
		t.Fatalf("failed to query setting: %v", err)
	}
	if val != "false" {
		t.Errorf("expected persisted value 'false', got %q", val)
	}
}

func TestSaveGrowthBadge_SetTrue(t *testing.T) {
	database := setupFinanceTestDB(t)

	body := strings.NewReader(`{"value":"true"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/settings/growth-badge", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.SaveGrowthBadge(database).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var val string
	err := database.QueryRow(`SELECT value FROM settings WHERE key='growth_badge_enabled'`).Scan(&val)
	if err != nil {
		t.Fatalf("failed to query setting: %v", err)
	}
	if val != "true" {
		t.Errorf("expected persisted value 'true', got %q", val)
	}
}

func TestSaveGrowthBadge_InvalidValue(t *testing.T) {
	database := setupFinanceTestDB(t)

	body := strings.NewReader(`{"value":"maybe"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/settings/growth-badge", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.SaveGrowthBadge(database).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid value, got %d", w.Code)
	}
}

func TestGetSettings_IncludesGrowthBadgeEnabled(t *testing.T) {
	database := setupSettingsDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	handlers.GetSettings(database).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		GrowthBadgeEnabled bool `json:"growth_badge_enabled"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// Default should be true
	if !resp.GrowthBadgeEnabled {
		t.Error("expected growth_badge_enabled=true by default in settings response")
	}
}

func TestGetSettings_GrowthBadgeDisabled(t *testing.T) {
	database := setupSettingsDB(t)

	_, err := database.Exec(
		`INSERT INTO settings (key, value) VALUES ('growth_badge_enabled', 'false')`,
	)
	if err != nil {
		t.Fatalf("failed to insert setting: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	handlers.GetSettings(database).ServeHTTP(w, req)

	var resp struct {
		GrowthBadgeEnabled bool `json:"growth_badge_enabled"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.GrowthBadgeEnabled {
		t.Error("expected growth_badge_enabled=false when setting is 'false'")
	}
}
