-- Powers the player profile's Elo-over-time graph: one row per Elo update
-- for a non-bot player, appended going forward (no backfill for matches
-- completed before this migration).
CREATE TABLE elo_history (
    elo_history_id BIGSERIAL PRIMARY KEY,
    user_id        BIGINT NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    elo            INT NOT NULL,
    recorded_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_elo_history_user_recorded ON elo_history (user_id, recorded_at);

-- The ERD has no move-history table (see internal/domain/chess/engine.go),
-- so the profile's "favorite opening" stat needs somewhere to persist just
-- the first move of each chess match, recorded as it's played.
ALTER TABLE matches
    ADD COLUMN first_move VARCHAR;
