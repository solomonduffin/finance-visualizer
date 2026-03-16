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

type groupGrowthData struct {
	GroupID int         `json:"group_id"`
	Name    string      `json:"name"`
	Growth  *growthData `json:"growth"`
}

type growthResponse struct {
	Liquid             *growthData       `json:"liquid"`
	Savings            *growthData       `json:"savings"`
	Investments        *growthData       `json:"investments"`
	GrowthBadgeEnabled bool              `json:"growth_badge_enabled"`
	Groups             []groupGrowthData `json:"groups"`
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

// addToPanel adds an amount to the appropriate panel based on panelType.
func (pt *panelTotals) addToPanel(panelType string, amount decimal.Decimal) {
	switch panelType {
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

		pt.addToPanel(effectiveType, amount)
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

// groupTotalRow holds a group's panel_type and the sum of its member balances.
type groupTotalRow struct {
	groupID   int
	name      string
	panelType string
	total     decimal.Decimal
	hasData   bool
}

// queryGroupTotals queries all groups and sums member balances from a balance query.
// The snapshotCondition is the WHERE clause for selecting the right snapshot
// (e.g., latest or 30-days-ago).
func queryGroupTotals(database *sql.DB, snapshotCondition string) ([]groupTotalRow, error) {
	query := `
		SELECT g.id, g.name, g.panel_type,
		       COALESCE(bs.balance, '0') AS balance
		FROM account_groups g
		JOIN group_members gm ON gm.group_id = g.id
		JOIN accounts a ON a.id = gm.account_id AND a.hidden_at IS NULL
		LEFT JOIN balance_snapshots bs ON bs.account_id = a.id
		  AND bs.balance_date = (
			` + snapshotCondition + `
		  )
		ORDER BY g.id
	`

	rows, err := database.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groupMap := map[int]*groupTotalRow{}
	var groupOrder []int

	for rows.Next() {
		var gID int
		var gName, gPanelType, balance string
		if err := rows.Scan(&gID, &gName, &gPanelType, &balance); err != nil {
			return nil, err
		}

		grp, exists := groupMap[gID]
		if !exists {
			grp = &groupTotalRow{
				groupID:   gID,
				name:      gName,
				panelType: gPanelType,
				total:     decimal.Zero,
			}
			groupMap[gID] = grp
			groupOrder = append(groupOrder, gID)
		}

		amount, parseErr := decimal.NewFromString(balance)
		if parseErr == nil && !amount.IsZero() {
			grp.total = grp.total.Add(amount)
			grp.hasData = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]groupTotalRow, 0, len(groupOrder))
	for _, id := range groupOrder {
		result = append(result, *groupMap[id])
	}
	return result, nil
}

// GetGrowth returns an http.HandlerFunc that handles GET /api/growth.
// It returns per-panel 30-day growth data using shopspring/decimal arithmetic.
// Grouped accounts are excluded from panel totals and instead contribute
// via their group's panel_type. Per-group growth is also returned.
// Returns nil for panels with zero prior total (no division by zero).
func GetGrowth(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Query current panel totals for STANDALONE accounts only (exclude grouped accounts)
		currentTotals, err := queryPanelTotals(database, `
			SELECT COALESCE(a.account_type_override, a.account_type) AS effective_type,
			       bs.balance
			FROM accounts a
			JOIN balance_snapshots bs ON bs.account_id = a.id
			WHERE a.hidden_at IS NULL
			  AND a.id NOT IN (SELECT account_id FROM group_members)
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

		// Query prior panel totals for STANDALONE accounts only
		priorTotals, err := queryPanelTotals(database, `
			SELECT COALESCE(a.account_type_override, a.account_type) AS effective_type,
			       bs.balance
			FROM accounts a
			JOIN balance_snapshots bs ON bs.account_id = a.id
			WHERE a.hidden_at IS NULL
			  AND a.id NOT IN (SELECT account_id FROM group_members)
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

		// Query group totals (current)
		currentGroupTotals, err := queryGroupTotals(database,
			`SELECT MAX(bs2.balance_date) FROM balance_snapshots bs2 WHERE bs2.account_id = a.id`,
		)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Query group totals (prior - 30 days ago)
		priorGroupTotals, err := queryGroupTotals(database,
			`SELECT MAX(bs2.balance_date) FROM balance_snapshots bs2 WHERE bs2.account_id = a.id AND bs2.balance_date <= DATE('now', '-30 days')`,
		)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Add group totals to the appropriate panels (based on group's panel_type)
		for _, g := range currentGroupTotals {
			if g.hasData {
				currentTotals.addToPanel(g.panelType, g.total)
			}
		}
		for _, g := range priorGroupTotals {
			if g.hasData {
				priorTotals.addToPanel(g.panelType, g.total)
			}
		}

		resp := growthResponse{
			Liquid:      computeGrowth(currentTotals.liquid, priorTotals.liquid, currentTotals.hasLiquid, priorTotals.hasLiquid),
			Savings:     computeGrowth(currentTotals.savings, priorTotals.savings, currentTotals.hasSavings, priorTotals.hasSavings),
			Investments: computeGrowth(currentTotals.investments, priorTotals.investments, currentTotals.hasInvest, priorTotals.hasInvest),
			Groups:      []groupGrowthData{},
		}

		// Compute per-group growth
		priorGroupMap := map[int]groupTotalRow{}
		for _, g := range priorGroupTotals {
			priorGroupMap[g.groupID] = g
		}
		for _, current := range currentGroupTotals {
			prior, hasPrior := priorGroupMap[current.groupID]
			ggd := groupGrowthData{
				GroupID: current.groupID,
				Name:    current.name,
				Growth:  computeGrowth(current.total, prior.total, current.hasData, hasPrior && prior.hasData),
			}
			resp.Groups = append(resp.Groups, ggd)
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
