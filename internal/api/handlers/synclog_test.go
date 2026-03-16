package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/solomon/finance-visualizer/internal/api/handlers"
)

func TestGetSyncLog_ReturnsLast7Entries(t *testing.T) {
	database := setupSettingsDB(t)

	// Insert 10 sync_log entries
	for i := 1; i <= 10; i++ {
		_, err := database.Exec(
			`INSERT INTO sync_log(started_at, finished_at, accounts_fetched, accounts_failed)
			 VALUES(datetime('now', ?), datetime('now', ?), ?, 0)`,
			fmt.Sprintf("-%d minutes", (10-i)*10),
			fmt.Sprintf("-%d minutes", (10-i)*10-5),
			i,
		)
		if err != nil {
			t.Fatalf("failed to insert sync_log entry %d: %v", i, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sync-log", nil)
	w := httptest.NewRecorder()
	handlers.GetSyncLog(database).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Entries []struct {
			ID              int64   `json:"id"`
			StartedAt       string  `json:"started_at"`
			FinishedAt      *string `json:"finished_at"`
			AccountsFetched int     `json:"accounts_fetched"`
			AccountsFailed  int     `json:"accounts_failed"`
			ErrorText       *string `json:"error_text"`
			Status          string  `json:"status"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Entries) != 7 {
		t.Fatalf("expected 7 entries, got %d", len(resp.Entries))
	}

	// Verify newest-first ordering: first entry should have highest ID
	if resp.Entries[0].ID < resp.Entries[6].ID {
		t.Errorf("expected newest-first ordering, got first ID=%d < last ID=%d", resp.Entries[0].ID, resp.Entries[6].ID)
	}
}

func TestGetSyncLog_EmptyDB(t *testing.T) {
	database := setupSettingsDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/sync-log", nil)
	w := httptest.NewRecorder()
	handlers.GetSyncLog(database).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Should return {"entries":[]} not {"entries":null}
	body := w.Body.String()
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	entries, ok := resp["entries"].([]interface{})
	if !ok {
		t.Fatalf("expected entries to be an array, got %T (%v)", resp["entries"], resp["entries"])
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestGetSyncLog_StatusFailed(t *testing.T) {
	database := setupSettingsDB(t)

	_, err := database.Exec(
		`INSERT INTO sync_log(started_at, finished_at, accounts_fetched, accounts_failed, error_text)
		 VALUES(CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0, 0, 'connection refused')`,
	)
	if err != nil {
		t.Fatalf("failed to insert sync_log: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sync-log", nil)
	w := httptest.NewRecorder()
	handlers.GetSyncLog(database).ServeHTTP(w, req)

	var resp struct {
		Entries []struct {
			Status string `json:"status"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp.Entries))
	}
	if resp.Entries[0].Status != "failed" {
		t.Errorf("expected status=failed, got %q", resp.Entries[0].Status)
	}
}

func TestGetSyncLog_StatusPartial(t *testing.T) {
	database := setupSettingsDB(t)

	_, err := database.Exec(
		`INSERT INTO sync_log(started_at, finished_at, accounts_fetched, accounts_failed)
		 VALUES(CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 3, 1)`,
	)
	if err != nil {
		t.Fatalf("failed to insert sync_log: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sync-log", nil)
	w := httptest.NewRecorder()
	handlers.GetSyncLog(database).ServeHTTP(w, req)

	var resp struct {
		Entries []struct {
			Status string `json:"status"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp.Entries))
	}
	if resp.Entries[0].Status != "partial" {
		t.Errorf("expected status=partial, got %q", resp.Entries[0].Status)
	}
}

func TestGetSyncLog_StatusSuccess(t *testing.T) {
	database := setupSettingsDB(t)

	_, err := database.Exec(
		`INSERT INTO sync_log(started_at, finished_at, accounts_fetched, accounts_failed)
		 VALUES(CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 5, 0)`,
	)
	if err != nil {
		t.Fatalf("failed to insert sync_log: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sync-log", nil)
	w := httptest.NewRecorder()
	handlers.GetSyncLog(database).ServeHTTP(w, req)

	var resp struct {
		Entries []struct {
			Status string `json:"status"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp.Entries))
	}
	if resp.Entries[0].Status != "success" {
		t.Errorf("expected status=success, got %q", resp.Entries[0].Status)
	}
}

func TestSanitizeErrorText_URLCredentials(t *testing.T) {
	input := "failed to connect to user:secretpass@api.simplefin.org:443"
	result := handlers.SanitizeErrorText(input)

	if result == input {
		t.Error("expected credentials to be sanitized, but input was returned unchanged")
	}
	expected := "failed to connect to [redacted-url]"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSanitizeErrorText_Base64Token(t *testing.T) {
	// 40+ char base64 string
	token := "aHR0cHM6Ly91c2VyOnBhc3NAYXBpLnNpbXBsZWZpbi5vcmcvYWNjb3VudHM="
	input := "auth failed with token " + token + " in request"
	result := handlers.SanitizeErrorText(input)

	if result == input {
		t.Error("expected base64 token to be sanitized, but input was returned unchanged")
	}
	expected := "auth failed with token [redacted-token] in request"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSanitizeErrorText_NoCredentials(t *testing.T) {
	input := "connection timeout after 30s"
	result := handlers.SanitizeErrorText(input)

	if result != input {
		t.Errorf("expected unchanged message %q, got %q", input, result)
	}
}
