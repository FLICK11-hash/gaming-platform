-- name: ListActiveMissions :many
SELECT * FROM missions WHERE is_active = true ORDER BY mission_id;

-- name: GetMission :one
SELECT * FROM missions WHERE mission_id = $1;

-- name: GetOrCreateUserMissionProgress :one
INSERT INTO user_mission_progress (user_id, mission_id, period_key, progress_count)
VALUES ($1, $2, $3, 0)
ON CONFLICT (user_id, mission_id, period_key) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING *;

-- name: GetOrCreateUserMissionProgressBatch :many
-- Same upsert as GetOrCreateUserMissionProgress, but for every active
-- mission in one round trip instead of one per mission: mission_ids and
-- period_keys are matched pairwise (unnest walks both arrays in lockstep),
-- so the missions list endpoint - polled repeatedly by every logged-in
-- client - doesn't do a query per mission on every load.
INSERT INTO user_mission_progress (user_id, mission_id, period_key, progress_count)
SELECT sqlc.arg('user_id')::bigint, unnest(sqlc.arg('mission_ids')::bigint[]), unnest(sqlc.arg('period_keys')::text[]), 0
ON CONFLICT (user_id, mission_id, period_key) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING *;

-- name: IncrementMissionProgress :one
INSERT INTO user_mission_progress (user_id, mission_id, period_key, progress_count)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, mission_id, period_key) DO UPDATE
SET progress_count = user_mission_progress.progress_count + $4
RETURNING *;

-- name: MarkMissionProgressCompleted :exec
UPDATE user_mission_progress
SET completed_at = now()
WHERE user_id = $1 AND mission_id = $2 AND period_key = $3 AND completed_at IS NULL;

-- name: ClaimMissionProgress :one
UPDATE user_mission_progress
SET claimed_at = now()
WHERE user_id = $1 AND mission_id = $2 AND period_key = $3
  AND completed_at IS NOT NULL AND claimed_at IS NULL
RETURNING *;

-- name: GetCosmeticItemByImageKey :one
SELECT * FROM cosmetic_items WHERE image_key = $1;

-- name: SetUserLoginStreak :one
UPDATE users
SET current_login_streak = $2, login_streak_started_on = $3
WHERE user_id = $1
RETURNING *;
