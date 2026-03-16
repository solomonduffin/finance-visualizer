CREATE TABLE IF NOT EXISTS alert_rules (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    name                TEXT NOT NULL,
    operands            TEXT NOT NULL,
    expression          TEXT NOT NULL,
    comparison          TEXT NOT NULL CHECK(comparison IN ('<', '<=', '>', '>=', '==')),
    threshold           TEXT NOT NULL,
    notify_on_recovery  INTEGER NOT NULL DEFAULT 1,
    enabled             INTEGER NOT NULL DEFAULT 1,
    last_state          TEXT NOT NULL DEFAULT 'normal' CHECK(last_state IN ('normal', 'triggered', 'recovered')),
    last_eval_at        DATETIME,
    last_value          TEXT,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS alert_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id     INTEGER NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    state       TEXT NOT NULL CHECK(state IN ('triggered', 'recovered')),
    value       TEXT,
    notified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_alert_history_rule
    ON alert_history(rule_id, notified_at DESC);
