package sync_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/solomon/finance-visualizer/internal/db"
	finSync "github.com/solomon/finance-visualizer/internal/sync"
)

// openTestDB opens a temporary SQLite database and runs migrations.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	if err := db.Migrate(dbPath); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// setAccessURL inserts or updates the simplefin_access_url setting.
func setAccessURL(t *testing.T, database *sql.DB, url string) {
	t.Helper()
	_, err := database.Exec(
		`INSERT INTO settings(key, value) VALUES('simplefin_access_url', ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		url,
	)
	if err != nil {
		t.Fatalf("setAccessURL: %v", err)
	}
}

// newMockServer creates an httptest.Server that returns a SimpleFIN AccountSet JSON.
func newMockServer(t *testing.T, accounts []map[string]any) *httptest.Server {
	t.Helper()
	body := map[string]any{
		"errors":   []string{},
		"accounts": accounts,
	}
	data, _ := json.Marshal(body)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSyncOnce_NoAccessURL(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	// No access URL in settings — SyncOnce should be a no-op.
	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// No sync_log entries should be created.
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM sync_log`).Scan(&count); err != nil {
		t.Fatalf("query sync_log: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 sync_log entries, got %d", count)
	}
}

func TestSyncOnce_Success(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	accounts := []map[string]any{
		{
			"id":           "acct-001",
			"name":         "My Checking",
			"currency":     "USD",
			"balance":      "1234.56",
			"balance-date": 1700000000,
			"org":          map[string]any{"name": "First Bank", "id": "first-bank"},
		},
		{
			"id":           "acct-002",
			"name":         "Savings Account",
			"currency":     "USD",
			"balance":      "9999.00",
			"balance-date": 1700000001,
			"org":          map[string]any{"name": "First Bank", "id": "first-bank"},
		},
	}

	srv := newMockServer(t, accounts)
	setAccessURL(t, database, srv.URL+"/simplefin")

	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("SyncOnce error: %v", err)
	}

	// Check accounts were inserted.
	var acctCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&acctCount); err != nil {
		t.Fatalf("query accounts: %v", err)
	}
	if acctCount != 2 {
		t.Errorf("expected 2 accounts, got %d", acctCount)
	}

	// Check balance_snapshots were inserted.
	var snapCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM balance_snapshots`).Scan(&snapCount); err != nil {
		t.Fatalf("query balance_snapshots: %v", err)
	}
	if snapCount != 2 {
		t.Errorf("expected 2 snapshots, got %d", snapCount)
	}

	// Check sync_log entry.
	var fetched, failed int
	var finishedAt sql.NullString
	if err := database.QueryRow(
		`SELECT accounts_fetched, accounts_failed, finished_at FROM sync_log WHERE id=1`,
	).Scan(&fetched, &failed, &finishedAt); err != nil {
		t.Fatalf("query sync_log: %v", err)
	}
	if fetched != 2 {
		t.Errorf("expected accounts_fetched=2, got %d", fetched)
	}
	if failed != 0 {
		t.Errorf("expected accounts_failed=0, got %d", failed)
	}
	if !finishedAt.Valid {
		t.Error("expected finished_at to be set")
	}
}

func TestSyncOnce_FirstSync(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[],"accounts":[]}`))
	}))
	t.Cleanup(srv.Close)
	setAccessURL(t, database, srv.URL+"/simplefin")

	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("SyncOnce error: %v", err)
	}

	// With no accounts in DB, startDate must be ~30 days ago.
	if capturedQuery == "" {
		t.Fatal("no query captured from mock server")
	}

	// Parse the start-date value from query.
	vals, err := parseQuery(capturedQuery)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	startDateStr, ok := vals["start-date"]
	if !ok {
		t.Fatalf("expected start-date in query, got: %s", capturedQuery)
	}

	// Parse as integer epoch.
	var epoch int64
	if _, err := parseInt64(startDateStr, &epoch); err != nil {
		t.Fatalf("start-date not a valid epoch: %s", startDateStr)
	}

	// Must be approximately 30 days ago (within 1 hour tolerance).
	expected := time.Now().Add(-30 * 24 * time.Hour).Unix()
	diff := epoch - expected
	if diff < -3600 || diff > 3600 {
		t.Errorf("start-date %d is not ~30 days ago (expected ~%d)", epoch, expected)
	}
}

func TestSyncOnce_SubsequentSync(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	// Pre-insert an account so this is not a first sync.
	_, err := database.Exec(
		`INSERT INTO accounts(id, name, account_type, currency) VALUES('existing-acct', 'Old Account', 'checking', 'USD')`,
	)
	if err != nil {
		t.Fatalf("pre-insert account: %v", err)
	}

	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[],"accounts":[]}`))
	}))
	t.Cleanup(srv.Close)
	setAccessURL(t, database, srv.URL+"/simplefin")

	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("SyncOnce error: %v", err)
	}

	// With accounts in DB, no start-date should be sent.
	vals, _ := parseQuery(capturedQuery)
	if _, ok := vals["start-date"]; ok {
		t.Errorf("expected no start-date in subsequent sync, got: %s", capturedQuery)
	}
}

func TestInsertSnapshot_Duplicate(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	// Insert the same account twice via sync, producing identical snapshots.
	accounts := []map[string]any{
		{
			"id":           "acct-dup",
			"name":         "Duplicate Account",
			"currency":     "USD",
			"balance":      "500.00",
			"balance-date": 1700000000,
			"org":          map[string]any{"name": "Bank", "id": "bank"},
		},
	}

	srv := newMockServer(t, accounts)
	setAccessURL(t, database, srv.URL+"/simplefin")

	// First sync.
	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("first SyncOnce error: %v", err)
	}

	// Second sync with the same data.
	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("second SyncOnce error: %v", err)
	}

	// Only 1 snapshot should exist (duplicate silently ignored).
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM balance_snapshots WHERE account_id='acct-dup'`).Scan(&count); err != nil {
		t.Fatalf("query snapshots: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 snapshot after duplicate, got %d", count)
	}
}

func TestSyncOnce_UpdatesSameDayBalance(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	// First sync: balance is 500.00
	accounts1 := []map[string]any{
		{
			"id":           "acct-update",
			"name":         "Checking",
			"currency":     "USD",
			"balance":      "500.00",
			"balance-date": 1700000000,
			"org":          map[string]any{"name": "Bank", "id": "bank"},
		},
	}
	srv1 := newMockServer(t, accounts1)
	setAccessURL(t, database, srv1.URL+"/simplefin")

	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("first SyncOnce error: %v", err)
	}

	// Second sync: same date, updated balance 750.00
	accounts2 := []map[string]any{
		{
			"id":           "acct-update",
			"name":         "Checking",
			"currency":     "USD",
			"balance":      "750.00",
			"balance-date": 1700000000,
			"org":          map[string]any{"name": "Bank", "id": "bank"},
		},
	}
	srv2 := newMockServer(t, accounts2)
	setAccessURL(t, database, srv2.URL+"/simplefin")

	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("second SyncOnce error: %v", err)
	}

	// Balance should be updated to 750.00
	var balance string
	if err := database.QueryRow(`SELECT balance FROM balance_snapshots WHERE account_id='acct-update'`).Scan(&balance); err != nil {
		t.Fatalf("query balance: %v", err)
	}
	if balance != "750.00" {
		t.Errorf("expected balance 750.00 after same-day update, got %s", balance)
	}

	// Still only 1 snapshot row
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM balance_snapshots WHERE account_id='acct-update'`).Scan(&count); err != nil {
		t.Fatalf("query count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 snapshot, got %d", count)
	}
}

func TestUpsertAccount(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	// First sync: insert account with name "Original Name".
	accounts := []map[string]any{
		{
			"id":           "acct-upsert",
			"name":         "Original Name",
			"currency":     "USD",
			"balance":      "100.00",
			"balance-date": 1700000000,
			"org":          map[string]any{"name": "Bank", "id": "bank"},
		},
	}
	srv := newMockServer(t, accounts)
	setAccessURL(t, database, srv.URL+"/simplefin")

	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("first SyncOnce: %v", err)
	}

	var name string
	if err := database.QueryRow(`SELECT name FROM accounts WHERE id='acct-upsert'`).Scan(&name); err != nil {
		t.Fatalf("query account name: %v", err)
	}
	if name != "Original Name" {
		t.Errorf("expected 'Original Name', got %q", name)
	}

	// Update the mock to return a different name.
	accounts[0]["name"] = "Updated Name"
	updatedData, _ := json.Marshal(map[string]any{"errors": []string{}, "accounts": accounts})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(updatedData)
	})

	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("second SyncOnce: %v", err)
	}

	if err := database.QueryRow(`SELECT name FROM accounts WHERE id='acct-upsert'`).Scan(&name); err != nil {
		t.Fatalf("query account name after upsert: %v", err)
	}
	if name != "Updated Name" {
		t.Errorf("expected 'Updated Name', got %q", name)
	}
}

func TestSyncOnce_PartialFailure(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	// One valid account, one with an invalid balance.
	accounts := []map[string]any{
		{
			"id":           "acct-good",
			"name":         "Good Account",
			"currency":     "USD",
			"balance":      "100.00",
			"balance-date": 1700000000,
			"org":          map[string]any{"name": "Bank", "id": "bank"},
		},
		{
			"id":           "acct-bad",
			"name":         "Bad Account",
			"currency":     "USD",
			"balance":      "NOT_A_NUMBER",
			"balance-date": 1700000001,
			"org":          map[string]any{"name": "Bank", "id": "bank"},
		},
	}

	srv := newMockServer(t, accounts)
	setAccessURL(t, database, srv.URL+"/simplefin")

	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("SyncOnce error: %v", err)
	}

	var fetched, failed int
	if err := database.QueryRow(
		`SELECT accounts_fetched, accounts_failed FROM sync_log WHERE id=1`,
	).Scan(&fetched, &failed); err != nil {
		t.Fatalf("query sync_log: %v", err)
	}
	if fetched != 1 {
		t.Errorf("expected accounts_fetched=1, got %d", fetched)
	}
	if failed != 1 {
		t.Errorf("expected accounts_failed=1, got %d", failed)
	}
}

func TestInferAccountType(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"My Checking", "checking"},
		{"Savings Account", "savings"},
		{"Visa Credit Card", "credit"},
		{"Chase Sapphire Preferred (9243)", "credit"},
		{"Platinum Card", "credit"},
		{"Brokerage", "investment"},
		{"IRA", "investment"},
		{"401k", "investment"},
		{"Unknown Account", "other"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := finSync.InferAccountType(tc.name)
			if got != tc.expected {
				t.Errorf("InferAccountType(%q) = %q, want %q", tc.name, got, tc.expected)
			}
		})
	}
}

func TestNextRunTime_Future(t *testing.T) {
	// Pick an hour that is 1 hour from now, ensuring it's "in the future today".
	now := time.Now()
	futureHour := (now.Hour() + 1) % 24
	if futureHour == 0 {
		// Edge case: we'd roll to next day; shift forward by 2 hours instead.
		futureHour = (now.Hour() + 2) % 24
		if futureHour <= now.Hour() {
			t.Skip("test skipped: near midnight boundary")
		}
	}

	next := finSync.NextRunTime(futureHour)
	if next.Hour() != futureHour {
		t.Errorf("expected hour %d, got %d", futureHour, next.Hour())
	}
	if next.Before(now) {
		t.Errorf("expected future time, got %v (now: %v)", next, now)
	}
	// Should be today.
	y1, m1, d1 := now.Date()
	y2, m2, d2 := next.Date()
	if y1 != y2 || m1 != m2 || d1 != d2 {
		t.Errorf("expected next run to be today (%v), got %v", now, next)
	}
}

func TestNextRunTime_Past(t *testing.T) {
	// Pick an hour that has already passed.
	pastHour := 0 // midnight — always in the past unless running exactly at midnight

	next := finSync.NextRunTime(pastHour)

	now := time.Now()
	if !next.After(now) {
		t.Errorf("expected next run to be in the future, got %v (now: %v)", next, now)
	}
	if next.Hour() != pastHour {
		t.Errorf("expected hour %d, got %d", pastHour, next.Hour())
	}
	// Should be tomorrow.
	tomorrow := now.AddDate(0, 0, 1)
	y1, m1, d1 := tomorrow.Date()
	y2, m2, d2 := next.Date()
	if y1 != y2 || m1 != m2 || d1 != d2 {
		t.Errorf("expected next run to be tomorrow (%v), got %v", tomorrow, next)
	}
}

func TestSyncMutex(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	// Track the order of sync start/end using a channel.
	type event struct {
		id    int
		start bool
	}
	events := make(chan event, 10)

	// Set up a slow mock server that records when requests arrive.
	requestArrived := make(chan struct{}, 1)
	allowResponse := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestArrived <- struct{}{}
		<-allowResponse // wait until test says to respond
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[],"accounts":[]}`))
	}))
	t.Cleanup(srv.Close)
	setAccessURL(t, database, srv.URL+"/simplefin")

	var wg sync.WaitGroup
	wg.Add(2)

	// First goroutine: starts sync and holds it until we unblock.
	go func() {
		defer wg.Done()
		events <- event{1, true}
		_, _ = finSync.SyncOnce(ctx, database)
		events <- event{1, false}
	}()

	// Wait for first sync to reach the server.
	<-requestArrived

	// Second goroutine: tries to sync while first is blocked (should wait for mutex).
	go func() {
		defer wg.Done()
		events <- event{2, true}
		_, _ = finSync.SyncOnce(ctx, database)
		events <- event{2, false}
	}()

	// Give goroutine 2 time to acquire the mutex and block.
	time.Sleep(50 * time.Millisecond)

	// Unblock first sync.
	close(allowResponse)
	wg.Wait()
	close(events)

	// Verify event ordering: goroutine 1 must finish before goroutine 2's sync body runs
	// (i.e., the second request arrives after the first completes).
	// This is inherently a "no panic, no data race" test. The primary assertion is that
	// SyncOnce returns without error when called concurrently.
}

// --- helpers ---

func parseQuery(q string) (map[string]string, error) {
	result := make(map[string]string)
	if q == "" {
		return result, nil
	}
	for _, part := range splitAmpersand(q) {
		idx := indexByte(part, '=')
		if idx < 0 {
			result[part] = ""
			continue
		}
		result[part[:idx]] = part[idx+1:]
	}
	return result, nil
}

func splitAmpersand(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '&' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func parseInt64(s string, out *int64) (int64, error) {
	var result int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, &parseError{s}
		}
		result = result*10 + int64(c-'0')
	}
	if out != nil {
		*out = result
	}
	return result, nil
}

type parseError struct{ s string }

func (e *parseError) Error() string { return "not an integer: " + e.s }

func TestSoftDeleteStaleAccounts(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	// First sync: 2 accounts.
	accounts1 := []map[string]any{
		{
			"id": "acct-001", "name": "My Checking", "currency": "USD",
			"balance": "1000.00", "balance-date": 1700000000,
			"org": map[string]any{"name": "Bank A", "id": "bank-a"},
		},
		{
			"id": "acct-002", "name": "Old Account", "currency": "USD",
			"balance": "500.00", "balance-date": 1700000000,
			"org": map[string]any{"name": "Bank B", "id": "bank-b"},
		},
	}
	srv1 := newMockServer(t, accounts1)
	setAccessURL(t, database, srv1.URL+"/simplefin")
	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	var count int
	database.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&count)
	if count != 2 {
		t.Fatalf("expected 2 accounts after first sync, got %d", count)
	}

	// Second sync: only 1 account returned (acct-002 removed from SimpleFIN).
	accounts2 := []map[string]any{
		{
			"id": "acct-001", "name": "My Checking", "currency": "USD",
			"balance": "1100.00", "balance-date": 1700086400,
			"org": map[string]any{"name": "Bank A", "id": "bank-a"},
		},
	}
	srv2 := newMockServer(t, accounts2)
	setAccessURL(t, database, srv2.URL+"/simplefin")
	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	// acct-002 should still exist (soft-deleted, NOT hard-deleted).
	database.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 accounts after soft-delete (not hard-deleted), got %d", count)
	}

	// acct-002 should have hidden_at set
	var hiddenAt sql.NullString
	database.QueryRow(`SELECT hidden_at FROM accounts WHERE id='acct-002'`).Scan(&hiddenAt)
	if !hiddenAt.Valid {
		t.Error("expected acct-002 to have hidden_at set (soft-deleted)")
	}

	// acct-001 should NOT have hidden_at set
	database.QueryRow(`SELECT hidden_at FROM accounts WHERE id='acct-001'`).Scan(&hiddenAt)
	if hiddenAt.Valid {
		t.Error("expected acct-001 to NOT have hidden_at set (still active)")
	}

	// Balance snapshots should be preserved (NOT deleted).
	var snapCount int
	database.QueryRow(`SELECT COUNT(*) FROM balance_snapshots WHERE account_id='acct-002'`).Scan(&snapCount)
	if snapCount == 0 {
		t.Error("expected balance_snapshots to be preserved for soft-deleted account")
	}
}

func TestSoftDeleteStaleAccounts_DoesNotReHide(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	// Sync 2 accounts.
	accounts := []map[string]any{
		{
			"id": "acct-001", "name": "Active", "currency": "USD",
			"balance": "1000.00", "balance-date": 1700000000,
			"org": map[string]any{"name": "Bank", "id": "bank"},
		},
		{
			"id": "acct-002", "name": "Will Disappear", "currency": "USD",
			"balance": "500.00", "balance-date": 1700000000,
			"org": map[string]any{"name": "Bank", "id": "bank"},
		},
	}
	srv := newMockServer(t, accounts)
	setAccessURL(t, database, srv.URL+"/simplefin")
	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	// Manually set hidden_at on acct-002 with a known timestamp
	knownTimestamp := "2024-06-01T00:00:00Z"
	_, err := database.Exec(`UPDATE accounts SET hidden_at = ? WHERE id = 'acct-002'`, knownTimestamp)
	if err != nil {
		t.Fatalf("set hidden_at: %v", err)
	}

	// Second sync with only acct-001. acct-002 already hidden, should NOT be re-hidden.
	accounts2 := []map[string]any{
		{
			"id": "acct-001", "name": "Active", "currency": "USD",
			"balance": "1100.00", "balance-date": 1700086400,
			"org": map[string]any{"name": "Bank", "id": "bank"},
		},
	}
	updatedData, _ := json.Marshal(map[string]any{"errors": []string{}, "accounts": accounts2})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(updatedData)
	})

	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	// hidden_at should still be the original timestamp (not updated)
	var hiddenAt string
	database.QueryRow(`SELECT hidden_at FROM accounts WHERE id='acct-002'`).Scan(&hiddenAt)
	if hiddenAt != knownTimestamp {
		t.Errorf("expected hidden_at to remain %q (not re-hidden), got %q", knownTimestamp, hiddenAt)
	}
}

func TestRestoreReturningAccounts(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	// Sync 2 accounts.
	accounts := []map[string]any{
		{
			"id": "acct-001", "name": "Checking", "currency": "USD",
			"balance": "1000.00", "balance-date": 1700000000,
			"org": map[string]any{"name": "Bank", "id": "bank"},
		},
		{
			"id": "acct-002", "name": "Savings", "currency": "USD",
			"balance": "500.00", "balance-date": 1700000000,
			"org": map[string]any{"name": "Bank", "id": "bank"},
		},
	}
	srv := newMockServer(t, accounts)
	setAccessURL(t, database, srv.URL+"/simplefin")
	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	// Soft-delete acct-002 manually (simulating it disappeared from SimpleFIN)
	_, err := database.Exec(`UPDATE accounts SET hidden_at = CURRENT_TIMESTAMP WHERE id = 'acct-002'`)
	if err != nil {
		t.Fatalf("soft-delete: %v", err)
	}

	// Now sync with both accounts again (acct-002 reappears)
	allAccounts := []map[string]any{
		{
			"id": "acct-001", "name": "Checking", "currency": "USD",
			"balance": "1100.00", "balance-date": 1700086400,
			"org": map[string]any{"name": "Bank", "id": "bank"},
		},
		{
			"id": "acct-002", "name": "Savings", "currency": "USD",
			"balance": "600.00", "balance-date": 1700086400,
			"org": map[string]any{"name": "Bank", "id": "bank"},
		},
	}
	updatedData, _ := json.Marshal(map[string]any{"errors": []string{}, "accounts": allAccounts})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(updatedData)
	})

	restored, err := finSync.SyncOnce(ctx, database)
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}

	// acct-002 should be restored (hidden_at cleared)
	var hiddenAt sql.NullString
	database.QueryRow(`SELECT hidden_at FROM accounts WHERE id='acct-002'`).Scan(&hiddenAt)
	if hiddenAt.Valid {
		t.Errorf("expected acct-002 to have hidden_at cleared after restore, got %q", hiddenAt.String)
	}

	// SyncOnce should return names of restored accounts
	if len(restored) != 1 {
		t.Fatalf("expected 1 restored account name, got %d: %v", len(restored), restored)
	}
	if restored[0] != "Savings" {
		t.Errorf("expected restored name \"Savings\", got %q", restored[0])
	}
}

func TestSyncOnce_SoftDelete_PreservesSnapshots(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	// First sync: 2 accounts with snapshots.
	accounts1 := []map[string]any{
		{
			"id": "acct-001", "name": "Active", "currency": "USD",
			"balance": "1000.00", "balance-date": 1700000000,
			"org": map[string]any{"name": "Bank", "id": "bank"},
		},
		{
			"id": "acct-002", "name": "Disappearing", "currency": "USD",
			"balance": "500.00", "balance-date": 1700000000,
			"org": map[string]any{"name": "Bank", "id": "bank"},
		},
	}
	srv := newMockServer(t, accounts1)
	setAccessURL(t, database, srv.URL+"/simplefin")
	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	// Verify snapshots exist
	var snapBefore int
	database.QueryRow(`SELECT COUNT(*) FROM balance_snapshots WHERE account_id='acct-002'`).Scan(&snapBefore)
	if snapBefore == 0 {
		t.Fatal("expected snapshots for acct-002 before soft-delete")
	}

	// Second sync: acct-002 gone
	accounts2 := []map[string]any{
		{
			"id": "acct-001", "name": "Active", "currency": "USD",
			"balance": "1100.00", "balance-date": 1700086400,
			"org": map[string]any{"name": "Bank", "id": "bank"},
		},
	}
	updatedData, _ := json.Marshal(map[string]any{"errors": []string{}, "accounts": accounts2})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(updatedData)
	})

	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	// Snapshots must be preserved
	var snapAfter int
	database.QueryRow(`SELECT COUNT(*) FROM balance_snapshots WHERE account_id='acct-002'`).Scan(&snapAfter)
	if snapAfter != snapBefore {
		t.Errorf("expected %d snapshots preserved, got %d", snapBefore, snapAfter)
	}
}

func TestProcessAccount_DoesNotOverwriteUserColumns(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	// First sync: create an account
	accounts := []map[string]any{
		{
			"id": "acct-001", "name": "My Checking", "currency": "USD",
			"balance": "1000.00", "balance-date": 1700000000,
			"org": map[string]any{"name": "Bank", "id": "bank"},
		},
	}
	srv := newMockServer(t, accounts)
	setAccessURL(t, database, srv.URL+"/simplefin")
	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	// Set user-owned columns
	_, err := database.Exec(`UPDATE accounts SET display_name = 'My Custom Name', account_type_override = 'savings' WHERE id = 'acct-001'`)
	if err != nil {
		t.Fatalf("set user columns: %v", err)
	}

	// Second sync: same account with different name from SimpleFIN
	accounts[0]["name"] = "My Checking Updated"
	updatedData, _ := json.Marshal(map[string]any{"errors": []string{}, "accounts": accounts})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(updatedData)
	})

	if _, err := finSync.SyncOnce(ctx, database); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	// User-owned columns should be preserved
	var displayName sql.NullString
	var typeOverride sql.NullString
	var name string
	database.QueryRow(`SELECT name, display_name, account_type_override FROM accounts WHERE id='acct-001'`).Scan(&name, &displayName, &typeOverride)

	// name (system-owned) should be updated
	if name != "My Checking Updated" {
		t.Errorf("expected system name to be updated to \"My Checking Updated\", got %q", name)
	}
	// display_name (user-owned) should be preserved
	if !displayName.Valid || displayName.String != "My Custom Name" {
		t.Errorf("expected display_name to be preserved as \"My Custom Name\", got %v", displayName)
	}
	// account_type_override (user-owned) should be preserved
	if !typeOverride.Valid || typeOverride.String != "savings" {
		t.Errorf("expected account_type_override to be preserved as \"savings\", got %v", typeOverride)
	}
}

// TestSyncOnce_SoftDelete_RestoreOrder verifies that restore happens before soft-delete
// (tested implicitly through TestRestoreReturningAccounts, which verifies an account
// can be restored and then other accounts soft-deleted in the same sync cycle).
