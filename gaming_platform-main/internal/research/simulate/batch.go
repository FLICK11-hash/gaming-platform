package simulate

import (
	"math/rand"
	"sort"

	"gaming_platform/internal/domain/bot"
)

// chessOpeningBook is a fixed set of well-known, legal first moves for
// White. Both chooseHardMove and chooseEasyMove are otherwise deterministic
// enough (chooseHardMove entirely so; chooseEasyMove only occasionally not)
// that letting the bot pick its own opening every game would barely vary
// it - sampling from here instead is what actually lets a batch explore
// different opening lines.
var chessOpeningBook = []string{
	"e2e4", // King's Pawn
	"d2d4", // Queen's Pawn
	"g1f3", // Reti
	"c2c4", // English
	"e2e3", // Van 't Kruijs
	"d2d3", // King's Indian Attack setup
	"b1c3", // Van Geet
	"g2g3", // King's Fianchetto
	"f2f4", // Bird's Opening
	"b2b3", // Nimzo-Larsen
}

// OpeningStat aggregates every game played that opened with a given move.
type OpeningStat struct {
	Games          int     `json:"games"`
	FirstMoverWins int     `json:"first_mover_wins"`
	Draws          int     `json:"draws"`
	WinRate        float64 `json:"win_rate"`
}

// BatchStats aggregates a batch of simulated games.
type BatchStats struct {
	Games              int                    `json:"games"`
	FirstMoverWins     int                    `json:"first_mover_wins"`
	SecondMoverWins    int                    `json:"second_mover_wins"`
	Draws              int                    `json:"draws"`
	FirstMoverWinRate  float64                `json:"first_mover_win_rate"`
	SecondMoverWinRate float64                `json:"second_mover_win_rate"`
	DrawRate           float64                `json:"draw_rate"`
	AveragePlies       float64                `json:"average_plies"`
	ByOpening          map[string]OpeningStat `json:"by_opening,omitempty"`
}

func (b *BatchStats) record(firstWon, secondWon bool, plies int, opening string) {
	draw := !firstWon && !secondWon
	b.Games++
	b.AveragePlies += float64(plies)
	switch {
	case firstWon:
		b.FirstMoverWins++
	case secondWon:
		b.SecondMoverWins++
	default:
		b.Draws++
	}

	if opening == "" {
		return
	}
	if b.ByOpening == nil {
		b.ByOpening = make(map[string]OpeningStat)
	}
	stat := b.ByOpening[opening]
	stat.Games++
	if firstWon {
		stat.FirstMoverWins++
	}
	if draw {
		stat.Draws++
	}
	b.ByOpening[opening] = stat
}

func (b *BatchStats) finalize() {
	if b.Games == 0 {
		return
	}
	b.FirstMoverWinRate = float64(b.FirstMoverWins) / float64(b.Games)
	b.SecondMoverWinRate = float64(b.SecondMoverWins) / float64(b.Games)
	b.DrawRate = float64(b.Draws) / float64(b.Games)
	b.AveragePlies /= float64(b.Games)

	for opening, stat := range b.ByOpening {
		stat.WinRate = float64(stat.FirstMoverWins) / float64(stat.Games)
		b.ByOpening[opening] = stat
	}
}

// RunChessBatch plays n chess games at the given difficulty (both sides
// play at the same strength, isolating move-order as the only asymmetry)
// and aggregates the results. Which side moves first is randomized every
// game. If randomizeOpening is true, the first mover's opening move is
// sampled from chessOpeningBook instead of chosen by its own AI, so the
// results can be broken down per opening; if false, the bot picks its own
// (much less varied) opening every game.
func RunChessBatch(n int, difficulty bot.Difficulty, randomizeOpening bool) (BatchStats, error) {
	var stats BatchStats
	for i := 0; i < n; i++ {
		opening := ""
		if randomizeOpening {
			opening = chessOpeningBook[rand.Intn(len(chessOpeningBook))]
		}
		result, err := PlayChessGame(difficulty, difficulty, opening)
		if err != nil {
			return BatchStats{}, err
		}
		stats.record(result.FirstMoverWon, result.SecondMoverWon, result.Plies, result.OpeningMove)
	}
	stats.finalize()
	return stats, nil
}

// RunConnectFourBatch mirrors RunChessBatch for Connect Four: n games at a
// single difficulty for both sides, first mover randomized every game, and
// (if randomizeOpening) the first mover's opening column sampled uniformly
// from all 7 columns instead of chosen by its own AI.
func RunConnectFourBatch(n int, difficulty bot.Difficulty, randomizeOpening bool) (BatchStats, error) {
	var stats BatchStats
	for i := 0; i < n; i++ {
		column := -1
		if randomizeOpening {
			column = rand.Intn(connectFourColumns)
		}
		result, err := PlayConnectFourGame(difficulty, difficulty, column)
		if err != nil {
			return BatchStats{}, err
		}
		opening := ""
		if randomizeOpening {
			opening = columnLabel(result.OpeningColumn)
		}
		stats.record(result.FirstMoverWon, result.SecondMoverWon, result.Plies, opening)
	}
	stats.finalize()
	return stats, nil
}

const connectFourColumns = 7

func columnLabel(col int) string {
	return "column " + string(rune('1'+col))
}

// SortedOpenings returns b's per-opening stats sorted by descending game
// count, so a report always lists the most-played openings first
// regardless of Go's randomized map iteration order.
func (b BatchStats) SortedOpenings() []struct {
	Opening string
	Stat    OpeningStat
} {
	out := make([]struct {
		Opening string
		Stat    OpeningStat
	}, 0, len(b.ByOpening))
	for opening, stat := range b.ByOpening {
		out = append(out, struct {
			Opening string
			Stat    OpeningStat
		}{opening, stat})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Stat.Games != out[j].Stat.Games {
			return out[i].Stat.Games > out[j].Stat.Games
		}
		return out[i].Opening < out[j].Opening
	})
	return out
}
