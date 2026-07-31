// Package simulate plays out complete bot-vs-bot games entirely in memory -
// no database, no websocket, no HTTP server - so the platform's own game
// engines and bot AI can be run thousands of times in a batch to answer
// questions a single live match never could: how much of an edge does the
// side that moves first actually have, and how much does the specific
// opening line change the outcome? It reuses the exact same chess/
// connectfour engines and bot move-choosing logic real matches use, so
// results reflect the platform's actual bots, not a separate model of them.
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
	// WhiteWon/BlackWon/Draw are mutually exclusive; exactly one is true.
	WhiteWon bool
	BlackWon bool
	Draw     bool
	// MovesPlayed holds every forced opening move actually played (in
	// from+to square form, e.g. "e2e4"), i.e. the prefix passed to
	// PlayChessGame as forcedMoves - not the whole game, just the part a
	// batch runner held fixed on purpose.
	MovesPlayed []string
	Plies       int
}

// PlayChessGame plays one complete game between whiteDifficulty (playing
// White) and blackDifficulty (playing Black). Chess always has White move
// first, so which side moves first isn't a variable to control here -
// which color you assign the stronger/weaker bot to is.
//
// forcedMoves, if non-empty, is a sequence of legal moves (from+to square
// form, e.g. []string{"e2e4", "c7c5"}) played in order at the very start of
// the game instead of consulting the bots' own move-choosing logic; play
// resumes with the bots once the sequence is exhausted. This lets a batch
// runner hold an opening (or a specific reply to it) fixed while varying
// everything downstream.
func PlayChessGame(whiteDifficulty, blackDifficulty bot.Difficulty, forcedMoves []string) (ChessGameResult, error) {
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
				result.WhiteWon = true
			case chess.BlackWon:
				result.BlackWon = true
			default:
				result.Draw = true
			}
			return result, nil
		}

		difficulty := whiteDifficulty
		if ply%2 == 1 {
			difficulty = blackDifficulty
		}

		var from, to, promo string
		if ply < len(forcedMoves) {
			from, to = forcedMoves[ply][:2], forcedMoves[ply][2:]
		} else {
			from, to, promo, err = bot.ChooseChessMove(state.FEN, difficulty)
			if err != nil {
				return ChessGameResult{}, fmt.Errorf("simulate: choose chess move: %w", err)
			}
		}

		if ply < len(forcedMoves) {
			result.MovesPlayed = append(result.MovesPlayed, from+to)
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
	RedWon   bool
	GreenWon bool
	Draw     bool
	// OpeningColumn is the column (0-indexed) Red, who always moves first,
	// dropped into.
	OpeningColumn int
	Plies         int
}

// PlayConnectFourGame plays one complete game between redDifficulty
// (playing Red, who always moves first) and greenDifficulty (playing
// Green). If forcedOpeningColumn is >= 0, Red is made to open there instead
// of consulting its own move-choosing logic, so a batch runner can hold the
// opening column fixed while varying everything downstream of it.
func PlayConnectFourGame(redDifficulty, greenDifficulty bot.Difficulty, forcedOpeningColumn int) (ConnectFourGameResult, error) {
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
				result.RedWon = true
			case connectfour.GreenWon:
				result.GreenWon = true
			default:
				result.Draw = true
			}
			return result, nil
		}

		difficulty := redDifficulty
		if ply%2 == 1 {
			difficulty = greenDifficulty
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
