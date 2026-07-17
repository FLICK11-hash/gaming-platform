-- name: ListDistinctWaitingBuckets :many
SELECT DISTINCT
    game_type,
    game_mode,
    coin_wager
FROM matchmaking_queue
WHERE status = 'waiting';

-- name: CreateQueueEntry :one
INSERT INTO matchmaking_queue (
    user_id,
    game_type,
    game_mode,
    coin_wager,
    min_elo,
    max_elo,
    status
)
VALUES ($1, $2, $3, $4, $5, $6, 'waiting')
RETURNING *;

-- name: GetWaitingQueueEntryForUser :one
SELECT *
FROM matchmaking_queue
WHERE user_id = $1
  AND status = 'waiting';

-- name: ListWaitingQueueEntries :many
SELECT *
FROM matchmaking_queue
WHERE game_type = $1
  AND game_mode = $2
  AND coin_wager = $3
  AND status = 'waiting'
ORDER BY created_at
FOR UPDATE SKIP LOCKED;

-- name: CancelQueueEntry :exec
UPDATE matchmaking_queue
SET status = 'cancelled'
WHERE queue_id = $1
  AND user_id = $2
  AND status = 'waiting';

-- name: MarkQueueEntryMatched :exec
UPDATE matchmaking_queue
SET status = 'matched'
WHERE queue_id = $1;
