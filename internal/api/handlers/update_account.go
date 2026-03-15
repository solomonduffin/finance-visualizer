package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// nullableString distinguishes between JSON null, absent, and a string value.
// When Set is true and Value is nil, the JSON field was explicitly null.
// When Set is true and Value is non-nil, the JSON field was a string.
// When Set is false, the JSON field was absent from the payload.
type nullableString struct {
	Value *string
	Set   bool
}

func (ns *nullableString) UnmarshalJSON(data []byte) error {
	ns.Set = true
	if string(data) == "null" {
		ns.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	ns.Value = &s
	return nil
}

// updateAccountRequest is the JSON body for PATCH /api/accounts/{id}.
type updateAccountRequest struct {
	DisplayName  nullableString `json:"display_name"`
	Hidden       *bool          `json:"hidden"`
	TypeOverride nullableString `json:"account_type_override"`
}

// validAccountTypes are the allowed values for account_type_override.
var validAccountTypes = map[string]bool{
	"checking":   true,
	"savings":    true,
	"credit":     true,
	"investment": true,
	"other":      true,
}

// UpdateAccount returns an http.HandlerFunc that handles PATCH /api/accounts/{id}.
// It updates account metadata: display_name, hidden (toggle), account_type_override.
// Fields not present in the request body are left unchanged.
// display_name: null clears, string sets. hidden: true sets hidden_at, false clears it.
// account_type_override: null clears, string sets (validated against allowed types).
func UpdateAccount(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accountID := chi.URLParam(r, "id")
		if accountID == "" {
			http.Error(w, `{"error":"missing account id"}`, http.StatusBadRequest)
			return
		}

		var req updateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		// Check that at least one field is being updated
		if !req.DisplayName.Set && req.Hidden == nil && !req.TypeOverride.Set {
			http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
			return
		}

		// Validate account_type_override if provided
		if req.TypeOverride.Set && req.TypeOverride.Value != nil {
			if !validAccountTypes[*req.TypeOverride.Value] {
				http.Error(w, `{"error":"invalid account_type_override"}`, http.StatusBadRequest)
				return
			}
		}

		// Verify account exists
		var exists bool
		err := database.QueryRowContext(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM accounts WHERE id = ?)`, accountID).Scan(&exists)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, `{"error":"account not found"}`, http.StatusNotFound)
			return
		}

		// Build dynamic UPDATE query
		setClauses := []string{}
		args := []interface{}{}

		if req.DisplayName.Set {
			if req.DisplayName.Value == nil {
				setClauses = append(setClauses, "display_name = NULL")
			} else {
				setClauses = append(setClauses, "display_name = ?")
				args = append(args, *req.DisplayName.Value)
			}
		}

		if req.Hidden != nil {
			if *req.Hidden {
				setClauses = append(setClauses, "hidden_at = CURRENT_TIMESTAMP")
			} else {
				setClauses = append(setClauses, "hidden_at = NULL")
			}
		}

		if req.TypeOverride.Set {
			if req.TypeOverride.Value == nil {
				setClauses = append(setClauses, "account_type_override = NULL")
			} else {
				setClauses = append(setClauses, "account_type_override = ?")
				args = append(args, *req.TypeOverride.Value)
			}
		}

		// Execute UPDATE
		if len(setClauses) > 0 {
			query := "UPDATE accounts SET "
			for i, clause := range setClauses {
				if i > 0 {
					query += ", "
				}
				query += clause
			}
			query += " WHERE id = ?"
			args = append(args, accountID)

			if _, err := database.ExecContext(r.Context(), query, args...); err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}
		}

		// SELECT the updated account with latest balance and return it
		var id, name, originalName, effectiveType, currency string
		var orgName sql.NullString
		var displayName sql.NullString
		var hiddenAt sql.NullString
		var accountTypeOverride sql.NullString
		var balance sql.NullString

		err = database.QueryRowContext(r.Context(), `
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
			WHERE a.id = ?
		`, accountID).Scan(&id, &name, &originalName, &effectiveType, &currency, &orgName,
			&displayName, &hiddenAt, &accountTypeOverride, &balance)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to read updated account: %s"}`, err), http.StatusInternalServerError)
			return
		}

		balanceStr := "0"
		if balance.Valid {
			balanceStr = balance.String
		}

		item := accountItem{
			ID:           id,
			Name:         name,
			OriginalName: originalName,
			Type:         effectiveType,
			Balance:      balanceStr,
			Currency:     currency,
		}
		if orgName.Valid {
			item.OrgName = orgName.String
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(item) //nolint:errcheck
	}
}
