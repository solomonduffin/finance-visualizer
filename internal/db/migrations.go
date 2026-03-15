package db

import (
	"embed"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite" // modernc-backed driver, NOT sqlite3
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate runs all pending database migrations against the SQLite database at dbPath.
// It is safe to call multiple times — ErrNoChange is handled gracefully.
// Uses embedded SQL files from internal/db/migrations/*.sql via go:embed.
func Migrate(dbPath string) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}

	dsn := dbPath
	if dbPath != ":memory:" {
		dsn = "file:" + dbPath
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, "sqlite://"+dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
