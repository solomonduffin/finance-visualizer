package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/shopspring/decimal"
)

type netWorthPoint struct {
	Date        string `json:"date"`
	Liquid      string `json:"liquid"`
	Savings     string `json:"savings"`
	Investments string `json:"investments"`
}

type netWorthStats struct {
	CurrentNetWorth    string  `json:"current_net_worth"`
	PeriodChangeDollar string  `json:"period_change_dollars"`
	PeriodChangePct    *string `json:"period_change_pct"` // nil if first total is zero
	AllTimeHigh        string  `json:"all_time_high"`
	AllTimeHighDate    string  `json:"all_time_high_date"`
}

type netWorthResponse struct {
	Points []netWorthPoint `json:"points"`
	Stats  *netWorthStats  `json:"stats"`
}

// GetNetWorth returns an http.HandlerFunc that handles GET /api/net-worth.
// It returns time-series data with per-panel breakdown (liquid, savings, investments)
// and summary statistics (current net worth, period change, all-time high).
//
// Query parameter: ?days=N limits results to the last N days (default 90).
// days=0 returns all data (no date filtering).
//
// Hidden accounts (hidden_at IS NOT NULL) are excluded.
// COALESCE(account_type_override, account_type) is used for effective type grouping.
// Missing panel data for a date carries forward the last known value.
// Empty state returns {"points":[], "stats":null}.
func GetNetWorth(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := netWorthResponse{
			Points: []netWorthPoint{},
		}

		// Parse optional ?days=N query parameter. Default is 90.
		days := 90
		if dStr := r.URL.Query().Get("days"); dStr != "" {
			if d, err := strconv.Atoi(dStr); err == nil && d >= 0 {
				days = d
			}
		}

		// Build SQL query.
		query := `
			SELECT DATE(bs.balance_date) AS bd,
			       COALESCE(a.account_type_override, a.account_type) AS effective_type,
			       bs.balance
			FROM balance_snapshots bs
			JOIN accounts a ON a.id = bs.account_id
			WHERE a.hidden_at IS NULL
			  AND COALESCE(a.account_type_override, a.account_type) IN ('checking', 'credit', 'savings', 'investment')`

		if days > 0 {
			query += fmt.Sprintf(" AND DATE(bs.balance_date) >= DATE('now', '-%d days')", days)
		}
		query += " ORDER BY bd, effective_type"

		rows, err := database.QueryContext(r.Context(), query)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Accumulate per-day panel totals using ordered map pattern.
		dateOrder := []string{}
		dayMap := map[string]*dayAccumulator{}

		for rows.Next() {
			var balanceDate, effectiveType, balance string
			if err := rows.Scan(&balanceDate, &effectiveType, &balance); err != nil {
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}

			amount, err := decimal.NewFromString(balance)
			if err != nil {
				continue // skip invalid balance values
			}

			acc, exists := dayMap[balanceDate]
			if !exists {
				acc = &dayAccumulator{}
				dayMap[balanceDate] = acc
				dateOrder = append(dateOrder, balanceDate)
			}

			switch effectiveType {
			case "checking":
				acc.sumChecking = acc.sumChecking.Add(amount)
				acc.hasChecking = true
			case "credit":
				acc.sumCredit = acc.sumCredit.Add(amount)
				acc.hasCredit = true
			case "savings":
				acc.sumSavings = acc.sumSavings.Add(amount)
				acc.hasSavings = true
			case "investment":
				acc.sumInvestments = acc.sumInvestments.Add(amount)
				acc.hasInvestments = true
			}
		}
		if err := rows.Err(); err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Sort chronologically (defensive).
		sort.Strings(dateOrder)

		if len(dateOrder) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp) //nolint:errcheck
			return
		}

		// Build points with carry-forward logic.
		// Track last known values per panel for carry-forward.
		lastLiquid := decimal.Zero
		lastSavings := decimal.Zero
		lastInvestments := decimal.Zero
		hasEverLiquid := false
		hasEverSavings := false
		hasEverInvestments := false

		// Track all-time high across the range.
		allTimeHigh := decimal.Zero
		allTimeHighDate := dateOrder[0]

		for _, date := range dateOrder {
			acc := dayMap[date]

			// Compute liquid for this day (checking + credit).
			var liquid decimal.Decimal
			if acc.hasChecking || acc.hasCredit {
				liquid = acc.sumChecking.Add(acc.sumCredit)
				lastLiquid = liquid
				hasEverLiquid = true
			} else if hasEverLiquid {
				liquid = lastLiquid // carry forward
			}

			// Compute savings for this day.
			var savings decimal.Decimal
			if acc.hasSavings {
				savings = acc.sumSavings
				lastSavings = savings
				hasEverSavings = true
			} else if hasEverSavings {
				savings = lastSavings // carry forward
			}

			// Compute investments for this day.
			var investments decimal.Decimal
			if acc.hasInvestments {
				investments = acc.sumInvestments
				lastInvestments = investments
				hasEverInvestments = true
			} else if hasEverInvestments {
				investments = lastInvestments // carry forward
			}

			resp.Points = append(resp.Points, netWorthPoint{
				Date:        date,
				Liquid:      liquid.StringFixed(2),
				Savings:     savings.StringFixed(2),
				Investments: investments.StringFixed(2),
			})

			// Track all-time high.
			total := liquid.Add(savings).Add(investments)
			if total.GreaterThan(allTimeHigh) {
				allTimeHigh = total
				allTimeHighDate = date
			}
		}

		// Compute stats.
		firstPoint := resp.Points[0]
		lastPoint := resp.Points[len(resp.Points)-1]

		firstLiquid, _ := decimal.NewFromString(firstPoint.Liquid)
		firstSavings, _ := decimal.NewFromString(firstPoint.Savings)
		firstInvestments, _ := decimal.NewFromString(firstPoint.Investments)
		firstTotal := firstLiquid.Add(firstSavings).Add(firstInvestments)

		lastLiquidDec, _ := decimal.NewFromString(lastPoint.Liquid)
		lastSavingsDec, _ := decimal.NewFromString(lastPoint.Savings)
		lastInvestmentsDec, _ := decimal.NewFromString(lastPoint.Investments)
		currentTotal := lastLiquidDec.Add(lastSavingsDec).Add(lastInvestmentsDec)

		periodChange := currentTotal.Sub(firstTotal)

		stats := &netWorthStats{
			CurrentNetWorth:    currentTotal.StringFixed(2),
			PeriodChangeDollar: periodChange.StringFixed(2),
			AllTimeHigh:        allTimeHigh.StringFixed(2),
			AllTimeHighDate:    allTimeHighDate,
		}

		// Compute period change percentage, nil if first total is zero.
		if !firstTotal.IsZero() {
			pct := periodChange.Div(firstTotal).Mul(decimal.NewFromInt(100))
			pctStr := pct.StringFixed(2)
			stats.PeriodChangePct = &pctStr
		}

		resp.Stats = stats

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}
