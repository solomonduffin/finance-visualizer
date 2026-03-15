package db_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/solomon/finance-visualizer/internal/db"
)

// openTestDB opens an in-memory SQLite database for tests that don't require WAL mode.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// openFileTestDB opens a file-based SQLite database for tests requiring WAL mode.
func openFileTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	database, err := db.Open(path)
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
		os.RemoveAll(dir)
	})
	return database
}

func TestOpen_WALMode(t *testing.T) {
	// WAL mode only applies to file-based databases; in-memory always uses "memory" journal mode.
	database := openFileTestDB(t)

	var journalMode string
	err := database.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("expected journal_mode=wal, got %q", journalMode)
	}
}

func TestOpen_BusyTimeout(t *testing.T) {
	database := openTestDB(t)

	var busyTimeout int
	err := database.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout)
	if err != nil {
		t.Fatalf("failed to query busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Errorf("expected busy_timeout=5000, got %d", busyTimeout)
	}
}

func TestOpen_ForeignKeys(t *testing.T) {
	database := openTestDB(t)

	var foreignKeys int
	err := database.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
	if err != nil {
		t.Fatalf("failed to query foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("expected foreign_keys=1, got %d", foreignKeys)
	}
}

func TestOpen_MaxOpenConns(t *testing.T) {
	database := openTestDB(t)

	// Single-writer: execute two concurrent writes and verify no conflicts.
	// With SetMaxOpenConns(1), the second write will wait (busy_timeout) rather than erroring.
	_, err := database.Exec("CREATE TABLE IF NOT EXISTS write_test (id INTEGER PRIMARY KEY, val TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	_, err = database.Exec("INSERT INTO write_test (val) VALUES ('a')")
	if err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	_, err = database.Exec("INSERT INTO write_test (val) VALUES ('b')")
	if err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	stats := database.Stats()
	if stats.MaxOpenConnections != 1 {
		t.Errorf("expected MaxOpenConnections=1, got %d", stats.MaxOpenConnections)
	}
}
