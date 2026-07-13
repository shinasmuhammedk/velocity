CREATE TABLE trades (
    id UUID PRIMARY KEY,

    buy_order_id UUID NOT NULL,
    sell_order_id UUID NOT NULL,

    buyer_id UUID NOT NULL,
    seller_id UUID NOT NULL,

    symbol TEXT NOT NULL
        REFERENCES symbols(symbol),

    price BIGINT NOT NULL,
    quantity BIGINT NOT NULL,

    executed_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_trades_symbol_time
ON trades(symbol, executed_at DESC);

CREATE INDEX idx_trades_buy_order
ON trades(buy_order_id);

CREATE INDEX idx_trades_sell_order
ON trades(sell_order_id);

CREATE INDEX idx_trades_buyer
ON trades(buyer_id);

CREATE INDEX idx_trades_seller
ON trades(seller_id);