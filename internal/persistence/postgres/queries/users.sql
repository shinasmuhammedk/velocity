-- name: CreateUser :one
INSERT INTO users (
    id,
    email,
    password_hash
)
VALUES (
    $1,
    $2,
    $3
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


-- name: ListUsers :many
SELECT *
FROM users
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;


-- name: DeleteUser :exec
DELETE
FROM users
WHERE id = $1;