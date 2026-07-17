DROP INDEX IF EXISTS idx_matchmaking_queue_lookup;

ALTER TABLE matchmaking_queue
    ADD COLUMN entry_fee NUMERIC(12, 2) NOT NULL DEFAULT 0;

ALTER TABLE matchmaking_queue
    DROP CONSTRAINT IF EXISTS matchmaking_queue_positive_coin_wager,
    DROP CONSTRAINT IF EXISTS matchmaking_queue_valid_game_mode,
    DROP COLUMN IF EXISTS coin_wager,
    DROP COLUMN IF EXISTS game_mode;

CREATE INDEX idx_matchmaking_queue_lookup
    ON matchmaking_queue (game_type, entry_fee, status);

ALTER TABLE tournaments
    DROP CONSTRAINT IF EXISTS tournaments_nonnegative_coin_values,
    DROP CONSTRAINT IF EXISTS tournaments_valid_game_mode,
    DROP COLUMN IF EXISTS bonus_coins,
    DROP COLUMN IF EXISTS winner_coin_prize,
    DROP COLUMN IF EXISTS total_coin_pool,
    DROP COLUMN IF EXISTS coin_wager,
    DROP COLUMN IF EXISTS game_mode;
