// Package chess wraps github.com/notnil/chess to provide per-match, in-memory
// chess game state. The ERD has no move-history column, so live game state
// lives only in memory here; only match status/winner/timestamps are
// persisted to the database.
package chess

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	nchess "github.com/notnil/chess"
)

var (
	ErrGameNotFound = errors.New("chess: game not found for match")
	ErrIllegalMove  = errors.New("chess: illegal move")
)

// Color mirrors nchess.Color so callers outside this package don't need to
// import notnil/chess directly.
type Color = nchess.Color

const (
	White = nchess.White
	Black = nchess.Black
)

// Outcome mirrors nchess.Outcome.
type Outcome = nchess.Outcome

const (
	NoOutcome = nchess.NoOutcome
	WhiteWon  = nchess.WhiteWon
	BlackWon  = nchess.BlackWon
	Draw      = nchess.Draw
)

// MoveResult describes the state of a match after a move is applied.
type MoveResult struct {
	FEN     string
	Turn    Color
	Outcome Outcome
	Method  nchess.Method
}

// Engine holds one live chess.Game per active match_id.
type Engine struct {
	mu    sync.Mutex
	games map[int64]*nchess.Game
}

func NewEngine() *Engine {
	return &Engine{games: make(map[int64]*nchess.Game)}
}

// StartGame creates a fresh game for matchID.
func (e *Engine) StartGame(matchID int64) MoveResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	game := nchess.NewGame()
	e.games[matchID] = game

	return MoveResult{
		FEN:     game.FEN(),
		Turn:    game.Position().Turn(),
		Outcome: game.Outcome(),
		Method:  game.Method(),
	}
}

// ApplyMove validates and applies a move given as a from/to square pair
// (e.g. "e2", "e4") plus an optional promotion piece letter ("q", "r", "b",
// "n"). Moves are matched against the position's legal moves by square
// rather than decoded from a notation string, since long algebraic
// notation requires a piece-letter prefix for non-pawn moves that a client
// has no reliable way to supply.
func (e *Engine) ApplyMove(matchID int64, from, to, promo string) (MoveResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	game, ok := e.games[matchID]
	if !ok {
		return MoveResult{}, ErrGameNotFound
	}

	fromSq, err := parseSquare(from)
	if err != nil {
		return MoveResult{}, ErrIllegalMove
	}
	toSq, err := parseSquare(to)
	if err != nil {
		return MoveResult{}, ErrIllegalMove
	}

	var chosen *nchess.Move
	for _, m := range game.ValidMoves() {
		if m.S1() != fromSq || m.S2() != toSq {
			continue
		}
		if promo != "" && !strings.EqualFold(m.Promo().String(), promo) {
			continue
		}
		chosen = m
		break
	}
	if chosen == nil {
		return MoveResult{}, ErrIllegalMove
	}

	if err := game.Move(chosen); err != nil {
		return MoveResult{}, ErrIllegalMove
	}

	return MoveResult{
		FEN:     game.FEN(),
		Turn:    game.Position().Turn(),
		Outcome: game.Outcome(),
		Method:  game.Method(),
	}, nil
}

func parseSquare(s string) (nchess.Square, error) {
	if len(s) != 2 {
		return nchess.NoSquare, fmt.Errorf("chess: invalid square %q", s)
	}
	file := nchess.File(s[0] - 'a')
	rank := nchess.Rank(s[1] - '1')
	if file < nchess.FileA || file > nchess.FileH || rank < nchess.Rank1 || rank > nchess.Rank8 {
		return nchess.NoSquare, fmt.Errorf("chess: invalid square %q", s)
	}
	return nchess.NewSquare(file, rank), nil
}

// Resign marks the game as resigned by the given color and returns the
// resulting outcome.
func (e *Engine) Resign(matchID int64, color Color) (MoveResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	game, ok := e.games[matchID]
	if !ok {
		return MoveResult{}, ErrGameNotFound
	}

	game.Resign(color)

	return MoveResult{
		FEN:     game.FEN(),
		Turn:    game.Position().Turn(),
		Outcome: game.Outcome(),
		Method:  game.Method(),
	}, nil
}

// State returns the current state of matchID's game without mutating it.
func (e *Engine) State(matchID int64) (MoveResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	game, ok := e.games[matchID]
	if !ok {
		return MoveResult{}, ErrGameNotFound
	}

	return MoveResult{
		FEN:     game.FEN(),
		Turn:    game.Position().Turn(),
		Outcome: game.Outcome(),
		Method:  game.Method(),
	}, nil
}

// EndGame drops the in-memory game once a match is complete.
func (e *Engine) EndGame(matchID int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.games, matchID)
}
