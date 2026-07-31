// Package simulate plays out complete bot-vs-bot games entirely in memory -
// no database, no websocket, no HTTP server - so the platform's own game
// engines and bot AI can be run thousands of times in a batch to answer
// questions a single live match never could: how much does moving first
// actually matter, and how much does the opening line change the outcome?
// It reuses the exact same chess/connectfour engines and bot move-choosing
// logic real matches use, so results reflect the platform's actual bots,
// not a separate model of them.
package simulate

import (
	"fmt"

	"gaming_platform/internal/domain/bot"
	"gaming_platform/internal/domain/chess"
	"gaming_platform/internal/domain/connectfour"
)

// simMatchID is arbitrary: every simulated game gets its own fresh engine
// instance, so there's never more than one game per engine to key by ID.
const simMatchID = 1

// maxPlies caps how long a single simulated game is allowed to run before
// it's abandoned as a draw. Real games always terminate (checkmate,
// stalemate, repetition, the 50-move rule, or a full Connect Four board),
// but a batch runner shouldn't be able to hang forever if a future engine
// change ever broke that guarantee.
const maxPlies = 300

// ChessGameResult is the outcome of one simulated chess game.
type ChessGameResult struct {
	// FirstMoverWon/SecondMoverWon/Draw are mutually exclusive; exactly one
	// is true, mirroring how a real match settles (see
	// connectfour.Outcome's tiebreak for the one platform-specific
	// exception, which doesn't apply here since chess draws stay draws).
	FirstMoverWon  bool
	SecondMoverWon bool
	Draw           bool
	// OpeningMove is the first move played, in from+to square form (e.g.
	// "e2e4"), letting a batch runner group results by opening.
	OpeningMove string
	Plies       int
}

// PlayChessGame plays one complete game between two bots, whichever moves
// first at firstDifficulty and whichever moves second at secondDifficulty.
// If forcedOpening is non-empty (a legal first move in from+to square form,
// e.g. "e2e4"), the first mover is made to play it instead of consulting
// its own move-choosing logic, so a batch runner can hold the opening fixed
// while varying everything downstream of it.
func PlayChessGame(firstDifficulty, secondDifficulty bot.Difficulty, forcedOpening string) (ChessGameResult, error) {
	engine := chess.NewEngine()
	engine.StartGame(simMatchID)
	defer engine.EndGame(simMatchID)

	result := ChessGameResult{}
	for ply := 0; ply < maxPlies; ply++ {
		state, err := engine.State(simMatchID)
		if err != nil {
			return ChessGameResult{}, fmt.Errorf("simulate: read chess state: %w", err)
		}
		if state.Outcome != chess.NoOutcome {
			result.Plies = ply
			switch state.Outcome {
			case chess.WhiteWon:
				result.FirstMoverWon = true
			case chess.BlackWon:
				result.SecondMoverWon = true
			default:
				result.Draw = true
			}
			return result, nil
		}

		difficulty := firstDifficulty
		if ply%2 == 1 {
			difficulty = secondDifficulty
		}

		var from, to, promo string
		if ply == 0 && forcedOpening != "" {
			from, to = forcedOpening[:2], forcedOpening[2:]
		} else {
			from, to, promo, err = bot.ChooseChessMove(state.FEN, difficulty)
			if err != nil {
				return ChessGameResult{}, fmt.Errorf("simulate: choose chess move: %w", err)
			}
		}

		if ply == 0 {
			result.OpeningMove = from + to
		}
		if _, err := engine.ApplyMove(simMatchID, from, to, promo); err != nil {
			return ChessGameResult{}, fmt.Errorf("simulate: apply chess move %s%s: %w", from, to, err)
		}
	}

	// maxPlies reached without a decisive outcome or engine-reported draw;
	// treat it as a draw rather than silently mislabeling a stalled game as
	// a win for either side.
	result.Draw = true
	result.Plies = maxPlies
	return result, nil
}

// ConnectFourGameResult is the outcome of one simulated Connect Four game.
type ConnectFourGameResult struct {
	FirstMoverWon  bool
	SecondMoverWon bool
	Draw           bool
	// OpeningColumn is the column (0-indexed) the first mover dropped into.
	OpeningColumn int
	Plies         int
}

// PlayConnectFourGame plays one complete game the same way PlayChessGame
// does: firstDifficulty moves first (as Red), secondDifficulty moves second
// (as Green), and forcedOpeningColumn (if >= 0) overrides the first mover's
// opening column.
func PlayConnectFourGame(firstDifficulty, secondDifficulty bot.Difficulty, forcedOpeningColumn int) (ConnectFourGameResult, error) {
	engine := connectfour.NewEngine()
	engine.StartGame(simMatchID)
	defer engine.EndGame(simMatchID)

	result := ConnectFourGameResult{}
	for ply := 0; ply < maxPlies; ply++ {
		state, err := engine.State(simMatchID)
		if err != nil {
			return ConnectFourGameResult{}, fmt.Errorf("simulate: read connect four state: %w", err)
		}
		if state.Outcome != connectfour.NoOutcome {
			result.Plies = ply
			switch state.Outcome {
			case connectfour.RedWon:
				result.FirstMoverWon = true
			case connectfour.GreenWon:
				result.SecondMoverWon = true
			default:
				result.Draw = true
			}
			return result, nil
		}

		difficulty := firstDifficulty
		if ply%2 == 1 {
			difficulty = secondDifficulty
		}

		column := forcedOpeningColumn
		if ply != 0 || forcedOpeningColumn < 0 {
			column, err = bot.ChooseConnectFourMove(state.Board, state.Turn, difficulty)
			if err != nil {
				return ConnectFourGameResult{}, fmt.Errorf("simulate: choose connect four move: %w", err)
			}
		}

		if ply == 0 {
			result.OpeningColumn = column
		}
		if _, err := engine.DropChip(simMatchID, column, state.Turn); err != nil {
			return ConnectFourGameResult{}, fmt.Errorf("simulate: drop chip in column %d: %w", column, err)
		}
	}

	result.Draw = true
	result.Plies = maxPlies
	return result, nil
}
