CREATE TABLE symbols (
    symbol TEXT PRIMARY KEY,

    display_name TEXT NOT NULL,

    tick_size BIGINT NOT NULL DEFAULT 1,
    lot_size BIGINT NOT NULL DEFAULT 1,

    is_active BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT tick_size_positive CHECK (tick_size > 0),
    CONSTRAINT lot_size_positive CHECK (lot_size > 0)
);