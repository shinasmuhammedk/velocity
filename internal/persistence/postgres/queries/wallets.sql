-- name: CreateWallet :one
INSERT INTO wallets (
    user_id,
    asset,
    available,
    locked
)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetWallet :one
SELECT *
FROM wallets
WHERE user_id = $1
  AND asset = $2;

-- name: UpdateWallet :exec
UPDATE wallets
SET
    available = $2,
    locked = $3,
    updated_at = now()
WHERE id = $1;

-- name: ListWallets :many
SELECT *
FROM wallets
WHERE user_id = $1;

-- name: LockWalletFunds :execrows
UPDATE wallets
SET
    available = available - $2,
    locked = locked + $2,
    updated_at = NOW()
WHERE id = $1
  AND available >= $2;