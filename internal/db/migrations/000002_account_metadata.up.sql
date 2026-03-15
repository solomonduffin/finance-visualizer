ALTER TABLE accounts ADD COLUMN display_name TEXT;
ALTER TABLE accounts ADD COLUMN hidden_at DATETIME;
ALTER TABLE accounts ADD COLUMN account_type_override TEXT
    CHECK(account_type_override IN ('checking', 'savings', 'credit', 'investment', 'other'));
