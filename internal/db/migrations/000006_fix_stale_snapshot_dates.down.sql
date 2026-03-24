-- Down migration: no-op.
-- Deleted rows cannot be recovered. The up migration removes corrupted data;
-- rolling back is not meaningful.
SELECT 1;
