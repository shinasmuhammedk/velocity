CREATE TABLE failed_settlements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Deliberately no foreign keys here. This table exists to reliably
    -- capture a settlement failure even in unusual states (e.g. the
    -- trade/order/user rows themselves being the source of the
    -- problem) - adding constraints would risk this insert itself
    -- failing and losing the record it exists to preserve.
    trade_id BIGINT NOT NULL,
    buy_order_id BIGINT NOT NULL,
    sell_order_id BIGINT NOT NULL,
    buyer_id BIGINT NOT NULL,
    seller_id BIGINT NOT NULL,
    symbol TEXT NOT NULL,
    price BIGINT NOT NULL,
    quantity BIGINT NOT NULL,

    error_message TEXT NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    resolved BOOLEAN NOT NULL DEFAULT false,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX idx_failed_settlements_unresolved
ON failed_settlements(resolved)
WHERE resolved = false;

CREATE INDEX idx_failed_settlements_trade
ON failed_settlements(trade_id);