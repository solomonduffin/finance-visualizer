package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type accountItem struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`                  // COALESCE(display_name, name)
	OriginalName        string  `json:"original_name"`         // raw SimpleFIN name
	Type                string  `json:"type"`                  // effective type
	Balance             string  `json:"balance"`
	Currency            string  `json:"currency"`
	OrgName             string  `json:"org_name,omitempty"`
	DisplayName         *string `json:"display_name"`          // user-set or null
	HiddenAt            *string `json:"hidden_at"`             // ISO timestamp or null
	AccountTypeOverride *string `json:"account_type_override"` // user override or null
}

type accountsResponse struct {
	Liquid      []accountItem `json:"liquid"`
	Savings     []accountItem `json:"savings"`
	Investments []accountItem `json:"investments"`
	Other       []accountItem `json:"other"`
}

// GetAccounts returns an http.HandlerFunc that handles GET /api/accounts.
// It returns all visible accounts grouped by effective type with the latest snapshot balance for each.
// Accounts with no snapshots show balance "0".
// Hidden accounts (hidden_at IS NOT NULL) are excluded.
// Display name is used when set (COALESCE), and account_type_override controls grouping.
// Empty groups are returned as empty JSON arrays, not null.
func GetAccounts(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := accountsResponse{
			Liquid:      []accountItem{},
			Savings:     []accountItem{},
			Investments: []accountItem{},
			Other:       []accountItem{},
		}

		// Query all visible accounts with their latest balance using LEFT JOIN + correlated subquery.
		// LEFT JOIN ensures accounts with no snapshots are still included (balance will be NULL).
		// COALESCE is used for display_name -> name fallback and account_type_override -> account_type fallback.
		rows, err := database.QueryContext(r.Context(), `
			SELECT a.id,
			       COALESCE(a.display_name, a.name) AS name,
			       a.name AS original_name,
			       COALESCE(a.account_type_override, a.account_type) AS effective_type,
			       a.currency, a.org_name,
			       a.display_name, a.hidden_at, a.account_type_override,
			       bs.balance
			FROM accounts a
			LEFT JOIN balance_snapshots bs ON bs.account_id = a.id
			  AND bs.balance_date = (
			      SELECT MAX(bs2.balance_date)
			      FROM balance_snapshots bs2
			      WHERE bs2.account_id = a.id
			  )
			WHERE a.hidden_at IS NULL
			ORDER BY effective_type, name
		`)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var id, name, originalName, effectiveType, currency string
			var orgName sql.NullString
			var balance sql.NullString
			var displayName sql.NullString
			var hiddenAt sql.NullString
			var accountTypeOverride sql.NullString

			if err := rows.Scan(&id, &name, &originalName, &effectiveType, &currency, &orgName,
				&displayName, &hiddenAt, &accountTypeOverride, &balance); err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}

			balanceStr := "0"
			if balance.Valid {
				balanceStr = balance.String
			}

			orgNameStr := ""
			if orgName.Valid {
				orgNameStr = orgName.String
			}

			item := accountItem{
				ID:           id,
				Name:         name,
				OriginalName: originalName,
				Type:         effectiveType,
				Balance:      balanceStr,
				Currency:     currency,
				OrgName:      orgNameStr,
			}
			if displayName.Valid {
				s := displayName.String
				item.DisplayName = &s
			}
			if hiddenAt.Valid {
				s := hiddenAt.String
				item.HiddenAt = &s
			}
			if accountTypeOverride.Valid {
				s := accountTypeOverride.String
				item.AccountTypeOverride = &s
			}

			switch effectiveType {
			case "checking", "credit":
				resp.Liquid = append(resp.Liquid, item)
			case "savings":
				resp.Savings = append(resp.Savings, item)
			case "investment":
				resp.Investments = append(resp.Investments, item)
			default:
				resp.Other = append(resp.Other, item)
			}
		}
		if err := rows.Err(); err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}
