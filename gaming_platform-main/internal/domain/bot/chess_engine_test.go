package bot

import (
	"os/exec"
	"testing"

	nchess "github.com/notnil/chess"
)

func findStockfishBinary(t *testing.T) string {
	t.Helper()
	for _, candidate := range []string{"stockfish", "/usr/games/stockfish", "/usr/bin/stockfish"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	t.Skip("stockfish binary not found on this machine; skipping")
	return ""
}

func TestChooseChessMoveEngine_WithoutConfiguredEngineFallsBackToBuiltIn(t *testing.T) {
	// This test only holds if no earlier test in this package has called
	// ConfigureStockfish and left it configured - true today since
	// TestChooseChessMoveEngine_UsesConfiguredStockfish (below) is the only
	// other place that does, and it doesn't offer a way to unconfigure.
	// Guard against silently testing the wrong path if that ever changes.
	if sharedEngine != nil {
		t.Skip("an engine is already configured in this test binary; fallback path isn't exercised")
	}

	game := nchess.NewGame()
	for _, difficulty := range []Difficulty{Easy, Medium, Hard} {
		from, to, _, err := ChooseChessMoveEngine(game.FEN(), difficulty)
		if err != nil {
			t.Fatalf("difficulty %v: unexpected error: %v", difficulty, err)
		}
		if from == "" || to == "" {
			t.Fatalf("difficulty %v: expected a non-empty fallback move, got %q %q", difficulty, from, to)
		}
	}
}

func TestChooseChessMoveEngine_UsesConfiguredStockfish(t *testing.T) {
	path := findStockfishBinary(t)

	if err := ConfigureStockfish(path); err != nil {
		t.Fatalf("ConfigureStockfish: %v", err)
	}

	game := nchess.NewGame()
	for _, difficulty := range []Difficulty{Easy, Medium, Hard} {
		from, to, _, err := ChooseChessMoveEngine(game.FEN(), difficulty)
		if err != nil {
			t.Fatalf("difficulty %v: unexpected error: %v", difficulty, err)
		}

		found := false
		for _, mv := range game.ValidMoves() {
			if mv.S1().String() == from && mv.S2().String() == to {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("difficulty %v: engine returned %s%s, not a legal move from the starting position", difficulty, from, to)
		}
	}
}
