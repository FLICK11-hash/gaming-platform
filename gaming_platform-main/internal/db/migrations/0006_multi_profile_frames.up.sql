-- Profile frames become a multi-equip slot (up to 5 at once) instead of a
-- single selection, so players can stack several pun pins at a time.

ALTER TABLE users
    ADD COLUMN selected_profile_frames VARCHAR[] NOT NULL DEFAULT '{}';

UPDATE users
SET selected_profile_frames = ARRAY[selected_profile_frame]
WHERE selected_profile_frame IS NOT NULL;

ALTER TABLE users
    DROP COLUMN selected_profile_frame;

ALTER TABLE users
    ADD CONSTRAINT users_max_five_profile_frames
        CHECK (array_length(selected_profile_frames, 1) IS NULL OR array_length(selected_profile_frames, 1) <= 5);
