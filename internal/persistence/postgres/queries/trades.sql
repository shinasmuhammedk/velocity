-- name: CreateTrade :one
INSERT INTO trades (
    id,
    buy_order_id,
    sell_order_id,
    buyer_id,
    seller_id,
    symbol,
    price,
    quantity,
    executed_at
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
    $9
)
RETURNING *;


-- name: GetTradeByID :one
SELECT *
FROM trades
WHERE id = $1;


-- name: GetTradesBySymbol :many
SELECT *
FROM trades
WHERE symbol = $1
ORDER BY executed_at DESC
LIMIT $2 OFFSET $3;


-- name: GetTradesByBuyer :many
SELECT *
FROM trades
WHERE buyer_id = $1
ORDER BY executed_at DESC
LIMIT $2 OFFSET $3;


-- name: GetTradesBySeller :many
SELECT *
FROM trades
WHERE seller_id = $1
ORDER BY executed_at DESC
LIMIT $2 OFFSET $3;


-- name: GetTradesByUser :many
SELECT *
FROM trades
WHERE buyer_id = $1
   OR seller_id = $1
ORDER BY executed_at DESC
LIMIT $2 OFFSET $3;


-- name: GetTradesBetweenTimes :many
SELECT *
FROM trades
WHERE executed_at BETWEEN $1 AND $2
ORDER BY executed_at ASC;


-- name: GetRecentTradesBySymbol :many
SELECT *
FROM trades
WHERE symbol = $1
ORDER BY executed_at DESC
LIMIT $2;


-- name: GetLastTradePrice :one
SELECT price
FROM trades
WHERE symbol = $1
ORDER BY executed_at DESC
LIMIT 1;


-- name: GetTradeVolumeBySymbol :one
SELECT COALESCE(SUM(quantity), 0)
FROM trades
WHERE symbol = $1;


-- name: GetTradeVolumeBetweenTimes :one
SELECT COALESCE(SUM(quantity), 0)
FROM trades
WHERE symbol = $1
AND executed_at BETWEEN $2 AND $3;

-- name: ListTradesByUser :many
SELECT *
FROM trades
WHERE buyer_id = $1
   OR seller_id = $1
ORDER BY executed_at DESC;

-- name: ListTradesBySymbol :many
SELECT *
FROM trades
WHERE symbol = $1
ORDER BY executed_at DESC;