-- name: CreateTournamentPlayer :one
INSERT INTO tournament_players (tournament_id, user_id, seed_number)
VALUES ($1, $2, $3)
RETURNING *;

-- name: SetTournamentPlayerResult :exec
UPDATE tournament_players
SET final_result = $3
WHERE tournament_id = $1 AND user_id = $2;
