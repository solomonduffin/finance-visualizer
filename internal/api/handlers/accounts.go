package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type accountItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Balance  string `json:"balance"`
	Currency string `json:"currency"`
	OrgName  string `json:"org_name,omitempty"`
}

type accountsResponse struct {
	Checking    []accountItem `json:"checking"`
	Savings     []accountItem `json:"savings"`
	Credit      []accountItem `json:"credit"`
	Investments []accountItem `json:"investments"`
	Other       []accountItem `json:"other"`
}

// GetAccounts returns an http.HandlerFunc that handles GET /api/accounts.
// It returns all accounts grouped by type with the latest snapshot balance for each.
// Accounts with no snapshots show balance "0".
// Empty groups are returned as empty JSON arrays, not null.
func GetAccounts(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := accountsResponse{
			Checking:    []accountItem{},
			Savings:     []accountItem{},
			Credit:      []accountItem{},
			Investments: []accountItem{},
			Other:       []accountItem{},
		}

		// Query all accounts with their latest balance using LEFT JOIN + correlated subquery.
		// LEFT JOIN ensures accounts with no snapshots are still included (balance will be NULL).
		rows, err := database.QueryContext(r.Context(), `
			SELECT a.id, a.name, a.account_type, a.currency, a.org_name,
			       bs.balance
			FROM accounts a
			LEFT JOIN balance_snapshots bs ON bs.account_id = a.id
			  AND bs.balance_date = (
			      SELECT MAX(bs2.balance_date)
			      FROM balance_snapshots bs2
			      WHERE bs2.account_id = a.id
			  )
			ORDER BY a.account_type, a.name
		`)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var id, name, accountType, currency string
			var orgName sql.NullString
			var balance sql.NullString

			if err := rows.Scan(&id, &name, &accountType, &currency, &orgName, &balance); err != nil {
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
				ID:       id,
				Name:     name,
				Type:     accountType,
				Balance:  balanceStr,
				Currency: currency,
				OrgName:  orgNameStr,
			}

			switch accountType {
			case "checking":
				resp.Checking = append(resp.Checking, item)
			case "savings":
				resp.Savings = append(resp.Savings, item)
			case "credit":
				resp.Credit = append(resp.Credit, item)
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
