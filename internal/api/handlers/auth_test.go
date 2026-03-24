package handlers_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/solomon/finance-visualizer/internal/api/handlers"
	"github.com/solomon/finance-visualizer/internal/auth"
	"github.com/solomon/finance-visualizer/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// setupTestDB creates a temp file SQLite DB with the schema migrated and
// inserts a password_hash into the settings table.
func setupTestDB(t *testing.T, password string) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err := db.Migrate(dbPath); err != nil {
		t.Fatalf("failed to migrate test DB: %v", err)
	}

	// Insert password hash into settings table
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}
	_, err = database.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('password_hash', ?)`, string(hash))
	if err != nil {
		t.Fatalf("failed to insert password hash: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestLoginHandler_Success(t *testing.T) {
	auth.Init("test-secret")
	database := setupTestDB(t, "correctpassword")

	body := strings.NewReader(`{"password":"correctpassword"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.Login(database)(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var jwtCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "jwt" {
			jwtCookie = c
			break
		}
	}
	if jwtCookie == nil {
		t.Fatal("expected 'jwt' cookie in response, got none")
	}
	if !jwtCookie.HttpOnly {
		t.Error("jwt cookie should be HttpOnly")
	}
	if jwtCookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("jwt cookie SameSite should be Strict, got %v", jwtCookie.SameSite)
	}
	if jwtCookie.MaxAge <= 0 {
		t.Errorf("jwt cookie MaxAge should be positive, got %d", jwtCookie.MaxAge)
	}
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	auth.Init("test-secret")
	database := setupTestDB(t, "correctpassword")

	body := strings.NewReader(`{"password":"wrongpassword"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.Login(database)(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		if c.Name == "jwt" {
			t.Error("unexpected jwt cookie in response for wrong password")
		}
	}
}

func TestLoginHandler_EmptyBody(t *testing.T) {
	auth.Init("test-secret")
	database := setupTestDB(t, "correctpassword")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.Login(database)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty body, got %d", w.Code)
	}
}

// TestPasswordHashUpsert_UpdatesOnRestart verifies that the password hash upsert
// SQL (as used in cmd/server/main.go) replaces the stored hash when the env var changes.
// This is a regression test for a bug where INSERT OR IGNORE silently ignored new passwords.
func TestPasswordHashUpsert_UpdatesOnRestart(t *testing.T) {
	auth.Init("test-secret")
	database := setupTestDB(t, "oldpassword")

	// Verify old password works
	body := strings.NewReader(`{"password":"oldpassword"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	w := httptest.NewRecorder()
	handlers.Login(database)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for old password, got %d", w.Code)
	}

	// Simulate "restart with new password" by upserting the hash using ON CONFLICT DO UPDATE
	// (this is the fixed SQL from cmd/server/main.go)
	newHash, err := bcrypt.GenerateFromPassword([]byte("newpassword"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash new password: %v", err)
	}
	_, err = database.Exec(
		`INSERT INTO settings (key, value) VALUES ('password_hash', ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		string(newHash),
	)
	if err != nil {
		t.Fatalf("upsert password hash: %v", err)
	}

	// Old password should now fail
	body = strings.NewReader(`{"password":"oldpassword"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	w = httptest.NewRecorder()
	handlers.Login(database)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for old password after upsert, got %d", w.Code)
	}

	// New password should work
	body = strings.NewReader(`{"password":"newpassword"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	w = httptest.NewRecorder()
	handlers.Login(database)(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for new password after upsert, got %d", w.Code)
	}
}

func TestLoginHandler_MalformedJSON(t *testing.T) {
	auth.Init("test-secret")
	database := setupTestDB(t, "correctpassword")

	body := strings.NewReader(`{not valid json}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.Login(database)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for malformed JSON, got %d", w.Code)
	}
}
