-- Daily & weekly missions: small recurring objectives that grant coins or
-- an exclusive badge on completion, to give players a reason to come back.
CREATE TABLE missions (
    mission_id             BIGSERIAL PRIMARY KEY,
    key                    VARCHAR(60) NOT NULL UNIQUE,
    reset_type             VARCHAR(10) NOT NULL, -- 'daily' | 'streak'
    title                  VARCHAR(100) NOT NULL,
    description            VARCHAR(255) NOT NULL,
    goal_type              VARCHAR(30) NOT NULL, -- wins | games_played | upset_win | login_streak
    goal_count             INT NOT NULL,
    reward_coins           BIGINT NOT NULL DEFAULT 0,
    reward_badge_image_key VARCHAR(100),
    is_active              BOOLEAN NOT NULL DEFAULT TRUE,

    CONSTRAINT missions_valid_reset_type CHECK (reset_type IN ('daily', 'streak')),
    CONSTRAINT missions_valid_goal_type CHECK (goal_type IN ('wins', 'games_played', 'upset_win', 'login_streak')),
    CONSTRAINT missions_positive_goal CHECK (goal_count > 0),
    CONSTRAINT missions_nonnegative_reward CHECK (reward_coins >= 0)
);

-- Progress toward a mission is bucketed by period_key, whose meaning
-- depends on the mission's reset_type: for 'daily' missions it's the UTC
-- date; for 'streak' missions it's "<date the current streak
-- started>#<which goal_count-sized chunk of the streak this is>", so a
-- streak mission naturally re-arms every goal_count consecutive days and a
-- broken-then-restarted streak never collides with an earlier attempt's row.
CREATE TABLE user_mission_progress (
    user_mission_progress_id BIGSERIAL PRIMARY KEY,
    user_id                  BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    mission_id               BIGINT NOT NULL REFERENCES missions(mission_id) ON DELETE CASCADE,
    period_key               VARCHAR(40) NOT NULL,
    progress_count           INT NOT NULL DEFAULT 0,
    completed_at             TIMESTAMPTZ,
    claimed_at               TIMESTAMPTZ,

    UNIQUE (user_id, mission_id, period_key)
);

CREATE INDEX idx_user_mission_progress_user ON user_mission_progress(user_id);

ALTER TABLE users
    ADD COLUMN current_login_streak INT NOT NULL DEFAULT 0,
    ADD COLUMN login_streak_started_on DATE;

ALTER TABLE coin_transactions DROP CONSTRAINT coin_transactions_valid_type;
ALTER TABLE coin_transactions ADD CONSTRAINT coin_transactions_valid_type
    CHECK (
        transaction_type IN (
            'starting_balance',
            'daily_login',
            'match_entry',
            'match_winnings',
            'tournament_entry',
            'tournament_winnings',
            'cosmetic_purchase',
            'refund',
            'admin_adjustment',
            'mission_reward'
        )
    );

INSERT INTO cosmetic_items (name, description, item_type, image_key, coin_price)
VALUES ('Weekly Warrior', 'Claimed a daily reward 7 days in a row without missing one.', 'badge', 'weekly-warrior', 0);

INSERT INTO missions (key, reset_type, title, description, goal_type, goal_count, reward_coins, reward_badge_image_key)
VALUES
    ('daily_win_2', 'daily', 'Win 2 Games', 'Win 2 matches today.', 'wins', 2, 100, NULL),
    ('daily_play_5', 'daily', 'Play 5 Matches', 'Complete 5 matches today, win or lose.', 'games_played', 5, 75, NULL),
    ('daily_giant_slayer', 'daily', 'Giant Slayer', 'Defeat an opponent rated 100+ Elo above you.', 'upset_win', 1, 250, NULL),
    ('weekly_login_streak', 'streak', 'Login 7 Days in a Row', 'Claim your daily reward 7 days in a row.', 'login_streak', 7, 0, 'weekly-warrior');
