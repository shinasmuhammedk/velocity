-- name: CreateWalletTransaction :one
INSERT INTO wallet_transactions (
    user_id,
    asset,
    amount,
    type,
    trade_id
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;


-- name: ListWalletTransactionsByUser :many
SELECT *
FROM wallet_transactions
WHERE user_id = $1
ORDER BY created_at DESC, id DESC;


-- name: ListWalletTransactionsByUserAndAsset :many
SELECT *
FROM wallet_transactions
WHERE user_id = $1
  AND asset = $2
ORDER BY created_at DESC, id DESC;