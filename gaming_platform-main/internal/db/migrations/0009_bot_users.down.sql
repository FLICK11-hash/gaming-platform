DELETE FROM users WHERE is_bot = true;
ALTER TABLE users DROP COLUMN is_bot;
