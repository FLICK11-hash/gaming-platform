-- Lifetime achievement unlocks: unlike missions (daily/weekly, resetting)
-- these are permanent, one-time milestones that grant an exclusive
-- reward-only cosmetic (coin_price = 0) the first time they're crossed.
INSERT INTO cosmetic_items (name, description, item_type, image_key, coin_price)
VALUES
    ('Bronze Knight', 'Win 10 games to unlock.', 'avatar', 'bronze-knight', 0),
    ('Silver Crown', 'Reach 1500 Elo to unlock.', 'avatar', 'silver-crown', 0),
    ('Champion Banner', 'Win 25 tournaments to unlock.', 'profile_frame', 'champion-banner', 0),
    ('Legendary Avatar', 'Maintain a 100-day login streak to unlock.', 'avatar', 'legendary-avatar', 0);
