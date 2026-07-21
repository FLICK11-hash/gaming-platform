CREATE TABLE fraud_reports (
    fraud_report_id  BIGSERIAL PRIMARY KEY,
    reporter_user_id BIGINT NOT NULL REFERENCES users (user_id),
    reported_user_id BIGINT NOT NULL REFERENCES users (user_id),
    tournament_id    BIGINT REFERENCES tournaments (tournament_id),
    match_id         BIGINT REFERENCES matches (match_id),
    reason           TEXT,
    status           VARCHAR,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE follows (
    following_user_id BIGINT NOT NULL REFERENCES users (user_id),
    followed_user_id  BIGINT NOT NULL REFERENCES users (user_id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (following_user_id, followed_user_id)
);

CREATE TABLE posts (
    post_id    BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users (user_id),
    title      VARCHAR,
    body       TEXT,
    status     VARCHAR,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_fraud_reports_reported_user ON fraud_reports (reported_user_id);
CREATE INDEX idx_follows_followed ON follows (followed_user_id);
CREATE INDEX idx_posts_user ON posts (user_id);
