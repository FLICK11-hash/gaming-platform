DELETE FROM cosmetic_items WHERE image_key = 'weekly-champion';

DROP TABLE leaderboard_reward_runs;

ALTER TABLE users
    DROP COLUMN current_win_streak,
    DROP COLUMN best_win_streak;
