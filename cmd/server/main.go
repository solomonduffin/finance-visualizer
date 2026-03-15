package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
	"github.com/solomon/finance-visualizer/internal/config"
	"github.com/solomon/finance-visualizer/internal/db"
)

func main() {
	// Initialize colorized structured logging for development.
	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen,
	}))
	slog.SetDefault(logger)

	// Load .env file in development (ignored if file does not exist).
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		slog.Warn("could not load .env file", "error", err)
	}

	// Load and validate configuration from environment variables.
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	// Ensure the data directory exists before opening the database.
	if err := os.MkdirAll("data", 0o750); err != nil {
		slog.Error("failed to create data directory", "error", err)
		os.Exit(1)
	}

	// Open the SQLite database (WAL mode + busy_timeout applied via connection hook).
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "path", cfg.DBPath, "error", err)
		os.Exit(1)
	}
	defer database.Close()
	slog.Info("database opened", "path", cfg.DBPath)

	// Run migrations to ensure the schema is up to date.
	if err := db.Migrate(cfg.DBPath); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations complete")

	// Placeholder: HTTP server will be wired in Plan 02.
	slog.Info("server startup complete — HTTP server to be added in Plan 02", "port", cfg.Port)
}
