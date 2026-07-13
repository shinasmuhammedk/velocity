-- name: CreatePosition :one
INSERT INTO positions (
    id,
    user_id,
    symbol,
    quantity,
    updated_at
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;


-- name: GetPosition :one
SELECT *
FROM positions
WHERE user_id = $1
AND symbol = $2;


-- name: GetPositionsByUser :many
SELECT *
FROM positions
WHERE user_id = $1
ORDER BY symbol ASC;


-- name: UpsertPosition :exec
INSERT INTO positions (
    user_id,
    symbol,
    quantity,
    updated_at
)
VALUES (
    $1,
    $2,
    $3,
    NOW()
)
ON CONFLICT (user_id, symbol)
DO UPDATE
SET
    quantity = positions.quantity + EXCLUDED.quantity,
    updated_at = NOW();


-- name: UpdatePosition :exec
UPDATE positions
SET
    quantity = $3,
    updated_at = NOW()
WHERE user_id = $1
AND symbol = $2;


-- name: DeletePosition :exec
DELETE FROM positions
WHERE user_id = $1
AND symbol = $2;


-- name: GetLongPositions :many
SELECT *
FROM positions
WHERE quantity > 0
ORDER BY quantity DESC;


-- name: GetShortPositions :many
SELECT *
FROM positions
WHERE quantity < 0
ORDER BY quantity ASC;

-- name: ListPositionsByUser :many
SELECT *
FROM positions
WHERE user_id = $1
ORDER BY updated_at DESC;