-- Bot opponents: seeded automatically for a human who waits too long in
-- matchmaking (see internal/domain/matchmaking and internal/domain/bot).
-- Both accounts share the same placeholder password hash below; login is
-- rejected for is_bot accounts in handlers_users.go regardless, since bots
-- never authenticate over HTTP.
ALTER TABLE users ADD COLUMN is_bot BOOLEAN NOT NULL DEFAULT false;

INSERT INTO users (
    username, email, password,
    current_elo, total_wins, total_losses, num_games,
    account_status, coin_balance, selected_avatar, is_bot
)
VALUES
    ('RookieBot', 'rookiebot@bots.internal',
     '$2a$10$y8e33TfrimyaQ1nKCE33L.TbqU7r2i1PmSfoh1NSk1YMf/XmzQ6Z.',
     1200, 0, 0, 0, 'active', 100000000, 'rook', true),
    ('GrandmasterBot', 'grandmasterbot@bots.internal',
     '$2a$10$y8e33TfrimyaQ1nKCE33L.TbqU7r2i1PmSfoh1NSk1YMf/XmzQ6Z.',
     2100, 0, 0, 0, 'active', 100000000, 'queen', true)
ON CONFLICT (username) DO NOTHING;
