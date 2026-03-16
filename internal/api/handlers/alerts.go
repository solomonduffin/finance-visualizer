package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/solomon/finance-visualizer/internal/alerts"
)

type createAlertRequest struct {
	Name             string          `json:"name"`
	Operands         json.RawMessage `json:"operands"`
	Comparison       string          `json:"comparison"`
	Threshold        string          `json:"threshold"`
	NotifyOnRecovery bool            `json:"notify_on_recovery"`
}

type alertRuleResponse struct {
	ID               int                 `json:"id"`
	Name             string              `json:"name"`
	Operands         json.RawMessage     `json:"operands"`
	Expression       string              `json:"expression"`
	Comparison       string              `json:"comparison"`
	Threshold        string              `json:"threshold"`
	NotifyOnRecovery bool                `json:"notify_on_recovery"`
	Enabled          bool                `json:"enabled"`
	LastState        string              `json:"last_state"`
	LastEvalAt       *string             `json:"last_eval_at"`
	LastValue        *string             `json:"last_value"`
	CreatedAt        string              `json:"created_at"`
	UpdatedAt        string              `json:"updated_at"`
	History          []alertHistoryEntry `json:"history"`
}

type alertHistoryEntry struct {
	ID         int     `json:"id"`
	State      string  `json:"state"`
	Value      *string `json:"value"`
	NotifiedAt string  `json:"notified_at"`
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}

// scanAlertRule scans a single alert_rules row into an alertRuleResponse.
// The row must have columns: id, name, operands, expression, comparison,
// threshold, notify_on_recovery, enabled, last_state, last_eval_at,
// last_value, created_at, updated_at.
func scanAlertRule(row interface{ Scan(dest ...interface{}) error }) (*alertRuleResponse, error) {
	var resp alertRuleResponse
	var operands string
	var notifyOnRecovery int
	var enabled int
	var lastEvalAt sql.NullString
	var lastValue sql.NullString

	if err := row.Scan(
		&resp.ID, &resp.Name, &operands, &resp.Expression,
		&resp.Comparison, &resp.Threshold, &notifyOnRecovery,
		&enabled, &resp.LastState, &lastEvalAt, &lastValue,
		&resp.CreatedAt, &resp.UpdatedAt,
	); err != nil {
		return nil, err
	}

	resp.Operands = json.RawMessage(operands)
	resp.NotifyOnRecovery = intToBool(notifyOnRecovery)
	resp.Enabled = intToBool(enabled)
	if lastEvalAt.Valid {
		resp.LastEvalAt = &lastEvalAt.String
	}
	if lastValue.Valid {
		resp.LastValue = &lastValue.String
	}
	resp.History = []alertHistoryEntry{}

	return &resp, nil
}

// fetchAlertHistory loads the last 10 history entries for a given rule.
func fetchAlertHistory(database *sql.DB, ruleID int) ([]alertHistoryEntry, error) {
	rows, err := database.Query(
		`SELECT id, state, value, notified_at FROM alert_history WHERE rule_id = ? ORDER BY notified_at DESC LIMIT 10`,
		ruleID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := []alertHistoryEntry{}
	for rows.Next() {
		var entry alertHistoryEntry
		var value sql.NullString
		if err := rows.Scan(&entry.ID, &entry.State, &value, &entry.NotifiedAt); err != nil {
			return nil, err
		}
		if value.Valid {
			entry.Value = &value.String
		}
		history = append(history, entry)
	}
	return history, rows.Err()
}

// CreateAlert returns an http.HandlerFunc that handles POST /api/alerts.
// It creates a new alert rule with validated operands and returns the created rule as JSON.
func CreateAlert(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createAlertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
			return
		}

		expression, err := alerts.CompileOperands(req.Operands, req.Comparison, req.Threshold)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		result, err := database.ExecContext(r.Context(),
			`INSERT INTO alert_rules (name, operands, expression, comparison, threshold, notify_on_recovery, enabled, last_state)
			 VALUES (?, ?, ?, ?, ?, ?, 1, 'normal')`,
			req.Name, string(req.Operands), expression, req.Comparison, req.Threshold, boolToInt(req.NotifyOnRecovery),
		)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		id, err := result.LastInsertId()
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		row := database.QueryRowContext(r.Context(),
			`SELECT id, name, operands, expression, comparison, threshold, notify_on_recovery,
			        enabled, last_state, last_eval_at, last_value, created_at, updated_at
			 FROM alert_rules WHERE id = ?`, id,
		)
		resp, err := scanAlertRule(row)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}

// ListAlerts returns an http.HandlerFunc that handles GET /api/alerts.
// It returns all alert rules with their history entries.
func ListAlerts(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := database.QueryContext(r.Context(),
			`SELECT id, name, operands, expression, comparison, threshold, notify_on_recovery,
			        enabled, last_state, last_eval_at, last_value, created_at, updated_at
			 FROM alert_rules ORDER BY created_at DESC`,
		)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Collect all rules first to close the rows cursor before nested queries.
		rules := []alertRuleResponse{}
		for rows.Next() {
			resp, err := scanAlertRule(rows)
			if err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}
			rules = append(rules, *resp)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		rows.Close()

		// Fetch history for each rule (rows cursor is now closed).
		for i := range rules {
			history, err := fetchAlertHistory(database, rules[i].ID)
			if err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}
			rules[i].History = history
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rules) //nolint:errcheck
	}
}

// UpdateAlert returns an http.HandlerFunc that handles PUT /api/alerts/{id}.
// It updates an existing rule's name, operands, comparison, threshold, and recovery setting.
func UpdateAlert(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid alert id"}`, http.StatusBadRequest)
			return
		}

		var req createAlertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
			return
		}

		expression, err := alerts.CompileOperands(req.Operands, req.Comparison, req.Threshold)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		result, err := database.ExecContext(r.Context(),
			`UPDATE alert_rules SET name=?, operands=?, expression=?, comparison=?, threshold=?,
			        notify_on_recovery=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			req.Name, string(req.Operands), expression, req.Comparison, req.Threshold,
			boolToInt(req.NotifyOnRecovery), id,
		)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			http.Error(w, `{"error":"alert not found"}`, http.StatusNotFound)
			return
		}

		row := database.QueryRowContext(r.Context(),
			`SELECT id, name, operands, expression, comparison, threshold, notify_on_recovery,
			        enabled, last_state, last_eval_at, last_value, created_at, updated_at
			 FROM alert_rules WHERE id = ?`, id,
		)
		resp, err := scanAlertRule(row)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}

// ToggleAlert returns an http.HandlerFunc that handles PATCH /api/alerts/{id}.
// It toggles a rule's enabled state.
func ToggleAlert(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid alert id"}`, http.StatusBadRequest)
			return
		}

		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		result, err := database.ExecContext(r.Context(),
			`UPDATE alert_rules SET enabled=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			boolToInt(body.Enabled), id,
		)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			http.Error(w, `{"error":"alert not found"}`, http.StatusNotFound)
			return
		}

		row := database.QueryRowContext(r.Context(),
			`SELECT id, name, operands, expression, comparison, threshold, notify_on_recovery,
			        enabled, last_state, last_eval_at, last_value, created_at, updated_at
			 FROM alert_rules WHERE id = ?`, id,
		)
		resp, err := scanAlertRule(row)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}

// DeleteAlert returns an http.HandlerFunc that handles DELETE /api/alerts/{id}.
// It removes a rule and its history (via CASCADE).
func DeleteAlert(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid alert id"}`, http.StatusBadRequest)
			return
		}

		_, err = database.ExecContext(r.Context(),
			`DELETE FROM alert_rules WHERE id=?`, id,
		)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
