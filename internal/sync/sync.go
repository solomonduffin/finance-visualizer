// Package sync provides sync orchestration for fetching financial account data
// from SimpleFIN and persisting it to the local SQLite database.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	gosync "sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/solomon/finance-visualizer/internal/alerts"
	"github.com/solomon/finance-visualizer/internal/simplefin"
)

// syncMu prevents concurrent SyncOnce executions.
var syncMu gosync.Mutex

// InferAccountType maps a human-readable account name to one of the schema
// enum values: checking, savings, credit, investment, other.
// Matching is case-insensitive; the first keyword hit wins.
func InferAccountType(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "saving"):
		return "savings"
	case strings.Contains(lower, "credit") || strings.Contains(lower, "card") ||
		strings.Contains(lower, "sapphire") || strings.Contains(lower, "platinum"):
		return "credit"
	case strings.Contains(lower, "invest") || strings.Contains(lower, "brokerage") ||
		strings.Contains(lower, "ira") || strings.Contains(lower, "401"):
		return "investment"
	case strings.Contains(lower, "check"):
		return "checking"
	default:
		return "other"
	}
}

// NextRunTime returns the next wall-clock time at which the given hour (0-23,
// local time) will occur. If that hour has already passed today, tomorrow is
// returned.
func NextRunTime(hour int) time.Time {
	now := time.Now()
	candidate := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !candidate.After(now) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate
}

// SyncOnce performs a single synchronisation cycle:
//  1. Reads simplefin_access_url from the settings table. If absent, returns nil (no-op).
//  2. Inserts a sync_log row and captures its ID.
//  3. Decides start-date (30 days ago on first sync, nil otherwise).
//  4. Fetches accounts from SimpleFIN.
//  5. Upserts each account and inserts an idempotent balance snapshot.
//  6. Restores any previously soft-deleted accounts that reappeared.
//  7. Soft-deletes accounts not in the latest fetch (sets hidden_at, preserves snapshots).
//  8. Finalises the sync_log row.
//
// Returns the display names of any restored accounts (for frontend toast notification).
// Per-account errors are isolated: one bad account does not abort the run.
// Concurrent calls are serialised via syncMu.
func SyncOnce(ctx context.Context, db *sql.DB, jwtSecret string) ([]string, error) {
	syncMu.Lock()
	defer syncMu.Unlock()

	slog.Info("sync: starting")

	// Read access URL from settings.
	var accessURL string
	err := db.QueryRowContext(ctx,
		`SELECT value FROM settings WHERE key='simplefin_access_url'`,
	).Scan(&accessURL)
	if err == sql.ErrNoRows || accessURL == "" {
		// No access URL configured — silent no-op.
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("sync: read settings: %w", err)
	}

	// Create sync_log entry.
	res, err := db.ExecContext(ctx,
		`INSERT INTO sync_log(started_at) VALUES(CURRENT_TIMESTAMP)`,
	)
	if err != nil {
		return nil, fmt.Errorf("sync: insert sync_log: %w", err)
	}
	logID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("sync: get sync_log id: %w", err)
	}

	// Helper to finalise sync_log regardless of outcome.
	finalize := func(fetched, failed int, errText *string) {
		_, ferr := db.ExecContext(ctx,
			`UPDATE sync_log SET finished_at=CURRENT_TIMESTAMP, accounts_fetched=?, accounts_failed=?, error_text=? WHERE id=?`,
			fetched, failed, errText, logID,
		)
		if ferr != nil {
			slog.Error("sync: failed to update sync_log", "err", ferr)
		}
	}

	// Determine start date: 30 days ago on first sync, nil on subsequent.
	var startDate *time.Time
	var accountCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounts`).Scan(&accountCount); err != nil {
		errText := err.Error()
		finalize(0, 0, &errText)
		return nil, fmt.Errorf("sync: count accounts: %w", err)
	}
	if accountCount == 0 {
		t := time.Now().Add(-30 * 24 * time.Hour)
		startDate = &t
	}

	// Fetch accounts from SimpleFIN (with holdings data for investment accounts).
	// Per the SimpleFIN spec, the /accounts endpoint is at {ACCESS_URL}/accounts.
	accountsURL := strings.TrimRight(accessURL, "/") + "/accounts"
	accountSet, err := simplefin.FetchAccountsWithHoldings(accountsURL, startDate)
	if err != nil {
		errText := err.Error()
		finalize(0, 0, &errText)
		return nil, fmt.Errorf("sync: fetch accounts: %w", err)
	}

	fetched := 0
	failed := 0
	seenIDs := make([]string, 0, len(accountSet.Accounts))

	for _, acct := range accountSet.Accounts {
		if err := processAccount(ctx, db, acct); err != nil {
			slog.Warn("sync: account failed", "account_id", acct.ID, "err", err)
			failed++
			continue
		}
		seenIDs = append(seenIDs, acct.ID)
		fetched++

		// Persist holdings for investment-type accounts with holdings data.
		acctType := InferAccountType(acct.Name)
		if acctType == "investment" && len(acct.Holdings) > 0 {
			if err := persistHoldings(ctx, db, acct.ID, acct.Holdings); err != nil {
				slog.Warn("sync: persist holdings failed", "account_id", acct.ID, "err", err)
			}
		}
	}

	// Restore and soft-delete in correct order: restore BEFORE soft-delete.
	var restored []string
	if len(seenIDs) > 0 {
		// (a) Restore accounts that reappeared in this sync.
		var restoreErr error
		restored, restoreErr = restoreReturningAccounts(ctx, db, seenIDs)
		if restoreErr != nil {
			slog.Warn("sync: restore returning accounts failed", "err", restoreErr)
		} else if len(restored) > 0 {
			slog.Info("sync: restored returning accounts", "names", restored)
		}

		// (b) Soft-delete accounts not in this sync (set hidden_at, preserve snapshots).
		softDeleted, sdErr := softDeleteStaleAccounts(ctx, db, seenIDs)
		if sdErr != nil {
			slog.Warn("sync: soft-delete stale accounts failed", "err", sdErr)
		} else if softDeleted > 0 {
			slog.Info("sync: soft-deleted stale accounts", "count", softDeleted)
		}
	}

	finalize(fetched, failed, nil)

	// Evaluate alert rules (best-effort, never fails sync)
	if fetched > 0 {
		if evalErr := alerts.EvaluateAll(ctx, db, jwtSecret); evalErr != nil {
			slog.Error("sync: alert evaluation failed", "err", evalErr)
		}
	}

	slog.Info("sync: complete", "fetched", fetched, "failed", failed)
	return restored, nil
}

// processAccount upserts an account row and inserts an idempotent balance snapshot.
//
// IMPORTANT: The ON CONFLICT SET clause only updates system-owned columns:
//   - name, account_type, currency, org_name, org_slug, updated_at
//
// User-owned columns are NOT included and must NOT be added here:
//   - display_name (user-set custom name)
//   - hidden_at (soft-delete state)
//   - account_type_override (user-set type override)
func processAccount(ctx context.Context, db *sql.DB, acct simplefin.Account) error {
	// Validate balance is a parseable decimal.
	if _, err := decimal.NewFromString(acct.Balance); err != nil {
		return fmt.Errorf("invalid balance %q: %w", acct.Balance, err)
	}

	// Derive balance_date from Unix epoch.
	balanceDate := time.Unix(acct.BalanceDate, 0).UTC().Format("2006-01-02")

	// Upsert account. Only system-owned columns are updated on conflict.
	// User-owned columns (display_name, hidden_at, account_type_override) are preserved.
	_, err := db.ExecContext(ctx, `
		INSERT INTO accounts(id, name, account_type, currency, org_name, org_slug)
		VALUES(?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			account_type=excluded.account_type,
			currency=excluded.currency,
			org_name=excluded.org_name,
			org_slug=excluded.org_slug,
			updated_at=CURRENT_TIMESTAMP
	`, acct.ID, acct.Name, InferAccountType(acct.Name), acct.Currency, acct.Org.Name, acct.Org.ID)
	if err != nil {
		return fmt.Errorf("upsert account %q: %w", acct.ID, err)
	}

	// Upsert snapshot — update balance if a snapshot for this date already exists.
	_, err = db.ExecContext(ctx, `
		INSERT INTO balance_snapshots(account_id, balance, balance_date)
		VALUES(?, ?, ?)
		ON CONFLICT(account_id, balance_date) DO UPDATE SET balance=excluded.balance
	`, acct.ID, acct.Balance, balanceDate)
	if err != nil {
		return fmt.Errorf("insert snapshot for %q: %w", acct.ID, err)
	}

	return nil
}

// persistHoldings replaces all holdings for an account with the latest data
// from SimpleFIN. It deletes existing holdings and inserts the new set within
// a transaction for atomicity.
func persistHoldings(ctx context.Context, db *sql.DB, accountID string, holdings []simplefin.Holding) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Delete all existing holdings for this account.
	if _, err := tx.ExecContext(ctx, `DELETE FROM holdings WHERE account_id = ?`, accountID); err != nil {
		return fmt.Errorf("delete stale holdings: %w", err)
	}

	// Insert each holding.
	for _, h := range holdings {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO holdings (id, account_id, symbol, description, shares, market_value, cost_basis, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			h.ID, accountID, h.Symbol, h.Description, h.Shares, h.MarketValue, h.CostBasis,
		); err != nil {
			return fmt.Errorf("insert holding %q: %w", h.ID, err)
		}
	}

	return tx.Commit()
}

// softDeleteStaleAccounts sets hidden_at on accounts that were not returned
// by the latest SimpleFIN fetch. Only targets accounts where hidden_at IS NULL
// (does not re-hide already hidden accounts). Does NOT delete balance_snapshots.
// Returns the number of newly soft-deleted accounts.
func softDeleteStaleAccounts(ctx context.Context, db *sql.DB, seenIDs []string) (int64, error) {
	placeholders := make([]string, len(seenIDs))
	args := make([]interface{}, len(seenIDs))
	for i, id := range seenIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	inClause := strings.Join(placeholders, ",")

	res, err := db.ExecContext(ctx,
		fmt.Sprintf(`UPDATE accounts SET hidden_at = CURRENT_TIMESTAMP WHERE id NOT IN (%s) AND hidden_at IS NULL`, inClause),
		args...)
	if err != nil {
		return 0, fmt.Errorf("soft-delete stale accounts: %w", err)
	}
	return res.RowsAffected()
}

// restoreReturningAccounts clears hidden_at for accounts that reappeared in the
// latest SimpleFIN fetch. Returns the display names (COALESCE(display_name, name))
// of restored accounts for frontend toast notification.
func restoreReturningAccounts(ctx context.Context, db *sql.DB, seenIDs []string) ([]string, error) {
	placeholders := make([]string, len(seenIDs))
	args := make([]interface{}, len(seenIDs))
	for i, id := range seenIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	inClause := strings.Join(placeholders, ",")

	// First, get display names of accounts that will be restored.
	rows, err := db.QueryContext(ctx,
		fmt.Sprintf(`SELECT COALESCE(display_name, name) FROM accounts WHERE id IN (%s) AND hidden_at IS NOT NULL`, inClause),
		args...)
	if err != nil {
		return nil, fmt.Errorf("query returning accounts: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan returning account name: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate returning accounts: %w", err)
	}

	if len(names) == 0 {
		return nil, nil
	}

	// Then clear hidden_at.
	_, err = db.ExecContext(ctx,
		fmt.Sprintf(`UPDATE accounts SET hidden_at = NULL WHERE id IN (%s) AND hidden_at IS NOT NULL`, inClause),
		args...)
	if err != nil {
		return nil, fmt.Errorf("restore returning accounts: %w", err)
	}

	return names, nil
}

// RunScheduler runs SyncOnce once per day at syncHour (0-23, local time).
// It blocks until ctx is cancelled.
func RunScheduler(ctx context.Context, syncHour int, db *sql.DB, jwtSecret string) {
	for {
		next := NextRunTime(syncHour)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if _, err := SyncOnce(ctx, db, jwtSecret); err != nil {
				slog.Error("sync: scheduler run failed", "err", err)
			}
		}
	}
}
