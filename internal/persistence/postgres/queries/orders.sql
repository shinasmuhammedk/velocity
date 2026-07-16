-- name: CreateOrder :one
INSERT INTO orders (
    id,
    user_id,
    symbol,
    side,
    order_type,
    time_in_force,
    status,
    price,
    stop_price,
    quantity,
    remaining,
    filled,
    created_at,
    updated_at
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
    $11,
    $12,
    $13,
    $14
)
RETURNING *;


-- name: GetOrderByID :one
SELECT *
FROM orders
WHERE id = $1;


-- name: GetOrdersByUser :many
SELECT *
FROM orders
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;


-- name: GetOrdersBySymbol :many
SELECT *
FROM orders
WHERE symbol = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;


-- name: GetOpenOrdersBySymbol :many
SELECT *
FROM orders
WHERE symbol = $1
AND status IN (
    'OPEN',
    'PARTIALLY_FILLED'
)
ORDER BY created_at ASC;


-- name: UpdateOrderAfterTrade :exec
UPDATE orders
SET
    remaining = $2,
    filled = $3,
    status = $4,
    updated_at = NOW()
WHERE id = $1;


-- name: CancelOrder :exec
UPDATE orders
SET
    status = 'CANCELLED',
    updated_at = NOW()
WHERE id = $1;


-- name: RejectOrder :exec
UPDATE orders
SET
    status = 'REJECTED',
    updated_at = NOW()
WHERE id = $1;


-- name: UpdateOrderStatus :exec
UPDATE orders
SET
    status = $2,
    updated_at = NOW()
WHERE id = $1;


-- name: RecoveryOrders :many
SELECT *
FROM orders
WHERE status IN (
    'OPEN',
    'PARTIALLY_FILLED'
)
ORDER BY symbol, created_at ASC;


-- name: GetPendingStopOrders :many
SELECT *
FROM orders
WHERE status = 'PENDING'
AND order_type IN (
    'STOP_MARKET',
    'STOP_LIMIT'
)
ORDER BY created_at ASC;

-- name: ListOrdersByUser :many
SELECT *
FROM orders
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: ListOpenOrders :many
SELECT *
FROM orders
WHERE symbol = $1
AND status IN (
    'OPEN',
    'PARTIALLY_FILLED',
    'PENDING'
)
ORDER BY created_at ASC;

-- name: UpdateOrderForModify :exec
UPDATE orders
SET
    price = $2,
    quantity = $3,
    remaining = $4,
    updated_at = NOW()
WHERE id = $1;