DROP INDEX IF EXISTS idx_failed_settlements_dead;

ALTER TABLE failed_settlements
DROP COLUMN IF EXISTS is_dead;