-- name: CreateSymbol :one
INSERT INTO symbols (
    symbol,
    display_name,
    tick_size,
    lot_size,
    is_active,
    created_at
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    NOW()
)
RETURNING *;


-- name: GetSymbol :one
SELECT *
FROM symbols
WHERE symbol = $1;


-- name: ListSymbols :many
SELECT *
FROM symbols
ORDER BY symbol;


-- name: ListActiveSymbols :many
SELECT *
FROM symbols
WHERE is_active = true
ORDER BY symbol;


-- name: UpdateSymbolStatus :exec
UPDATE symbols
SET
    is_active = $2
WHERE symbol = $1;


-- name: DeleteSymbol :exec
DELETE FROM symbols
WHERE symbol = $1;