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

-- name: ApplyMoveClocks :exec
-- Applies both clocks' post-move state in a single round trip instead of two
-- separate reads plus two separate writes: the mover's new remaining time
-- and the turn handoff are set together, since every move touches exactly
-- these two rows. Bullet chess gives each side 60 seconds total, so cutting
-- per-move database round trips matters for keeping the clock fair.
UPDATE match_clocks
SET
    remaining_seconds = CASE WHEN user_id = sqlc.arg('mover_user_id') THEN sqlc.arg('mover_remaining_seconds') ELSE remaining_seconds END,
    is_current_turn = (user_id = sqlc.arg('opponent_user_id')),
    last_started_at = sqlc.arg('last_started_at')
WHERE match_id = sqlc.arg('match_id')
  AND user_id IN (sqlc.arg('mover_user_id'), sqlc.arg('opponent_user_id'));
