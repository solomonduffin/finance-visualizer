package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

// validPanelTypes are the allowed values for group panel_type.
var validPanelTypes = map[string]bool{
	"checking":   true,
	"savings":    true,
	"investment": true,
}

type createGroupRequest struct {
	Name      string `json:"name"`
	PanelType string `json:"panel_type"`
}

type updateGroupRequest struct {
	Name      *string `json:"name"`
	PanelType *string `json:"panel_type"`
}

type addMemberRequest struct {
	AccountID string `json:"account_id"`
}

type groupMember struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	OriginalName string  `json:"original_name"`
	Balance      string  `json:"balance"`
	Currency     string  `json:"currency"`
	OrgName      string  `json:"org_name"`
	DisplayName  *string `json:"display_name"`
}

type groupResponse struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	PanelType    string        `json:"panel_type"`
	TotalBalance string        `json:"total_balance"`
	Members      []groupMember `json:"members"`
}

// fetchGroupResponse builds a full groupResponse by querying the group and its members.
func fetchGroupResponse(database *sql.DB, groupID int64) (*groupResponse, error) {
	var grp groupResponse
	err := database.QueryRow(
		`SELECT id, name, panel_type FROM account_groups WHERE id = ?`, groupID,
	).Scan(&grp.ID, &grp.Name, &grp.PanelType)
	if err != nil {
		return nil, err
	}

	rows, err := database.Query(`
		SELECT a.id,
		       COALESCE(a.display_name, a.name) AS name,
		       a.name AS original_name,
		       a.currency, a.org_name, a.display_name,
		       COALESCE(bs.balance, '0') AS balance
		FROM group_members gm
		JOIN accounts a ON a.id = gm.account_id
		LEFT JOIN balance_snapshots bs ON bs.account_id = a.id
		  AND bs.balance_date = (
		      SELECT MAX(bs2.balance_date) FROM balance_snapshots bs2 WHERE bs2.account_id = a.id
		  )
		WHERE gm.group_id = ?
		ORDER BY name
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	totalBalance := decimal.Zero
	grp.Members = []groupMember{}

	for rows.Next() {
		var m groupMember
		var orgName sql.NullString
		var displayName sql.NullString
		var balance string

		if err := rows.Scan(&m.ID, &m.Name, &m.OriginalName, &m.Currency, &orgName, &displayName, &balance); err != nil {
			return nil, err
		}

		m.Balance = balance
		if orgName.Valid {
			m.OrgName = orgName.String
		}
		if displayName.Valid {
			s := displayName.String
			m.DisplayName = &s
		}

		amount, parseErr := decimal.NewFromString(balance)
		if parseErr == nil {
			totalBalance = totalBalance.Add(amount)
		}

		grp.Members = append(grp.Members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	grp.TotalBalance = totalBalance.StringFixed(2)
	return &grp, nil
}

// CreateGroup returns an http.HandlerFunc that handles POST /api/groups.
// It creates a new account group with a name and panel_type.
// Returns 201 with the created group (empty members, total_balance "0.00").
func CreateGroup(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
			return
		}
		if !validPanelTypes[req.PanelType] {
			http.Error(w, `{"error":"invalid panel_type; must be checking, savings, or investment"}`, http.StatusBadRequest)
			return
		}

		result, err := database.ExecContext(r.Context(),
			`INSERT INTO account_groups (name, panel_type) VALUES (?, ?)`,
			req.Name, req.PanelType,
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

		grp, err := fetchGroupResponse(database, id)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(grp) //nolint:errcheck
	}
}

// UpdateGroup returns an http.HandlerFunc that handles PATCH /api/groups/{id}.
// It updates the group name and/or panel_type. Returns 200 with the updated group.
func UpdateGroup(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		groupID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid group id"}`, http.StatusBadRequest)
			return
		}

		var req updateGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.Name == nil && req.PanelType == nil {
			http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
			return
		}

		// Validate panel_type if provided
		if req.PanelType != nil && !validPanelTypes[*req.PanelType] {
			http.Error(w, `{"error":"invalid panel_type; must be checking, savings, or investment"}`, http.StatusBadRequest)
			return
		}

		// Build dynamic UPDATE
		setClauses := []string{}
		args := []interface{}{}

		if req.Name != nil {
			setClauses = append(setClauses, "name = ?")
			args = append(args, *req.Name)
		}
		if req.PanelType != nil {
			setClauses = append(setClauses, "panel_type = ?")
			args = append(args, *req.PanelType)
		}

		setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")

		query := "UPDATE account_groups SET "
		for i, clause := range setClauses {
			if i > 0 {
				query += ", "
			}
			query += clause
		}
		query += " WHERE id = ?"
		args = append(args, groupID)

		result, err := database.ExecContext(r.Context(), query, args...)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			http.Error(w, `{"error":"group not found"}`, http.StatusNotFound)
			return
		}

		grp, err := fetchGroupResponse(database, groupID)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(grp) //nolint:errcheck
	}
}

// DeleteGroup returns an http.HandlerFunc that handles DELETE /api/groups/{id}.
// It deletes the group. CASCADE handles member cleanup. Returns 204 No Content.
// Returns 404 if the group does not exist.
func DeleteGroup(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		groupID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid group id"}`, http.StatusBadRequest)
			return
		}

		result, err := database.ExecContext(r.Context(),
			`DELETE FROM account_groups WHERE id = ?`, groupID,
		)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			http.Error(w, `{"error":"group not found"}`, http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// AddGroupMember returns an http.HandlerFunc that handles POST /api/groups/{id}/members.
// It adds an account to a group. Returns 409 if the account is already in any group.
// Returns 200 with the updated group including members and computed total_balance.
func AddGroupMember(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		groupID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid group id"}`, http.StatusBadRequest)
			return
		}

		var req addMemberRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.AccountID == "" {
			http.Error(w, `{"error":"account_id is required"}`, http.StatusBadRequest)
			return
		}

		tx, err := database.BeginTx(r.Context(), nil)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		defer tx.Rollback() //nolint:errcheck

		// Verify group exists
		var exists bool
		err = tx.QueryRowContext(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM account_groups WHERE id = ?)`, groupID,
		).Scan(&exists)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, `{"error":"group not found"}`, http.StatusNotFound)
			return
		}

		// Check account not already in any group
		var existingGroupID sql.NullInt64
		err = tx.QueryRowContext(r.Context(),
			`SELECT group_id FROM group_members WHERE account_id = ?`, req.AccountID,
		).Scan(&existingGroupID)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		if existingGroupID.Valid {
			http.Error(w, `{"error":"account already belongs to a group"}`, http.StatusConflict)
			return
		}

		// Insert member
		_, err = tx.ExecContext(r.Context(),
			`INSERT INTO group_members (group_id, account_id) VALUES (?, ?)`,
			groupID, req.AccountID,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to add member: %s"}`, err), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		grp, err := fetchGroupResponse(database, groupID)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(grp) //nolint:errcheck
	}
}

// RemoveGroupMember returns an http.HandlerFunc that handles DELETE /api/groups/{id}/members/{accountId}.
// It removes an account from a group. If the last member is removed, the group is auto-deleted.
// Returns 200 with {deleted_group: true/false}.
func RemoveGroupMember(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		groupID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid group id"}`, http.StatusBadRequest)
			return
		}

		accountID := chi.URLParam(r, "accountId")
		if accountID == "" {
			http.Error(w, `{"error":"missing account id"}`, http.StatusBadRequest)
			return
		}

		tx, err := database.BeginTx(r.Context(), nil)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		defer tx.Rollback() //nolint:errcheck

		// Delete the member
		result, err := tx.ExecContext(r.Context(),
			`DELETE FROM group_members WHERE group_id = ? AND account_id = ?`,
			groupID, accountID,
		)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			http.Error(w, `{"error":"member not found"}`, http.StatusNotFound)
			return
		}

		// Check remaining members
		var remaining int
		err = tx.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM group_members WHERE group_id = ?`, groupID,
		).Scan(&remaining)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		deletedGroup := false
		if remaining == 0 {
			_, err = tx.ExecContext(r.Context(),
				`DELETE FROM account_groups WHERE id = ?`, groupID,
			)
			if err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}
			deletedGroup = true
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"deleted_group": deletedGroup}) //nolint:errcheck
	}
}
