-- Cosmetic shop repricing: everything costs roughly 10x what it used to.
-- Kind of a Big Dill gets an extra bump (20x) to live up to its name.

UPDATE cosmetic_items SET coin_price = 5000  WHERE image_key = 'knight';
UPDATE cosmetic_items SET coin_price = 15000 WHERE image_key = 'king';

UPDATE cosmetic_items SET coin_price = 2500  WHERE image_key = 'first-victory';
UPDATE cosmetic_items SET coin_price = 7500  WHERE image_key = 'hot-streak';
UPDATE cosmetic_items SET coin_price = 25000 WHERE image_key = 'grandmaster';

UPDATE cosmetic_items SET coin_price = 4000  WHERE image_key = 'big-dill';
UPDATE cosmetic_items SET coin_price = 3500  WHERE image_key = 'lettuce-celebrate';
UPDATE cosmetic_items SET coin_price = 6000  WHERE image_key = 'nacho-average';
UPDATE cosmetic_items SET coin_price = 9000  WHERE image_key = 'holy-guacamole';
UPDATE cosmetic_items SET coin_price = 14000 WHERE image_key = 'donut-even';
UPDATE cosmetic_items SET coin_price = 20000 WHERE image_key = 'legend-dairy';
