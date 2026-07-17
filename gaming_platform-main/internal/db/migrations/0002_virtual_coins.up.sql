-- Replace the real-money balance with a virtual-coin economy.
-- Coins have no cash value and cannot be withdrawn.

ALTER TABLE users
    ADD COLUMN coin_balance BIGINT NOT NULL DEFAULT 1000,
    ADD COLUMN last_daily_reward_at TIMESTAMPTZ,
    ADD COLUMN selected_avatar VARCHAR NOT NULL DEFAULT 'rook',
    ADD COLUMN selected_badge VARCHAR;

CREATE TABLE coin_transactions (
    coin_transaction_id BIGSERIAL PRIMARY KEY,
    user_id              BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    amount               BIGINT NOT NULL,
    transaction_type     VARCHAR(40) NOT NULL,
    description          VARCHAR(255),
    related_match_id     BIGINT REFERENCES matches(match_id) ON DELETE SET NULL,
    related_tournament_id BIGINT REFERENCES tournaments(tournament_id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT coin_transactions_nonzero_amount
        CHECK (amount <> 0),

    CONSTRAINT coin_transactions_valid_type
        CHECK (
            transaction_type IN (
                'starting_balance',
                'daily_login',
                'match_entry',
                'match_winnings',
                'tournament_entry',
                'tournament_winnings',
                'cosmetic_purchase',
                'refund',
                'admin_adjustment'
            )
        )
);

CREATE TABLE cosmetic_items (
    cosmetic_item_id BIGSERIAL PRIMARY KEY,
    name             VARCHAR(100) NOT NULL,
    description      TEXT,
    item_type        VARCHAR(30) NOT NULL,
    image_key        VARCHAR(100) NOT NULL,
    coin_price       BIGINT NOT NULL,
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT cosmetic_items_nonnegative_price
        CHECK (coin_price >= 0),

    CONSTRAINT cosmetic_items_valid_type
        CHECK (item_type IN ('avatar', 'badge', 'profile_frame', 'board_theme'))
);

CREATE TABLE user_cosmetics (
    user_id          BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    cosmetic_item_id BIGINT NOT NULL REFERENCES cosmetic_items(cosmetic_item_id) ON DELETE CASCADE,
    purchased_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, cosmetic_item_id)
);

CREATE INDEX idx_coin_transactions_user_created
    ON coin_transactions(user_id, created_at DESC);

CREATE INDEX idx_user_cosmetics_user
    ON user_cosmetics(user_id);

CREATE INDEX idx_cosmetic_items_type_active
    ON cosmetic_items(item_type, is_active);

-- Starter cosmetic catalog.
INSERT INTO cosmetic_items
    (name, description, item_type, image_key, coin_price)
VALUES
    ('Rookie Rook', 'A classic rook avatar.', 'avatar', 'rook', 0),
    ('Knight Rider', 'A silver knight avatar.', 'avatar', 'knight', 500),
    ('Royal King', 'A gold king avatar.', 'avatar', 'king', 1500),
    ('First Victory', 'Awarded to confident competitors.', 'badge', 'first-victory', 250),
    ('Hot Streak', 'A badge for players on a winning run.', 'badge', 'hot-streak', 750),
    ('Grandmaster', 'A premium competitive badge.', 'badge', 'grandmaster', 2500);

-- Record the starting balance for existing users.
INSERT INTO coin_transactions
    (user_id, amount, transaction_type, description)
SELECT
    user_id,
    1000,
    'starting_balance',
    'Initial virtual coin balance'
FROM users;
