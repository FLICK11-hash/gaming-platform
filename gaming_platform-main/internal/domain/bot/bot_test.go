package bot

import (
	"testing"

	nchess "github.com/notnil/chess"

	"gaming_platform/internal/domain/connectfour"
)

func TestChooseChessMove_TakesForcedMate(t *testing.T) {
	// White to move, back-rank mate in one: Ra1-a8#.
	const fen = "6k1/5ppp/8/8/8/8/8/R5K1 w - - 0 1"

	for _, difficulty := range []Difficulty{Easy, Hard} {
		from, to, _, err := ChooseChessMove(fen, difficulty)
		if err != nil {
			t.Fatalf("difficulty %v: unexpected error: %v", difficulty, err)
		}
		if from != "a1" || to != "a8" {
			t.Fatalf("difficulty %v: expected a1a8 mate, got %s%s", difficulty, from, to)
		}

		opt, err := nchess.FEN(fen)
		if err != nil {
			t.Fatalf("FEN: %v", err)
		}
		game := nchess.NewGame(opt)
		applyMove(t, game, from, to)
		if game.Outcome() != nchess.WhiteWon {
			t.Fatalf("difficulty %v: expected white to have won, outcome was %v", difficulty, game.Outcome())
		}
	}
}

func TestChooseChessMove_ReturnsLegalMove(t *testing.T) {
	game := nchess.NewGame()
	for _, difficulty := range []Difficulty{Easy, Hard} {
		from, to, promo, err := ChooseChessMove(game.FEN(), difficulty)
		if err != nil {
			t.Fatalf("difficulty %v: unexpected error: %v", difficulty, err)
		}
		if from == "" || to == "" {
			t.Fatalf("difficulty %v: expected a non-empty move, got %q %q %q", difficulty, from, to, promo)
		}
	}
}

func applyMove(t *testing.T, game *nchess.Game, from, to string) {
	t.Helper()
	for _, mv := range game.ValidMoves() {
		if mv.S1().String() == from && mv.S2().String() == to {
			if err := game.Move(mv); err != nil {
				t.Fatalf("applying %s%s: %v", from, to, err)
			}
			return
		}
	}
	t.Fatalf("move %s%s not found among legal moves", from, to)
}

func TestChooseConnectFourMove_TakesImmediateWin(t *testing.T) {
	var board connectfour.Board
	// Red has three in a row vertically in column 0; the 4th completes it.
	board[5][0] = int(connectfour.Red)
	board[4][0] = int(connectfour.Red)
	board[3][0] = int(connectfour.Red)
	board[5][1] = int(connectfour.Green)
	board[5][2] = int(connectfour.Green)

	for _, difficulty := range []Difficulty{Easy, Hard} {
		col, err := ChooseConnectFourMove(board, connectfour.Red, difficulty)
		if err != nil {
			t.Fatalf("difficulty %v: unexpected error: %v", difficulty, err)
		}
		if col != 0 {
			t.Fatalf("difficulty %v: expected column 0 to win, got %d", difficulty, col)
		}
	}
}

func TestChooseConnectFourMove_HardBlocksForcedLoss(t *testing.T) {
	var board connectfour.Board
	// Green has three in a row vertically in column 3; Red (to move) must
	// block at row 2, column 3 or lose next turn.
	board[5][3] = int(connectfour.Green)
	board[4][3] = int(connectfour.Green)
	board[3][3] = int(connectfour.Green)
	board[5][0] = int(connectfour.Red)
	board[5][1] = int(connectfour.Red)

	col, err := ChooseConnectFourMove(board, connectfour.Red, Hard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if col != 3 {
		t.Fatalf("expected the hard bot to block column 3, got %d", col)
	}
}

func TestChooseConnectFourMove_ReturnsLegalColumn(t *testing.T) {
	var board connectfour.Board
	for _, difficulty := range []Difficulty{Easy, Hard} {
		col, err := ChooseConnectFourMove(board, connectfour.Red, difficulty)
		if err != nil {
			t.Fatalf("difficulty %v: unexpected error: %v", difficulty, err)
		}
		if col < 0 || col >= connectfour.Columns {
			t.Fatalf("difficulty %v: column %d out of range", difficulty, col)
		}
	}
}

func TestChooseConnectFourMove_EasyUsuallyBlocks(t *testing.T) {
	var board connectfour.Board
	board[5][3] = int(connectfour.Green)
	board[4][3] = int(connectfour.Green)
	board[3][3] = int(connectfour.Green)
	board[5][0] = int(connectfour.Red)
	board[5][1] = int(connectfour.Red)

	blocks := 0
	const trials = 200
	for i := 0; i < trials; i++ {
		col, err := ChooseConnectFourMove(board, connectfour.Red, Easy)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if col == 3 {
			blocks++
		}
	}
	if blocks < trials/2 {
		t.Fatalf("expected Easy to block a forced loss most of the time, only blocked %d/%d", blocks, trials)
	}
}

func TestDifficultyForElo(t *testing.T) {
	if DifficultyForElo(1200) != Easy {
		t.Fatalf("expected 1200 elo to be Easy")
	}
	if DifficultyForElo(2100) != Hard {
		t.Fatalf("expected 2100 elo to be Hard")
	}
}

func TestUsernameForOpponentElo(t *testing.T) {
	if got := UsernameForOpponentElo(1400); got != RookieBotUsername {
		t.Fatalf("expected RookieBot for elo below threshold, got %s", got)
	}
	if got := UsernameForOpponentElo(1500); got != GrandmasterBotUsername {
		t.Fatalf("expected GrandmasterBot for elo at threshold, got %s", got)
	}
}
