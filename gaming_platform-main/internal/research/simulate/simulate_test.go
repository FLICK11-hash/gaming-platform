package simulate

import (
	"testing"

	"gaming_platform/internal/domain/bot"
)

func TestPlayChessGame_TerminatesWithExactlyOneOutcome(t *testing.T) {
	result, err := PlayChessGame(bot.Easy, bot.Easy, "")
	if err != nil {
		t.Fatalf("PlayChessGame: %v", err)
	}
	if result.Plies == 0 {
		t.Fatalf("game ended after 0 plies")
	}

	outcomes := 0
	for _, v := range []bool{result.FirstMoverWon, result.SecondMoverWon, result.Draw} {
		if v {
			outcomes++
		}
	}
	if outcomes != 1 {
		t.Fatalf("expected exactly one outcome flag set, got FirstMoverWon=%v SecondMoverWon=%v Draw=%v",
			result.FirstMoverWon, result.SecondMoverWon, result.Draw)
	}
}

func TestPlayChessGame_ForcedOpeningIsPlayedFirst(t *testing.T) {
	result, err := PlayChessGame(bot.Easy, bot.Easy, "d2d4")
	if err != nil {
		t.Fatalf("PlayChessGame: %v", err)
	}
	if result.OpeningMove != "d2d4" {
		t.Fatalf("OpeningMove = %q, want %q", result.OpeningMove, "d2d4")
	}
}

func TestPlayConnectFourGame_TerminatesWithExactlyOneOutcome(t *testing.T) {
	result, err := PlayConnectFourGame(bot.Easy, bot.Easy, -1)
	if err != nil {
		t.Fatalf("PlayConnectFourGame: %v", err)
	}
	if result.Plies == 0 {
		t.Fatalf("game ended after 0 plies")
	}

	outcomes := 0
	for _, v := range []bool{result.FirstMoverWon, result.SecondMoverWon, result.Draw} {
		if v {
			outcomes++
		}
	}
	if outcomes != 1 {
		t.Fatalf("expected exactly one outcome flag set, got FirstMoverWon=%v SecondMoverWon=%v Draw=%v",
			result.FirstMoverWon, result.SecondMoverWon, result.Draw)
	}
}

func TestPlayConnectFourGame_ForcedOpeningColumnIsPlayedFirst(t *testing.T) {
	result, err := PlayConnectFourGame(bot.Easy, bot.Easy, 3)
	if err != nil {
		t.Fatalf("PlayConnectFourGame: %v", err)
	}
	if result.OpeningColumn != 3 {
		t.Fatalf("OpeningColumn = %d, want 3", result.OpeningColumn)
	}
}

func TestRunChessBatch_AggregatesConsistently(t *testing.T) {
	const n = 20
	stats, err := RunChessBatch(n, bot.Easy, true)
	if err != nil {
		t.Fatalf("RunChessBatch: %v", err)
	}
	if stats.Games != n {
		t.Fatalf("Games = %d, want %d", stats.Games, n)
	}
	if got := stats.FirstMoverWins + stats.SecondMoverWins + stats.Draws; got != n {
		t.Fatalf("FirstMoverWins+SecondMoverWins+Draws = %d, want %d", got, n)
	}

	openingGames := 0
	for _, o := range stats.ByOpening {
		openingGames += o.Games
	}
	if openingGames != n {
		t.Fatalf("sum of per-opening games = %d, want %d", openingGames, n)
	}
}

func TestRunConnectFourBatch_AggregatesConsistently(t *testing.T) {
	const n = 20
	stats, err := RunConnectFourBatch(n, bot.Easy, true)
	if err != nil {
		t.Fatalf("RunConnectFourBatch: %v", err)
	}
	if stats.Games != n {
		t.Fatalf("Games = %d, want %d", stats.Games, n)
	}
	if got := stats.FirstMoverWins + stats.SecondMoverWins + stats.Draws; got != n {
		t.Fatalf("FirstMoverWins+SecondMoverWins+Draws = %d, want %d", got, n)
	}
}

func TestSortedOpenings_OrderedByGameCountDescending(t *testing.T) {
	stats := BatchStats{
		ByOpening: map[string]OpeningStat{
			"e2e4": {Games: 5, FirstMoverWins: 3},
			"d2d4": {Games: 9, FirstMoverWins: 4},
			"c2c4": {Games: 9, FirstMoverWins: 1},
		},
	}
	sorted := stats.SortedOpenings()
	if len(sorted) != 3 {
		t.Fatalf("len(sorted) = %d, want 3", len(sorted))
	}
	if sorted[0].Stat.Games != 9 || sorted[1].Stat.Games != 9 {
		t.Fatalf("expected the two 9-game openings first, got %+v", sorted)
	}
	// Tied game counts break by opening name ascending.
	if sorted[0].Opening != "c2c4" || sorted[1].Opening != "d2d4" {
		t.Fatalf("expected tie-break by name (c2c4 before d2d4), got %q then %q", sorted[0].Opening, sorted[1].Opening)
	}
	if sorted[2].Opening != "e2e4" {
		t.Fatalf("expected e2e4 last, got %q", sorted[2].Opening)
	}
}
