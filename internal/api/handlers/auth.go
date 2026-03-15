package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/solomon/finance-visualizer/internal/auth"
)

type loginRequest struct {
	Password string `json:"password"`
}

// Login returns an http.HandlerFunc that handles POST /api/auth/login.
// It decodes the JSON body, verifies the password against the stored bcrypt hash
// in the settings table, and issues a 30-day HttpOnly JWT cookie on success.
func Login(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse and validate request body
		if r.Body == nil {
			http.Error(w, `{"error":"request body required"}`, http.StatusBadRequest)
			return
		}

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}

		if req.Password == "" {
			http.Error(w, `{"error":"password is required"}`, http.StatusBadRequest)
			return
		}

		// Read stored password hash from settings table
		var storedHash string
		err := database.QueryRowContext(r.Context(),
			`SELECT value FROM settings WHERE key = 'password_hash'`,
		).Scan(&storedHash)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Verify password against stored bcrypt hash
		if err := auth.VerifyPassword(storedHash, req.Password); err != nil {
			http.Error(w, `{"error":"invalid password"}`, http.StatusUnauthorized)
			return
		}

		// Generate JWT token
		tokenString, err := auth.CreateToken()
		if err != nil {
			http.Error(w, `{"error":"failed to create token"}`, http.StatusInternalServerError)
			return
		}

		// Set HttpOnly cookie — name MUST be "jwt" (jwtauth.TokenFromCookie looks for this exact name)
		http.SetCookie(w, &http.Cookie{
			Name:     "jwt",
			Value:    tokenString,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   30 * 24 * 3600, // 30 days in seconds
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	}
}
