-- name: InsertEloHistory :exec
INSERT INTO elo_history (user_id, elo) VALUES ($1, $2);

-- name: ListEloHistoryForUser :many
SELECT elo, recorded_at FROM elo_history WHERE user_id = $1 ORDER BY recorded_at ASC;

-- name: SetMatchFirstMove :exec
UPDATE matches SET first_move = $2 WHERE match_id = $1;

-- name: ListChessFirstMovesForUser :many
SELECT first_move FROM matches
WHERE (player_1_id = $1 OR player_2_id = $1)
  AND game_type = 'chess'
  AND first_move IS NOT NULL;

-- name: ListRecentCompletedMatchesForUser :many
SELECT
    m.match_id,
    m.game_type,
    m.completed_at,
    m.winner_user_id,
    (CASE WHEN m.player_1_id = sqlc.arg('user_id')::bigint THEN p2.username ELSE p1.username END)::varchar AS opponent_username
FROM matches m
JOIN users p1 ON p1.user_id = m.player_1_id
JOIN users p2 ON p2.user_id = m.player_2_id
WHERE (m.player_1_id = sqlc.arg('user_id')::bigint OR m.player_2_id = sqlc.arg('user_id')::bigint)
  AND m.status = 'completed'
ORDER BY m.completed_at DESC
LIMIT 10;

-- name: CountTournamentWinsForUser :one
SELECT COUNT(*) FROM tournaments WHERE winner_user_id = $1 AND status = 'completed';

-- name: AvgGameLengthSecondsForUser :one
SELECT
    COUNT(*) AS games,
    COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at))), 0)::float8 AS avg_seconds
FROM matches
WHERE (player_1_id = $1 OR player_2_id = $1)
  AND status = 'completed'
  AND started_at IS NOT NULL
  AND completed_at IS NOT NULL;
