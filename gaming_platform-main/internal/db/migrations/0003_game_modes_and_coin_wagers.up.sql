-- Add support for both 1v1 and 4-player tournament matchmaking.
-- Replace real-money entry fees with virtual coin wagers.

ALTER TABLE matchmaking_queue
    ADD COLUMN game_mode VARCHAR(30) NOT NULL DEFAULT 'one_vs_one',
    ADD COLUMN coin_wager BIGINT NOT NULL DEFAULT 50;

ALTER TABLE matchmaking_queue
    ADD CONSTRAINT matchmaking_queue_valid_game_mode
        CHECK (game_mode IN ('one_vs_one', 'tournament'));

ALTER TABLE matchmaking_queue
    ADD CONSTRAINT matchmaking_queue_positive_coin_wager
        CHECK (coin_wager > 0);

-- The old entry_fee column is no longer used by the MVP.
ALTER TABLE matchmaking_queue
    DROP COLUMN entry_fee;

DROP INDEX IF EXISTS idx_matchmaking_queue_lookup;

CREATE INDEX idx_matchmaking_queue_lookup
    ON matchmaking_queue (
        game_type,
        game_mode,
        coin_wager,
        status,
        created_at
    );

-- Store virtual-coin values on tournaments instead of dollar amounts.
ALTER TABLE tournaments
    ADD COLUMN game_mode VARCHAR(30) NOT NULL DEFAULT 'tournament',
    ADD COLUMN coin_wager BIGINT NOT NULL DEFAULT 50,
    ADD COLUMN total_coin_pool BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN winner_coin_prize BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN bonus_coins BIGINT NOT NULL DEFAULT 0;

ALTER TABLE tournaments
    ADD CONSTRAINT tournaments_valid_game_mode
        CHECK (game_mode IN ('one_vs_one', 'tournament'));

ALTER TABLE tournaments
    ADD CONSTRAINT tournaments_nonnegative_coin_values
        CHECK (
            coin_wager >= 0
            AND total_coin_pool >= 0
            AND winner_coin_prize >= 0
            AND bonus_coins >= 0
        );

ALTER TABLE tournaments
    DROP COLUMN entry_fee,
    DROP COLUMN total_pool,
    DROP COLUMN platform_fee,
    DROP COLUMN grand_prize;
