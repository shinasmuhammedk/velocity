-- name: UpsertCandle :one

INSERT INTO candles (
    symbol,
    interval,
    open_time,
    close_time,
    open,
    high,
    low,
    close,
    volume,
    quote_volume,
    trade_count
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11
)
ON CONFLICT (symbol, interval, open_time) DO UPDATE SET
    close_time   = EXCLUDED.close_time,
    open         = EXCLUDED.open,
    high         = EXCLUDED.high,
    low          = EXCLUDED.low,
    close        = EXCLUDED.close,
    volume       = EXCLUDED.volume,
    quote_volume = EXCLUDED.quote_volume,
    trade_count  = EXCLUDED.trade_count,
    updated_at   = now()
RETURNING *;


-- name: ListCandlesBySymbolInterval :many
SELECT *
FROM candles
WHERE symbol = $1
  AND interval = $2
ORDER BY open_time DESC
LIMIT $3;


-- name: ListCandlesBySymbolIntervalRange :many
SELECT *
FROM candles
WHERE symbol = $1
  AND interval = $2
  AND open_time >= $3
  AND open_time <= $4
ORDER BY open_time DESC
LIMIT $5;