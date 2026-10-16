-- name: TopUsersByElo :many
SELECT user_id, username, selected_avatar, selected_badge, current_elo
FROM users
WHERE is_bot = false AND account_status = 'active'
ORDER BY current_elo DESC NULLS LAST
LIMIT 10;

-- name: TopUsersByCoins :many
SELECT user_id, username, selected_avatar, selected_badge, coin_balance
FROM users
WHERE is_bot = false AND account_status = 'active'
ORDER BY coin_balance DESC
LIMIT 10;

-- name: TopUsersByWinStreak :many
SELECT user_id, username, selected_avatar, selected_badge, best_win_streak
FROM users
WHERE is_bot = false AND account_status = 'active' AND best_win_streak > 0
ORDER BY best_win_streak DESC
LIMIT 10;

-- name: TopUsersByTournamentChampionships :many
SELECT u.user_id, u.username, u.selected_avatar, u.selected_badge, COUNT(*) AS championships
FROM tournaments t
JOIN users u ON u.user_id = t.winner_user_id
WHERE t.status = 'completed' AND u.is_bot = false
GROUP BY u.user_id, u.username, u.selected_avatar, u.selected_badge
ORDER BY championships DESC
LIMIT 10;

-- name: TopUsersByWeeklyWins :many
SELECT u.user_id, u.username, u.selected_avatar, u.selected_badge, COUNT(*) AS weekly_wins
FROM matches m
JOIN users u ON u.user_id = m.winner_user_id
WHERE m.completed_at >= sqlc.arg('since')::timestamptz
  AND m.completed_at < sqlc.arg('until')::timestamptz
  AND u.is_bot = false
GROUP BY u.user_id, u.username, u.selected_avatar, u.selected_badge
ORDER BY weekly_wins DESC
LIMIT 10;

-- name: RecordWinStreak :one
UPDATE users
SET current_win_streak = current_win_streak + 1,
    best_win_streak = GREATEST(best_win_streak, current_win_streak + 1)
WHERE user_id = $1
RETURNING *;

-- name: ResetWinStreak :exec
UPDATE users SET current_win_streak = 0 WHERE user_id = $1 AND current_win_streak != 0;

-- name: HasLeaderboardRewardRun :one
SELECT EXISTS(SELECT 1 FROM leaderboard_reward_runs WHERE week_key = $1);

-- name: RecordLeaderboardRewardRun :exec
INSERT INTO leaderboard_reward_runs (week_key) VALUES ($1)
ON CONFLICT DO NOTHING;
