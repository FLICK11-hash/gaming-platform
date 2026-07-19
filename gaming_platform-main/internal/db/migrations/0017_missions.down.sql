DELETE FROM missions;
DELETE FROM cosmetic_items WHERE image_key = 'weekly-warrior';

ALTER TABLE coin_transactions DROP CONSTRAINT coin_transactions_valid_type;
ALTER TABLE coin_transactions ADD CONSTRAINT coin_transactions_valid_type
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
    );

ALTER TABLE users
    DROP COLUMN current_login_streak,
    DROP COLUMN login_streak_started_on;

DROP TABLE user_mission_progress;
DROP TABLE missions;
