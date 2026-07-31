package research

import (
	"os/exec"
	"testing"
	"time"
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

func TestRunner_StartThenSnapshotTransitionsToDone(t *testing.T) {
	path := findStockfishBinary(t)
	// Even at 1 game per opening, this plays 3 Elo tiers x 2 books x 10
	// openings x 1 game = 60 real Stockfish games, and weak/limited-strength
	// play tends to run long (routinely 80-100+ plies before a decisive
	// result or draw) rather than resolving quickly, so this is a slow test
	// by nature - it exercises the real engine end to end on purpose.
	r := NewRunner(1, path)

	if status, result, _, running := r.Snapshot(); status != StatusIdle || result != nil || running != 0 {
		t.Fatalf("fresh runner: status = %q, result = %v, running = %v, want idle/nil/0", status, result, running)
	}

	if err := r.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if status, _, _, running := r.Snapshot(); status != StatusRunning || running <= 0 {
		t.Fatalf("status right after Start = %q, running = %v, want running/>0", status, running)
	}

	if err := r.Start(); err != ErrAlreadyRunning {
		t.Fatalf("second Start while running: err = %v, want ErrAlreadyRunning", err)
	}

	deadline := time.Now().Add(120 * time.Second)
	var status Status
	var result *Result
	var errMsg string
	for time.Now().Before(deadline) {
		status, result, errMsg, _ = r.Snapshot()
		if status != StatusRunning {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if status != StatusDone {
		t.Fatalf("final status = %q, error = %q, want done", status, errMsg)
	}
	if result == nil {
		t.Fatal("result is nil after a done run")
	}
	if len(result.ChessByElo) != 3 {
		t.Fatalf("expected 3 chess Elo tiers, got %d", len(result.ChessByElo))
	}
	for _, tier := range result.ChessByElo {
		if len(tier.WhiteOpenings) == 0 || len(tier.BlackOpenings) == 0 {
			t.Fatalf("tier %s (%d elo): expected non-empty opening reports, got white=%d black=%d",
				tier.Label, tier.Elo, len(tier.WhiteOpenings), len(tier.BlackOpenings))
		}
	}
	if len(result.ConnectFour) != 7 {
		t.Fatalf("expected 7 Connect Four column reports, got %d", len(result.ConnectFour))
	}
	for _, c := range result.ConnectFour {
		if c.Games == 0 {
			t.Fatalf("column %d: expected non-zero games", c.Column)
		}
	}

	// A second run is allowed once the first has finished.
	if err := r.Start(); err != nil {
		t.Fatalf("Start after completion: %v", err)
	}
}
