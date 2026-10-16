-- Rebalances the bot accounts to match real Stockfish UCI_Elo targets
-- (Stockfish's own strength limiter can't go below 1320; see
-- internal/domain/bot.RookieBotUCIElo) and adds a third tier - ExpertBot -
-- for the human Elo band between RookieBot and GrandmasterBot. See
-- internal/domain/bot/bot.go for the matching Easy/Medium/Hard difficulty
-- thresholds.
UPDATE users SET current_elo = 1320 WHERE username = 'RookieBot' AND is_bot = true;
UPDATE users SET current_elo = 2500 WHERE username = 'GrandmasterBot' AND is_bot = true;

INSERT INTO users (
    username, email, password,
    current_elo, total_wins, total_losses, num_games,
    account_status, coin_balance, selected_avatar, is_bot
)
VALUES
    ('ExpertBot', 'expertbot@bots.internal',
     '$2a$10$y8e33TfrimyaQ1nKCE33L.TbqU7r2i1PmSfoh1NSk1YMf/XmzQ6Z.',
     2000, 0, 0, 0, 'active', 100000000, 'knight', true)
ON CONFLICT (username) DO NOTHING;
