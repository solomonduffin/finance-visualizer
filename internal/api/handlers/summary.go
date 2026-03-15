package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/shopspring/decimal"
)

type summaryResponse struct {
	Liquid       string  `json:"liquid"`
	Savings      string  `json:"savings"`
	Investments  string  `json:"investments"`
	LastSyncedAt *string `json:"last_synced_at"`
}

// GetSummary returns an http.HandlerFunc that handles GET /api/summary.
// It returns liquid, savings, and investments totals as string fields, along with
// the timestamp of the most recent successful sync.
//
// Liquid = sum(checking) + sum(credit), where credit balances are already negative.
// Accounts with type "other" are excluded from all panel totals.
func GetSummary(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Query all accounts with their latest snapshot balance using a correlated subquery.
		rows, err := database.QueryContext(r.Context(), `
			SELECT a.account_type, bs.balance
			FROM accounts a
			JOIN balance_snapshots bs ON bs.account_id = a.id
			WHERE bs.balance_date = (
				SELECT MAX(bs2.balance_date)
				FROM balance_snapshots bs2
				WHERE bs2.account_id = a.id
			)
		`)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		liquid := decimal.Zero
		savings := decimal.Zero
		investments := decimal.Zero

		for rows.Next() {
			var accountType, balance string
			if err := rows.Scan(&accountType, &balance); err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}

			amount, err := decimal.NewFromString(balance)
			if err != nil {
				// Skip invalid balance values
				continue
			}

			switch accountType {
			case "checking", "credit":
				liquid = liquid.Add(amount)
			case "savings":
				savings = savings.Add(amount)
			case "investment":
				investments = investments.Add(amount)
			// "other" type is intentionally excluded
			}
		}
		if err := rows.Err(); err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Query last successful sync timestamp.
		var finishedAt sql.NullString
		err = database.QueryRowContext(r.Context(), `
			SELECT finished_at FROM sync_log
			WHERE error_text IS NULL AND finished_at IS NOT NULL
			ORDER BY id DESC LIMIT 1
		`).Scan(&finishedAt)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		resp := summaryResponse{
			Liquid:      liquid.StringFixed(2),
			Savings:     savings.StringFixed(2),
			Investments: investments.StringFixed(2),
		}
		if finishedAt.Valid {
			resp.LastSyncedAt = &finishedAt.String
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}
