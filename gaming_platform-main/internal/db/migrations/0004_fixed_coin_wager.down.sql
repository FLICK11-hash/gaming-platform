ALTER TABLE matchmaking_queue
    ALTER COLUMN coin_wager SET DEFAULT 50;

ALTER TABLE tournaments
    ALTER COLUMN coin_wager SET DEFAULT 50;
