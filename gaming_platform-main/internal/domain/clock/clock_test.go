package clock

import (
	"testing"
	"time"
)

func TestRemainingAt_NotCurrentTurn(t *testing.T) {
	s := State{RemainingSeconds: 45, IsCurrentTurn: false}
	if got := RemainingAt(s, time.Now().Add(time.Hour)); got != 45 {
		t.Errorf("RemainingAt = %d, want 45 (clock not running)", got)
	}
}

func TestRemainingAt_CountsDown(t *testing.T) {
	start := time.Now()
	s := State{RemainingSeconds: 60, IsCurrentTurn: true, LastStartedAt: start}

	got := RemainingAt(s, start.Add(10*time.Second))
	if got != 50 {
		t.Errorf("RemainingAt after 10s = %d, want 50", got)
	}
}

func TestRemainingAt_FloorsAtZero(t *testing.T) {
	start := time.Now()
	s := State{RemainingSeconds: 60, IsCurrentTurn: true, LastStartedAt: start}

	got := RemainingAt(s, start.Add(90*time.Second))
	if got != 0 {
		t.Errorf("RemainingAt after 90s on a 60s clock = %d, want 0", got)
	}
}

func TestExpired(t *testing.T) {
	start := time.Now()
	s := State{RemainingSeconds: 60, IsCurrentTurn: true, LastStartedAt: start}

	if Expired(s, start.Add(30*time.Second)) {
		t.Error("clock should not be expired after 30s of a 60s allotment")
	}
	if !Expired(s, start.Add(61*time.Second)) {
		t.Error("clock should be expired after 61s of a 60s allotment")
	}
}
