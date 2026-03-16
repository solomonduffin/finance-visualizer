package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
)

// Regex patterns for sanitizing sensitive data from error text.
var (
	// Matches user:pass@host patterns in URLs.
	reURLCredentials = regexp.MustCompile(`[a-zA-Z0-9+/=_.-]+:[a-zA-Z0-9+/=_.-]+@[^\s]+`)
	// Matches base64 tokens of 40+ characters.
	reBase64Token = regexp.MustCompile(`[A-Za-z0-9+/=]{40,}`)
)

// SanitizeErrorText strips credentials and base64 tokens from error messages.
// Exported for testing.
func SanitizeErrorText(raw string) string {
	result := reURLCredentials.ReplaceAllString(raw, "[redacted-url]")
	result = reBase64Token.ReplaceAllString(result, "[redacted-token]")
	return result
}

type syncLogEntry struct {
	ID              int64   `json:"id"`
	StartedAt       string  `json:"started_at"`
	FinishedAt      *string `json:"finished_at"`
	AccountsFetched int     `json:"accounts_fetched"`
	AccountsFailed  int     `json:"accounts_failed"`
	ErrorText       *string `json:"error_text"`
	Status          string  `json:"status"`
}

type syncLogResponse struct {
	Entries []syncLogEntry `json:"entries"`
}

// GetSyncLog returns an http.HandlerFunc that handles GET /api/sync-log.
// It returns the last 7 sync_log entries with derived status and sanitized error text.
func GetSyncLog(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := database.QueryContext(r.Context(),
			`SELECT id, started_at, finished_at, accounts_fetched, accounts_failed, error_text
			 FROM sync_log ORDER BY id DESC LIMIT 7`,
		)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		entries := []syncLogEntry{} // non-nil so JSON encodes as [] not null

		for rows.Next() {
			var e syncLogEntry
			var finishedAt sql.NullString
			var errorText sql.NullString

			if err := rows.Scan(&e.ID, &e.StartedAt, &finishedAt, &e.AccountsFetched, &e.AccountsFailed, &errorText); err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}

			if finishedAt.Valid {
				e.FinishedAt = &finishedAt.String
			}

			if errorText.Valid {
				sanitized := SanitizeErrorText(errorText.String)
				e.ErrorText = &sanitized
			}

			// Derive status
			switch {
			case errorText.Valid && e.AccountsFetched == 0:
				e.Status = "failed"
			case e.AccountsFailed > 0:
				e.Status = "partial"
			default:
				e.Status = "success"
			}

			entries = append(entries, e)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(syncLogResponse{Entries: entries}) //nolint:errcheck
	}
}
