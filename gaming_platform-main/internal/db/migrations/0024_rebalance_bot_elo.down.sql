DELETE FROM users WHERE username = 'ExpertBot' AND is_bot = true;
UPDATE users SET current_elo = 2100 WHERE username = 'GrandmasterBot' AND is_bot = true;
UPDATE users SET current_elo = 1200 WHERE username = 'RookieBot' AND is_bot = true;
