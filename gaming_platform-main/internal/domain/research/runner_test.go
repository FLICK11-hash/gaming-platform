package research

import (
	"testing"
	"time"
)

func TestRunner_StartThenSnapshotTransitionsToDone(t *testing.T) {
	r := NewRunner(2, 7)

	if status, result, _ := r.Snapshot(); status != StatusIdle || result != nil {
		t.Fatalf("fresh runner: status = %q, result = %v, want idle/nil", status, result)
	}

	if err := r.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if status, _, _ := r.Snapshot(); status != StatusRunning {
		t.Fatalf("status right after Start = %q, want running", status)
	}

	if err := r.Start(); err != ErrAlreadyRunning {
		t.Fatalf("second Start while running: err = %v, want ErrAlreadyRunning", err)
	}

	deadline := time.Now().Add(30 * time.Second)
	var status Status
	var result *Result
	for time.Now().Before(deadline) {
		status, result, _ = r.Snapshot()
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
	if result.ConnectFour.Games == 0 {
		t.Fatal("expected non-zero Connect Four games")
	}

	// A second run is allowed once the first has finished.
	if err := r.Start(); err != nil {
		t.Fatalf("Start after completion: %v", err)
	}
}
