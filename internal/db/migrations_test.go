package db_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/solomon/finance-visualizer/internal/db"
)

// openMigratedDB opens a fresh file-based SQLite DB and runs migrations against it.
func openMigratedDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := db.Migrate(dbPath); err != nil {
		t.Fatalf("db.Migrate failed: %v", err)
	}
	return database
}

func tableExists(t *testing.T, database *sql.DB, tableName string) bool {
	t.Helper()
	var name string
	err := database.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
		tableName,
	).Scan(&name)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("failed to query sqlite_master: %v", err)
	}
	return true
}

func indexExists(t *testing.T, database *sql.DB, indexName string) bool {
	t.Helper()
	var name string
	err := database.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='index' AND name=?",
		indexName,
	).Scan(&name)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("failed to query sqlite_master for index: %v", err)
	}
	return true
}

func TestMigrate_CreatesSettingsTable(t *testing.T) {
	database := openMigratedDB(t)
	if !tableExists(t, database, "settings") {
		t.Error("expected 'settings' table to exist after migration")
	}
}

func TestMigrate_CreatesAccountsTable(t *testing.T) {
	database := openMigratedDB(t)
	if !tableExists(t, database, "accounts") {
		t.Error("expected 'accounts' table to exist after migration")
	}

	// Verify key columns exist by inserting a valid row.
	_, err := database.Exec(`INSERT INTO accounts (id, name, account_type, currency) VALUES ('acc-1', 'Test Checking', 'checking', 'USD')`)
	if err != nil {
		t.Errorf("accounts table missing expected columns: %v", err)
	}
}

func TestMigrate_CreatesBalanceSnapshotsTable(t *testing.T) {
	database := openMigratedDB(t)
	if !tableExists(t, database, "balance_snapshots") {
		t.Error("expected 'balance_snapshots' table to exist after migration")
	}

	// Insert a valid account first (FK constraint).
	_, err := database.Exec(`INSERT INTO accounts (id, name, account_type) VALUES ('acc-2', 'Savings', 'savings')`)
	if err != nil {
		t.Fatalf("failed to insert account: %v", err)
	}

	// Insert a valid balance snapshot.
	_, err = database.Exec(`INSERT INTO balance_snapshots (account_id, balance, balance_date) VALUES ('acc-2', '1234.56', '2026-01-01')`)
	if err != nil {
		t.Errorf("balance_snapshots insert failed: %v", err)
	}
}

func TestMigrate_CreatesSyncLogTable(t *testing.T) {
	database := openMigratedDB(t)
	if !tableExists(t, database, "sync_log") {
		t.Error("expected 'sync_log' table to exist after migration")
	}
}

func TestMigrate_CreatesIndex(t *testing.T) {
	database := openMigratedDB(t)
	if !indexExists(t, database, "idx_balance_snapshots_account_date") {
		t.Error("expected index 'idx_balance_snapshots_account_date' to exist after migration")
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "idempotent.db")

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	defer database.Close()

	// First run.
	if err := db.Migrate(dbPath); err != nil {
		t.Fatalf("first Migrate failed: %v", err)
	}

	// Second run should not error (ErrNoChange handled gracefully).
	if err := db.Migrate(dbPath); err != nil {
		t.Errorf("second Migrate returned unexpected error: %v", err)
	}
}

func TestMigrate_AccountTypeCheck(t *testing.T) {
	database := openMigratedDB(t)

	// Inserting an invalid account type should fail the CHECK constraint.
	_, err := database.Exec(`INSERT INTO accounts (id, name, account_type) VALUES ('acc-bad', 'Bad', 'invalid_type')`)
	if err == nil {
		t.Error("expected CHECK constraint violation for invalid account_type, got nil error")
	}
}

// TestMigration006_DeletesStaleDateMismatchRows verifies the cleanup SQL from
// migration 000006. Rows where DATE(fetched_at) != balance_date represent
// snapshots inserted on a day different from the bank's reported balance date
// (the stale-date bug). The migration must delete these and preserve correct ones.
func TestMigration006_DeletesStaleDateMismatchRows(t *testing.T) {
	database := openMigratedDB(t)

	_, err := database.Exec(`INSERT INTO accounts (id, name, account_type) VALUES ('acc-clean', 'Checking', 'checking')`)
	if err != nil {
		t.Fatalf("insert account: %v", err)
	}

	// Row A: balance_date matches the INSERT day (fetched_at) — should be KEPT.
	// Use today's date so DATE(fetched_at) = balance_date.
	_, err = database.Exec(`
		INSERT INTO balance_snapshots (account_id, balance, balance_date, fetched_at)
		VALUES ('acc-clean', '500.00', DATE('now'), CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("insert matching row: %v", err)
	}

	// Row B: balance_date is a past date but fetched_at is today —
	// represents the stale-bank-date bug (sync ran today, bank reported March 1).
	// Should be DELETED.
	_, err = database.Exec(`
		INSERT INTO balance_snapshots (account_id, balance, balance_date, fetched_at)
		VALUES ('acc-clean', '400.00', '2020-01-01', CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("insert mismatched row: %v", err)
	}

	// Run the migration 006 cleanup SQL.
	_, err = database.Exec(`DELETE FROM balance_snapshots WHERE DATE(fetched_at) != balance_date`)
	if err != nil {
		t.Fatalf("cleanup DELETE failed: %v", err)
	}

	// Row A (matching dates) must still exist.
	var countGood int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM balance_snapshots WHERE account_id='acc-clean' AND balance='500.00'`,
	).Scan(&countGood); err != nil {
		t.Fatalf("count good row: %v", err)
	}
	if countGood != 1 {
		t.Errorf("expected matching row to be preserved, got count=%d", countGood)
	}

	// Row B (mismatched dates) must be gone.
	var countBad int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM balance_snapshots WHERE account_id='acc-clean' AND balance='400.00'`,
	).Scan(&countBad); err != nil {
		t.Fatalf("count bad row: %v", err)
	}
	if countBad != 0 {
		t.Errorf("expected mismatched row to be deleted, got count=%d", countBad)
	}
}

func TestMigrate_BalanceSnapshotUniqueConstraint(t *testing.T) {
	database := openMigratedDB(t)

	_, err := database.Exec(`INSERT INTO accounts (id, name, account_type) VALUES ('acc-3', 'Checking', 'checking')`)
	if err != nil {
		t.Fatalf("failed to insert account: %v", err)
	}

	// First snapshot.
	_, err = database.Exec(`INSERT INTO balance_snapshots (account_id, balance, balance_date) VALUES ('acc-3', '100.00', '2026-01-15')`)
	if err != nil {
		t.Fatalf("first snapshot insert failed: %v", err)
	}

	// Second snapshot with same account_id + balance_date should violate the UNIQUE constraint.
	_, err = database.Exec(`INSERT INTO balance_snapshots (account_id, balance, balance_date) VALUES ('acc-3', '200.00', '2026-01-15')`)
	if err == nil {
		t.Error("expected UNIQUE constraint violation for duplicate (account_id, balance_date), got nil error")
	}
}
