package api_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/solomon/finance-visualizer/internal/api"
	"github.com/solomon/finance-visualizer/internal/auth"
	"github.com/solomon/finance-visualizer/internal/db"
	"golang.org/x/crypto/bcrypt"
)

const testSecret = "router-test-secret-key"
const testPassword = "routertestpass"

// setupRouterTestDB creates a temp file SQLite DB with schema and password hash.
func setupRouterTestDB(t *testing.T) *sql.DB {
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
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
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

func TestProtectedRoute_NoAuth(t *testing.T) {
	auth.Init(testSecret)
	database := setupRouterTestDB(t)
	router := api.NewRouter(auth.TokenAuth(), database, "test-secret-key-for-testing")

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", w.Code)
	}
}

func TestProtectedRoute_WithAuth(t *testing.T) {
	auth.Init(testSecret)
	database := setupRouterTestDB(t)
	router := api.NewRouter(auth.TokenAuth(), database, "test-secret-key-for-testing")

	// Get a valid JWT first via login
	loginBody := strings.NewReader(`{"password":"` + testPassword + `"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody)
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)

	if loginW.Code != http.StatusOK {
		t.Fatalf("login failed with status %d; cannot proceed to protected route test", loginW.Code)
	}

	var jwtCookie *http.Cookie
	for _, c := range loginW.Result().Cookies() {
		if c.Name == "jwt" {
			jwtCookie = c
			break
		}
	}
	if jwtCookie == nil {
		t.Fatal("no jwt cookie returned from login")
	}

	// Access protected route with cookie
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.AddCookie(jwtCookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with valid jwt cookie, got %d", w.Code)
	}
}

func TestProtectedRoute_ExpiredToken(t *testing.T) {
	auth.Init(testSecret)
	database := setupRouterTestDB(t)
	router := api.NewRouter(auth.TokenAuth(), database, "test-secret-key-for-testing")

	// Create a token that expired in the past
	ta := auth.TokenAuth()
	claims := map[string]interface{}{
		"exp": time.Now().Add(-1 * time.Hour).UTC().Unix(),
	}
	_, expiredToken, err := ta.Encode(claims)
	if err != nil {
		t.Fatalf("failed to encode expired token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: expiredToken})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with expired token, got %d", w.Code)
	}
}

func TestSettingsRoute_NoAuth(t *testing.T) {
	auth.Init(testSecret)
	database := setupRouterTestDB(t)
	router := api.NewRouter(auth.TokenAuth(), database, "test-secret-key-for-testing")

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth on GET /api/settings, got %d", w.Code)
	}
}

func TestSettingsRoute_WithAuth(t *testing.T) {
	auth.Init(testSecret)
	database := setupRouterTestDB(t)
	router := api.NewRouter(auth.TokenAuth(), database, "test-secret-key-for-testing")

	// Login first to get a JWT cookie
	loginBody := strings.NewReader(`{"password":"` + testPassword + `"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody)
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)
	if loginW.Code != http.StatusOK {
		t.Fatalf("login failed with status %d", loginW.Code)
	}

	var jwtCookie *http.Cookie
	for _, c := range loginW.Result().Cookies() {
		if c.Name == "jwt" {
			jwtCookie = c
			break
		}
	}
	if jwtCookie == nil {
		t.Fatal("no jwt cookie returned from login")
	}

	// Access settings with valid JWT
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	req.AddCookie(jwtCookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on GET /api/settings with auth, got %d", w.Code)
	}

	// Response must contain "configured" field
	body := w.Body.String()
	if !strings.Contains(body, "configured") {
		t.Errorf("expected 'configured' field in response body, got: %s", body)
	}
}

func TestLoginRateLimit(t *testing.T) {
	auth.Init(testSecret)
	database := setupRouterTestDB(t)
	router := api.NewRouter(auth.TokenAuth(), database, "test-secret-key-for-testing")

	// Send 6 login requests rapidly from the same IP — the 6th should be 429
	var lastCode int
	for i := 0; i < 6; i++ {
		body := strings.NewReader(`{"password":"wrongpassword"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.0.2.1:12345" // fixed IP to hit rate limit
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		lastCode = w.Code
	}

	if lastCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 on 6th login attempt, got %d", lastCode)
	}
}
