package match

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
	"gaming_platform/internal/domain/achievements"
	"gaming_platform/internal/domain/hub"
	"gaming_platform/internal/domain/leaderboard"
	"gaming_platform/internal/domain/matchhistory"
	"gaming_platform/internal/domain/missions"
)

// TestFinishMatch_ConcurrentCallsPayOutOnce is a regression test for a real
// bug: finishMatch used to check the match's status, then write the
// completed result as two separate steps, so two concurrent triggers for
// the same match (e.g. a move delivering checkmate racing a clock-watcher
// timeout, or a resign racing a disconnect forfeit) could both observe
// "still active" and both run Elo updates and prize payout. CompleteMatch
// now only affects a row still in status 'active', so only one of two
// concurrent finishMatch calls for the same match can ever complete it -
// this proves the winner is credited their prize exactly once, not twice.
func TestFinishMatch_ConcurrentCallsPayOutOnce(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping concurrency regression test")
	}

	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	q := db.New(pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	winner, err := q.CreateUser(ctx, db.CreateUserParams{Username: "concurrency_winner_" + suffix, Email: "cw_" + suffix + "@example.com", Password: "x"})
	if err != nil {
		t.Fatalf("create winner: %v", err)
	}
	loser, err := q.CreateUser(ctx, db.CreateUserParams{Username: "concurrency_loser_" + suffix, Email: "cl_" + suffix + "@example.com", Password: "x"})
	if err != nil {
		t.Fatalf("create loser: %v", err)
	}

	const coinWager = int64(200)
	const winnerPrize = coinWager * 2

	tour, err := q.CreateTournament(ctx, db.CreateTournamentParams{
		GameType:        "chess",
		GameMode:        "one_vs_one",
		Status:          "waiting",
		CoinWager:       coinWager,
		TotalCoinPool:   winnerPrize,
		WinnerCoinPrize: winnerPrize,
		BonusCoins:      0,
	})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	for i, uid := range []int64{winner.UserID, loser.UserID} {
		if _, err := q.CreateTournamentPlayer(ctx, db.CreateTournamentPlayerParams{
			TournamentID: tour.TournamentID,
			UserID:       uid,
			SeedNumber:   dbutil.IntToPGInt4(i + 1),
		}); err != nil {
			t.Fatalf("create tournament player: %v", err)
		}
	}

	m, err := q.CreateMatch(ctx, db.CreateMatchParams{
		TournamentID: tour.TournamentID,
		GameType:     "chess",
		Round:        "final",
		MatchNumber:  dbutil.IntToPGInt4(1),
		Player1ID:    winner.UserID,
		Player2ID:    loser.UserID,
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}
	if _, err := q.ActivateMatch(ctx, db.ActivateMatchParams{
		MatchID:           m.MatchID,
		CurrentTurnUserID: dbutil.Int64ToPGInt8(winner.UserID),
	}); err != nil {
		t.Fatalf("activate match: %v", err)
	}

	startingBalance := winner.CoinBalance

	svc := NewService(pool, hub.New(), missions.NewService(pool), leaderboard.NewService(pool), matchhistory.NewService(pool), achievements.NewService(pool))

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_ = svc.finishMatch(ctx, m.MatchID, winner.UserID, "")
		}()
	}
	close(start)
	wg.Wait()

	finalWinner, err := q.GetUserByID(ctx, winner.UserID)
	if err != nil {
		t.Fatalf("get winner: %v", err)
	}

	gotDelta := finalWinner.CoinBalance - startingBalance
	if gotDelta != winnerPrize {
		t.Fatalf("winner coin balance changed by %d, want exactly %d (prize paid out %d times)", gotDelta, winnerPrize, gotDelta/winnerPrize)
	}

	gotWins := dbutil.PGInt4ToInt(finalWinner.TotalWins) - dbutil.PGInt4ToInt(winner.TotalWins)
	if gotWins != 1 {
		t.Fatalf("winner total_wins increased by %d, want exactly 1", gotWins)
	}

	finalMatch, err := q.GetMatch(ctx, m.MatchID)
	if err != nil {
		t.Fatalf("get match: %v", err)
	}
	if finalMatch.Status.String != "completed" {
		t.Fatalf("match status = %q, want completed", finalMatch.Status.String)
	}
}
