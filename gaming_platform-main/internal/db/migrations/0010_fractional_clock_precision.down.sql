ALTER TABLE match_clocks ALTER COLUMN remaining_seconds TYPE integer USING round(remaining_seconds)::integer;
