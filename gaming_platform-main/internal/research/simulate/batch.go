package simulate

import (
	"math/rand"
	"sort"

	"gaming_platform/internal/domain/bot"
)

// OpeningStat aggregates every game played that opened with a given move.
type OpeningStat struct {
	Games          int     `json:"games"`
	FirstMoverWins int     `json:"first_mover_wins"`
	Draws          int     `json:"draws"`
	WinRate        float64 `json:"win_rate"`
}

// BatchStats aggregates a batch of simulated games in which both sides play
// at the same difficulty, so the only asymmetry being measured is who moves
// first: White in chess, Red in Connect Four (both games fix that by rule -
// there's nothing to randomize there, only to measure).
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

// RunChessBatch plays n chess games at the given difficulty and aggregates
// the results from White's perspective (White always moves first by rule).
// If randomizeOpening is true, White's opening move is sampled from
// WhiteChessOpenings instead of chosen by its own AI, so the results can be
// broken down per opening; if false, the bot picks its own (much less
// varied) opening every game.
func RunChessBatch(n int, difficulty bot.Difficulty, randomizeOpening bool) (BatchStats, error) {
	var stats BatchStats
	for i := 0; i < n; i++ {
		var forced []string
		if randomizeOpening {
			forced = []string{WhiteChessOpenings[rand.Intn(len(WhiteChessOpenings))]}
		}
		result, err := PlayChessGame(difficulty, difficulty, forced)
		if err != nil {
			return BatchStats{}, err
		}
		opening := ""
		if len(result.MovesPlayed) > 0 {
			opening = result.MovesPlayed[0]
		}
		stats.record(result.WhiteWon, result.BlackWon, result.Plies, opening)
	}
	stats.finalize()
	return stats, nil
}

// RunConnectFourBatch mirrors RunChessBatch for Connect Four: n games at a
// single difficulty for both sides, aggregated from Red's perspective (Red
// always moves first by rule), and (if randomizeOpening) Red's opening
// column sampled uniformly from all 7 columns instead of chosen by its own
// AI.
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
		stats.record(result.RedWon, result.GreenWon, result.Plies, opening)
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

// OpeningColor selects which side of the board RunChessOpeningBook tests -
// White's opening choices, or Black's replies to BlackOpeningReplyTo.
type OpeningColor int

const (
	White OpeningColor = iota
	Black
)

// ChessOpeningReport aggregates every game played testing one specific
// opening (for White) or reply (for Black), from that color's perspective.
type ChessOpeningReport struct {
	Opening      string  `json:"opening"`
	Games        int     `json:"games"`
	Wins         int     `json:"wins"`
	Losses       int     `json:"losses"`
	Draws        int     `json:"draws"`
	WinRate      float64 `json:"win_rate"`
	AveragePlies float64 `json:"average_plies"`
}

// RunChessOpeningBook plays gamesPerOpening games for every opening in
// color's book (WhiteChessOpenings or BlackChessOpenings), always seating
// the bot under study on that color - a "defense" only means something
// played as Black, the same way an "opening" only means something played
// as White. Both bots play at difficulty.
func RunChessOpeningBook(gamesPerOpening int, difficulty bot.Difficulty, color OpeningColor) ([]ChessOpeningReport, error) {
	book := WhiteChessOpenings
	if color == Black {
		book = BlackChessOpenings
	}

	reports := make([]ChessOpeningReport, 0, len(book))
	for _, opening := range book {
		forced := []string{opening}
		if color == Black {
			forced = []string{BlackOpeningReplyTo, opening}
		}

		report := ChessOpeningReport{Opening: opening}
		for i := 0; i < gamesPerOpening; i++ {
			result, err := PlayChessGame(difficulty, difficulty, forced)
			if err != nil {
				return nil, err
			}
			report.Games++
			report.AveragePlies += float64(result.Plies)

			won := result.WhiteWon
			if color == Black {
				won = result.BlackWon
			}
			switch {
			case won:
				report.Wins++
			case result.Draw:
				report.Draws++
			default:
				report.Losses++
			}
		}
		if report.Games > 0 {
			report.WinRate = float64(report.Wins) / float64(report.Games)
			report.AveragePlies /= float64(report.Games)
		}
		reports = append(reports, report)
	}
	return reports, nil
}
