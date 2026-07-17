ALTER TABLE matches
    DROP CONSTRAINT matches_valid_game_type;

ALTER TABLE matches
    DROP COLUMN game_type;
