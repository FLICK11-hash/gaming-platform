-- name: CreateFollow :exec
INSERT INTO follows (following_user_id, followed_user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DeleteFollow :exec
DELETE FROM follows WHERE following_user_id = $1 AND followed_user_id = $2;

-- name: ListFollowers :many
SELECT * FROM follows WHERE followed_user_id = $1 ORDER BY created_at DESC;

-- name: ListFollowing :many
SELECT * FROM follows WHERE following_user_id = $1 ORDER BY created_at DESC;
