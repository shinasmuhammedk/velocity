CREATE TABLE candles (
    symbol TEXT NOT NULL
        REFERENCES symbols(symbol),

    interval TEXT NOT NULL,

    open_time  TIMESTAMPTZ NOT NULL,
    close_time TIMESTAMPTZ NOT NULL,

    open  BIGINT NOT NULL,
    high  BIGINT NOT NULL,
    low   BIGINT NOT NULL,
    close BIGINT NOT NULL,

    volume       BIGINT NOT NULL,
    quote_volume BIGINT NOT NULL,
    trade_count  BIGINT NOT NULL,

    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (symbol, interval, open_time)
);

CREATE INDEX idx_candles_symbol_interval_time
ON candles(symbol, interval, open_time DESC);