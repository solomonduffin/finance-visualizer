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
	case strings.Contains(lower, "credit") || strings.Contains(lower, "card"):
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
//  6. Finalises the sync_log row.
//
// Per-account errors are isolated: one bad account does not abort the run.
// Concurrent calls are serialised via syncMu.
func SyncOnce(ctx context.Context, db *sql.DB) error {
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
		return nil
	}
	if err != nil {
		return fmt.Errorf("sync: read settings: %w", err)
	}

	// Create sync_log entry.
	res, err := db.ExecContext(ctx,
		`INSERT INTO sync_log(started_at) VALUES(CURRENT_TIMESTAMP)`,
	)
	if err != nil {
		return fmt.Errorf("sync: insert sync_log: %w", err)
	}
	logID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("sync: get sync_log id: %w", err)
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
		return fmt.Errorf("sync: count accounts: %w", err)
	}
	if accountCount == 0 {
		t := time.Now().Add(-30 * 24 * time.Hour)
		startDate = &t
	}

	// Fetch accounts from SimpleFIN.
	// Per the SimpleFIN spec, the /accounts endpoint is at {ACCESS_URL}/accounts.
	accountsURL := strings.TrimRight(accessURL, "/") + "/accounts"
	accountSet, err := simplefin.FetchAccounts(accountsURL, startDate)
	if err != nil {
		errText := err.Error()
		finalize(0, 0, &errText)
		return fmt.Errorf("sync: fetch accounts: %w", err)
	}

	fetched := 0
	failed := 0

	for _, acct := range accountSet.Accounts {
		if err := processAccount(ctx, db, acct); err != nil {
			slog.Warn("sync: account failed", "account_id", acct.ID, "err", err)
			failed++
			continue
		}
		fetched++
	}

	finalize(fetched, failed, nil)
	slog.Info("sync: complete", "fetched", fetched, "failed", failed)
	return nil
}

// processAccount upserts an account row and inserts an idempotent balance snapshot.
func processAccount(ctx context.Context, db *sql.DB, acct simplefin.Account) error {
	// Validate balance is a parseable decimal.
	if _, err := decimal.NewFromString(acct.Balance); err != nil {
		return fmt.Errorf("invalid balance %q: %w", acct.Balance, err)
	}

	// Derive balance_date from Unix epoch.
	balanceDate := time.Unix(acct.BalanceDate, 0).UTC().Format("2006-01-02")

	// Upsert account.
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

	// Insert snapshot (silently ignored on duplicate account_id + balance_date).
	_, err = db.ExecContext(ctx, `
		INSERT OR IGNORE INTO balance_snapshots(account_id, balance, balance_date)
		VALUES(?, ?, ?)
	`, acct.ID, acct.Balance, balanceDate)
	if err != nil {
		return fmt.Errorf("insert snapshot for %q: %w", acct.ID, err)
	}

	return nil
}

// RunScheduler runs SyncOnce once per day at syncHour (0-23, local time).
// It blocks until ctx is cancelled.
func RunScheduler(ctx context.Context, syncHour int, db *sql.DB) {
	for {
		next := NextRunTime(syncHour)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if err := SyncOnce(ctx, db); err != nil {
				slog.Error("sync: scheduler run failed", "err", err)
			}
		}
	}
}
