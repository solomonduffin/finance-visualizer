package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/shopspring/decimal"
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

type groupMemberItem struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	OriginalName        string  `json:"original_name"`
	Balance             string  `json:"balance"`
	Currency            string  `json:"currency"`
	OrgName             string  `json:"org_name"`
	DisplayName         *string `json:"display_name"`
	AccountTypeOverride *string `json:"account_type_override"`
}

type groupItem struct {
	ID           int               `json:"id"`
	Name         string            `json:"name"`
	PanelType    string            `json:"panel_type"`
	TotalBalance string            `json:"total_balance"`
	Members      []groupMemberItem `json:"members"`
}

type accountsResponse struct {
	Liquid      []accountItem `json:"liquid"`
	Savings     []accountItem `json:"savings"`
	Investments []accountItem `json:"investments"`
	Other       []accountItem `json:"other"`
	Groups      []groupItem   `json:"groups"`
}

// GetAccounts returns an http.HandlerFunc that handles GET /api/accounts.
// It returns accounts grouped by effective type with the latest snapshot balance for each.
// Accounts with no snapshots show balance "0".
// Hidden accounts (hidden_at IS NOT NULL) are excluded by default.
// Pass ?include_hidden=true to include hidden accounts (for Settings page).
// Display name is used when set (COALESCE), and account_type_override controls grouping.
// Empty groups are returned as empty JSON arrays, not null.
// Grouped accounts are excluded from standalone panel arrays and appear in the groups array.
func GetAccounts(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := accountsResponse{
			Liquid:      []accountItem{},
			Savings:     []accountItem{},
			Investments: []accountItem{},
			Other:       []accountItem{},
			Groups:      []groupItem{},
		}

		includeHidden := r.URL.Query().Get("include_hidden") == "true"

		// Query standalone accounts (NOT in any group) with their latest balance.
		// Grouped accounts are excluded via NOT IN subquery.
		query := `
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
			WHERE a.id NOT IN (SELECT account_id FROM group_members)
		`
		if !includeHidden {
			query += " AND a.hidden_at IS NULL"
		}
		query += " ORDER BY effective_type, name"

		rows, err := database.QueryContext(r.Context(), query)
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

		// Query groups with their members
		groupRows, err := database.QueryContext(r.Context(), `
			SELECT g.id, g.name, g.panel_type,
			       gm.account_id,
			       COALESCE(a.display_name, a.name) AS account_name,
			       a.name AS original_name,
			       a.currency, a.org_name,
			       a.display_name, a.account_type_override,
			       COALESCE(bs.balance, '0') AS balance
			FROM account_groups g
			LEFT JOIN group_members gm ON gm.group_id = g.id
			LEFT JOIN accounts a ON a.id = gm.account_id AND a.hidden_at IS NULL
			LEFT JOIN balance_snapshots bs ON bs.account_id = a.id
			  AND bs.balance_date = (
			      SELECT MAX(bs2.balance_date) FROM balance_snapshots bs2 WHERE bs2.account_id = a.id
			  )
			ORDER BY g.id, account_name
		`)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		defer groupRows.Close()

		groupMap := map[int]*groupItem{}
		var groupOrder []int

		for groupRows.Next() {
			var gID int
			var gName, gPanelType string
			var accountID sql.NullString
			var accountName sql.NullString
			var originalName sql.NullString
			var currency sql.NullString
			var orgName sql.NullString
			var displayName sql.NullString
			var accountTypeOverride sql.NullString
			var balance sql.NullString

			if err := groupRows.Scan(&gID, &gName, &gPanelType,
				&accountID, &accountName, &originalName,
				&currency, &orgName, &displayName, &accountTypeOverride,
				&balance); err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}

			grp, exists := groupMap[gID]
			if !exists {
				grp = &groupItem{
					ID:        gID,
					Name:      gName,
					PanelType: gPanelType,
					Members:   []groupMemberItem{},
				}
				groupMap[gID] = grp
				groupOrder = append(groupOrder, gID)
			}

			// If there's a member account (LEFT JOIN may yield NULL for empty groups)
			if accountID.Valid && accountName.Valid {
				member := groupMemberItem{
					ID:           accountID.String,
					Name:         accountName.String,
					Balance:      "0",
					Currency:     currency.String,
				}
				if originalName.Valid {
					member.OriginalName = originalName.String
				}
				if orgName.Valid {
					member.OrgName = orgName.String
				}
				if displayName.Valid {
					s := displayName.String
					member.DisplayName = &s
				}
				if accountTypeOverride.Valid {
					s := accountTypeOverride.String
					member.AccountTypeOverride = &s
				}
				if balance.Valid {
					member.Balance = balance.String
				}
				grp.Members = append(grp.Members, member)
			}
		}
		if err := groupRows.Err(); err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Compute total_balance for each group using shopspring/decimal
		for _, gID := range groupOrder {
			grp := groupMap[gID]
			total := decimal.Zero
			for _, m := range grp.Members {
				amount, parseErr := decimal.NewFromString(m.Balance)
				if parseErr == nil {
					total = total.Add(amount)
				}
			}
			grp.TotalBalance = total.StringFixed(2)
			resp.Groups = append(resp.Groups, *grp)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}
