-- name: CreateMatch :one
INSERT INTO matches (tournament_id, game_type, round, match_number, player_1_id, player_2_id, current_turn_user_id, status)
VALUES ($1, $2, $3, $4, $5, $6, NULL, 'waiting')
RETURNING *;

-- name: GetMatch :one
SELECT * FROM matches WHERE match_id = $1;

-- name: ListMatchesByTournament :many
SELECT * FROM matches WHERE tournament_id = $1 ORDER BY match_number;

-- name: GetActiveMatchForUser :one
SELECT * FROM matches
WHERE (player_1_id = $1 OR player_2_id = $1)
  AND status IN ('waiting', 'active')
ORDER BY match_id DESC
LIMIT 1;

-- name: ActivateMatch :one
UPDATE matches
SET status = 'active', started_at = now(), current_turn_user_id = $2
WHERE match_id = $1
RETURNING *;

-- name: SetMatchTurn :exec
UPDATE matches SET current_turn_user_id = $2 WHERE match_id = $1;

-- name: CompleteMatch :one
UPDATE matches
SET status = 'completed', winner_user_id = $2, completed_at = now(), current_turn_user_id = NULL
WHERE match_id = $1
RETURNING *;
