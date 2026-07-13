CREATE TABLE orders (
    id UUID PRIMARY KEY,

    user_id UUID NOT NULL
        REFERENCES users(id),

    symbol TEXT NOT NULL
        REFERENCES symbols(symbol),

    side TEXT NOT NULL,

    order_type TEXT NOT NULL,

    time_in_force TEXT NOT NULL,

    status TEXT NOT NULL,

    price BIGINT,

    stop_price BIGINT NOT NULL DEFAULT 0,

    quantity BIGINT NOT NULL,

    remaining BIGINT NOT NULL,

    filled BIGINT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT orders_side_check
        CHECK (side IN ('BUY', 'SELL')),

    CONSTRAINT orders_type_check
        CHECK (
            order_type IN (
                'LIMIT',
                'MARKET',
                'STOP_MARKET',
                'STOP_LIMIT'
            )
        ),

    CONSTRAINT orders_tif_check
        CHECK (
            time_in_force IN (
                'GTC',
                'IOC',
                'FOK',
                'POST_ONLY'
            )
        ),

    CONSTRAINT orders_status_check
        CHECK (
            status IN (
                'PENDING',
                'OPEN',
                'PARTIALLY_FILLED',
                'FILLED',
                'CANCELLED',
                'REJECTED'
            )
        )
);

-- Indexes
CREATE INDEX idx_orders_user
ON orders(user_id);

CREATE INDEX idx_orders_symbol
ON orders(symbol);

CREATE INDEX idx_orders_status
ON orders(status);

CREATE INDEX idx_orders_recovery
ON orders(symbol, created_at)
WHERE status IN (
    'OPEN',
    'PARTIALLY_FILLED',
    'PENDING'
);