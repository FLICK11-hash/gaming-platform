-- Everything from bat-guy onward (the original bat joke plus the first
-- character-mashup batch) didn't land well visually - removing all of it
-- and replacing with a fresh set of Greek-mythology frames, rendered as
-- hand-built SVG icons on the frontend rather than emoji.
DELETE FROM cosmetic_items WHERE image_key IN (
    'bat-guy',
    'sherlock-holmes-gamer', 'dracula-drivethru', 'robin-hood-gig',
    'winnie-pooh-hedgefund', 'frankenstein-gym', 'king-arthur-wholesale',
    'alice-wonderland-ai', 'medusa-salon'
);

INSERT INTO cosmetic_items
    (name, description, item_type, image_key, coin_price)
VALUES
    ('Twelve Labors, Zero Losses', 'Every impossible task, cleared without breaking a sweat. The lion pelt was just the first trophy.', 'profile_frame', 'hercules', 75000),
    ('One Weak Spot, Zero Excuses', 'Nearly invincible, deeply allergic to being reminded about the heel thing.', 'profile_frame', 'achilles', 80000),
    ('Flew Too Close to the W', 'Built the wings himself, ignored every warning label, won anyway.', 'profile_frame', 'icarus', 85000),
    ('Making Waves, Taking Names', 'Rules the whole board like it''s his ocean. Everyone else is just visiting.', 'profile_frame', 'poseidon', 90000),
    ('Lord of the Underdogs', 'Nobody wanted this matchup. He took it, ruled it, and never gave it back.', 'profile_frame', 'hades', 95000),
    ('Opened the Box, Closed the Game', 'Everyone said don''t open it. He opened it. Now look what got out.', 'profile_frame', 'pandora', 100000),
    ('Midas Touch, Golden Record', 'Everything this account touches turns into a win. Allegedly a curse. Doesn''t feel like one.', 'profile_frame', 'midas', 105000),
    ('Carrying the Whole Team', 'Holding the entire match up on his shoulders and somehow still smiling about it.', 'profile_frame', 'atlas', 110000),
    ('Stole Fire, Stole the Win', 'Took something the gods were hoarding and handed it straight to the people. Worth the punishment.', 'profile_frame', 'prometheus', 115000);
