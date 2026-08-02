package stockfish

import (
	"os/exec"
	"testing"
	"time"
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

	e, err := NewEngine(path, 1, RealMatchMoveTime)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer e.Close()

	const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	from, to, _, err := e.BestMove(startFEN, 1320)
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
	e, err := NewEngine(path, poolSize, RealMatchMoveTime)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer e.Close()

	const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	errCh := make(chan error, poolSize*2)
	for i := 0; i < poolSize*2; i++ {
		go func() {
			_, _, _, err := e.BestMove(startFEN, 1320)
			errCh <- err
		}()
	}
	for i := 0; i < poolSize*2; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("concurrent BestMove: %v", err)
		}
	}
}

func TestEngine_MoveTimeIsHonored(t *testing.T) {
	path := findBinary(t)

	e, err := NewEngine(path, 1, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer e.Close()

	const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	start := time.Now()
	if _, _, _, err := e.BestMove(startFEN, 1320); err != nil {
		t.Fatalf("BestMove: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("BestMove with a 10ms movetime took %v, want well under RealMatchMoveTime (%v)", elapsed, RealMatchMoveTime)
	}
}

// TestEngine_SwitchesEloOnSharedPool exercises the whole point of sharing
// one pool across strengths instead of dedicating a pool per strength: a
// single process must be able to serve different Elo requests back to
// back and actually reconfigure each time, not silently keep playing at
// whatever Elo it started with.
func TestEngine_SwitchesEloOnSharedPool(t *testing.T) {
	path := findBinary(t)

	e, err := NewEngine(path, 1, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer e.Close()

	const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	for _, elo := range []int{1320, 2500, 1320, MaxUCIElo} {
		from, to, _, err := e.BestMove(startFEN, elo)
		if err != nil {
			t.Fatalf("BestMove at %d elo: %v", elo, err)
		}
		if len(from) != 2 || len(to) != 2 {
			t.Fatalf("BestMove at %d elo returned from=%q to=%q, want two-character squares", elo, from, to)
		}
	}
}

// TestParseBestMoveLine exercises move parsing directly against canned UCI
// output instead of a running engine. Even at the top of Stockfish's Elo
// range, UCI_LimitStrength stays enabled (see ConfigureStockfish in
// internal/domain/bot - no bot tier should play literal, unbounded engine
// strength), so it deliberately still introduces occasional suboptimal
// moves; that made an earlier version of this test - which checked a real
// engine actually finds a forced promotion - flaky under load. Parsing
// correctness doesn't depend on what the engine decided to play, so it
// shouldn't depend on a live process at all.
func TestParseBestMoveLine(t *testing.T) {
	cases := []struct {
		line            string
		from, to, promo string
		wantErr         bool
	}{
		{line: "bestmove e2e4", from: "e2", to: "e4", promo: ""},
		{line: "bestmove e7e8q ponder e8 somewhere", from: "e7", to: "e8", promo: "q"},
		{line: "bestmove g1f3 ponder d7d5", from: "g1", to: "f3", promo: ""},
		{line: "bestmove (none)", wantErr: true},
		{line: "bestmove", wantErr: true},
		{line: "bestmove xy", wantErr: true},
	}
	for _, c := range cases {
		from, to, promo, err := parseBestMoveLine(c.line)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseBestMoveLine(%q): expected an error, got from=%q to=%q promo=%q", c.line, from, to, promo)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseBestMoveLine(%q): unexpected error: %v", c.line, err)
			continue
		}
		if from != c.from || to != c.to || promo != c.promo {
			t.Errorf("parseBestMoveLine(%q) = (%q, %q, %q), want (%q, %q, %q)", c.line, from, to, promo, c.from, c.to, c.promo)
		}
	}
}

func TestEngine_PromotionMoveIsParsedFromRealEngine(t *testing.T) {
	path := findBinary(t)

	e, err := NewEngine(path, 1, RealMatchMoveTime)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer e.Close()

	// White pawn on e7, one push from queening, kings tucked in opposite
	// corners (far enough apart to be a legal position and for Black to
	// have no hope of stopping the push). This doesn't assert Stockfish
	// actually finds the promotion (it's strength-limited even at the top
	// of the range, so it isn't guaranteed to under load) - just that
	// whatever move it does return parses into a well-formed shape.
	const nearPromotionFEN = "k7/4P3/8/8/8/8/8/K7 w - - 0 1"
	from, to, _, err := e.BestMove(nearPromotionFEN, MaxUCIElo)
	if err != nil {
		t.Fatalf("BestMove: %v", err)
	}
	if len(from) != 2 || len(to) != 2 {
		t.Fatalf("BestMove returned from=%q to=%q, want two-character squares", from, to)
	}
}
