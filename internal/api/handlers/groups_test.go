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

// groupResponseJSON mirrors the JSON structure returned by group endpoints.
type groupResponseJSON struct {
	ID           int                   `json:"id"`
	Name         string                `json:"name"`
	PanelType    string                `json:"panel_type"`
	TotalBalance string                `json:"total_balance"`
	Members      []groupMemberRespJSON `json:"members"`
}

type groupMemberRespJSON struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	OriginalName string  `json:"original_name"`
	Balance      string  `json:"balance"`
	Currency     string  `json:"currency"`
	OrgName      string  `json:"org_name"`
	DisplayName  *string `json:"display_name"`
}


func TestGroupCreate_Success(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/groups", handlers.CreateGroup(database))

	body := strings.NewReader(`{"name":"Coinbase","panel_type":"investment"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/groups", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp groupResponseJSON
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Name != "Coinbase" {
		t.Errorf("name: got %q, want %q", resp.Name, "Coinbase")
	}
	if resp.PanelType != "investment" {
		t.Errorf("panel_type: got %q, want %q", resp.PanelType, "investment")
	}
	if resp.TotalBalance != "0.00" {
		t.Errorf("total_balance: got %q, want %q", resp.TotalBalance, "0.00")
	}
	if len(resp.Members) != 0 {
		t.Errorf("expected empty members, got %d", len(resp.Members))
	}
	if resp.ID == 0 {
		t.Error("expected non-zero group ID")
	}
}

func TestGroupCreate_InvalidPanelType(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/groups", handlers.CreateGroup(database))

	body := strings.NewReader(`{"name":"Bad Group","panel_type":"foo"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/groups", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid panel_type, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGroupCreate_EmptyName(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/groups", handlers.CreateGroup(database))

	body := strings.NewReader(`{"name":"","panel_type":"investment"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/groups", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGroupAddMember_Success(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "acct1", "name": "Coinbase BTC", "account_type": "investment", "currency": "USD", "org_name": "Coinbase"},
	})
	seedSnapshots(t, database, []map[string]string{
		{"account_id": "acct1", "balance": "5000.00", "balance_date": "2024-01-01"},
	})

	r := chi.NewRouter()
	r.Post("/api/groups", handlers.CreateGroup(database))
	r.Post("/api/groups/{id}/members", handlers.AddGroupMember(database))

	// Create group first
	body := strings.NewReader(`{"name":"Coinbase","panel_type":"investment"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/groups", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created groupResponseJSON
	json.Unmarshal(w.Body.Bytes(), &created)

	// Add member
	memberBody := strings.NewReader(`{"account_id":"acct1"}`)
	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/groups/%d/members", created.ID), memberBody)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp groupResponseJSON
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(resp.Members))
	}
	if resp.Members[0].ID != "acct1" {
		t.Errorf("member id: got %q, want %q", resp.Members[0].ID, "acct1")
	}
	if resp.TotalBalance != "5000.00" {
		t.Errorf("total_balance: got %q, want %q", resp.TotalBalance, "5000.00")
	}
}

func TestGroupAddMember_AlreadyInGroup(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "acct1", "name": "Account 1", "account_type": "investment", "currency": "USD", "org_name": ""},
	})

	r := chi.NewRouter()
	r.Post("/api/groups", handlers.CreateGroup(database))
	r.Post("/api/groups/{id}/members", handlers.AddGroupMember(database))

	// Create two groups
	body1 := strings.NewReader(`{"name":"Group A","panel_type":"investment"}`)
	req1 := httptest.NewRequest(http.MethodPost, "/api/groups", body1)
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	var grpA groupResponseJSON
	json.Unmarshal(w1.Body.Bytes(), &grpA)

	body2 := strings.NewReader(`{"name":"Group B","panel_type":"savings"}`)
	req2 := httptest.NewRequest(http.MethodPost, "/api/groups", body2)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var grpB groupResponseJSON
	json.Unmarshal(w2.Body.Bytes(), &grpB)

	// Add acct1 to group A
	memberBody := strings.NewReader(`{"account_id":"acct1"}`)
	addReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/groups/%d/members", grpA.ID), memberBody)
	addReq.Header.Set("Content-Type", "application/json")
	addW := httptest.NewRecorder()
	r.ServeHTTP(addW, addReq)

	// Try to add acct1 to group B => 409
	memberBody2 := strings.NewReader(`{"account_id":"acct1"}`)
	addReq2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/groups/%d/members", grpB.ID), memberBody2)
	addReq2.Header.Set("Content-Type", "application/json")
	addW2 := httptest.NewRecorder()
	r.ServeHTTP(addW2, addReq2)

	if addW2.Code != http.StatusConflict {
		t.Errorf("expected 409 for account already in group, got %d: %s", addW2.Code, addW2.Body.String())
	}
}

func TestGroupRemoveMember_LastMemberDeletesGroup(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "acct1", "name": "Account 1", "account_type": "investment", "currency": "USD", "org_name": ""},
	})

	r := chi.NewRouter()
	r.Post("/api/groups", handlers.CreateGroup(database))
	r.Post("/api/groups/{id}/members", handlers.AddGroupMember(database))
	r.Delete("/api/groups/{id}/members/{accountId}", handlers.RemoveGroupMember(database))

	// Create group + add member
	body := strings.NewReader(`{"name":"Solo Group","panel_type":"investment"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/groups", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var grp groupResponseJSON
	json.Unmarshal(w.Body.Bytes(), &grp)

	memberBody := strings.NewReader(`{"account_id":"acct1"}`)
	addReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/groups/%d/members", grp.ID), memberBody)
	addReq.Header.Set("Content-Type", "application/json")
	addW := httptest.NewRecorder()
	r.ServeHTTP(addW, addReq)

	// Remove the only member => group should auto-delete
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/groups/%d/members/acct1", grp.ID), nil)
	delW := httptest.NewRecorder()
	r.ServeHTTP(delW, delReq)

	if delW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", delW.Code, delW.Body.String())
	}

	var delResp struct {
		DeletedGroup bool `json:"deleted_group"`
	}
	json.Unmarshal(delW.Body.Bytes(), &delResp)

	if !delResp.DeletedGroup {
		t.Error("expected deleted_group=true when last member removed")
	}

	// Verify group is gone from database
	var count int
	err := database.QueryRow(`SELECT COUNT(*) FROM account_groups WHERE id = ?`, grp.ID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query groups: %v", err)
	}
	if count != 0 {
		t.Errorf("expected group to be deleted, but found %d rows", count)
	}
}

func TestGroupUpdate_Name(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/groups", handlers.CreateGroup(database))
	r.Patch("/api/groups/{id}", handlers.UpdateGroup(database))

	// Create group
	body := strings.NewReader(`{"name":"Original","panel_type":"investment"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/groups", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var grp groupResponseJSON
	json.Unmarshal(w.Body.Bytes(), &grp)

	// Update name
	updateBody := strings.NewReader(`{"name":"Renamed"}`)
	updateReq := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/groups/%d", grp.ID), updateBody)
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", updateW.Code, updateW.Body.String())
	}

	var resp groupResponseJSON
	json.Unmarshal(updateW.Body.Bytes(), &resp)

	if resp.Name != "Renamed" {
		t.Errorf("name: got %q, want %q", resp.Name, "Renamed")
	}
	if resp.PanelType != "investment" {
		t.Errorf("panel_type should be unchanged: got %q, want %q", resp.PanelType, "investment")
	}
}

func TestGroupUpdate_PanelType(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/groups", handlers.CreateGroup(database))
	r.Patch("/api/groups/{id}", handlers.UpdateGroup(database))

	body := strings.NewReader(`{"name":"Test","panel_type":"investment"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/groups", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var grp groupResponseJSON
	json.Unmarshal(w.Body.Bytes(), &grp)

	updateBody := strings.NewReader(`{"panel_type":"savings"}`)
	updateReq := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/groups/%d", grp.ID), updateBody)
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", updateW.Code, updateW.Body.String())
	}

	var resp groupResponseJSON
	json.Unmarshal(updateW.Body.Bytes(), &resp)

	if resp.PanelType != "savings" {
		t.Errorf("panel_type: got %q, want %q", resp.PanelType, "savings")
	}
}

func TestGroupDelete_Success(t *testing.T) {
	database := setupFinanceTestDB(t)

	seedAccounts(t, database, []map[string]string{
		{"id": "acct1", "name": "Account 1", "account_type": "investment", "currency": "USD", "org_name": ""},
	})

	r := chi.NewRouter()
	r.Post("/api/groups", handlers.CreateGroup(database))
	r.Post("/api/groups/{id}/members", handlers.AddGroupMember(database))
	r.Delete("/api/groups/{id}", handlers.DeleteGroup(database))

	// Create group + add member
	body := strings.NewReader(`{"name":"ToDelete","panel_type":"investment"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/groups", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var grp groupResponseJSON
	json.Unmarshal(w.Body.Bytes(), &grp)

	memberBody := strings.NewReader(`{"account_id":"acct1"}`)
	addReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/groups/%d/members", grp.ID), memberBody)
	addReq.Header.Set("Content-Type", "application/json")
	addW := httptest.NewRecorder()
	r.ServeHTTP(addW, addReq)

	// Delete group
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/groups/%d", grp.ID), nil)
	delW := httptest.NewRecorder()
	r.ServeHTTP(delW, delReq)

	if delW.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", delW.Code, delW.Body.String())
	}

	// Verify group and members are gone
	var groupCount int
	database.QueryRow(`SELECT COUNT(*) FROM account_groups WHERE id = ?`, grp.ID).Scan(&groupCount)
	if groupCount != 0 {
		t.Error("expected group to be deleted")
	}

	var memberCount int
	database.QueryRow(`SELECT COUNT(*) FROM group_members WHERE group_id = ?`, grp.ID).Scan(&memberCount)
	if memberCount != 0 {
		t.Error("expected members to be cascade-deleted")
	}
}

func TestGroupDelete_NotFound(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Delete("/api/groups/{id}", handlers.DeleteGroup(database))

	delReq := httptest.NewRequest(http.MethodDelete, "/api/groups/99999", nil)
	delW := httptest.NewRecorder()
	r.ServeHTTP(delW, delReq)

	if delW.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent group, got %d: %s", delW.Code, delW.Body.String())
	}
}
