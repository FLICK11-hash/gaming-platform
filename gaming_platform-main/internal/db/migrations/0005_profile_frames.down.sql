DELETE FROM cosmetic_items
WHERE item_type = 'profile_frame'
  AND image_key IN (
    'big-dill',
    'lettuce-celebrate',
    'nacho-average',
    'holy-guacamole',
    'donut-even',
    'legend-dairy'
  );

ALTER TABLE users
    DROP COLUMN selected_profile_frame;
