package simplefin_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/solomon/finance-visualizer/internal/simplefin"
)

// sampleAccountSet is valid SimpleFIN JSON with two accounts.
var sampleAccountSet = `{
  "errors": [],
  "accounts": [
    {
      "id": "acct-001",
      "name": "My Checking",
      "currency": "USD",
      "balance": "1234.56",
      "balance-date": 1700000000,
      "org": {
        "name": "First Bank",
        "id": "first-bank"
      }
    },
    {
      "id": "acct-002",
      "name": "Savings Account",
      "currency": "USD",
      "balance": "9999.00",
      "balance-date": 1700000001,
      "org": {
        "name": "First Bank",
        "id": "first-bank"
      }
    }
  ]
}`

func TestFetchAccounts_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleAccountSet))
	}))
	defer srv.Close()

	result, err := simplefin.FetchAccounts(srv.URL+"/accounts", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(result.Accounts))
	}

	a := result.Accounts[0]
	if a.ID != "acct-001" {
		t.Errorf("expected ID acct-001, got %s", a.ID)
	}
	if a.Name != "My Checking" {
		t.Errorf("expected name 'My Checking', got %s", a.Name)
	}
	if a.Currency != "USD" {
		t.Errorf("expected currency USD, got %s", a.Currency)
	}
	if a.Balance != "1234.56" {
		t.Errorf("expected balance 1234.56, got %s", a.Balance)
	}
	if a.BalanceDate != 1700000000 {
		t.Errorf("expected balance-date 1700000000, got %d", a.BalanceDate)
	}
	if a.Org.Name != "First Bank" {
		t.Errorf("expected org name 'First Bank', got %s", a.Org.Name)
	}
	if a.Org.ID != "first-bank" {
		t.Errorf("expected org id 'first-bank', got %s", a.Org.ID)
	}
}

func TestFetchAccounts_WithStartDate(t *testing.T) {
	var capturedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errors":[],"accounts":[]}`))
	}))
	defer srv.Close()

	startDate := time.Unix(1700000000, 0)
	_, err := simplefin.FetchAccounts(srv.URL+"/accounts", &startDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedURL, "start-date=1700000000") {
		t.Errorf("expected start-date=1700000000 in URL, got: %s", capturedURL)
	}
}

func TestFetchAccounts_BalancesOnly(t *testing.T) {
	var capturedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errors":[],"accounts":[]}`))
	}))
	defer srv.Close()

	_, err := simplefin.FetchAccounts(srv.URL+"/accounts", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedURL, "balances-only=1") {
		t.Errorf("expected balances-only=1 in URL, got: %s", capturedURL)
	}
}

func TestFetchAccounts_BasicAuth(t *testing.T) {
	var capturedAuthHeader string
	var capturedURLHost string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get("Authorization")
		capturedURLHost = r.Host
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errors":[],"accounts":[]}`))
	}))
	defer srv.Close()

	// Build a URL with embedded credentials, pointing to our test server.
	// Replace http:// with http://user:pass@
	accessURL := strings.Replace(srv.URL, "http://", "http://myuser:mypass@", 1) + "/accounts"

	_, err := simplefin.FetchAccounts(accessURL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The Authorization header must be Basic auth with correct credentials.
	expectedCreds := base64.StdEncoding.EncodeToString([]byte("myuser:mypass"))
	expectedAuth := "Basic " + expectedCreds
	if capturedAuthHeader != expectedAuth {
		t.Errorf("expected Authorization header %q, got %q", expectedAuth, capturedAuthHeader)
	}

	// The URL sent must NOT contain credentials in the host.
	if strings.Contains(capturedURLHost, "myuser") || strings.Contains(capturedURLHost, "mypass") {
		t.Errorf("credentials should not appear in request host, got: %s", capturedURLHost)
	}
}

func TestFetchAccounts_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := simplefin.FetchAccounts(srv.URL+"/accounts", nil)
	if err == nil {
		t.Fatal("expected error on 403 response, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected error to contain '403', got: %v", err)
	}
}

func TestFetchAccounts_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{ invalid json }`))
	}))
	defer srv.Close()

	_, err := simplefin.FetchAccounts(srv.URL+"/accounts", nil)
	if err == nil {
		t.Fatal("expected error on invalid JSON, got nil")
	}
}

func TestFetchAccounts_NetworkError(t *testing.T) {
	// Use a URL that won't connect (port 1 is reserved/unreachable).
	_, err := simplefin.FetchAccounts("http://127.0.0.1:1/accounts", nil)
	if err == nil {
		t.Fatal("expected error on network failure, got nil")
	}
}

func TestClaimSetupToken_Success(t *testing.T) {
	// Fake access URL that the claim endpoint returns.
	fakeAccessURL := "https://user:pass@beta-bridge.simplefin.org/simplefin"

	claimSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fakeAccessURL))
	}))
	defer claimSrv.Close()

	// Encode the claim server URL as a base64 setup token.
	setupToken := base64.StdEncoding.EncodeToString([]byte(claimSrv.URL))

	got, err := simplefin.ClaimSetupToken(setupToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != fakeAccessURL {
		t.Errorf("expected access URL %q, got %q", fakeAccessURL, got)
	}
}

func TestClaimSetupToken_InvalidBase64(t *testing.T) {
	_, err := simplefin.ClaimSetupToken("not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
	if !strings.Contains(err.Error(), "base64") {
		t.Errorf("expected error to mention base64, got: %v", err)
	}
}

func TestClaimSetupToken_ClaimFails(t *testing.T) {
	claimSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("token already claimed"))
	}))
	defer claimSrv.Close()

	setupToken := base64.StdEncoding.EncodeToString([]byte(claimSrv.URL))
	_, err := simplefin.ClaimSetupToken(setupToken)
	if err == nil {
		t.Fatal("expected error on 403 response, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected error to contain '403', got: %v", err)
	}
}

func TestIsSetupToken(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"aHR0cHM6Ly9iZXRhLWJyaWRnZS5zaW1wbGVmaW4ub3Jn", true},
		{"https://user:pass@beta-bridge.simplefin.org/simplefin", false},
		{"http://localhost:8080/simplefin", false},
		{"", false},
		{"  aHR0cHM6Ly9iZXRhLWJyaWRnZS5zaW1wbGVmaW4ub3Jn  ", true},
	}
	for _, tt := range tests {
		got := simplefin.IsSetupToken(tt.input)
		if got != tt.expected {
			t.Errorf("IsSetupToken(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

// Ensure JSON marshalling round-trips the struct correctly (types check).
func TestAccountSet_JSONTypes(t *testing.T) {
	var as simplefin.AccountSet
	if err := json.Unmarshal([]byte(sampleAccountSet), &as); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if len(as.Accounts) != 2 {
		t.Fatalf("expected 2 accounts after unmarshal, got %d", len(as.Accounts))
	}
	if as.Accounts[0].BalanceDate != 1700000000 {
		t.Errorf("expected BalanceDate 1700000000, got %d", as.Accounts[0].BalanceDate)
	}
}
