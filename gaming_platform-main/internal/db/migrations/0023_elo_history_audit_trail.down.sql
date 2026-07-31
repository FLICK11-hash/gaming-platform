DROP INDEX IF EXISTS idx_elo_history_match;

ALTER TABLE elo_history
    DROP COLUMN match_id,
    DROP COLUMN elo_before;
