package config_test

import (
	"os"
	"testing"

	"github.com/solomon/finance-visualizer/internal/config"
)

func clearEnv(t *testing.T) {
	t.Helper()
	vars := []string{"PASSWORD", "PASSWORD_HASH", "JWT_SECRET", "PORT", "DB_PATH", "SYNC_HOUR"}
	for _, v := range vars {
		os.Unsetenv(v)
	}
	t.Cleanup(func() {
		for _, v := range vars {
			os.Unsetenv(v)
		}
	})
}

func TestLoad_SyncHourDefault(t *testing.T) {
	clearEnv(t)
	os.Setenv("PASSWORD", "mysecretpassword")
	os.Setenv("JWT_SECRET", "myjwtsecret")
	// SYNC_HOUR not set — should default to 6

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SyncHour != 6 {
		t.Errorf("expected default SyncHour=6, got %d", cfg.SyncHour)
	}
}

func TestLoad_SyncHourCustom(t *testing.T) {
	clearEnv(t)
	os.Setenv("PASSWORD", "mysecretpassword")
	os.Setenv("JWT_SECRET", "myjwtsecret")
	os.Setenv("SYNC_HOUR", "14")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SyncHour != 14 {
		t.Errorf("expected SyncHour=14, got %d", cfg.SyncHour)
	}
}

func TestLoad_SyncHourInvalid(t *testing.T) {
	clearEnv(t)
	os.Setenv("PASSWORD", "mysecretpassword")
	os.Setenv("JWT_SECRET", "myjwtsecret")
	os.Setenv("SYNC_HOUR", "abc")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error (invalid SYNC_HOUR should not fail): %v", err)
	}
	if cfg.SyncHour != 6 {
		t.Errorf("expected SyncHour=6 for invalid input, got %d", cfg.SyncHour)
	}
}

func TestConfig_Load_Success(t *testing.T) {
	clearEnv(t)
	os.Setenv("PASSWORD", "mysecretpassword")
	os.Setenv("JWT_SECRET", "myjwtsecret")
	os.Setenv("PORT", "9090")
	os.Setenv("DB_PATH", "/tmp/test.db")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.PasswordHash == "" {
		t.Error("expected PasswordHash to be set")
	}
	if cfg.JWTSecret != "myjwtsecret" {
		t.Errorf("expected JWTSecret=myjwtsecret, got %q", cfg.JWTSecret)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected Port=9090, got %q", cfg.Port)
	}
	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("expected DBPath=/tmp/test.db, got %q", cfg.DBPath)
	}
}

func TestConfig_Load_MissingPassword(t *testing.T) {
	clearEnv(t)
	os.Setenv("JWT_SECRET", "myjwtsecret")
	// Neither PASSWORD nor PASSWORD_HASH set

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when neither PASSWORD nor PASSWORD_HASH is set, got nil")
	}
}

func TestConfig_Load_MissingJWTSecret(t *testing.T) {
	clearEnv(t)
	os.Setenv("PASSWORD", "mysecretpassword")
	// JWT_SECRET not set

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET is not set, got nil")
	}
}

func TestConfig_Load_DefaultPort(t *testing.T) {
	clearEnv(t)
	os.Setenv("PASSWORD", "mysecretpassword")
	os.Setenv("JWT_SECRET", "myjwtsecret")
	// PORT not set — should default to 8080

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected default Port=8080, got %q", cfg.Port)
	}
}

func TestConfig_Load_PasswordHash(t *testing.T) {
	clearEnv(t)
	// Use a pre-hashed bcrypt hash for "testpassword"
	// Generated with bcrypt cost 12
	preHashed := "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj3oW5bRnl0y"
	os.Setenv("PASSWORD_HASH", preHashed)
	os.Setenv("JWT_SECRET", "myjwtsecret")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PasswordHash != preHashed {
		t.Errorf("expected PasswordHash to be pre-hashed value, got %q", cfg.PasswordHash)
	}
}

func TestConfig_Load_DefaultDBPath(t *testing.T) {
	clearEnv(t)
	os.Setenv("PASSWORD", "mysecretpassword")
	os.Setenv("JWT_SECRET", "myjwtsecret")
	// DB_PATH not set — should default to "data/finance.db"

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DBPath != "data/finance.db" {
		t.Errorf("expected default DBPath=data/finance.db, got %q", cfg.DBPath)
	}
}
