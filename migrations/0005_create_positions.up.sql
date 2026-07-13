CREATE TABLE positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL
        REFERENCES users(id),

    symbol TEXT NOT NULL
        REFERENCES symbols(symbol),

    quantity BIGINT NOT NULL DEFAULT 0,

    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE(user_id, symbol)
);

CREATE INDEX idx_positions_user
ON positions(user_id);