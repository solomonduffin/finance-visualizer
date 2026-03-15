package handlers

import (
	"database/sql"
	"net/http"
)

// UpdateAccount returns an http.HandlerFunc that handles PATCH /api/accounts/{id}.
// It updates account metadata: display_name, hidden (toggle), account_type_override.
func UpdateAccount(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Stub: not implemented yet
		http.Error(w, `{"error":"not implemented"}`, http.StatusNotImplemented)
	}
}
