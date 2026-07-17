ALTER TABLE users
    DROP CONSTRAINT users_max_five_profile_frames;

ALTER TABLE users
    ADD COLUMN selected_profile_frame VARCHAR;

UPDATE users
SET selected_profile_frame = selected_profile_frames[1]
WHERE array_length(selected_profile_frames, 1) > 0;

ALTER TABLE users
    DROP COLUMN selected_profile_frames;
