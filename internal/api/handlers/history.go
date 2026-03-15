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

type balancePoint struct {
	Date    string `json:"date"`
	Balance string `json:"balance"`
}

type historyResponse struct {
	Liquid      []balancePoint `json:"liquid"`
	Savings     []balancePoint `json:"savings"`
	Investments []balancePoint `json:"investments"`
}

// dayAccumulator holds running decimal totals for a single calendar day.
type dayAccumulator struct {
	sumChecking    decimal.Decimal
	sumCredit      decimal.Decimal
	sumSavings     decimal.Decimal
	sumInvestments decimal.Decimal
	hasChecking    bool
	hasCredit      bool
	hasSavings     bool
	hasInvestments bool
}

// GetBalanceHistory returns an http.HandlerFunc that handles GET /api/balance-history.
// It returns time-series data grouped into three panels: liquid, savings, and investments.
//
// Liquid per day = sum(checking balances that day) + sum(credit balances that day).
// Credit balances are already negative, so they reduce the liquid total.
// "Other" type accounts are excluded from all three panels.
// Hidden accounts (hidden_at IS NOT NULL) are excluded.
// COALESCE(account_type_override, account_type) is used for effective type grouping.
//
// Optional query parameter: ?days=N limits results to the last N days (N must be positive integer).
// Invalid or non-positive values are ignored and all data is returned.
//
// Empty state returns {"liquid":[],"savings":[],"investments":[]} -- never null.
func GetBalanceHistory(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := historyResponse{
			Liquid:      []balancePoint{},
			Savings:     []balancePoint{},
			Investments: []balancePoint{},
		}

		// Parse optional ?days=N query parameter.
		days := 0
		if dStr := r.URL.Query().Get("days"); dStr != "" {
			if d, err := strconv.Atoi(dStr); err == nil && d > 0 {
				days = d
			}
		}

		// Build SQL query. Only include account types that map to dashboard panels.
		// Use DATE() to normalize balance_date to YYYY-MM-DD string format.
		// Use COALESCE for effective type and filter out hidden accounts.
		query := `
			SELECT DATE(bs.balance_date),
			       COALESCE(a.account_type_override, a.account_type) AS effective_type,
			       bs.balance
			FROM balance_snapshots bs
			JOIN accounts a ON a.id = bs.account_id
			WHERE a.hidden_at IS NULL
			  AND COALESCE(a.account_type_override, a.account_type) IN ('checking', 'credit', 'savings', 'investment')`

		if days > 0 {
			query += fmt.Sprintf(" AND bs.balance_date >= date('now', '-%d days')", days)
		}
		query += " ORDER BY bs.balance_date ASC, effective_type"

		rows, err := database.QueryContext(r.Context(), query)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Accumulate per-day totals using an ordered map (slice of dates + map of accumulators).
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
				// Skip rows with invalid balance values
				continue
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

		// SQL ORDER BY guarantees ascending date order, but sort defensively in Go
		// in case multiple rows per date get interleaved in unexpected ways.
		sort.Strings(dateOrder)

		// Build output slices. Only append to a panel if at least one account
		// of that panel's types had a snapshot on the given date.
		for _, date := range dateOrder {
			acc := dayMap[date]

			if acc.hasChecking || acc.hasCredit {
				liquid := acc.sumChecking.Add(acc.sumCredit)
				resp.Liquid = append(resp.Liquid, balancePoint{
					Date:    date,
					Balance: liquid.StringFixed(2),
				})
			}

			if acc.hasSavings {
				resp.Savings = append(resp.Savings, balancePoint{
					Date:    date,
					Balance: acc.sumSavings.StringFixed(2),
				})
			}

			if acc.hasInvestments {
				resp.Investments = append(resp.Investments, balancePoint{
					Date:    date,
					Balance: acc.sumInvestments.StringFixed(2),
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}
