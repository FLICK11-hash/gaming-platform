-- The second character-mashup batch (Scrooge, Jekyll/Hyde, Little Red
-- Riding Hood, the Big Bad Wolf, Quasimodo, Ahab, Long John Silver, Puss in
-- Boots) didn't land well - removing it entirely rather than leaving
-- unwanted items purchasable in the shop.
DELETE FROM cosmetic_items WHERE image_key IN (
    'scrooge-refunds', 'jekyll-hyde-video-call', 'red-riding-hood-influencer',
    'big-bad-wolf-realtor', 'quasimodo-notifications', 'ahab-viral',
    'long-john-silver-substitute', 'puss-in-boots-fashion'
);
