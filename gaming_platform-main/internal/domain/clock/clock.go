// Package clock implements bullet-chess countdown clock math.
package clock

import "time"

const DefaultStartingSeconds = 60

// State is a snapshot of one player's clock. RemainingSeconds carries
// fractional precision so small costs (like the pre-move penalty) actually
// take effect instead of vanishing to integer truncation.
type State struct {
	RemainingSeconds float64
	IsCurrentTurn    bool
	LastStartedAt    time.Time
}

// RemainingAt returns how many seconds are left on the clock at `now`. If it
// isn't this player's turn, the clock isn't running and the stored
// remaining value is authoritative.
func RemainingAt(s State, now time.Time) float64 {
	if !s.IsCurrentTurn {
		return s.RemainingSeconds
	}
	elapsed := now.Sub(s.LastStartedAt).Seconds()
	remaining := s.RemainingSeconds - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Expired reports whether the player's time has run out.
func Expired(s State, now time.Time) bool {
	return RemainingAt(s, now) <= 0
}

// Pause freezes a running clock at `now`, returning the remaining seconds to
// persist.
func Pause(s State, now time.Time) float64 {
	return RemainingAt(s, now)
}
