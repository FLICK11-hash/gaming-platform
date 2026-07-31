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
	"gaming_platform/internal/engine/stockfish"
	"gaming_platform/internal/research/simulate"
)

// ErrAlreadyRunning is returned by Start when a run is already in progress.
var ErrAlreadyRunning = errors.New("research: a simulation is already running")

// DefaultGamesPerOpening is how many games a production run plays per
// opening/column - the same count for every chess opening in both books,
// every chess Elo tier, and every Connect Four column, so every bar in
// every chart reflects an equal-sized sample and none is quietly noisier
// than its neighbors.
const DefaultGamesPerOpening = 20

// connectFourDifficulty is fixed at Easy for every run. Connect Four's bot
// strength isn't part of this project's Elo question - only chess's is
// (see chessEloTiers) - and Easy is the only pure-Go tier with real
// move-to-move randomness (Medium/Hard's fallback search is fully
// deterministic), so it's what makes repeated games actually vary.
const connectFourDifficulty = bot.Easy

// connectFourDifficultyLabel is connectFourDifficulty rendered as the
// string the website displays, kept next to the constant so the two can't
// drift apart.
const connectFourDifficultyLabel = "easy"

// batchMoveTime is how long each Stockfish search gets per move during a
// batch run - far shorter than stockfish.RealMatchMoveTime. That's
// deliberate: a live match wants a real "thinking" pause, but this harness
// plays thousands of individual moves per run (2 opening books x 10
// openings x DefaultGamesPerOpening games, x3 Elo tiers) and cares about
// the statistical playing pattern across many games, not any single move's
// depth.
const batchMoveTime = 20 * time.Millisecond

// batchEnginePoolSize is how many games can be in flight at once per Elo
// tier (see simulate.RunChessOpeningBookConcurrent) - independent of, and
// not competing with, the persistent pools real matches use
// (bot.ConfigureStockfish), since this runner starts and tears down its
// own engines for the duration of a single run.
const batchEnginePoolSize = 4

// chessEloTier is one bot-Elo level to run the full chess opening book
// against.
type chessEloTier struct {
	Label string
	Elo   int
}

// chessEloTiers are exactly the three Stockfish targets the platform's own
// three bot accounts play at in real matches (see internal/domain/bot), so
// this measures whether/how the same edges - moving first, a specific
// opening - hold up as the live bots get stronger, using their actual Elo
// settings rather than an arbitrary separate scale.
var chessEloTiers = []chessEloTier{
	{Label: bot.RookieBotUsername, Elo: bot.RookieBotUCIElo},
	{Label: bot.ExpertBotUsername, Elo: bot.ExpertBotUCIElo},
	{Label: bot.GrandmasterBotUsername, Elo: bot.GrandmasterBotUCIElo},
}

// ChessEloTierResult is one Elo tier's full opening-book result.
type ChessEloTierResult struct {
	Label         string                        `json:"label"`
	Elo           int                           `json:"elo"`
	WhiteOpenings []simulate.ChessOpeningReport `json:"white_openings"`
	BlackOpenings []simulate.ChessOpeningReport `json:"black_openings"`
}

// Result is one finished simulation run's full output.
type Result struct {
	StartedAt       time.Time `json:"started_at"`
	FinishedAt      time.Time `json:"finished_at"`
	GamesPerOpening int       `json:"games_per_opening"`
	// ChessByElo holds one full opening-book result per real Stockfish Elo
	// tier (see chessEloTiers) - both sides of every game play at the same
	// Elo within a tier, so the only variable being measured within a tier
	// is move order/opening choice; comparing across tiers is what answers
	// "does anything change as the bots get stronger."
	ChessByElo []ChessEloTierResult `json:"chess_by_elo"`
	// ConnectFourDifficulty is the bot AI tier both Connect Four sides
	// played at - not an Elo rating, since Connect Four's bot strength
	// isn't part of this comparison; both simulated bots are identical,
	// isolating move order/opening choice as the only variable.
	ConnectFourDifficulty string                              `json:"connect_four_difficulty"`
	ConnectFour           []simulate.ConnectFourOpeningReport `json:"connect_four"`
}

// Runner holds the single latest simulation run in memory. There is only
// ever one active or most-recent run for the whole site - this is a shared
// research feature, not a per-player one - and results don't survive a
// server restart; that's an accepted tradeoff for how cheap it keeps the
// feature.
type Runner struct {
	gamesPerOpening int
	stockfishPath   string

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
// chess opening (in both books, at every Elo tier) and every Connect Four
// column on each run, using stockfishPath to start the chess Elo tiers'
// engines (the same binary bot.ConfigureStockfish uses for real matches;
// see internal/config's STOCKFISH_PATH). Use DefaultGamesPerOpening in
// production; tests can pass a much smaller value to stay fast.
func NewRunner(gamesPerOpening int, stockfishPath string) *Runner {
	return &Runner{
		gamesPerOpening: gamesPerOpening,
		stockfishPath:   stockfishPath,
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

	chessByElo, err := r.runChessEloTiers()
	if err != nil {
		r.fail(err)
		return
	}

	connectFour, err := simulate.RunConnectFourOpeningBook(r.gamesPerOpening, connectFourDifficulty)
	if err != nil {
		r.fail(err)
		return
	}

	result := &Result{
		StartedAt:             started,
		FinishedAt:            time.Now(),
		GamesPerOpening:       r.gamesPerOpening,
		ChessByElo:            chessByElo,
		ConnectFourDifficulty: connectFourDifficultyLabel,
		ConnectFour:           connectFour,
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = StatusDone
	r.result = result
}

// runChessEloTiers plays the full White and Black opening books at every
// entry in chessEloTiers, each against its own freshly-started Stockfish
// engine pool (started and torn down within this one run, independent of
// bot.ConfigureStockfish's persistent real-match pools). A real engine is
// required here, not just preferred: the pure-Go fallback's Medium and
// Hard tiers share the exact same evaluation (see chess.go's
// ChooseChessMove), so without Stockfish there's no genuine third data
// point to compare - the run fails loudly instead of silently producing a
// chart where two of the three tiers are identical.
func (r *Runner) runChessEloTiers() ([]ChessEloTierResult, error) {
	results := make([]ChessEloTierResult, 0, len(chessEloTiers))
	for _, tier := range chessEloTiers {
		engine, err := stockfish.NewEngine(r.stockfishPath, tier.Elo, batchEnginePoolSize, batchMoveTime)
		if err != nil {
			return nil, fmt.Errorf("research: start Stockfish for %s (%d elo): %w", tier.Label, tier.Elo, err)
		}

		mover := func(fen string) (string, string, string, error) { return engine.BestMove(fen) }
		white, err := simulate.RunChessOpeningBookConcurrent(r.gamesPerOpening, mover, simulate.White, batchEnginePoolSize)
		if err != nil {
			engine.Close()
			return nil, fmt.Errorf("research: %s white openings: %w", tier.Label, err)
		}
		black, err := simulate.RunChessOpeningBookConcurrent(r.gamesPerOpening, mover, simulate.Black, batchEnginePoolSize)
		engine.Close()
		if err != nil {
			return nil, fmt.Errorf("research: %s black openings: %w", tier.Label, err)
		}

		results = append(results, ChessEloTierResult{
			Label:         tier.Label,
			Elo:           tier.Elo,
			WhiteOpenings: white,
			BlackOpenings: black,
		})
	}
	return results, nil
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
