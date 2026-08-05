-- name: CreateUser :one
INSERT INTO users (
    id,
    email,
    created_at,
    updated_at
)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;


-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1;


-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1;


-- name: ExistsUser :one
SELECT EXISTS(
    SELECT 1
    FROM users
    WHERE id = $1
);


-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;


-- name: ListUsers :many
SELECT *
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;