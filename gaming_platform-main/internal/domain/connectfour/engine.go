// Package connectfour implements per-match, in-memory Connect Four game
// state: a 7-column by 6-row board, gravity-drop moves, and four-in-a-row
// win detection in all four directions.
package connectfour

import (
	"errors"
	"sync"
)

const (
	Columns = 7
	Rows    = 6
)

// Color identifies a player's chips: Red always moves first (player 1),
// Green moves second (player 2), mirroring chess's White/Black convention.
type Color int

const (
	Red Color = iota + 1
	Green
)

// Outcome describes how (or whether) a game has ended.
type Outcome int

const (
	NoOutcome Outcome = iota
	RedWon
	GreenWon
	Draw
)

var (
	ErrGameNotFound  = errors.New("connectfour: game not found for match")
	ErrGameOver      = errors.New("connectfour: game is already over")
	ErrInvalidColumn = errors.New("connectfour: invalid column")
	ErrColumnFull    = errors.New("connectfour: column is full")
	ErrNotYourTurn   = errors.New("connectfour: not your turn")
)

// Board is Rows tall by Columns wide; Board[0] is the top row. A cell holds
// 0 (empty), or the int value of Red/Green.
type Board [Rows][Columns]int

// MoveResult is a snapshot of a game after StartGame or DropChip.
type MoveResult struct {
	Board   Board
	Turn    Color
	Outcome Outcome
	// WinningCells holds every cell in the connected line that won the game
	// (at least 4, occasionally more), so a client can highlight them. Only
	// populated when Outcome is RedWon or GreenWon.
	WinningCells [][2]int
}

type gameState struct {
	board        Board
	turn         Color
	outcome      Outcome
	winningCells [][2]int
}

// Engine holds one live Connect Four game per active match_id.
type Engine struct {
	mu    sync.Mutex
	games map[int64]*gameState
}

func NewEngine() *Engine {
	return &Engine{games: make(map[int64]*gameState)}
}

// StartGame creates a fresh game for matchID; Red moves first.
func (e *Engine) StartGame(matchID int64) MoveResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	g := &gameState{turn: Red}
	e.games[matchID] = g
	return snapshot(g)
}

// DropChip drops color's chip into column (0-indexed) for matchID.
func (e *Engine) DropChip(matchID int64, column int, color Color) (MoveResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	g, ok := e.games[matchID]
	if !ok {
		return MoveResult{}, ErrGameNotFound
	}
	if g.outcome != NoOutcome {
		return MoveResult{}, ErrGameOver
	}
	if column < 0 || column >= Columns {
		return MoveResult{}, ErrInvalidColumn
	}
	if g.turn != color {
		return MoveResult{}, ErrNotYourTurn
	}

	row := -1
	for r := Rows - 1; r >= 0; r-- {
		if g.board[r][column] == 0 {
			row = r
			break
		}
	}
	if row == -1 {
		return MoveResult{}, ErrColumnFull
	}

	g.board[row][column] = int(color)

	won, winningCells := checkWin(g.board, row, column, int(color))
	switch {
	case won:
		if color == Red {
			g.outcome = RedWon
		} else {
			g.outcome = GreenWon
		}
		g.winningCells = winningCells
	case boardFull(g.board):
		g.outcome = Draw
	default:
		if color == Red {
			g.turn = Green
		} else {
			g.turn = Red
		}
	}

	return snapshot(g), nil
}

// State returns the current state of matchID's game without mutating it.
func (e *Engine) State(matchID int64) (MoveResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	g, ok := e.games[matchID]
	if !ok {
		return MoveResult{}, ErrGameNotFound
	}
	return snapshot(g), nil
}

// EndGame drops the in-memory game once a match is complete.
func (e *Engine) EndGame(matchID int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.games, matchID)
}

func snapshot(g *gameState) MoveResult {
	return MoveResult{Board: g.board, Turn: g.turn, Outcome: g.outcome, WinningCells: g.winningCells}
}

func boardFull(board Board) bool {
	for c := 0; c < Columns; c++ {
		if board[0][c] == 0 {
			return false
		}
	}
	return true
}

// checkWin reports whether the chip just placed at (row, col) completes a
// run of four for color, checking all four axes (horizontal, vertical, and
// both diagonals) by collecting outward from the placed chip in each
// direction along an axis. When it finds a win, it also returns every cell
// in that connected line (in order), so a client can highlight them.
func checkWin(board Board, row, col, color int) (bool, [][2]int) {
	directions := [4][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	for _, d := range directions {
		forward := collectDirection(board, row, col, d[0], d[1], color)
		backward := collectDirection(board, row, col, -d[0], -d[1], color)
		if len(forward)+len(backward)+1 >= 4 {
			cells := make([][2]int, 0, len(forward)+len(backward)+1)
			for i := len(backward) - 1; i >= 0; i-- {
				cells = append(cells, backward[i])
			}
			cells = append(cells, [2]int{row, col})
			cells = append(cells, forward...)
			return true, cells
		}
	}
	return false, nil
}

func collectDirection(board Board, row, col, dRow, dCol, color int) [][2]int {
	var cells [][2]int
	r, c := row+dRow, col+dCol
	for r >= 0 && r < Rows && c >= 0 && c < Columns && board[r][c] == color {
		cells = append(cells, [2]int{r, c})
		r += dRow
		c += dCol
	}
	return cells
}
