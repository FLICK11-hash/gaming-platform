-- All matches and tournaments now use a flat 200-coin wager instead of a
-- player-selectable amount.

ALTER TABLE matchmaking_queue
    ALTER COLUMN coin_wager SET DEFAULT 200;

ALTER TABLE tournaments
    ALTER COLUMN coin_wager SET DEFAULT 200;
