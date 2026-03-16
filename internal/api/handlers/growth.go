package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/shopspring/decimal"
)

type growthData struct {
	CurrentTotal string `json:"current_total"`
	PriorTotal   string `json:"prior_total"`
	DollarChange string `json:"dollar_change"`
	PctChange    string `json:"pct_change"`
}

type growthResponse struct {
	Liquid             *growthData `json:"liquid"`
	Savings            *growthData `json:"savings"`
	Investments        *growthData `json:"investments"`
	GrowthBadgeEnabled bool        `json:"growth_badge_enabled"`
}

// panelTotals holds decimal sums for each panel category.
type panelTotals struct {
	liquid      decimal.Decimal
	savings     decimal.Decimal
	investments decimal.Decimal
	hasLiquid   bool
	hasSavings  bool
	hasInvest   bool
}

// queryPanelTotals runs a query returning (effective_type, balance) rows
// and accumulates totals per panel.
func queryPanelTotals(database *sql.DB, query string, args ...interface{}) (*panelTotals, error) {
	rows, err := database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pt := &panelTotals{
		liquid:      decimal.Zero,
		savings:     decimal.Zero,
		investments: decimal.Zero,
	}

	for rows.Next() {
		var effectiveType, balance string
		if err := rows.Scan(&effectiveType, &balance); err != nil {
			return nil, err
		}

		amount, err := decimal.NewFromString(balance)
		if err != nil {
			continue // skip invalid balance values
		}

		switch effectiveType {
		case "checking", "credit":
			pt.liquid = pt.liquid.Add(amount)
			pt.hasLiquid = true
		case "savings":
			pt.savings = pt.savings.Add(amount)
			pt.hasSavings = true
		case "investment":
			pt.investments = pt.investments.Add(amount)
			pt.hasInvest = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pt, nil
}

// computeGrowth calculates the growth data for a single panel.
// Returns nil if prior total is zero (to avoid division by zero) or if no data exists.
func computeGrowth(current, prior decimal.Decimal, hasCurrent, hasPrior bool) *growthData {
	if !hasCurrent || !hasPrior {
		return nil
	}
	if prior.IsZero() {
		return nil
	}

	change := current.Sub(prior)
	pct := change.Div(prior).Mul(decimal.NewFromInt(100))

	return &growthData{
		CurrentTotal: current.StringFixed(2),
		PriorTotal:   prior.StringFixed(2),
		DollarChange: change.StringFixed(2),
		PctChange:    pct.StringFixed(2),
	}
}

// GetGrowth returns an http.HandlerFunc that handles GET /api/growth.
// It returns per-panel 30-day growth data using shopspring/decimal arithmetic.
// Returns nil for panels with zero prior total (no division by zero).
func GetGrowth(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Query current panel totals (latest snapshot for each visible account)
		currentTotals, err := queryPanelTotals(database, `
			SELECT COALESCE(a.account_type_override, a.account_type) AS effective_type,
			       bs.balance
			FROM accounts a
			JOIN balance_snapshots bs ON bs.account_id = a.id
			WHERE a.hidden_at IS NULL
			  AND bs.balance_date = (
				SELECT MAX(bs2.balance_date)
				FROM balance_snapshots bs2
				WHERE bs2.account_id = a.id
			)
		`)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Query prior panel totals (closest snapshot at or before 30 days ago)
		priorTotals, err := queryPanelTotals(database, `
			SELECT COALESCE(a.account_type_override, a.account_type) AS effective_type,
			       bs.balance
			FROM accounts a
			JOIN balance_snapshots bs ON bs.account_id = a.id
			WHERE a.hidden_at IS NULL
			  AND bs.balance_date = (
				SELECT MAX(bs2.balance_date)
				FROM balance_snapshots bs2
				WHERE bs2.account_id = a.id
				  AND bs2.balance_date <= DATE('now', '-30 days')
			)
		`)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		resp := growthResponse{
			Liquid:      computeGrowth(currentTotals.liquid, priorTotals.liquid, currentTotals.hasLiquid, priorTotals.hasLiquid),
			Savings:     computeGrowth(currentTotals.savings, priorTotals.savings, currentTotals.hasSavings, priorTotals.hasSavings),
			Investments: computeGrowth(currentTotals.investments, priorTotals.investments, currentTotals.hasInvest, priorTotals.hasInvest),
		}

		// Query growth_badge_enabled setting (defaults to true)
		var growthEnabled string
		err = database.QueryRowContext(r.Context(),
			`SELECT value FROM settings WHERE key='growth_badge_enabled'`,
		).Scan(&growthEnabled)
		if err == sql.ErrNoRows {
			resp.GrowthBadgeEnabled = true
		} else if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		} else {
			resp.GrowthBadgeEnabled = growthEnabled != "false"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}
