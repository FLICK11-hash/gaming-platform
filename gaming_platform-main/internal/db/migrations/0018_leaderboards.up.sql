-- Global leaderboards: win-streak tracking on users, plus a small table so
-- a slow-ticking background job can idempotently check "has this ISO week
-- already had its top-10 reward granted?" without a cron scheduler.
ALTER TABLE users
    ADD COLUMN current_win_streak INT NOT NULL DEFAULT 0,
    ADD COLUMN best_win_streak INT NOT NULL DEFAULT 0;

CREATE TABLE leaderboard_reward_runs (
    leaderboard_reward_run_id BIGSERIAL PRIMARY KEY,
    week_key                  VARCHAR(20) NOT NULL UNIQUE,
    run_at                    TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO cosmetic_items (name, description, item_type, image_key, coin_price)
VALUES ('Weekly Champion', 'Finished in the top 10 for wins in a weekly leaderboard.', 'badge', 'weekly-champion', 0);
