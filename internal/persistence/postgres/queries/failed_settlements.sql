-- name: CreateFailedSettlement :one
INSERT INTO failed_settlements (
    trade_id,
    buy_order_id,
    sell_order_id,
    buyer_id,
    seller_id,
    symbol,
    price,
    quantity,
    error_message
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


-- name: ListUnresolvedFailedSettlements :many
SELECT *
FROM failed_settlements
WHERE resolved = false
  AND is_dead = false
ORDER BY created_at ASC;


-- name: GetFailedSettlement :one
SELECT *
FROM failed_settlements
WHERE id = $1;


-- name: IncrementFailedSettlementRetryCount :exec
UPDATE failed_settlements
SET retry_count = retry_count + 1
WHERE id = $1;


-- name: ResolveFailedSettlement :exec
UPDATE failed_settlements
SET resolved = true,
    resolved_at = now()
WHERE id = $1;


-- name: MarkFailedSettlementDead :exec
UPDATE failed_settlements
SET is_dead = true
WHERE id = $1
  AND resolved = false;