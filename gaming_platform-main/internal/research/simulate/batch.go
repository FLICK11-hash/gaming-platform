package simulate

import (
	"math/rand"
	"sort"
	"sync"

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

// RunChessBatch plays n chess games with mover playing both sides and
// aggregates the results from White's perspective (White always moves
// first by rule). If randomizeOpening is true, White's opening move is
// sampled from WhiteChessOpenings instead of chosen by mover, so the
// results can be broken down per opening; if false, mover picks its own
// (much less varied) opening every game.
func RunChessBatch(n int, mover ChessMover, randomizeOpening bool) (BatchStats, error) {
	var stats BatchStats
	for i := 0; i < n; i++ {
		var forced []string
		if randomizeOpening {
			forced = []string{WhiteChessOpenings[rand.Intn(len(WhiteChessOpenings))]}
		}
		result, err := PlayChessGame(mover, mover, forced)
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
// as White. mover plays both sides.
func RunChessOpeningBook(gamesPerOpening int, mover ChessMover, color OpeningColor) ([]ChessOpeningReport, error) {
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
			result, err := PlayChessGame(mover, mover, forced)
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

// RunChessOpeningBookConcurrent is RunChessOpeningBook, but plays up to
// concurrency games at once instead of one at a time. A mover backed by a
// pooled real engine (see internal/engine/stockfish) is safe to call this
// way - each call checks out its own engine process for the duration of
// its game - and a search that takes real wall-clock time per move (unlike
// the platform's instant hand-rolled evaluation) makes that concurrency
// worth having: gamesPerOpening x len(book) games run sequentially would
// otherwise take gamesPerOpening x len(book) times as long as one game.
func RunChessOpeningBookConcurrent(gamesPerOpening int, mover ChessMover, color OpeningColor, concurrency int) ([]ChessOpeningReport, error) {
	if concurrency < 1 {
		concurrency = 1
	}

	book := WhiteChessOpenings
	if color == Black {
		book = BlackChessOpenings
	}

	type job struct {
		openingIdx int
		forced     []string
	}
	type outcome struct {
		openingIdx int
		result     ChessGameResult
	}

	jobs := make(chan job, len(book)*gamesPerOpening)
	for i, opening := range book {
		forced := []string{opening}
		if color == Black {
			forced = []string{BlackOpeningReplyTo, opening}
		}
		for g := 0; g < gamesPerOpening; g++ {
			jobs <- job{openingIdx: i, forced: forced}
		}
	}
	close(jobs)

	outcomes := make(chan outcome, len(book)*gamesPerOpening)
	errs := make(chan error, concurrency)
	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				result, err := PlayChessGame(mover, mover, j.forced)
				if err != nil {
					errs <- err
					return
				}
				outcomes <- outcome{j.openingIdx, result}
			}
		}()
	}
	wg.Wait()
	close(outcomes)
	close(errs)

	if err := <-errs; err != nil {
		return nil, err
	}

	reports := make([]ChessOpeningReport, len(book))
	for i, opening := range book {
		reports[i].Opening = opening
	}
	for o := range outcomes {
		r := &reports[o.openingIdx]
		r.Games++
		r.AveragePlies += float64(o.result.Plies)
		won := o.result.WhiteWon
		if color == Black {
			won = o.result.BlackWon
		}
		switch {
		case won:
			r.Wins++
		case o.result.Draw:
			r.Draws++
		default:
			r.Losses++
		}
	}
	for i := range reports {
		if reports[i].Games > 0 {
			reports[i].WinRate = float64(reports[i].Wins) / float64(reports[i].Games)
			reports[i].AveragePlies /= float64(reports[i].Games)
		}
	}
	return reports, nil
}

// ConnectFourOpeningReport aggregates every game played testing one
// specific opening column for Red, who always moves first.
type ConnectFourOpeningReport struct {
	Column       int     `json:"column"`
	Games        int     `json:"games"`
	Wins         int     `json:"wins"`
	Losses       int     `json:"losses"`
	Draws        int     `json:"draws"`
	WinRate      float64 `json:"win_rate"`
	AveragePlies float64 `json:"average_plies"`
}

// RunConnectFourOpeningBook plays gamesPerColumn games for every one of
// Connect Four's 7 opening columns, always seating the bot under study as
// Red (who always moves first). Unlike RunConnectFourBatch, every column
// gets exactly the same number of games instead of a random share of a
// fixed total - the same guarantee RunChessOpeningBook gives each opening -
// so a per-column win rate isn't quietly built on a smaller, noisier
// sample than its neighbors.
func RunConnectFourOpeningBook(gamesPerColumn int, difficulty bot.Difficulty) ([]ConnectFourOpeningReport, error) {
	reports := make([]ConnectFourOpeningReport, 0, connectFourColumns)
	for col := 0; col < connectFourColumns; col++ {
		report := ConnectFourOpeningReport{Column: col}
		for i := 0; i < gamesPerColumn; i++ {
			result, err := PlayConnectFourGame(difficulty, difficulty, col)
			if err != nil {
				return nil, err
			}
			report.Games++
			report.AveragePlies += float64(result.Plies)
			switch {
			case result.RedWon:
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
