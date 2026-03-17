CREATE TABLE IF NOT EXISTS projection_account_settings (
    account_id  TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    apy         TEXT NOT NULL DEFAULT '0',
    compound    INTEGER NOT NULL DEFAULT 1,
    included    INTEGER NOT NULL DEFAULT 1,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS projection_holding_settings (
    holding_id  TEXT PRIMARY KEY,
    account_id  TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    apy         TEXT NOT NULL DEFAULT '0',
    compound    INTEGER NOT NULL DEFAULT 1,
    included    INTEGER NOT NULL DEFAULT 1,
    allocation  TEXT NOT NULL DEFAULT '0',
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_projection_holding_settings_account
    ON projection_holding_settings(account_id);

CREATE TABLE IF NOT EXISTS holdings (
    id          TEXT PRIMARY KEY,
    account_id  TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    symbol      TEXT,
    description TEXT NOT NULL,
    shares      TEXT,
    market_value TEXT NOT NULL,
    cost_basis  TEXT,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_holdings_account
    ON holdings(account_id);

CREATE TABLE IF NOT EXISTS projection_income_settings (
    id                  INTEGER PRIMARY KEY CHECK(id = 1),
    enabled             INTEGER NOT NULL DEFAULT 0,
    annual_income       TEXT NOT NULL DEFAULT '0',
    monthly_savings_pct TEXT NOT NULL DEFAULT '0',
    allocation_json     TEXT NOT NULL DEFAULT '{}',
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
