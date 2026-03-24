-- Migration 000006: Remove balance_snapshots that were stored under the wrong date.
--
-- Root cause: prior to this fix, processAccount used SimpleFIN's BalanceDate
-- (the bank's last-update timestamp) rather than today's wall-clock date.
-- If a bank hadn't posted new activity, BalanceDate could lag by days or weeks.
-- Every sync would overwrite that stale past-date snapshot with the current
-- balance, so the graph showed today's balance for historical dates.
--
-- Identification: rows where DATE(fetched_at) != balance_date were inserted
-- on a calendar day that differed from the bank's reported date. These are
-- definitively wrong — the balance recorded in them is the balance as of the
-- actual sync day, not the bank-reported date.
--
-- Rows where DATE(fetched_at) = balance_date were inserted on the same
-- calendar day the bank reported; they are treated as valid historical records.
DELETE FROM balance_snapshots
WHERE DATE(fetched_at) != balance_date;
