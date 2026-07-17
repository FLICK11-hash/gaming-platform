-- name: CreateTournament :one
INSERT INTO tournaments (game_type, game_mode, status, coin_wager, total_coin_pool, winner_coin_prize, bonus_coins)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetTournament :one
SELECT * FROM tournaments WHERE tournament_id = $1;

-- name: StartTournament :one
UPDATE tournaments
SET status = 'active', started_at = now()
WHERE tournament_id = $1
RETURNING *;

-- name: CompleteTournament :one
UPDATE tournaments
SET status = 'completed', winner_user_id = $2, completed_at = now()
WHERE tournament_id = $1
RETURNING *;
