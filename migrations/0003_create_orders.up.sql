-- 0003_create_orders.up.sql
-- Orders: durable record of every order accepted by the engine. The engine's
-- own in-memory book is the authoritative state while an order is live
-- (architecture spec ADR-05); this table is what survives a restart and is
-- replayed by internal/engine/recovery (Section 11.5 / 6.10).

CREATE TABLE orders (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id)   ON DELETE RESTRICT,
    symbol      TEXT        NOT NULL REFERENCES symbols(symbol) ON DELETE RESTRICT,
    side        TEXT        NOT NULL,               -- BUY | SELL
    order_type  TEXT        NOT NULL,               -- LIMIT | MARKET
    price       BIGINT,                              -- integer, smallest currency unit (paise); NULL for MARKET
    quantity    BIGINT      NOT NULL,
    filled_qty  BIGINT      NOT NULL DEFAULT 0,
    status      TEXT        NOT NULL DEFAULT 'OPEN', -- OPEN | PARTIAL | FILLED | CANCELLED
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT orders_side_check           CHECK (side IN ('BUY', 'SELL')),
    CONSTRAINT orders_type_check           CHECK (order_type IN ('LIMIT', 'MARKET')),
    CONSTRAINT orders_status_check         CHECK (status IN ('OPEN', 'PARTIAL', 'FILLED', 'CANCELLED')),
    CONSTRAINT orders_quantity_positive    CHECK (quantity > 0),
    CONSTRAINT orders_filled_qty_valid     CHECK (filled_qty >= 0 AND filled_qty <= quantity),
    CONSTRAINT orders_limit_price_required CHECK (order_type <> 'LIMIT' OR price IS NOT NULL),
    CONSTRAINT orders_price_positive       CHECK (price IS NULL OR price > 0)
);

-- Own-order lookups: GET /orders for a user, most recent first.
CREATE INDEX idx_orders_user_created ON orders (user_id, created_at DESC);

-- General per-symbol chronological access (trade history joins, admin views).
CREATE INDEX idx_orders_symbol_created ON orders (symbol, created_at);

-- Recovery replay (architecture spec Section 11.5): on restart, the engine
-- rebuilds each symbol's book by replaying OPEN/PARTIAL orders in arrival
-- order. This partial index covers exactly that query and stays small
-- because FILLED/CANCELLED rows -- the overwhelming majority over time --
-- are excluded from it.
CREATE INDEX idx_orders_recovery
    ON orders (symbol, created_at)
    WHERE status IN ('OPEN', 'PARTIAL');