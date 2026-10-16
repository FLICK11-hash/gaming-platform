package bot

import (
	"math/rand"
	"testing"

	nchess "github.com/notnil/chess"

	"gaming_platform/internal/domain/connectfour"
)

// referenceNegamax is negamax without alpha-beta bounds - full-width,
// deliberately naive - so pruned negamax's result can be checked against it.
// A pruning bug that changes which move looks best wouldn't show up in a
// "does it still block a forced loss" test if the bug only affects a
// position with no forced tactic in it; comparing raw scores against an
// unpruned reference catches that regardless of the position.
func referenceNegamax(board connectfour.Board, toMove connectfour.Color, depth int) int {
	cols := legalColumns(board)
	if len(cols) == 0 {
		return 0
	}
	if depth == 0 {
		return heuristic(board, toMove)
	}

	best := -c4WinScore * 2
	for _, c := range cols {
		row := dropRow(board, c)
		b := board
		b[row][c] = int(toMove)

		var score int
		if connectfour.CheckWin(b, row, c, toMove) {
			score = c4WinScore
		} else {
			score = -referenceNegamax(b, other(toMove), depth-1)
		}
		if score > best {
			best = score
		}
	}
	return best
}

func TestNegamax_PrunedMatchesUnprunedReference(t *testing.T) {
	rng := rand.New(rand.NewSource(1))

	for trial := 0; trial < 30; trial++ {
		var board connectfour.Board
		turn := connectfour.Red
		// Play a random handful of legal moves to reach a varied,
		// realistic mid-game position.
		for i := 0; i < 6; i++ {
			cols := legalColumns(board)
			if len(cols) == 0 {
				break
			}
			c := cols[rng.Intn(len(cols))]
			row := dropRow(board, c)
			if row < 0 {
				continue
			}
			board[row][c] = int(turn)
			if connectfour.CheckWin(board, row, c, turn) {
				break
			}
			turn = other(turn)
		}

		const depth = 3
		got := negamax(board, turn, depth, -c4WinScore*2, c4WinScore*2)
		want := referenceNegamax(board, turn, depth)
		if got != want {
			t.Fatalf("trial %d: pruned negamax = %d, unpruned reference = %d, board=%v turn=%v", trial, got, want, board, turn)
		}
	}
}

// referenceMinimax is minimax without alpha-beta bounds, for the same
// reason referenceNegamax exists above.
func referenceMinimax(g *nchess.Game, depth int) int {
	moves := g.ValidMoves()
	if len(moves) == 0 {
		switch g.Outcome() {
		case nchess.WhiteWon:
			return mateScore
		case nchess.BlackWon:
			return -mateScore
		default:
			return 0
		}
	}
	if depth == 0 {
		return evaluateMaterial(g)
	}

	turn := g.Position().Turn()
	if turn == nchess.White {
		best := -mateScore * 2
		for _, mv := range moves {
			clone := g.Clone()
			_ = clone.Move(mv)
			if v := referenceMinimax(clone, depth-1); v > best {
				best = v
			}
		}
		return best
	}

	best := mateScore * 2
	for _, mv := range moves {
		clone := g.Clone()
		_ = clone.Move(mv)
		if v := referenceMinimax(clone, depth-1); v < best {
			best = v
		}
	}
	return best
}

func TestMinimax_PrunedMatchesUnprunedReference(t *testing.T) {
	rng := rand.New(rand.NewSource(2))

	for trial := 0; trial < 20; trial++ {
		game := nchess.NewGame()
		for i := 0; i < 6; i++ {
			moves := game.ValidMoves()
			if len(moves) == 0 {
				break
			}
			if err := game.Move(moves[rng.Intn(len(moves))]); err != nil {
				t.Fatalf("applying random move: %v", err)
			}
			if game.Outcome() != nchess.NoOutcome {
				break
			}
		}
		if game.Outcome() != nchess.NoOutcome {
			continue
		}

		const depth = 2
		got := minimax(game, depth, -mateScore*2, mateScore*2)
		want := referenceMinimax(game, depth)
		if got != want {
			t.Fatalf("trial %d: pruned minimax = %d, unpruned reference = %d, FEN=%s", trial, got, want, game.FEN())
		}
	}
}
