-- More profile frames, themed on public-domain literary/folklore figures
-- (not trademarked characters) so they're free to use. Priced above the
-- existing top frame (legend-dairy, 20000) so the shop keeps escalating.

INSERT INTO cosmetic_items
    (name, description, item_type, image_key, coin_price)
VALUES
    ('Fangs for the Win', 'Sinks its teeth into victory. Undead serious about winning.', 'profile_frame', 'dracula', 25000),
    ('Elementary, My Dear Winner', 'The game is afoot, and so is the W.', 'profile_frame', 'sherlock-holmes', 30000),
    ('It''s Alive! (And Winning)', 'Stitched together from spare parts and pure victory.', 'profile_frame', 'frankenstein', 35000),
    ('Petrifying Performance', 'One look at this board and your opponent turns to stone.', 'profile_frame', 'medusa', 40000),
    ('Robbing the Rich (of Their Coins)', 'Steals wins from the wealthy and keeps them all.', 'profile_frame', 'robin-hood', 45000),
    ('Sword in the Stone-Cold Killer', 'Only the truly worthy can pull off a win like this.', 'profile_frame', 'king-arthur', 50000),
    ('Monkey Business Champion', '72 transformations, 0 losses.', 'profile_frame', 'sun-wukong', 55000),
    ('Thunderously Good', 'Strikes down opponents from Mount Olympus.', 'profile_frame', 'zeus', 60000),
    ('Larger Than Life W', 'So big even the win needed its own legend.', 'profile_frame', 'paul-bunyan', 65000),
    ('I Want YOU... To Lose', 'Points a finger at your opponent. It means business.', 'profile_frame', 'uncle-sam', 70000);
