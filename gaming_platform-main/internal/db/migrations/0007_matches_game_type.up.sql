-- Matches now record which game they're playing, so a single match service
-- can host both chess and Connect Four. Denormalized from the tournament's
-- game_type onto matches directly, since HandleMove needs it on every move
-- and shouldn't have to join out to tournaments for that.

ALTER TABLE matches
    ADD COLUMN game_type VARCHAR NOT NULL DEFAULT 'chess';

ALTER TABLE matches
    ADD CONSTRAINT matches_valid_game_type
        CHECK (game_type IN ('chess', 'connect_four'));

UPDATE matches m
SET game_type = t.game_type
FROM tournaments t
WHERE m.tournament_id = t.tournament_id
  AND t.game_type IN ('chess', 'connect_four');
