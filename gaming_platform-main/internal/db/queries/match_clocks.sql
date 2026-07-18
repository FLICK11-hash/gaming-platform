-- name: CreateMatchClock :one
INSERT INTO match_clocks (match_id, user_id, starting_seconds, remaining_seconds, is_current_turn, last_started_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListMatchClocksByMatch :many
SELECT * FROM match_clocks WHERE match_id = $1;

-- name: GetMatchClockForUser :one
SELECT * FROM match_clocks WHERE match_id = $1 AND user_id = $2;

-- name: UpdateMatchClock :one
UPDATE match_clocks
SET remaining_seconds = $2, is_current_turn = $3, last_started_at = $4
WHERE match_clock_id = $1
RETURNING *;
