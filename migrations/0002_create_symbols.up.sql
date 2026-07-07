-- 0002_create_symbols.up.sql
-- Symbols: reference table backing the "symbol exists" validation rule
-- (architecture spec Section 4.5) and giving the Engine Registry a durable
-- list of tradable instruments instead of a hardcoded set. Not present in
-- the original four-table brief -- added because orders/trades/positions
-- all need a symbol to reference, and a hardcoded whitelist in application
-- code cannot be extended without a deploy.

CREATE TABLE symbols (
    symbol        TEXT        PRIMARY KEY,
    display_name  TEXT        NOT NULL,
    tick_size     BIGINT      NOT NULL DEFAULT 1,   -- smallest allowed price increment
    lot_size      BIGINT      NOT NULL DEFAULT 1,   -- smallest allowed quantity increment
    is_active     BOOLEAN     NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT symbols_tick_size_positive CHECK (tick_size > 0),
    CONSTRAINT symbols_lot_size_positive  CHECK (lot_size > 0)
);

-- Seed the symbols used across the roadmap's worked examples.
INSERT INTO symbols (symbol, display_name) VALUES
    ('DRFT', 'Draft Co.'),
    ('AAPL', 'Apple Inc.'),
    ('TSLA', 'Tesla Inc.');