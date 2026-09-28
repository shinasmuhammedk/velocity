ALTER TABLE failed_settlements
ADD COLUMN executed_at TIMESTAMPTZ NOT NULL DEFAULT now();