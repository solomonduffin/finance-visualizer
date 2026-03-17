package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// projectionAccountSetting represents one account's projection configuration.
type projectionAccountSetting struct {
	AccountID   string                     `json:"account_id"`
	AccountName string                     `json:"account_name"`
	AccountType string                     `json:"account_type"`
	Balance     string                     `json:"balance"`
	APY         string                     `json:"apy"`
	Compound    bool                       `json:"compound"`
	Included    bool                       `json:"included"`
	Holdings    []projectionHoldingSetting `json:"holdings"`
}

// projectionHoldingSetting represents one holding's projection configuration.
type projectionHoldingSetting struct {
	HoldingID   string `json:"holding_id"`
	Symbol      string `json:"symbol"`
	Description string `json:"description"`
	MarketValue string `json:"market_value"`
	APY         string `json:"apy"`
	Compound    bool   `json:"compound"`
	Included    bool   `json:"included"`
	Allocation  string `json:"allocation"`
}

// projectionIncomeSettings represents the income modeling configuration.
type projectionIncomeSettings struct {
	Enabled           bool   `json:"enabled"`
	AnnualIncome      string `json:"annual_income"`
	MonthlySavingsPct string `json:"monthly_savings_pct"`
	AllocationJSON    string `json:"allocation_json"`
}

// projectionSettingsResponse is the full response for GET /api/projections/settings.
type projectionSettingsResponse struct {
	Accounts []projectionAccountSetting `json:"accounts"`
	Income   projectionIncomeSettings   `json:"income"`
}

// investmentTypes are the account types that can have holdings.
var investmentTypes = map[string]bool{
	"brokerage":  true,
	"retirement": true,
	"crypto":     true,
	"investment": true,
}

// GetProjectionSettings returns an http.HandlerFunc that handles GET /api/projections/settings.
// It returns all non-hidden accounts with their projection settings (apy, compound, included)
// and, for investment-type accounts, nested holdings with their own settings.
// Also includes the income modeling settings singleton.
func GetProjectionSettings(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Query all non-hidden accounts with LEFT JOIN to projection_account_settings
		// and the latest balance snapshot.
		rows, err := db.QueryContext(r.Context(), `
			SELECT
				a.id,
				COALESCE(a.display_name, a.name) as account_name,
				COALESCE(a.account_type_override, a.account_type) as effective_type,
				COALESCE(bs.balance, '0') as balance,
				COALESCE(ps.apy, '0') as apy,
				COALESCE(ps.compound, 1) as compound,
				COALESCE(ps.included, 1) as included
			FROM accounts a
			LEFT JOIN projection_account_settings ps ON a.id = ps.account_id
			LEFT JOIN (
				SELECT account_id, balance
				FROM balance_snapshots
				WHERE (account_id, balance_date) IN (
					SELECT account_id, MAX(balance_date) FROM balance_snapshots GROUP BY account_id
				)
			) bs ON a.id = bs.account_id
			WHERE a.hidden_at IS NULL
			ORDER BY COALESCE(a.account_type_override, a.account_type), COALESCE(a.display_name, a.name)
		`)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		accounts := []projectionAccountSetting{}
		for rows.Next() {
			var acct projectionAccountSetting
			var compound, included int
			if err := rows.Scan(
				&acct.AccountID, &acct.AccountName, &acct.AccountType,
				&acct.Balance, &acct.APY, &compound, &included,
			); err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}
			acct.Compound = intToBool(compound)
			acct.Included = intToBool(included)
			acct.Holdings = []projectionHoldingSetting{}
			accounts = append(accounts, acct)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		rows.Close()

		// For investment-type accounts, query holdings with LEFT JOIN to projection_holding_settings.
		for i := range accounts {
			if !investmentTypes[accounts[i].AccountType] {
				continue
			}

			holdingRows, err := db.QueryContext(r.Context(), `
				SELECT
					h.id, COALESCE(h.symbol, '') as symbol, h.description, h.market_value,
					COALESCE(phs.apy, '0') as apy,
					COALESCE(phs.compound, 1) as compound,
					COALESCE(phs.included, 1) as included,
					COALESCE(phs.allocation, '0') as allocation
				FROM holdings h
				LEFT JOIN projection_holding_settings phs ON h.id = phs.holding_id
				WHERE h.account_id = ?
				ORDER BY h.description
			`, accounts[i].AccountID)
			if err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}

			holdings := []projectionHoldingSetting{}
			for holdingRows.Next() {
				var h projectionHoldingSetting
				var compound, included int
				if err := holdingRows.Scan(
					&h.HoldingID, &h.Symbol, &h.Description, &h.MarketValue,
					&h.APY, &compound, &included, &h.Allocation,
				); err != nil {
					holdingRows.Close()
					http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
					return
				}
				h.Compound = intToBool(compound)
				h.Included = intToBool(included)
				holdings = append(holdings, h)
			}
			if err := holdingRows.Err(); err != nil {
				holdingRows.Close()
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}
			holdingRows.Close()

			accounts[i].Holdings = holdings
		}

		// Query income settings singleton.
		income := projectionIncomeSettings{
			Enabled:           false,
			AnnualIncome:      "0",
			MonthlySavingsPct: "0",
			AllocationJSON:    "{}",
		}

		var enabled int
		err = db.QueryRowContext(r.Context(),
			`SELECT enabled, annual_income, monthly_savings_pct, allocation_json
			 FROM projection_income_settings WHERE id = 1`,
		).Scan(&enabled, &income.AnnualIncome, &income.MonthlySavingsPct, &income.AllocationJSON)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		if err == nil {
			income.Enabled = intToBool(enabled)
		}

		resp := projectionSettingsResponse{
			Accounts: accounts,
			Income:   income,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}

// saveProjectionSettingsRequest is the body for PUT /api/projections/settings.
type saveProjectionSettingsRequest struct {
	Accounts []saveAccountSetting  `json:"accounts"`
	Holdings []saveHoldingSetting  `json:"holdings"`
}

type saveAccountSetting struct {
	AccountID string `json:"account_id"`
	APY       string `json:"apy"`
	Compound  bool   `json:"compound"`
	Included  bool   `json:"included"`
}

type saveHoldingSetting struct {
	HoldingID  string `json:"holding_id"`
	AccountID  string `json:"account_id"`
	APY        string `json:"apy"`
	Compound   bool   `json:"compound"`
	Included   bool   `json:"included"`
	Allocation string `json:"allocation"`
}

// SaveProjectionSettings returns an http.HandlerFunc that handles PUT /api/projections/settings.
// It upserts projection_account_settings and projection_holding_settings rows
// within a single transaction.
func SaveProjectionSettings(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req saveProjectionSettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		tx, err := db.BeginTx(r.Context(), nil)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		defer tx.Rollback() //nolint:errcheck

		for _, acct := range req.Accounts {
			_, err := tx.ExecContext(r.Context(),
				`INSERT OR REPLACE INTO projection_account_settings
				 (account_id, apy, compound, included, updated_at)
				 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`,
				acct.AccountID, acct.APY, boolToInt(acct.Compound), boolToInt(acct.Included),
			)
			if err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}
		}

		for _, h := range req.Holdings {
			_, err := tx.ExecContext(r.Context(),
				`INSERT OR REPLACE INTO projection_holding_settings
				 (holding_id, account_id, apy, compound, included, allocation, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
				h.HoldingID, h.AccountID, h.APY, boolToInt(h.Compound), boolToInt(h.Included), h.Allocation,
			)
			if err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	}
}

// SaveIncomeSettings returns an http.HandlerFunc that handles PUT /api/projections/income.
// It upserts the projection_income_settings singleton row.
func SaveIncomeSettings(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req projectionIncomeSettings
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		_, err := db.ExecContext(r.Context(),
			`INSERT INTO projection_income_settings
			 (id, enabled, annual_income, monthly_savings_pct, allocation_json, updated_at)
			 VALUES (1, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			 ON CONFLICT(id) DO UPDATE SET
			   enabled=excluded.enabled,
			   annual_income=excluded.annual_income,
			   monthly_savings_pct=excluded.monthly_savings_pct,
			   allocation_json=excluded.allocation_json,
			   updated_at=CURRENT_TIMESTAMP`,
			boolToInt(req.Enabled), req.AnnualIncome, req.MonthlySavingsPct, req.AllocationJSON,
		)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	}
}
