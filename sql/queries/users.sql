-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, password)
VALUES (
    gen_random_uuid(), NOW(), NOW(), $1, $2
)
RETURNING *;

-- name: UpdateUser :one
UPDATE users SET email=$2, password = $3,
updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpgradeUser :one
UPDATE users SET is_chirpy_red = TRUE,
updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetUserByMail :one
SELECT * from users where email = $1 ORDER BY created_at ASC LIMIT 1;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: GetUserByRefreshToken :one
SELECT u.*
FROM users u
JOIN refresh_tokens rt ON rt.user_id = u.id
WHERE rt.token = $1
  AND (rt.revoked_at IS NULL)
  AND (rt.expires_at > NOW());