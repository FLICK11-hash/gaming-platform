-- name: CreateUser :one
INSERT INTO users (
    username,
    email,
    password,
    current_elo,
    total_wins,
    total_losses,
    num_games,
    account_status,
    coin_balance,
    selected_avatar
)
VALUES (
    $1,
    $2,
    $3,
    1200,
    0,
    0,
    0,
    'active',
    1000,
    'rook'
)
RETURNING *;
-- name: GetUserByID :one
SELECT * FROM users WHERE user_id = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUserEloAndStats :one
UPDATE users
SET current_elo = $2,
    total_wins = $3,
    total_losses = $4,
    num_games = $5
WHERE user_id = $1
RETURNING *;

-- name: ClaimDailyLoginReward :one
-- Grants the daily reward once per rolling 24h window. Matches zero rows
-- (pgx.ErrNoRows) if the caller already claimed within the last 24h.
UPDATE users
SET coin_balance = coin_balance + sqlc.arg('reward_amount')::bigint,
    last_daily_reward_at = now()
WHERE user_id = sqlc.arg('user_id')
  AND (last_daily_reward_at IS NULL OR last_daily_reward_at <= now() - INTERVAL '24 hours')
RETURNING *;

-- name: SetSelectedAvatar :one
UPDATE users SET selected_avatar = $2 WHERE user_id = $1
RETURNING *;

-- name: SetSelectedBadge :one
UPDATE users SET selected_badge = $2 WHERE user_id = $1
RETURNING *;
