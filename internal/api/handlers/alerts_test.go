package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/solomon/finance-visualizer/internal/api/handlers"
)

// alertRuleJSON mirrors the JSON structure returned by alert endpoints.
type alertRuleJSON struct {
	ID               int               `json:"id"`
	Name             string            `json:"name"`
	Operands         json.RawMessage   `json:"operands"`
	Expression       string            `json:"expression"`
	Comparison       string            `json:"comparison"`
	Threshold        string            `json:"threshold"`
	NotifyOnRecovery bool              `json:"notify_on_recovery"`
	Enabled          bool              `json:"enabled"`
	LastState        string            `json:"last_state"`
	LastEvalAt       *string           `json:"last_eval_at"`
	LastValue        *string           `json:"last_value"`
	CreatedAt        string            `json:"created_at"`
	UpdatedAt        string            `json:"updated_at"`
	History          []alertHistoryJSON `json:"history"`
}

type alertHistoryJSON struct {
	ID         int     `json:"id"`
	State      string  `json:"state"`
	Value      *string `json:"value"`
	NotifiedAt string  `json:"notified_at"`
}

func TestCreateAlert_Valid(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/alerts", handlers.CreateAlert(database))

	body := strings.NewReader(`{
		"name": "Low Checking",
		"operands": [{"type":"bucket","ref":"liquid","label":"Liquid","operator":"+"}],
		"comparison": "<",
		"threshold": "1000"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/alerts", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp alertRuleJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Name != "Low Checking" {
		t.Errorf("name: got %q, want %q", resp.Name, "Low Checking")
	}
	if resp.Expression == "" {
		t.Error("expected non-empty expression")
	}
	if resp.Comparison != "<" {
		t.Errorf("comparison: got %q, want %q", resp.Comparison, "<")
	}
	if resp.Threshold != "1000" {
		t.Errorf("threshold: got %q, want %q", resp.Threshold, "1000")
	}
	if !resp.Enabled {
		t.Error("expected enabled=true by default")
	}
	if resp.LastState != "normal" {
		t.Errorf("last_state: got %q, want %q", resp.LastState, "normal")
	}
}

func TestCreateAlert_MissingName(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/alerts", handlers.CreateAlert(database))

	body := strings.NewReader(`{
		"name": "",
		"operands": [{"type":"bucket","ref":"liquid","label":"Liquid","operator":"+"}],
		"comparison": "<",
		"threshold": "1000"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/alerts", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "name is required") {
		t.Errorf("expected error about name, got: %s", w.Body.String())
	}
}

func TestCreateAlert_InvalidComparison(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/alerts", handlers.CreateAlert(database))

	body := strings.NewReader(`{
		"name": "Bad",
		"operands": [{"type":"bucket","ref":"liquid","label":"Liquid","operator":"+"}],
		"comparison": "!=",
		"threshold": "1000"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/alerts", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateAlert_EmptyOperands(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/alerts", handlers.CreateAlert(database))

	body := strings.NewReader(`{
		"name": "Bad",
		"operands": [],
		"comparison": "<",
		"threshold": "1000"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/alerts", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListAlerts_Empty(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Get("/api/alerts", handlers.ListAlerts(database))

	req := httptest.NewRequest(http.MethodGet, "/api/alerts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []alertRuleJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp == nil {
		t.Fatal("expected empty array, got null")
	}
	if len(resp) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(resp))
	}
}

func TestListAlerts_WithRules(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/alerts", handlers.CreateAlert(database))
	r.Get("/api/alerts", handlers.ListAlerts(database))

	// Create two rules.
	for _, name := range []string{"Rule A", "Rule B"} {
		body := strings.NewReader(fmt.Sprintf(`{
			"name": %q,
			"operands": [{"type":"bucket","ref":"liquid","label":"Liquid","operator":"+"}],
			"comparison": "<",
			"threshold": "500"
		}`, name))
		req := httptest.NewRequest(http.MethodPost, "/api/alerts", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %s: expected 201, got %d: %s", name, w.Code, w.Body.String())
		}
	}

	// List all rules.
	req := httptest.NewRequest(http.MethodGet, "/api/alerts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []alertRuleJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(resp))
	}

	// Each rule should have a history array (empty).
	for _, rule := range resp {
		if rule.History == nil {
			t.Errorf("rule %d: expected history array, got nil", rule.ID)
		}
	}
}

func TestUpdateAlert(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/alerts", handlers.CreateAlert(database))
	r.Put("/api/alerts/{id}", handlers.UpdateAlert(database))

	// Create a rule.
	body := strings.NewReader(`{
		"name": "Original",
		"operands": [{"type":"bucket","ref":"liquid","label":"Liquid","operator":"+"}],
		"comparison": "<",
		"threshold": "1000"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/alerts", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created alertRuleJSON
	json.Unmarshal(w.Body.Bytes(), &created)

	// Update the rule.
	updateBody := strings.NewReader(`{
		"name": "Updated Name",
		"operands": [{"type":"bucket","ref":"savings","label":"Savings","operator":"+"}],
		"comparison": ">",
		"threshold": "2000"
	}`)
	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/alerts/%d", created.ID), updateBody)
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", updateW.Code, updateW.Body.String())
	}

	var resp alertRuleJSON
	if err := json.Unmarshal(updateW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Name != "Updated Name" {
		t.Errorf("name: got %q, want %q", resp.Name, "Updated Name")
	}
	if resp.Comparison != ">" {
		t.Errorf("comparison: got %q, want %q", resp.Comparison, ">")
	}
	if resp.Threshold != "2000" {
		t.Errorf("threshold: got %q, want %q", resp.Threshold, "2000")
	}
}

func TestToggleAlert(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/alerts", handlers.CreateAlert(database))
	r.Patch("/api/alerts/{id}", handlers.ToggleAlert(database))

	// Create a rule (enabled by default).
	body := strings.NewReader(`{
		"name": "Toggle Test",
		"operands": [{"type":"bucket","ref":"liquid","label":"Liquid","operator":"+"}],
		"comparison": "<",
		"threshold": "1000"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/alerts", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created alertRuleJSON
	json.Unmarshal(w.Body.Bytes(), &created)

	// Disable the rule.
	toggleBody := strings.NewReader(`{"enabled": false}`)
	toggleReq := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/alerts/%d", created.ID), toggleBody)
	toggleReq.Header.Set("Content-Type", "application/json")
	toggleW := httptest.NewRecorder()
	r.ServeHTTP(toggleW, toggleReq)

	if toggleW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", toggleW.Code, toggleW.Body.String())
	}

	var resp alertRuleJSON
	if err := json.Unmarshal(toggleW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Enabled {
		t.Error("expected enabled=false after toggle")
	}
}

func TestDeleteAlert(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/alerts", handlers.CreateAlert(database))
	r.Delete("/api/alerts/{id}", handlers.DeleteAlert(database))
	r.Get("/api/alerts", handlers.ListAlerts(database))

	// Create a rule.
	body := strings.NewReader(`{
		"name": "To Delete",
		"operands": [{"type":"bucket","ref":"liquid","label":"Liquid","operator":"+"}],
		"comparison": "<",
		"threshold": "1000"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/alerts", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created alertRuleJSON
	json.Unmarshal(w.Body.Bytes(), &created)

	// Delete the rule.
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/alerts/%d", created.ID), nil)
	delW := httptest.NewRecorder()
	r.ServeHTTP(delW, delReq)

	if delW.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", delW.Code, delW.Body.String())
	}

	// List should be empty.
	listReq := httptest.NewRequest(http.MethodGet, "/api/alerts", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)

	var rules []alertRuleJSON
	json.Unmarshal(listW.Body.Bytes(), &rules)
	if len(rules) != 0 {
		t.Errorf("expected 0 alerts after delete, got %d", len(rules))
	}
}
