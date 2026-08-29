ALTER TABLE failed_settlements
ADD COLUMN is_dead BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX idx_failed_settlements_dead
ON failed_settlements(is_dead)
WHERE is_dead = true;