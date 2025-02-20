-- name: CreateUser :one
INSERT INTO users(id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING id, created_at, updated_at, email, hashed_password;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: GetAllChirps :many
SELECT id, created_at, updated_at, body, user_id
FROM chirps
ORDER BY created_at ASC;

-- name: GetChirp :one
SELECT id, created_at, updated_at, body, user_id
FROM chirps
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, created_at, updated_at, email, hashed_password, is_chirpy_red
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, created_at, updated_at, email, hashed_password, is_chirpy_red
FROM users
WHERE id = $1;

-- name: UpdateUser :one
UPDATE users
SET email = $1, hashed_password = $2, updated_at = now()
WHERE id = $3
RETURNING id, email, created_at, updated_at;

-- name: UpgradeUsertoChirpyRed :one
UPDATE users
SET is_chirpy_red = true, updated_at = now()
WHERE id = $1
RETURNING id, created_at, updated_at, email, hashed_password, is_chirpy_red;