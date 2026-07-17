// Package clock implements bullet-chess countdown clock math.
package clock

import "time"

const DefaultStartingSeconds = 60

// State is a snapshot of one player's clock.
type State struct {
	RemainingSeconds int
	IsCurrentTurn    bool
	LastStartedAt    time.Time
}

// RemainingAt returns how many whole seconds are left on the clock at `now`.
// If it isn't this player's turn, the clock isn't running and the stored
// remaining value is authoritative.
func RemainingAt(s State, now time.Time) int {
	if !s.IsCurrentTurn {
		return s.RemainingSeconds
	}
	elapsed := int(now.Sub(s.LastStartedAt).Seconds())
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
func Pause(s State, now time.Time) int {
	return RemainingAt(s, now)
}
