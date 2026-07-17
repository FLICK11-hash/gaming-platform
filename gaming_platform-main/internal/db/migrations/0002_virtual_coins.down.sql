DROP INDEX IF EXISTS idx_cosmetic_items_type_active;
DROP INDEX IF EXISTS idx_user_cosmetics_user;
DROP INDEX IF EXISTS idx_coin_transactions_user_created;

DROP TABLE IF EXISTS user_cosmetics;
DROP TABLE IF EXISTS cosmetic_items;
DROP TABLE IF EXISTS coin_transactions;

ALTER TABLE users
    DROP COLUMN IF EXISTS selected_badge,
    DROP COLUMN IF EXISTS selected_avatar,
    DROP COLUMN IF EXISTS last_daily_reward_at,
    DROP COLUMN IF EXISTS coin_balance;
