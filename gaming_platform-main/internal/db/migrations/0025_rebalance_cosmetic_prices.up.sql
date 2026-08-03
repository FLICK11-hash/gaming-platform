-- The 10x price bump in migration 0008 overshot once combined with the
-- shop's later growth to dozens of items priced on a steep, mostly
-- arithmetic +5000-per-tier ladder: the single rarest profile frame ended
-- up at 115,000 coins - worth 575 match wins at the standard 200-coin
-- wager, not a long-term goal but an effectively unreachable number for
-- almost every player. This rescales every purchasable cosmetic onto a
-- much gentler geometric curve - cheapest paid item at 300, the rarest
-- frame at 10,000 (about 50 wins) - while preserving the exact same
-- relative ordering, so nothing that used to feel rarer than something
-- else stops feeling that way.

-- Avatars
UPDATE cosmetic_items SET coin_price = 600  WHERE image_key = 'knight';
UPDATE cosmetic_items SET coin_price = 2000 WHERE image_key = 'king';

-- Badges
UPDATE cosmetic_items SET coin_price = 300  WHERE image_key = 'first-victory';
UPDATE cosmetic_items SET coin_price = 1200 WHERE image_key = 'hot-streak';
UPDATE cosmetic_items SET coin_price = 3500 WHERE image_key = 'grandmaster';

-- Profile frames, cheapest to priciest
UPDATE cosmetic_items SET coin_price = 400   WHERE image_key = 'lettuce-celebrate';
UPDATE cosmetic_items SET coin_price = 450   WHERE image_key = 'big-dill';
UPDATE cosmetic_items SET coin_price = 550   WHERE image_key = 'nacho-average';
UPDATE cosmetic_items SET coin_price = 650   WHERE image_key = 'holy-guacamole';
UPDATE cosmetic_items SET coin_price = 750   WHERE image_key = 'donut-even';
UPDATE cosmetic_items SET coin_price = 850   WHERE image_key = 'legend-dairy';
UPDATE cosmetic_items SET coin_price = 1000  WHERE image_key = 'dracula';
UPDATE cosmetic_items SET coin_price = 1150  WHERE image_key = 'sherlock-holmes';
UPDATE cosmetic_items SET coin_price = 1350  WHERE image_key = 'frankenstein';
UPDATE cosmetic_items SET coin_price = 1550  WHERE image_key = 'medusa';
UPDATE cosmetic_items SET coin_price = 1800  WHERE image_key = 'robin-hood';
UPDATE cosmetic_items SET coin_price = 2050  WHERE image_key = 'king-arthur';
UPDATE cosmetic_items SET coin_price = 2350  WHERE image_key = 'sun-wukong';
UPDATE cosmetic_items SET coin_price = 2700  WHERE image_key = 'zeus';
UPDATE cosmetic_items SET coin_price = 3100  WHERE image_key = 'paul-bunyan';
UPDATE cosmetic_items SET coin_price = 3550  WHERE image_key = 'uncle-sam';
UPDATE cosmetic_items SET coin_price = 4050  WHERE image_key = 'hercules';
UPDATE cosmetic_items SET coin_price = 4600  WHERE image_key = 'achilles';
UPDATE cosmetic_items SET coin_price = 5200  WHERE image_key = 'icarus';
UPDATE cosmetic_items SET coin_price = 5850  WHERE image_key = 'poseidon';
UPDATE cosmetic_items SET coin_price = 6550  WHERE image_key = 'hades';
UPDATE cosmetic_items SET coin_price = 7300  WHERE image_key = 'pandora';
UPDATE cosmetic_items SET coin_price = 8100  WHERE image_key = 'midas';
UPDATE cosmetic_items SET coin_price = 9000  WHERE image_key = 'atlas';
UPDATE cosmetic_items SET coin_price = 10000 WHERE image_key = 'prometheus';
