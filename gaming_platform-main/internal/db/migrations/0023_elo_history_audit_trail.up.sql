-- Turns elo_history from a bare sequence of after-values into a
-- self-contained audit trail: each row now records which match caused the
-- change and what the rating was immediately before it, so a bad Elo update
-- can be diagnosed (and, if needed, manually reconstructed) without having
-- to infer elo_before by sorting on recorded_at and hoping nothing raced.
-- Nullable/no backfill for the same reason as the original table (see
-- migration 0019): existing rows predate this column.
ALTER TABLE elo_history
    ADD COLUMN match_id   BIGINT REFERENCES matches (match_id) ON DELETE SET NULL,
    ADD COLUMN elo_before INT;

CREATE INDEX idx_elo_history_match ON elo_history (match_id);
