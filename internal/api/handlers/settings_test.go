package handlers_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/solomon/finance-visualizer/internal/api/handlers"
	"github.com/solomon/finance-visualizer/internal/db"
)

// setupSettingsDB creates a temp file SQLite DB with schema.
func setupSettingsDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "settings_test.db")

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err := db.Migrate(dbPath); err != nil {
		t.Fatalf("failed to migrate test DB: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestGetSettings_NotConfigured(t *testing.T) {
	database := setupSettingsDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	handlers.GetSettings(database).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	configured, ok := resp["configured"].(bool)
	if !ok || configured {
		t.Errorf("expected configured=false, got %v", resp["configured"])
	}
	if resp["last_sync_at"] != nil {
		t.Errorf("expected last_sync_at=null, got %v", resp["last_sync_at"])
	}
	if resp["last_sync_status"] != nil {
		t.Errorf("expected last_sync_status=null, got %v", resp["last_sync_status"])
	}
}

func TestGetSettings_Configured(t *testing.T) {
	database := setupSettingsDB(t)

	_, err := database.Exec(
		`INSERT INTO settings (key, value) VALUES ('simplefin_access_url', 'https://user:pass@host/simplefin')`,
	)
	if err != nil {
		t.Fatalf("failed to insert settings: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	handlers.GetSettings(database).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	configured, ok := resp["configured"].(bool)
	if !ok || !configured {
		t.Errorf("expected configured=true, got %v", resp["configured"])
	}
}

func TestGetSettings_WithSyncHistory(t *testing.T) {
	database := setupSettingsDB(t)

	// Insert a completed sync log entry with no error
	_, err := database.Exec(
		`INSERT INTO sync_log(started_at, finished_at, accounts_fetched, accounts_failed) VALUES(CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 3, 0)`,
	)
	if err != nil {
		t.Fatalf("failed to insert sync_log: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	handlers.GetSettings(database).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["last_sync_at"] == nil {
		t.Error("expected last_sync_at to be set, got null")
	}
	status, ok := resp["last_sync_status"].(string)
	if !ok || status != "success" {
		t.Errorf("expected last_sync_status=success, got %v", resp["last_sync_status"])
	}
}

func TestSaveSettings_Success(t *testing.T) {
	database := setupSettingsDB(t)

	body := strings.NewReader(`{"access_url":"https://user:pass@host/simplefin"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/settings", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.SaveSettings(database).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	ok, hasOk := resp["ok"].(bool)
	if !hasOk || !ok {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}

	// Verify the URL was actually saved
	var savedURL string
	err := database.QueryRow(`SELECT value FROM settings WHERE key='simplefin_access_url'`).Scan(&savedURL)
	if err != nil {
		t.Fatalf("failed to query saved URL: %v", err)
	}
	if savedURL != "https://user:pass@host/simplefin" {
		t.Errorf("expected saved URL, got %q", savedURL)
	}
}

func TestSaveSettings_EmptyURL(t *testing.T) {
	database := setupSettingsDB(t)

	body := strings.NewReader(`{"access_url":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/settings", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.SaveSettings(database).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty access_url, got %d", w.Code)
	}
}

func TestSaveSettings_TriggerSync(t *testing.T) {
	database := setupSettingsDB(t)

	body := strings.NewReader(`{"access_url":"https://user:pass@host/simplefin"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/settings", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.SaveSettings(database).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// The URL was saved; a goroutine was launched to run SyncOnce.
	// SyncOnce will try to fetch from SimpleFIN (which will fail on a fake URL),
	// but it writes a sync_log entry. Give it a moment.
	time.Sleep(200 * time.Millisecond)

	// Verify sync was triggered by checking sync_log
	var count int
	err := database.QueryRow(`SELECT COUNT(*) FROM sync_log`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query sync_log: %v", err)
	}
	if count == 0 {
		t.Error("expected sync_log to have at least one entry after SaveSettings trigger")
	}
}

func TestSyncNow_Success(t *testing.T) {
	database := setupSettingsDB(t)

	// Configure access URL so SyncOnce doesn't no-op immediately.
	// The URL is unreachable but SyncNow still returns ok:true.
	_, err := database.Exec(
		`INSERT INTO settings (key, value) VALUES ('simplefin_access_url', 'https://user:pass@host/simplefin')`,
	)
	if err != nil {
		t.Fatalf("failed to insert access URL: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sync/now", nil)
	w := httptest.NewRecorder()
	handlers.SyncNow(database).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	ok, hasOk := resp["ok"].(bool)
	if !hasOk || !ok {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}

	// Verify restored field is present as an array (empty since no accounts were actually restored)
	restoredRaw, hasRestored := resp["restored"]
	if !hasRestored {
		t.Error("expected 'restored' field in response")
	}
	if restoredArr, ok := restoredRaw.([]interface{}); !ok {
		t.Errorf("expected 'restored' to be an array, got %T", restoredRaw)
	} else if len(restoredArr) != 0 {
		t.Errorf("expected empty restored array, got %v", restoredArr)
	}
}

func TestSyncNow_NoConfig(t *testing.T) {
	database := setupSettingsDB(t)
	// No access URL — SyncOnce should no-op gracefully

	req := httptest.NewRequest(http.MethodPost, "/api/sync/now", nil)
	w := httptest.NewRecorder()
	handlers.SyncNow(database).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	ok, hasOk := resp["ok"].(bool)
	if !hasOk || !ok {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}
}
