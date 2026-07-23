ALTER TABLE users ADD COLUMN account_balance NUMERIC(12, 2);

CREATE TABLE payments (
    payment_id              BIGSERIAL PRIMARY KEY,
    user_id                 BIGINT NOT NULL REFERENCES users (user_id),
    tournament_id           BIGINT REFERENCES tournaments (tournament_id),
    amount                  NUMERIC(12, 2) NOT NULL,
    payment_type            VARCHAR,
    payment_status          VARCHAR,
    external_transaction_id VARCHAR,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_history_log (
    user_history_log_id BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL REFERENCES users (user_id),
    tournament_id       BIGINT NOT NULL REFERENCES tournaments (tournament_id),
    match_id            BIGINT REFERENCES matches (match_id),
    result              VARCHAR,
    wins_after_game     INT,
    losses_after_game   INT,
    elo_before          INT,
    elo_after           INT,
    balance_before      NUMERIC(12, 2),
    balance_after       NUMERIC(12, 2),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
