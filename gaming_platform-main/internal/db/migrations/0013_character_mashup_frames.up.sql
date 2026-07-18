-- "Public domain character x modern situation" mashup frames. Each keeps
-- the exact character + scenario the mashup is built on, but swaps any
-- specific trademarked brand name (a game title, a restaurant chain, a
-- delivery app, a wholesale retailer, an AI product) for an equivalent
-- generic activity, since the brand names themselves aren't public domain
-- the way the characters are.
INSERT INTO cosmetic_items
    (name, description, item_type, image_key, coin_price)
VALUES
    ('Elementary, My Dear K/D Ratio', 'The world''s greatest detective, now deducing enemy spawn points instead of crime scenes. Solves every case except why he keeps getting outplayed by a 12-year-old with a headset.', 'profile_frame', 'sherlock-holmes-gamer', 75000),
    ('Sucks at Small Talk, Great at Fries', 'Immortal for 500 years, employee of the month for three. Turns out biting necks and taking drive-thru orders both come down to good timing.', 'profile_frame', 'dracula-drivethru', 80000),
    ('Steals From the Rich, Delivers to Everyone', 'Still redistributing wealth one order at a time, except now the wealth is your burrito and he''s eating a fry from the bag on the way. Five-star rating, questionable ethics.', 'profile_frame', 'robin-hood-gig', 85000),
    ('Bearish Market, Bullish on Honey', 'Turned "oh bother" into a diversified portfolio. Ninety percent of his holdings are hunny futures and he still somehow outperforms the index.', 'profile_frame', 'winnie-pooh-hedgefund', 90000),
    ('Built Different (Literally)', 'Stitched together from spare parts and pure gains. Posts a lot about protein, mostly because he''s not entirely sure what he''s made of anymore.', 'profile_frame', 'frankenstein-gym', 95000),
    ('Sword in the Bulk Aisle', 'Only the truly worthy can lift the 48-pack of paper towels one-handed. Came for a rotisserie chicken, left having pulled a legendary weapon from a pallet.', 'profile_frame', 'king-arthur-wholesale', 100000),
    ('Curiouser and Curiouser Prompts', 'Fell down a rabbit hole of confidently wrong answers and somehow found it less confusing than the actual rabbit hole. Still trying to get the chatbot to admit it doesn''t know either.', 'profile_frame', 'alice-wonderland-ai', 105000),
    ('Snake Charmer Salon', 'Every client leaves stunning. Some leave literally stone-faced. Booked solid for weeks, mostly by people who never leave a bad review.', 'profile_frame', 'medusa-salon', 110000);
