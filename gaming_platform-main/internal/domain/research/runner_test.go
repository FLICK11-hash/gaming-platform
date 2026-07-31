package research

import (
	"testing"
	"time"
)

func TestRunner_StartThenSnapshotTransitionsToDone(t *testing.T) {
	r := NewRunner(2)

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

	deadline := time.Now().Add(30 * time.Second)
	var status Status
	var result *Result
	for time.Now().Before(deadline) {
		status, result, _, _ = r.Snapshot()
		if status != StatusRunning {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if status != StatusDone {
		t.Fatalf("final status = %q, want done", status)
	}
	if result == nil {
		t.Fatal("result is nil after a done run")
	}
	if len(result.WhiteOpenings) == 0 || len(result.BlackOpenings) == 0 {
		t.Fatalf("expected non-empty opening reports, got white=%d black=%d",
			len(result.WhiteOpenings), len(result.BlackOpenings))
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
