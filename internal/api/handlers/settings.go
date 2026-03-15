package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	gosync "github.com/solomon/finance-visualizer/internal/sync"
)

type saveSettingsRequest struct {
	AccessURL string `json:"access_url"`
}

type settingsResponse struct {
	Configured     bool    `json:"configured"`
	LastSyncAt     *string `json:"last_sync_at"`
	LastSyncStatus *string `json:"last_sync_status"`
}

// GetSettings returns an http.HandlerFunc that handles GET /api/settings.
// It returns whether SimpleFIN is configured and the last sync status.
func GetSettings(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := settingsResponse{}

		// Check if access URL is configured.
		var accessURL string
		err := database.QueryRowContext(r.Context(),
			`SELECT value FROM settings WHERE key='simplefin_access_url'`,
		).Scan(&accessURL)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		resp.Configured = (err == nil && accessURL != "")

		// Query most recent sync_log entry.
		var finishedAt sql.NullString
		var errorText sql.NullString
		err = database.QueryRowContext(r.Context(),
			`SELECT finished_at, error_text FROM sync_log ORDER BY id DESC LIMIT 1`,
		).Scan(&finishedAt, &errorText)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		if err == nil && finishedAt.Valid {
			resp.LastSyncAt = &finishedAt.String
			if errorText.Valid && errorText.String != "" {
				resp.LastSyncStatus = &errorText.String
			} else {
				s := "success"
				resp.LastSyncStatus = &s
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}

// SaveSettings returns an http.HandlerFunc that handles POST /api/settings.
// It saves the SimpleFIN access URL and triggers an immediate first sync.
func SaveSettings(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, `{"error":"request body required"}`, http.StatusBadRequest)
			return
		}

		var req saveSettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}

		if req.AccessURL == "" {
			http.Error(w, `{"error":"access_url is required"}`, http.StatusBadRequest)
			return
		}

		// Upsert the access URL into settings.
		_, err := database.ExecContext(r.Context(),
			`INSERT INTO settings (key, value) VALUES ('simplefin_access_url', ?)
			 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
			req.AccessURL,
		)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Trigger immediate first sync in a goroutine.
		// Use context.Background() so the sync isn't cancelled when the HTTP response completes.
		go gosync.SyncOnce(context.Background(), database) //nolint:errcheck

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	}
}

// SyncNow returns an http.HandlerFunc that handles POST /api/sync/now.
// It launches a background sync and returns immediately.
func SyncNow(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Launch sync in a goroutine — returns immediately regardless of sync outcome.
		// The sync mutex prevents concurrent runs.
		go gosync.SyncOnce(context.Background(), database) //nolint:errcheck

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	}
}
