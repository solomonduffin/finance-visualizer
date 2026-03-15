package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
	"github.com/solomon/finance-visualizer/internal/api"
	"github.com/solomon/finance-visualizer/internal/auth"
	"github.com/solomon/finance-visualizer/internal/config"
	"github.com/solomon/finance-visualizer/internal/db"
	gosync "github.com/solomon/finance-visualizer/internal/sync"
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

	// Seed the password hash into the settings table if not already present.
	// This uses INSERT OR IGNORE so existing values are never overwritten.
	_, err = database.Exec(
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('password_hash', ?)`,
		cfg.PasswordHash,
	)
	if err != nil {
		slog.Error("failed to seed password hash", "error", err)
		os.Exit(1)
	}

	// Create a cancellable context tied to OS signals for graceful shutdown.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Initialize the JWT auth package with the secret key.
	auth.Init(cfg.JWTSecret)
	slog.Info("auth initialized")

	// Create the chi router with all middleware and routes.
	router := api.NewRouter(auth.TokenAuth(), database)

	// Start the daily sync scheduler goroutine.
	go gosync.RunScheduler(ctx, cfg.SyncHour, database)
	slog.Info("sync scheduler started", "hour", cfg.SyncHour)

	// Start the HTTP server.
	addr := ":" + cfg.Port
	slog.Info("starting HTTP server", "addr", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("HTTP server error", "error", err)
		os.Exit(1)
	}
}
