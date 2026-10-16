-- Schema transcribed from ERD for UCHI Summer Student Project
-- https://dbdiagram.io/d/6a4806e14ac62e474c277575

CREATE TABLE users (
    user_id         BIGSERIAL PRIMARY KEY,
    username        VARCHAR NOT NULL UNIQUE,
    email           VARCHAR NOT NULL UNIQUE,
    password        VARCHAR NOT NULL,
    current_elo     INT,
    total_wins      INT,
    total_losses    INT,
    num_games       INT,
    account_balance NUMERIC(12, 2),
    account_status  VARCHAR, -- active, suspended, banned
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tournaments (
    tournament_id  BIGSERIAL PRIMARY KEY,
    game_type      VARCHAR NOT NULL, -- chess, connect_four, hangman
    status         VARCHAR NOT NULL, -- waiting, active, completed, cancelled
    entry_fee      NUMERIC(12, 2) NOT NULL,
    total_pool     NUMERIC(12, 2) NOT NULL,
    platform_fee   NUMERIC(12, 2) NOT NULL, -- around 10% of total_pool
    grand_prize    NUMERIC(12, 2) NOT NULL,
    winner_user_id BIGINT REFERENCES users (user_id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ
);

CREATE TABLE tournament_players (
    tournament_player_id BIGSERIAL PRIMARY KEY,
    tournament_id         BIGINT NOT NULL REFERENCES tournaments (tournament_id),
    user_id               BIGINT NOT NULL REFERENCES users (user_id),
    seed_number           INT, -- 1, 2, 3, 4
    final_result          VARCHAR, -- champion, finalist, semifinal_loss
    joined_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE matches (
    match_id             BIGSERIAL PRIMARY KEY,
    tournament_id        BIGINT NOT NULL REFERENCES tournaments (tournament_id),
    round                VARCHAR NOT NULL, -- semifinal, final
    match_number         INT, -- 1, 2, or 3
    player_1_id          BIGINT NOT NULL REFERENCES users (user_id),
    player_2_id          BIGINT NOT NULL REFERENCES users (user_id),
    current_turn_user_id BIGINT REFERENCES users (user_id),
    winner_user_id       BIGINT REFERENCES users (user_id),
    status               VARCHAR, -- waiting, active, completed, cancelled
    started_at           TIMESTAMPTZ,
    completed_at         TIMESTAMPTZ
);

CREATE TABLE match_clocks (
    match_clock_id    BIGSERIAL PRIMARY KEY,
    match_id          BIGINT NOT NULL REFERENCES matches (match_id),
    user_id           BIGINT NOT NULL REFERENCES users (user_id),
    starting_seconds  INT NOT NULL, -- 60
    remaining_seconds INT NOT NULL,
    is_current_turn   BOOLEAN,
    last_started_at   TIMESTAMPTZ -- when this player's clock began counting down
);

-- Reserved for the payments milestone: table created for ERD/schema fidelity,
-- but no queries or service code read/write it yet.
CREATE TABLE payments (
    payment_id              BIGSERIAL PRIMARY KEY,
    user_id                 BIGINT NOT NULL REFERENCES users (user_id),
    tournament_id           BIGINT REFERENCES tournaments (tournament_id),
    amount                  NUMERIC(12, 2) NOT NULL,
    payment_type            VARCHAR, -- deposit, entry_fee, payout, refund, platform_fee
    payment_status          VARCHAR, -- pending, completed, failed, reversed
    external_transaction_id VARCHAR,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Reserved for the payments milestone: table created for ERD/schema fidelity,
-- but no queries or service code read/write it yet.
CREATE TABLE user_history_log (
    user_history_log_id BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL REFERENCES users (user_id),
    tournament_id       BIGINT NOT NULL REFERENCES tournaments (tournament_id),
    match_id            BIGINT REFERENCES matches (match_id),
    result              VARCHAR, -- won, lost, champion, finalist
    wins_after_game     INT,
    losses_after_game   INT,
    elo_before          INT,
    elo_after           INT,
    balance_before      NUMERIC(12, 2),
    balance_after       NUMERIC(12, 2),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE matchmaking_queue (
    queue_id   BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users (user_id),
    game_type  VARCHAR NOT NULL,
    entry_fee  NUMERIC(12, 2) NOT NULL,
    min_elo    INT,
    max_elo    INT,
    status     VARCHAR, -- waiting, matched, cancelled
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE fraud_reports (
    fraud_report_id  BIGSERIAL PRIMARY KEY,
    reporter_user_id BIGINT NOT NULL REFERENCES users (user_id),
    reported_user_id BIGINT NOT NULL REFERENCES users (user_id),
    tournament_id    BIGINT REFERENCES tournaments (tournament_id),
    match_id         BIGINT REFERENCES matches (match_id),
    reason           TEXT,
    status           VARCHAR, -- open, reviewed, dismissed, confirmed
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
    status     VARCHAR, -- draft, published, hidden
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tournament_players_tournament ON tournament_players (tournament_id);
CREATE INDEX idx_tournament_players_user ON tournament_players (user_id);
CREATE INDEX idx_matches_tournament ON matches (tournament_id);
CREATE INDEX idx_match_clocks_match ON match_clocks (match_id);
CREATE INDEX idx_matchmaking_queue_lookup ON matchmaking_queue (game_type, entry_fee, status);
CREATE INDEX idx_fraud_reports_reported_user ON fraud_reports (reported_user_id);
CREATE INDEX idx_follows_followed ON follows (followed_user_id);
CREATE INDEX idx_posts_user ON posts (user_id);
