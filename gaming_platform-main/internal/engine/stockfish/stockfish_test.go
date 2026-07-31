package stockfish

import (
	"os/exec"
	"testing"
)

// findBinary locates a real Stockfish binary for the test to exercise
// against. Tests here are skipped (not failed) if none is found, since
// Stockfish is an external dependency this repo doesn't vendor.
func findBinary(t *testing.T) string {
	t.Helper()
	for _, candidate := range []string{"stockfish", "/usr/games/stockfish", "/usr/bin/stockfish"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	t.Skip("stockfish binary not found on this machine; skipping")
	return ""
}

func TestNewEngine_StartsAndReportsBestMove(t *testing.T) {
	path := findBinary(t)

	e, err := NewEngine(path, 1320, 1)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer e.Close()

	const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	from, to, _, err := e.BestMove(startFEN)
	if err != nil {
		t.Fatalf("BestMove: %v", err)
	}
	if len(from) != 2 || len(to) != 2 {
		t.Fatalf("BestMove returned from=%q to=%q, want two-character squares", from, to)
	}
}

func TestEngine_HandlesConcurrentCallsAcrossPool(t *testing.T) {
	path := findBinary(t)

	const poolSize = 3
	e, err := NewEngine(path, 1320, poolSize)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer e.Close()

	const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	errCh := make(chan error, poolSize*2)
	for i := 0; i < poolSize*2; i++ {
		go func() {
			_, _, _, err := e.BestMove(startFEN)
			errCh <- err
		}()
	}
	for i := 0; i < poolSize*2; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("concurrent BestMove: %v", err)
		}
	}
}

func TestEngine_PromotionMoveIsParsed(t *testing.T) {
	path := findBinary(t)

	// Max strength here, deliberately: UCI_LimitStrength intentionally
	// plays weaker-than-best moves sometimes (that's the whole point of
	// simulating a lower rating), so a heavily-limited engine can't be
	// trusted to find the obvious win in a trivial forced position. Parsing
	// a promotion move correctly doesn't depend on strength; forcing the
	// engine to actually produce one here does.
	e, err := NewEngine(path, MaxUCIElo, 1)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer e.Close()

	// White pawn on e7, one push from queening, kings tucked in opposite
	// corners (far enough apart to be a legal position and for Black to
	// have no hope of stopping the push).
	const nearPromotionFEN = "k7/4P3/8/8/8/8/8/K7 w - - 0 1"
	from, to, promo, err := e.BestMove(nearPromotionFEN)
	if err != nil {
		t.Fatalf("BestMove: %v", err)
	}
	if from != "e7" || to != "e8" {
		t.Fatalf("BestMove = %s%s, want e7e8", from, to)
	}
	if promo == "" {
		t.Fatalf("expected a promotion piece letter, got empty string")
	}
}
