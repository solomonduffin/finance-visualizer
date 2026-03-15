package db

import (
	"context"
	"database/sql"

	"modernc.org/sqlite"
	_ "modernc.org/sqlite"
)

const initSQL = `
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA foreign_keys = ON;
`

// Open opens a SQLite database at the given path (use ":memory:" for in-memory).
// It registers a connection hook that sets WAL mode, busy_timeout, and foreign keys
// on every connection opened by the pool. SetMaxOpenConns(1) ensures single-writer safety.
func Open(path string) (*sql.DB, error) {
	sqlite.RegisterConnectionHook(func(conn sqlite.ExecQuerierContext, _ string) error {
		_, err := conn.ExecContext(context.Background(), initSQL, nil)
		return err
	})

	dsn := path
	if path != ":memory:" {
		dsn = "file:" + path
	}

	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Single writer — prevents "database is locked" under concurrent writes.
	database.SetMaxOpenConns(1)

	// Ping to verify the connection and apply the hook immediately.
	if err := database.Ping(); err != nil {
		database.Close()
		return nil, err
	}

	return database, nil
}
