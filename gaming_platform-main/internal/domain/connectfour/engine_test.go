package connectfour

import "testing"

func TestDropChip_HorizontalWin(t *testing.T) {
	e := NewEngine()
	e.StartGame(1)

	// Red: 0,1,2,3 (bottom row); Green plays elsewhere in between.
	moves := []struct {
		col   int
		color Color
	}{
		{0, Red}, {0, Green},
		{1, Red}, {1, Green},
		{2, Red}, {2, Green},
	}
	for _, m := range moves {
		if _, err := e.DropChip(1, m.col, m.color); err != nil {
			t.Fatalf("DropChip(%d, %v) failed: %v", m.col, m.color, err)
		}
	}

	result, err := e.DropChip(1, 3, Red)
	if err != nil {
		t.Fatalf("winning move failed: %v", err)
	}
	if result.Outcome != RedWon {
		t.Errorf("Outcome = %v, want RedWon", result.Outcome)
	}
}

func TestDropChip_VerticalWin(t *testing.T) {
	e := NewEngine()
	e.StartGame(1)

	moves := []struct {
		col   int
		color Color
	}{
		{0, Red}, {1, Green},
		{0, Red}, {1, Green},
		{0, Red}, {1, Green},
	}
	for _, m := range moves {
		if _, err := e.DropChip(1, m.col, m.color); err != nil {
			t.Fatalf("DropChip(%d, %v) failed: %v", m.col, m.color, err)
		}
	}

	result, err := e.DropChip(1, 0, Red)
	if err != nil {
		t.Fatalf("winning move failed: %v", err)
	}
	if result.Outcome != RedWon {
		t.Errorf("Outcome = %v, want RedWon", result.Outcome)
	}
}

func TestDropChip_DiagonalWin(t *testing.T) {
	e := NewEngine()
	e.StartGame(1)

	// Builds an ascending diagonal of Red at (5,0),(4,1),(3,2),(2,3) (rows
	// counted from the top, so this is a "/" diagonal from the bottom-left
	// going up-right). col6 absorbs one harmless Red move so the turn
	// count works out to land the winning move on Red.
	moves := []struct {
		col   int
		color Color
	}{
		{6, Red},          // harmless
		{1, Green},        // col1 filler
		{1, Red},          // target (4,1)
		{2, Green},        // col2 filler
		{0, Red},          // target (5,0)
		{2, Green},        // col2 filler
		{2, Red},          // target (3,2)
		{3, Green},        // col3 filler
		{3, Red},          // col3 filler (not on the diagonal)
		{3, Green},        // col3 filler
	}
	for _, m := range moves {
		if _, err := e.DropChip(1, m.col, m.color); err != nil {
			t.Fatalf("DropChip(%d, %v) failed: %v", m.col, m.color, err)
		}
	}

	result, err := e.DropChip(1, 3, Red) // target (2,3): completes the diagonal
	if err != nil {
		t.Fatalf("winning move failed: %v", err)
	}
	if result.Outcome != RedWon {
		t.Errorf("Outcome = %v, want RedWon", result.Outcome)
	}
}

func TestDropChip_NotYourTurn(t *testing.T) {
	e := NewEngine()
	e.StartGame(1)

	if _, err := e.DropChip(1, 0, Green); err != ErrNotYourTurn {
		t.Errorf("err = %v, want ErrNotYourTurn", err)
	}
}

func TestDropChip_ColumnFull(t *testing.T) {
	e := NewEngine()
	e.StartGame(1)

	for i := 0; i < Rows; i++ {
		color := Red
		if i%2 == 1 {
			color = Green
		}
		if _, err := e.DropChip(1, 0, color); err != nil {
			t.Fatalf("DropChip #%d failed: %v", i, err)
		}
	}

	if _, err := e.DropChip(1, 0, Red); err != ErrColumnFull {
		t.Errorf("err = %v, want ErrColumnFull", err)
	}
}

// TestDropChip_DrawOnFullBoard searches (via backtracking, using the real
// engine) for some legal sequence of drops that fills the board without
// either color ever completing four in a row, then asserts it ends in Draw.
// The board is small enough that this search is effectively instant, and it
// avoids depending on a hand-derived fill pattern being correct.
func TestDropChip_DrawOnFullBoard(t *testing.T) {
	sequence := findNonWinningFillSequence(t)

	e := NewEngine()
	e.StartGame(1)
	turn := Red
	var last MoveResult
	for i, col := range sequence {
		result, err := e.DropChip(1, col, turn)
		if err != nil {
			t.Fatalf("replaying move %d (col %d) failed: %v", i, col, err)
		}
		last = result
		turn = otherColor(turn)
	}

	if last.Outcome != Draw {
		t.Fatalf("Outcome = %v, want Draw", last.Outcome)
	}
}

func otherColor(c Color) Color {
	if c == Red {
		return Green
	}
	return Red
}

// findNonWinningFillSequence does a complete DFS over column choices to find
// a full 42-move sequence where no move ever completes four in a row. It
// works on a plain [Rows][Columns]int board rather than the real Engine:
// Go arrays are values, so trying a candidate and backtracking is just
// "make a modified copy," with no undo logic needed.
func findNonWinningFillSequence(t *testing.T) []int {
	t.Helper()

	var sequence []int

	var search func(board [Rows][Columns]int, turn Color) bool
	search = func(board [Rows][Columns]int, turn Color) bool {
		if len(sequence) == Rows*Columns {
			return true
		}

		for col := 0; col < Columns; col++ {
			row := -1
			for r := Rows - 1; r >= 0; r-- {
				if board[r][col] == 0 {
					row = r
					break
				}
			}
			if row == -1 {
				continue // column full
			}

			next := board
			next[row][col] = int(turn)
			if checkWin(next, row, col, int(turn)) {
				continue // this column completes a line; not a safe move
			}

			sequence = append(sequence, col)
			if search(next, otherColor(turn)) {
				return true
			}
			sequence = sequence[:len(sequence)-1]
		}

		return false
	}

	if !search([Rows][Columns]int{}, Red) {
		t.Fatal("could not find any non-winning fill sequence for a 7x6 board")
	}

	return sequence
}

func TestState_UnknownMatch(t *testing.T) {
	e := NewEngine()
	if _, err := e.State(999); err != ErrGameNotFound {
		t.Errorf("err = %v, want ErrGameNotFound", err)
	}
}

func TestEndGame_RemovesState(t *testing.T) {
	e := NewEngine()
	e.StartGame(1)
	e.EndGame(1)

	if _, err := e.State(1); err != ErrGameNotFound {
		t.Errorf("err = %v, want ErrGameNotFound after EndGame", err)
	}
}
