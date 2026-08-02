package bot

import (
	"errors"
	"log"
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
// `depth` plies deeper. alpha/beta are the standard alpha-beta bounds: once
// a branch is proven at least as good for one side as the bound the other
// side can already force elsewhere, the rest of that branch is skipped -
// this returns the exact same score a full-width search would, just without
// evaluating the moves that can't change it.
func minimax(g *nchess.Game, depth, alpha, beta int) int {
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
			if v := minimax(clone, depth-1, alpha, beta); v > best {
				best = v
			}
			if best > alpha {
				alpha = best
			}
			if alpha >= beta {
				break
			}
		}
		return best
	}

	best := mateScore * 2
	for _, mv := range moves {
		clone := g.Clone()
		_ = clone.Move(mv)
		if v := minimax(clone, depth-1, alpha, beta); v < best {
			best = v
		}
		if best < beta {
			beta = best
		}
		if alpha >= beta {
			break
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

// easyBlunderChance is how often RookieBot ignores its own evaluation and
// plays a purely random legal move instead - enough to keep it clearly
// weaker than GrandmasterBot's full 2-ply search, without being the
// mostly-random opponent it used to be.
const easyBlunderChance = 0.15

// chooseEasyMove greedily grabs whatever material is available one ply
// ahead (i.e. without considering the opponent's reply), so it develops
// sensibly and doesn't hang free pieces sitting in front of it, but it can't
// see far enough to avoid a two-move tactic the way chooseHardMove can. It
// occasionally (easyBlunderChance) plays a random move instead, so it still
// makes real mistakes.
func chooseEasyMove(game *nchess.Game, moves []*nchess.Move) *nchess.Move {
	if rand.Float64() < easyBlunderChance {
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
	alpha, beta := -mateScore*2, mateScore*2

	var best *nchess.Move
	bestScore := 0
	for i, mv := range moves {
		clone := game.Clone()
		_ = clone.Move(mv)
		score := minimax(clone, 1, alpha, beta)
		if i == 0 || betterFor(turn, score, bestScore) {
			bestScore = score
			best = mv
		}
		// Tighten whichever bound is turn's own, the same way the recursive
		// calls do, so later sibling moves at this level get to prune too.
		if turn == nchess.White && bestScore > alpha {
			alpha = bestScore
		} else if turn == nchess.Black && bestScore < beta {
			beta = bestScore
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

	// Medium has no distinct pure-Go evaluation of its own - it shares
	// chooseHardMove's 2-ply search. That's fine here: this fallback path
	// only runs at all when Stockfish is unavailable or errors, so it's a
	// safety net, not the primary way Medium-tier bots play.
	var chosen *nchess.Move
	if difficulty == Easy {
		chosen = chooseEasyMove(game, moves)
	} else {
		chosen = chooseHardMove(game, moves)
	}

	return squareStr(chosen.S1()), squareStr(chosen.S2()), promoStr(chosen.Promo()), nil
}

// easyExtraBlunderChance further weakens RookieBot's Stockfish-backed play
// beyond UCI_Elo's own floor (RookieBotUCIElo, Stockfish's minimum of
// 1320) to approximate its intended ~1200 rating more closely: Stockfish
// can't be told to play weaker than 1320 through UCI_Elo alone, so this
// occasionally overrides its move with a random legal one instead - the
// same mechanism as easyBlunderChance above, layered on top of a real
// engine rather than the hand-rolled evaluation. This is an approximation:
// there's no way to verify the resulting Elo without running it through
// many rated games, which isn't practical here.
const easyExtraBlunderChance = 0.08

// ChooseChessMoveEngine is ChooseChessMove's real-match counterpart: it
// prefers a real Stockfish engine (see ConfigureStockfish) over the
// hand-rolled evaluation ChooseChessMove uses, falling back to it
// automatically if no engine is configured or a call to one fails (a
// crashed or slow-to-start Stockfish process shouldn't be able to stall a
// live match).
func ChooseChessMoveEngine(fen string, difficulty Difficulty) (from, to, promo string, err error) {
	if sharedEngine == nil {
		return ChooseChessMove(fen, difficulty)
	}

	var elo int
	switch difficulty {
	case Medium:
		elo = ExpertBotUCIElo
	case Hard:
		elo = GrandmasterBotUCIElo
	default:
		elo = RookieBotUCIElo
	}

	from, to, promo, err = sharedEngine.BestMove(fen, elo)
	if err != nil {
		log.Printf("bot: stockfish move failed, falling back to built-in logic: %v", err)
		return ChooseChessMove(fen, difficulty)
	}

	if difficulty == Easy && rand.Float64() < easyExtraBlunderChance {
		if opt, ferr := nchess.FEN(fen); ferr == nil {
			if moves := nchess.NewGame(opt).ValidMoves(); len(moves) > 0 {
				mv := moves[rand.Intn(len(moves))]
				return squareStr(mv.S1()), squareStr(mv.S2()), promoStr(mv.Promo()), nil
			}
		}
	}

	return from, to, promo, nil
}
