-- Comedic, non-chess "profile frame" flair. Unlike avatars/badges (small
-- icons in the shop and equip slots), a profile frame is meant to be shown
-- big and loud right next to the player's name.

ALTER TABLE users
    ADD COLUMN selected_profile_frame VARCHAR;

INSERT INTO cosmetic_items
    (name, description, item_type, image_key, coin_price)
VALUES
    ('Kind of a Big Dill', 'For players who take themselves a little too seriously.', 'profile_frame', 'big-dill', 200),
    ('Lettuce Celebrate', 'Romaine calm, you earned this one.', 'profile_frame', 'lettuce-celebrate', 350),
    ('Nacho Average Player', 'Definitely not your average player. Probably.', 'profile_frame', 'nacho-average', 600),
    ('Holy Guacamole', 'For those unbe-leaf-ably good games.', 'profile_frame', 'holy-guacamole', 900),
    ('Donut Even Try It', 'A hole lot of confidence, glazed on top.', 'profile_frame', 'donut-even', 1400),
    ('Legend-dairy', 'The most legend-dairy title in the shop.', 'profile_frame', 'legend-dairy', 2000);
