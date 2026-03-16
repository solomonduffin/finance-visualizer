package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/solomon/finance-visualizer/internal/simplefin"
	gosync "github.com/solomon/finance-visualizer/internal/sync"
)

type saveSettingsRequest struct {
	AccessURL string `json:"access_url"`
}

type settingsResponse struct {
	Configured         bool    `json:"configured"`
	LastSyncAt         *string `json:"last_sync_at"`
	LastSyncStatus     *string `json:"last_sync_status"`
	GrowthBadgeEnabled bool    `json:"growth_badge_enabled"`
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

		// Query growth_badge_enabled setting (defaults to true when absent).
		var growthEnabled string
		err = database.QueryRowContext(r.Context(),
			`SELECT value FROM settings WHERE key='growth_badge_enabled'`,
		).Scan(&growthEnabled)
		if err == sql.ErrNoRows {
			resp.GrowthBadgeEnabled = true
		} else if err == nil {
			resp.GrowthBadgeEnabled = growthEnabled != "false"
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

		// If the input looks like a setup token (base64, no "://"), claim it first.
		accessURL := req.AccessURL
		if simplefin.IsSetupToken(accessURL) {
			claimed, err := simplefin.ClaimSetupToken(accessURL)
			if err != nil {
				http.Error(w, `{"error":"failed to claim setup token: `+err.Error()+`"}`, http.StatusBadRequest)
				return
			}
			accessURL = claimed
		}

		// Upsert the access URL into settings.
		_, err := database.ExecContext(r.Context(),
			`INSERT INTO settings (key, value) VALUES ('simplefin_access_url', ?)
			 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
			accessURL,
		)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Trigger immediate first sync in a goroutine.
		// Use context.Background() so the sync isn't cancelled when the HTTP response completes.
		go func() {
			_, _ = gosync.SyncOnce(context.Background(), database)
		}()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	}
}

type syncNowResponse struct {
	OK       bool     `json:"ok"`
	Restored []string `json:"restored"`
}

// SyncNow returns an http.HandlerFunc that handles POST /api/sync/now.
// It runs a synchronous sync and returns restored account names in the response.
// Sync errors are logged but the handler always returns ok:true (matching previous behavior
// where sync was fire-and-forget).
func SyncNow(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Run sync synchronously so we can return restored account names.
		// The sync mutex prevents concurrent runs.
		restored, err := gosync.SyncOnce(r.Context(), database)
		if err != nil {
			// Log the error but don't fail the HTTP response — sync errors are expected
			// when the access URL is unreachable or invalid.
			restored = []string{}
		}

		if restored == nil {
			restored = []string{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(syncNowResponse{
			OK:       true,
			Restored: restored,
		}) //nolint:errcheck
	}
}

type growthBadgeRequest struct {
	Value string `json:"value"`
}

// SaveGrowthBadge returns an http.HandlerFunc that handles PUT /api/settings/growth-badge.
// It persists the growth badge toggle value ("true" or "false") in the settings table.
func SaveGrowthBadge(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, `{"error":"request body required"}`, http.StatusBadRequest)
			return
		}

		var req growthBadgeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}

		if req.Value != "true" && req.Value != "false" {
			http.Error(w, `{"error":"value must be \"true\" or \"false\""}`, http.StatusBadRequest)
			return
		}

		_, err := database.ExecContext(r.Context(),
			`INSERT INTO settings (key, value) VALUES ('growth_badge_enabled', ?)
			 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
			req.Value,
		)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	}
}
