-- Pre-moving costs a fractional 0.2s penalty (see match/service.go
-- PremoveCostSeconds), so match clocks need sub-second precision instead of
-- whole seconds.
ALTER TABLE match_clocks ALTER COLUMN remaining_seconds TYPE double precision;
