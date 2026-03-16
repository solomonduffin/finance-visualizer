package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/solomon/finance-visualizer/internal/alerts"
)

type emailConfigRequest struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	To       string `json:"to"`
}

// SaveEmailConfig returns an http.HandlerFunc that handles POST /api/email/config.
// It saves SMTP configuration with encrypted password to the settings table.
func SaveEmailConfig(database *sql.DB, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req emailConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.Host == "" {
			http.Error(w, `{"error":"host is required"}`, http.StatusBadRequest)
			return
		}

		// Encrypt the password before saving.
		encryptedPassword := ""
		if req.Password != "" {
			key := alerts.DeriveKey(jwtSecret)
			encrypted, err := alerts.Encrypt(req.Password, key)
			if err != nil {
				http.Error(w, `{"error":"failed to encrypt password"}`, http.StatusInternalServerError)
				return
			}
			encryptedPassword = encrypted
		}

		// Upsert each SMTP setting using the same pattern as existing settings handlers.
		settings := map[string]string{
			"smtp_host":     req.Host,
			"smtp_port":     req.Port,
			"smtp_username": req.Username,
			"smtp_password": encryptedPassword,
			"smtp_from":     req.From,
			"smtp_to":       req.To,
		}

		for key, value := range settings {
			_, err := database.ExecContext(r.Context(),
				`INSERT INTO settings (key, value) VALUES (?, ?)
				 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
				key, value,
			)
			if err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	}
}

// GetEmailConfig returns an http.HandlerFunc that handles GET /api/email/config.
// It returns SMTP fields (without password value) and configured boolean.
func GetEmailConfig(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Query all SMTP settings except password.
		keys := []string{"smtp_host", "smtp_port", "smtp_username", "smtp_from", "smtp_to"}
		settings := make(map[string]string)

		for _, key := range keys {
			var value string
			err := database.QueryRowContext(r.Context(),
				`SELECT value FROM settings WHERE key = ?`, key,
			).Scan(&value)
			if err == sql.ErrNoRows {
				continue
			}
			if err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}
			settings[key] = value
		}

		host := settings["smtp_host"]
		if host == "" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"configured":false}`)) //nolint:errcheck
			return
		}

		resp := map[string]interface{}{
			"configured": true,
			"host":       settings["smtp_host"],
			"port":       settings["smtp_port"],
			"username":   settings["smtp_username"],
			"from":       settings["smtp_from"],
			"to":         settings["smtp_to"],
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}

// TestEmail returns an http.HandlerFunc that handles POST /api/email/test.
// It sends a test email using provided form values without saving them.
func TestEmail(database *sql.DB, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req emailConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.Host == "" {
			http.Error(w, `{"error":"host is required"}`, http.StatusBadRequest)
			return
		}

		port := 587
		if req.Port != "" {
			p, err := strconv.Atoi(req.Port)
			if err == nil {
				port = p
			}
		}

		cfg := alerts.SMTPConfig{
			Host:        req.Host,
			Port:        port,
			Username:    req.Username,
			Password:    req.Password,
			FromAddress: req.From,
			ToAddress:   req.To,
		}

		detail := alerts.AlertDetail{
			RuleName:      "Test Email",
			Status:        "triggered",
			ComputedValue: "N/A",
			Threshold:     "N/A",
			Comparison:    "N/A",
			Operands:      []alerts.Operand{},
			OperandValues: map[string]string{},
			Timestamp:     time.Now(),
		}

		if err := alerts.SendAlert(r.Context(), cfg, detail); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}) //nolint:errcheck
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	}
}
