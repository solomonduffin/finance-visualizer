package config

import (
	"errors"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// PasswordHash is the bcrypt hash used to verify login attempts.
	// Sourced from PASSWORD_HASH env var directly, or generated from PASSWORD env var.
	PasswordHash string

	// JWTSecret is the HMAC signing key for JWT tokens.
	JWTSecret string

	// Port is the TCP port the HTTP server listens on. Defaults to "8080".
	Port string

	// DBPath is the file path for the SQLite database. Defaults to "data/finance.db".
	DBPath string

	// SyncHour is the hour of day (0-23, local time) at which the daily SimpleFIN
	// sync runs. Sourced from SYNC_HOUR env var. Defaults to 6 (6:00 AM).
	SyncHour int
}

// Load reads configuration from environment variables and returns a validated Config.
// Required: JWT_SECRET and one of PASSWORD or PASSWORD_HASH.
// Defaults: PORT=8080, DB_PATH=data/finance.db.
func Load() (*Config, error) {
	cfg := &Config{}

	// Resolve password: prefer PASSWORD_HASH, fall back to hashing PASSWORD.
	passwordHash := os.Getenv("PASSWORD_HASH")
	password := os.Getenv("PASSWORD")

	switch {
	case passwordHash != "":
		cfg.PasswordHash = passwordHash
	case password != "":
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		cfg.PasswordHash = string(hash)
	default:
		return nil, errors.New("config: neither PASSWORD nor PASSWORD_HASH environment variable is set; set one to protect the app")
	}

	// JWT_SECRET is required.
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, errors.New("config: JWT_SECRET environment variable is not set; it is required to sign authentication tokens")
	}
	cfg.JWTSecret = jwtSecret

	// PORT defaults to 8080.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	cfg.Port = port

	// DB_PATH defaults to data/finance.db.
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/finance.db"
	}
	cfg.DBPath = dbPath

	// SYNC_HOUR defaults to 6; invalid values fall back to 6 (not an error).
	syncHour := 6
	if h := os.Getenv("SYNC_HOUR"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil {
			syncHour = parsed
		}
	}
	cfg.SyncHour = syncHour

	return cfg, nil
}
