-- 0005_create_positions.up.sql
-- Positions: each user's net holding per symbol, upserted as part of the
-- atomic 4-step trade-persistence transaction (architecture spec Section
-- 3.2 of the roadmap / Section 11): order status, counterparty status,
-- trade insert, and this position upsert all commit or roll back together.

CREATE TABLE positions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id)   ON DELETE RESTRICT,
    symbol     TEXT        NOT NULL REFERENCES symbols(symbol) ON DELETE RESTRICT,
    quantity   BIGINT      NOT NULL DEFAULT 0,   -- net position; buys increase, sells decrease
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT positions_user_symbol_unique UNIQUE (user_id, symbol)
);

-- GET /positions -- a user's full portfolio.
CREATE INDEX idx_positions_user ON positions (user_id);

-- positions_user_symbol_unique is also what the persistence worker's
-- upsert relies on:
--
--   INSERT INTO positions (user_id, symbol, quantity)
--   VALUES ($1, $2, $3)
--   ON CONFLICT (user_id, symbol)
--   DO UPDATE SET quantity   = positions.quantity + EXCLUDED.quantity,
--                 updated_at = now();