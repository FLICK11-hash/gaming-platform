// Package research runs the platform's Monte Carlo bot-vs-bot simulation
// (internal/research/simulate) as a background job the website can kick off
// and poll, instead of only from the standalone cmd/simulate CLI.
package research

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"gaming_platform/internal/domain/bot"
	"gaming_platform/internal/research/simulate"
)

// ErrAlreadyRunning is returned by Start when a run is already in progress.
var ErrAlreadyRunning = errors.New("research: a simulation is already running")

// DefaultGamesPerOpening is how many games a production run plays per
// opening - the same count for every chess opening in both books and every
// Connect Four column, so every bar in every chart reflects an equal-sized
// sample and none is quietly noisier than its neighbors.
const DefaultGamesPerOpening = 20

// runDifficulty is Easy for every on-demand run. Hard-difficulty bots are
// fully deterministic - no randomness in their move choice at all - so
// replaying the same forced opening at Hard difficulty just reproduces the
// exact same game every time; Easy's occasional blunders are what make
// repeated games actually vary, which is the whole point of a Monte Carlo
// batch.
const runDifficulty = bot.Easy

// Result is one finished simulation run's full output.
type Result struct {
	StartedAt       time.Time                           `json:"started_at"`
	FinishedAt      time.Time                           `json:"finished_at"`
	GamesPerOpening int                                 `json:"games_per_opening"`
	WhiteOpenings   []simulate.ChessOpeningReport       `json:"white_openings"`
	BlackOpenings   []simulate.ChessOpeningReport       `json:"black_openings"`
	ConnectFour     []simulate.ConnectFourOpeningReport `json:"connect_four"`
}

// Runner holds the single latest simulation run in memory. There is only
// ever one active or most-recent run for the whole site - this is a shared
// research feature, not a per-player one - and results don't survive a
// server restart; that's an accepted tradeoff for how cheap it keeps the
// feature.
type Runner struct {
	gamesPerOpening int

	mu           sync.Mutex
	status       Status
	result       *Result
	errMsg       string
	runningSince time.Time
}

// Status is the lifecycle state of the Runner's latest run.
type Status string

const (
	StatusIdle    Status = "idle"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusError   Status = "error"
)

// NewRunner builds a Runner that plays gamesPerOpening games for every
// chess opening (in both books) and every Connect Four column on each run.
// Use DefaultGamesPerOpening in production; tests can pass a much smaller
// value to stay fast.
func NewRunner(gamesPerOpening int) *Runner {
	return &Runner{
		gamesPerOpening: gamesPerOpening,
		status:          StatusIdle,
	}
}

// Start kicks off a new run in the background and returns immediately.
// It returns ErrAlreadyRunning without starting anything if a run is
// already in progress.
func (r *Runner) Start() error {
	r.mu.Lock()
	if r.status == StatusRunning {
		r.mu.Unlock()
		return ErrAlreadyRunning
	}
	r.status = StatusRunning
	r.runningSince = time.Now()
	r.mu.Unlock()

	go r.run()
	return nil
}

// run does the actual work in the background goroutine Start spawns. The
// deferred recover is load-bearing: without it, a panic anywhere in the
// simulation (this runs thousands of games unattended) would kill the
// goroutine silently and leave status stuck at StatusRunning forever, with
// no error ever surfacing to the website.
func (r *Runner) run() {
	defer func() {
		if p := recover(); p != nil {
			r.fail(fmt.Errorf("panic: %v", p))
		}
	}()

	started := time.Now()

	whiteOpenings, err := simulate.RunChessOpeningBook(r.gamesPerOpening, runDifficulty, simulate.White)
	if err != nil {
		r.fail(err)
		return
	}
	blackOpenings, err := simulate.RunChessOpeningBook(r.gamesPerOpening, runDifficulty, simulate.Black)
	if err != nil {
		r.fail(err)
		return
	}
	connectFour, err := simulate.RunConnectFourOpeningBook(r.gamesPerOpening, runDifficulty)
	if err != nil {
		r.fail(err)
		return
	}

	result := &Result{
		StartedAt:       started,
		FinishedAt:      time.Now(),
		GamesPerOpening: r.gamesPerOpening,
		WhiteOpenings:   whiteOpenings,
		BlackOpenings:   blackOpenings,
		ConnectFour:     connectFour,
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = StatusDone
	r.result = result
}

func (r *Runner) fail(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = StatusError
	r.errMsg = err.Error()
}

// Snapshot returns the runner's current status, its latest completed result
// (nil if none has finished yet), an error message (only meaningful when
// status is StatusError), and how long the current or most recent run has
// been going (zero if none has ever started).
func (r *Runner) Snapshot() (Status, *Result, string, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var running time.Duration
	if !r.runningSince.IsZero() && r.status == StatusRunning {
		running = time.Since(r.runningSince)
	}
	return r.status, r.result, r.errMsg, running
}
