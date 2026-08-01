package bot

import (
	"errors"
	"math/rand"

	"gaming_platform/internal/domain/connectfour"
)

// ErrNoLegalColumns means every column is full, which should never happen
// for an in-progress match (the game would already be a draw).
var ErrNoLegalColumns = errors.New("bot: no legal columns")

// c4WinScore is a magic score large enough to dominate any heuristic value.
const c4WinScore = 1_000_000

func other(c connectfour.Color) connectfour.Color {
	if c == connectfour.Red {
		return connectfour.Green
	}
	return connectfour.Red
}

// dropRow returns the row a chip dropped into col would land on, or -1 if
// the column is full.
func dropRow(board connectfour.Board, col int) int {
	for r := connectfour.Rows - 1; r >= 0; r-- {
		if board[r][col] == 0 {
			return r
		}
	}
	return -1
}

func legalColumns(board connectfour.Board) []int {
	var cols []int
	for c := 0; c < connectfour.Columns; c++ {
		if board[0][c] == 0 {
			cols = append(cols, c)
		}
	}
	return cols
}

// findWinningColumn reports the first column (if any) where dropping a
// color chip would immediately win.
func findWinningColumn(board connectfour.Board, color connectfour.Color, cols []int) (int, bool) {
	for _, c := range cols {
		row := dropRow(board, c)
		b := board
		b[row][c] = int(color)
		if connectfour.CheckWin(b, row, c, color) {
			return c, true
		}
	}
	return 0, false
}

// centerWeights favors central columns, which participate in more possible
// four-in-a-row lines than the edges.
var centerWeights = [connectfour.Columns]int{1, 2, 3, 4, 3, 2, 1}

// heuristic scores board from color's own perspective: positive is good for
// color, negative is good for its opponent.
func heuristic(board connectfour.Board, color connectfour.Color) int {
	opp := other(color)
	score := 0
	for r := 0; r < connectfour.Rows; r++ {
		for c := 0; c < connectfour.Columns; c++ {
			switch board[r][c] {
			case int(color):
				score += centerWeights[c]
			case int(opp):
				score -= centerWeights[c]
			}
		}
	}
	return score
}

// negamax returns the best score toMove can achieve from its own
// perspective, searching down to depth plies (0 = evaluate immediately via
// heuristic). alpha/beta are the standard alpha-beta bounds: any branch
// once found to be at least as good for the opponent as beta is cut short,
// since toMove's opponent would never let play reach it - this returns the
// exact same score minimax (full-width, no pruning) would, just without
// evaluating the branches that can't change the outcome.
func negamax(board connectfour.Board, toMove connectfour.Color, depth, alpha, beta int) int {
	cols := legalColumns(board)
	if len(cols) == 0 {
		return 0 // board full: a draw
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
			score = -negamax(b, other(toMove), depth-1, -beta, -alpha)
		}
		if score > best {
			best = score
		}
		if best > alpha {
			alpha = best
		}
		if alpha >= beta {
			break // beta cutoff: the opponent already has a better option elsewhere
		}
	}
	return best
}

// chooseEasyC4Move blocks the opponent's immediate win almost always, and
// otherwise greedily plays the best single-ply center-weighted column
// instead of a purely random one, so it develops toward the center like a
// real (if short-sighted) player rather than scattering chips at random. It
// still blunders occasionally (easyBlunderChance) with a random column.
func chooseEasyC4Move(board connectfour.Board, turn connectfour.Color, cols []int) int {
	if blockCol, ok := findWinningColumn(board, other(turn), cols); ok && rand.Float64() > easyBlunderChance {
		return blockCol
	}

	if rand.Float64() < easyBlunderChance {
		return cols[rand.Intn(len(cols))]
	}

	bestCol := cols[0]
	bestScore := -c4WinScore * 2
	for _, c := range cols {
		row := dropRow(board, c)
		b := board
		b[row][c] = int(turn)
		score := heuristic(b, turn)
		if connectfour.CheckWin(b, row, c, turn) {
			score = c4WinScore
		}
		if score > bestScore {
			bestScore = score
			bestCol = c
		}
	}
	return bestCol
}

// chooseHardC4Move searches several plies ahead (own move + reply, repeated)
// with a center-column heuristic at the leaves, so it reliably takes forced
// wins and blocks forced losses a few moves out, not just the very next one.
func chooseHardC4Move(board connectfour.Board, turn connectfour.Color, cols []int) int {
	const depth = 5
	alpha := -c4WinScore * 2

	bestCol := cols[0]
	bestScore := -c4WinScore * 2
	for _, c := range cols {
		row := dropRow(board, c)
		b := board
		b[row][c] = int(turn)

		var score int
		if connectfour.CheckWin(b, row, c, turn) {
			score = c4WinScore
		} else {
			score = -negamax(b, other(turn), depth-1, -c4WinScore*2, -alpha)
		}
		if score > bestScore {
			bestScore = score
			bestCol = c
			if score > alpha {
				alpha = score
			}
		}
	}
	return bestCol
}

// ChooseConnectFourMove picks a column for turn to drop into, at the given
// difficulty. It always takes an immediate win if one is available,
// regardless of difficulty.
func ChooseConnectFourMove(board connectfour.Board, turn connectfour.Color, difficulty Difficulty) (int, error) {
	cols := legalColumns(board)
	if len(cols) == 0 {
		return 0, ErrNoLegalColumns
	}

	if winCol, ok := findWinningColumn(board, turn, cols); ok {
		return winCol, nil
	}

	if difficulty == Easy {
		return chooseEasyC4Move(board, turn, cols), nil
	}
	return chooseHardC4Move(board, turn, cols), nil
}
