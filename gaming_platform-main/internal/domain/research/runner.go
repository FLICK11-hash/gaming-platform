// Package research runs the platform's Monte Carlo bot-vs-bot simulation
// (internal/research/simulate) as a background job the website can kick off
// and poll, instead of only from the standalone cmd/simulate CLI.
package research

import (
	"errors"
	"sync"
	"time"

	"gaming_platform/internal/domain/bot"
	"gaming_platform/internal/research/simulate"
)

// ErrAlreadyRunning is returned by Start when a run is already in progress.
var ErrAlreadyRunning = errors.New("research: a simulation is already running")

// DefaultGamesPerOpening is how many games a production run plays per
// opening in each chess book, and DefaultConnectFourGames is the total
// Connect Four games played (spread across its 7 possible opening
// columns). Both are sized to keep a whole run to roughly a minute or two
// on a single shared core - enough games for the win rates to mean
// something without tying up the server for long.
const (
	DefaultGamesPerOpening  = 20
	DefaultConnectFourGames = 140
)

// runDifficulty is Easy for every on-demand run. Hard-difficulty bots are
// fully deterministic - no randomness in their move choice at all - so
// replaying the same forced opening at Hard difficulty just reproduces the
// exact same game every time; Easy's occasional blunders are what make
// repeated games actually vary, which is the whole point of a Monte Carlo
// batch.
const runDifficulty = bot.Easy

// Result is one finished simulation run's full output.
type Result struct {
	StartedAt       time.Time                     `json:"started_at"`
	FinishedAt      time.Time                     `json:"finished_at"`
	GamesPerOpening int                           `json:"games_per_opening"`
	WhiteOpenings   []simulate.ChessOpeningReport `json:"white_openings"`
	BlackOpenings   []simulate.ChessOpeningReport `json:"black_openings"`
	ConnectFour     simulate.BatchStats           `json:"connect_four"`
}

// Runner holds the single latest simulation run in memory. There is only
// ever one active or most-recent run for the whole site - this is a shared
// research feature, not a per-player one - and results don't survive a
// server restart; that's an accepted tradeoff for how cheap it keeps the
// feature.
type Runner struct {
	gamesPerOpening  int
	connectFourGames int

	mu     sync.Mutex
	status Status
	result *Result
	errMsg string
}

// Status is the lifecycle state of the Runner's latest run.
type Status string

const (
	StatusIdle    Status = "idle"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusError   Status = "error"
)

// NewRunner builds a Runner that plays gamesPerOpening games per chess
// opening and connectFourGames total Connect Four games on each run. Use
// DefaultGamesPerOpening/DefaultConnectFourGames in production; tests can
// pass much smaller values to stay fast.
func NewRunner(gamesPerOpening, connectFourGames int) *Runner {
	return &Runner{
		gamesPerOpening:  gamesPerOpening,
		connectFourGames: connectFourGames,
		status:           StatusIdle,
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
	r.mu.Unlock()

	go r.run()
	return nil
}

func (r *Runner) run() {
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
	connectFour, err := simulate.RunConnectFourBatch(r.connectFourGames, runDifficulty, true)
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
// (nil if none has finished yet), and an error message (only meaningful
// when status is StatusError).
func (r *Runner) Snapshot() (Status, *Result, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status, r.result, r.errMsg
}
