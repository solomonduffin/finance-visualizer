package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/solomon/finance-visualizer/internal/api/handlers"
)

// withChiURLParam creates a request with a chi route context URL parameter.
func withChiURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestUpdateAccount_SetDisplayName(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Chase Checking 1234", "account_type": "checking", "currency": "USD", "org_name": "Chase"},
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/accounts/chk1",
		strings.NewReader(`{"display_name": "My Checking"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "id", "chk1")

	w := httptest.NewRecorder()
	handlers.UpdateAccount(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp accountItemJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Name != "My Checking" {
		t.Errorf("name: got %q, want %q", resp.Name, "My Checking")
	}
	if resp.DisplayName == nil || *resp.DisplayName != "My Checking" {
		t.Errorf("display_name: got %v, want \"My Checking\"", resp.DisplayName)
	}
	if resp.OriginalName != "Chase Checking 1234" {
		t.Errorf("original_name: got %q, want %q", resp.OriginalName, "Chase Checking 1234")
	}
}

func TestUpdateAccount_ClearDisplayName(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Chase Checking 1234", "account_type": "checking", "currency": "USD", "org_name": "Chase", "display_name": "My Checking"},
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/accounts/chk1",
		strings.NewReader(`{"display_name": null}`))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "id", "chk1")

	w := httptest.NewRecorder()
	handlers.UpdateAccount(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp accountItemJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// display_name cleared: name should revert to original
	if resp.Name != "Chase Checking 1234" {
		t.Errorf("name: got %q, want %q", resp.Name, "Chase Checking 1234")
	}
	if resp.DisplayName != nil {
		t.Errorf("display_name: got %v, want nil", resp.DisplayName)
	}
}

func TestUpdateAccount_HideAccount(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Chase Checking", "account_type": "checking", "currency": "USD", "org_name": "Chase"},
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/accounts/chk1",
		strings.NewReader(`{"hidden": true}`))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "id", "chk1")

	w := httptest.NewRecorder()
	handlers.UpdateAccount(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp accountItemJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.HiddenAt == nil {
		t.Error("hidden_at should be non-null after hiding")
	}
}

func TestUpdateAccount_UnhideAccount(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Chase Checking", "account_type": "checking", "currency": "USD", "org_name": "Chase", "hidden_at": "2024-01-15T10:00:00Z"},
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/accounts/chk1",
		strings.NewReader(`{"hidden": false}`))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "id", "chk1")

	w := httptest.NewRecorder()
	handlers.UpdateAccount(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp accountItemJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.HiddenAt != nil {
		t.Errorf("hidden_at should be null after unhiding, got %v", resp.HiddenAt)
	}
}

func TestUpdateAccount_SetTypeOverride(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/accounts/chk1",
		strings.NewReader(`{"account_type_override": "savings"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "id", "chk1")

	w := httptest.NewRecorder()
	handlers.UpdateAccount(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp accountItemJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.AccountTypeOverride == nil || *resp.AccountTypeOverride != "savings" {
		t.Errorf("account_type_override: got %v, want \"savings\"", resp.AccountTypeOverride)
	}
	if resp.Type != "savings" {
		t.Errorf("effective type: got %q, want %q", resp.Type, "savings")
	}
}

func TestUpdateAccount_InvalidTypeOverride(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/accounts/chk1",
		strings.NewReader(`{"account_type_override": "invalid"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "id", "chk1")

	w := httptest.NewRecorder()
	handlers.UpdateAccount(database)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid type, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAccount_CombinedFields(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Chase Checking", "account_type": "checking", "currency": "USD", "org_name": "Chase"},
	})

	body := `{"display_name": "My Main", "hidden": true, "account_type_override": "savings"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/accounts/chk1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "id", "chk1")

	w := httptest.NewRecorder()
	handlers.UpdateAccount(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp accountItemJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Name != "My Main" {
		t.Errorf("name: got %q, want %q", resp.Name, "My Main")
	}
	if resp.HiddenAt == nil {
		t.Error("hidden_at should be non-null")
	}
	if resp.AccountTypeOverride == nil || *resp.AccountTypeOverride != "savings" {
		t.Errorf("type_override: got %v, want \"savings\"", resp.AccountTypeOverride)
	}
}

func TestUpdateAccount_NotFound(t *testing.T) {
	database := setupFinanceTestDB(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/accounts/nonexistent",
		strings.NewReader(`{"display_name": "test"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "id", "nonexistent")

	w := httptest.NewRecorder()
	handlers.UpdateAccount(database)(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent account, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAccount_EmptyBody(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Checking", "account_type": "checking", "currency": "USD", "org_name": ""},
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/accounts/chk1",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "id", "chk1")

	w := httptest.NewRecorder()
	handlers.UpdateAccount(database)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAccount_ReturnsBalance(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "chk1", "name": "Chase Checking", "account_type": "checking", "currency": "USD", "org_name": "Chase"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "chk1", "balance": "1234.56", "balance_date": "2024-01-01"},
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/accounts/chk1",
		strings.NewReader(`{"display_name": "My Checking"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "id", "chk1")

	w := httptest.NewRecorder()
	handlers.UpdateAccount(database)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp accountItemJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Balance != "1234.56" {
		t.Errorf("balance: got %q, want %q", resp.Balance, "1234.56")
	}
}
