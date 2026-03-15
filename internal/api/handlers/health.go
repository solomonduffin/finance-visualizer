// Package handlers provides HTTP handler functions for the finance-visualizer API.
package handlers

import (
	"encoding/json"
	"net/http"
)

// Health handles GET /api/health. Returns {"status":"ok"} with 200.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"}) //nolint:errcheck
}
