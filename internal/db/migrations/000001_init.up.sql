CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS accounts (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    account_type TEXT NOT NULL CHECK(account_type IN ('checking', 'savings', 'credit', 'investment', 'other')),
    currency     TEXT NOT NULL DEFAULT 'USD',
    org_name     TEXT,
    org_slug     TEXT,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS balance_snapshots (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id   TEXT NOT NULL REFERENCES accounts(id),
    balance      TEXT NOT NULL,
    balance_date DATE NOT NULL,
    fetched_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, balance_date)
);

CREATE INDEX IF NOT EXISTS idx_balance_snapshots_account_date
    ON balance_snapshots(account_id, balance_date DESC);

CREATE TABLE IF NOT EXISTS sync_log (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at       DATETIME NOT NULL,
    finished_at      DATETIME,
    accounts_fetched INTEGER DEFAULT 0,
    accounts_failed  INTEGER DEFAULT 0,
    error_text       TEXT
);
