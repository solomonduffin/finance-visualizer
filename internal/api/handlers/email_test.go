package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/solomon/finance-visualizer/internal/alerts"
	"github.com/solomon/finance-visualizer/internal/api/handlers"
)

const testJWTSecret = "test-secret"

func TestSaveEmailConfig(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/email/config", handlers.SaveEmailConfig(database, testJWTSecret))

	body := strings.NewReader(`{
		"host": "smtp.example.com",
		"port": "587",
		"username": "user@example.com",
		"password": "secret123",
		"from": "alerts@example.com",
		"to": "user@example.com"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/email/config", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}

	// Verify smtp_host was saved.
	var host string
	err := database.QueryRow(`SELECT value FROM settings WHERE key='smtp_host'`).Scan(&host)
	if err != nil {
		t.Fatalf("failed to query smtp_host: %v", err)
	}
	if host != "smtp.example.com" {
		t.Errorf("smtp_host: got %q, want %q", host, "smtp.example.com")
	}

	// Verify smtp_password is encrypted (not plaintext).
	var encryptedPwd string
	err = database.QueryRow(`SELECT value FROM settings WHERE key='smtp_password'`).Scan(&encryptedPwd)
	if err != nil {
		t.Fatalf("failed to query smtp_password: %v", err)
	}
	if encryptedPwd == "secret123" {
		t.Error("smtp_password should be encrypted, but found plaintext")
	}

	// Verify we can decrypt it back.
	key := alerts.DeriveKey(testJWTSecret)
	decrypted, err := alerts.Decrypt(encryptedPwd, key)
	if err != nil {
		t.Fatalf("failed to decrypt smtp_password: %v", err)
	}
	if decrypted != "secret123" {
		t.Errorf("decrypted password: got %q, want %q", decrypted, "secret123")
	}
}

func TestGetEmailConfig_NotConfigured(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Get("/api/email/config", handlers.GetEmailConfig(database))

	req := httptest.NewRequest(http.MethodGet, "/api/email/config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	configured, ok := resp["configured"].(bool)
	if !ok || configured {
		t.Errorf("expected configured=false, got %v", resp["configured"])
	}
}

func TestGetEmailConfig_Configured(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/email/config", handlers.SaveEmailConfig(database, testJWTSecret))
	r.Get("/api/email/config", handlers.GetEmailConfig(database))

	// Save config first.
	saveBody := strings.NewReader(`{
		"host": "smtp.example.com",
		"port": "587",
		"username": "user@example.com",
		"password": "secret123",
		"from": "alerts@example.com",
		"to": "user@example.com"
	}`)
	saveReq := httptest.NewRequest(http.MethodPost, "/api/email/config", saveBody)
	saveReq.Header.Set("Content-Type", "application/json")
	saveW := httptest.NewRecorder()
	r.ServeHTTP(saveW, saveReq)

	// Get config.
	getReq := httptest.NewRequest(http.MethodGet, "/api/email/config", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getW.Code, getW.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(getW.Body.Bytes(), &resp)

	configured, ok := resp["configured"].(bool)
	if !ok || !configured {
		t.Errorf("expected configured=true, got %v", resp["configured"])
	}

	if resp["host"] != "smtp.example.com" {
		t.Errorf("host: got %v, want %q", resp["host"], "smtp.example.com")
	}
	if resp["username"] != "user@example.com" {
		t.Errorf("username: got %v, want %q", resp["username"], "user@example.com")
	}

	// Must NOT return password.
	if _, hasPassword := resp["password"]; hasPassword {
		t.Error("response should NOT include password field")
	}
}

func TestTestEmail_MissingHost(t *testing.T) {
	database := setupFinanceTestDB(t)

	r := chi.NewRouter()
	r.Post("/api/email/test", handlers.TestEmail(database, testJWTSecret))

	body := strings.NewReader(`{
		"host": "",
		"port": "587",
		"username": "user@example.com",
		"password": "secret123",
		"from": "alerts@example.com",
		"to": "user@example.com"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/email/test", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
