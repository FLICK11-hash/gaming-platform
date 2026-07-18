package bot

import (
	"errors"
	"math/rand"

	nchess "github.com/notnil/chess"
)

// ErrNoLegalMoves means the FEN represents a position with no legal moves,
// which should never happen for an in-progress match (the game would have
// already ended by checkmate/stalemate).
var ErrNoLegalMoves = errors.New("bot: position has no legal moves")

// pieceValue is the standard material valuation used for both difficulty
// tiers' evaluation function.
func pieceValue(t nchess.PieceType) int {
	switch t {
	case nchess.Pawn:
		return 1
	case nchess.Knight, nchess.Bishop:
		return 3
	case nchess.Rook:
		return 5
	case nchess.Queen:
		return 9
	default:
		return 0
	}
}

// evaluateMaterial scores a position from White's perspective: positive
// favors White, negative favors Black.
func evaluateMaterial(g *nchess.Game) int {
	score := 0
	for _, piece := range g.Position().Board().SquareMap() {
		v := pieceValue(piece.Type())
		if piece.Color() == nchess.White {
			score += v
		} else {
			score -= v
		}
	}
	return score
}

const mateScore = 100000

// minimax returns the best score achievable from the position's perspective
// (positive always favors White, mirroring evaluateMaterial), searching
// `depth` plies deeper via full-width search - no pruning, since the small
// depths used here (<=2) keep the branching factor manageable.
func minimax(g *nchess.Game, depth int) int {
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
			if v := minimax(clone, depth-1); v > best {
				best = v
			}
		}
		return best
	}

	best := mateScore * 2
	for _, mv := range moves {
		clone := g.Clone()
		_ = clone.Move(mv)
		if v := minimax(clone, depth-1); v < best {
			best = v
		}
	}
	return best
}

// betterFor reports whether score is a better outcome for turn than best.
func betterFor(turn nchess.Color, score, best int) bool {
	if turn == nchess.White {
		return score > best
	}
	return score < best
}

// isWinFor reports whether outcome is a decisive win for color.
func isWinFor(o nchess.Outcome, color nchess.Color) bool {
	switch o {
	case nchess.WhiteWon:
		return color == nchess.White
	case nchess.BlackWon:
		return color == nchess.Black
	}
	return false
}

func squareStr(sq nchess.Square) string {
	return sq.String()
}

func promoStr(t nchess.PieceType) string {
	return t.String()
}

// chooseEasyMove plays mostly at random, but about half the time greedily
// grabs whatever material is available one ply ahead (i.e. without
// considering the opponent's reply), so it isn't oblivious to hanging a
// free piece right in front of it.
func chooseEasyMove(game *nchess.Game, moves []*nchess.Move) *nchess.Move {
	if rand.Intn(2) == 0 {
		return moves[rand.Intn(len(moves))]
	}

	turn := game.Position().Turn()
	var best *nchess.Move
	bestScore := 0
	for i, mv := range moves {
		clone := game.Clone()
		_ = clone.Move(mv)
		score := evaluateMaterial(clone)
		if i == 0 || betterFor(turn, score, bestScore) {
			bestScore = score
			best = mv
		}
	}
	return best
}

// chooseHardMove looks two plies ahead (its move, then the opponent's best
// reply) and picks whichever move leaves it in the best material position
// after that reply, i.e. it won't hang a piece to any single-move tactic.
func chooseHardMove(game *nchess.Game, moves []*nchess.Move) *nchess.Move {
	turn := game.Position().Turn()
	var best *nchess.Move
	bestScore := 0
	for i, mv := range moves {
		clone := game.Clone()
		_ = clone.Move(mv)
		score := minimax(clone, 1)
		if i == 0 || betterFor(turn, score, bestScore) {
			bestScore = score
			best = mv
		}
	}
	return best
}

// ChooseChessMove picks a from/to/promotion move for the side to move in
// fen, at the given difficulty. It always takes an immediate checkmate if
// one is available, regardless of difficulty.
func ChooseChessMove(fen string, difficulty Difficulty) (from, to, promo string, err error) {
	opt, err := nchess.FEN(fen)
	if err != nil {
		return "", "", "", err
	}
	game := nchess.NewGame(opt)

	moves := game.ValidMoves()
	if len(moves) == 0 {
		return "", "", "", ErrNoLegalMoves
	}

	turn := game.Position().Turn()
	for _, mv := range moves {
		clone := game.Clone()
		if err := clone.Move(mv); err != nil {
			continue
		}
		if isWinFor(clone.Outcome(), turn) {
			return squareStr(mv.S1()), squareStr(mv.S2()), promoStr(mv.Promo()), nil
		}
	}

	var chosen *nchess.Move
	if difficulty == Easy {
		chosen = chooseEasyMove(game, moves)
	} else {
		chosen = chooseHardMove(game, moves)
	}

	return squareStr(chosen.S1()), squareStr(chosen.S2()), promoStr(chosen.Promo()), nil
}
