-- This platform only ever deals in virtual coins (see migration
-- 0002_virtual_coins), which have no cash value and can't be withdrawn or
-- exchanged for money. These leftovers from the original real-money ERD
-- were never wired to any query or handler - dropping them removes any
-- appearance that the app processes real payments, which it never has.
ALTER TABLE users DROP COLUMN account_balance;
DROP TABLE payments;
DROP TABLE user_history_log;
