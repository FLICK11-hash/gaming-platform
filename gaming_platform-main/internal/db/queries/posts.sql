-- name: CreatePost :one
INSERT INTO posts (user_id, title, body, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPost :one
SELECT * FROM posts WHERE post_id = $1;

-- name: ListPostsByUser :many
SELECT * FROM posts WHERE user_id = $1 ORDER BY created_at DESC;

-- name: UpdatePostStatus :one
UPDATE posts SET status = $2 WHERE post_id = $1
RETURNING *;
