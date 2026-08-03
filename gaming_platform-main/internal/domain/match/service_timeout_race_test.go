package match

import (
	"fmt"
	"os"
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

// TestHandleMove_ExpiredDeadlineFinishesMatch is a regression test for a
// real bug: HandleMove detected an already-expired deadline via
// popIfExpired (which deleted the map entry) and then called completeMatch,
// which tried to look the same entry up a second time to find out who to
// award the win to - but it was already gone, so completeMatch silently
// returned nil without ever finishing the match. A move (including an
// auto-fired pre-move) arriving even slightly after the mover's clock
// actually expired would vanish into this gap: no match_over broadcast, no
// defeat screen, and the match just sat there. popExpiredDeadline now pops
// and returns the deadline in one step, so whichever caller wins the race
// gets it and finishes the match - this proves a move submitted after the
// deadline has passed correctly loses on time instead of being silently
// dropped.
func TestHandleMove_ExpiredDeadlineFinishesMatch(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping timeout-race regression test")
	}

	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	q := db.New(pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	mover, err := q.CreateUser(ctx, db.CreateUserParams{Username: "timeout_mover_" + suffix, Email: "tm_" + suffix + "@example.com", Password: "x"})
	if err != nil {
		t.Fatalf("create mover: %v", err)
	}
	opponent, err := q.CreateUser(ctx, db.CreateUserParams{Username: "timeout_opponent_" + suffix, Email: "to_" + suffix + "@example.com", Password: "x"})
	if err != nil {
		t.Fatalf("create opponent: %v", err)
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
	for i, uid := range []int64{mover.UserID, opponent.UserID} {
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
		Player1ID:    mover.UserID,
		Player2ID:    opponent.UserID,
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}
	if _, err := q.ActivateMatch(ctx, db.ActivateMatchParams{
		MatchID:           m.MatchID,
		CurrentTurnUserID: dbutil.Int64ToPGInt8(mover.UserID),
	}); err != nil {
		t.Fatalf("activate match: %v", err)
	}

	svc := NewService(pool, hub.New(), missions.NewService(pool), leaderboard.NewService(pool), matchhistory.NewService(pool), achievements.NewService(pool))

	// Simulate the mover's clock having already run out (e.g. an auto-fired
	// pre-move that lands microseconds after the deadline) by seeding an
	// already-past deadline directly, the same shape startMatch/HandleMove
	// would have set.
	svc.mu.Lock()
	svc.deadlines[m.MatchID] = deadline{
		MatchID:        m.MatchID,
		OpponentUserID: opponent.UserID,
		At:             time.Now().Add(-time.Second),
	}
	svc.mu.Unlock()

	if err := svc.HandleMove(ctx, m.MatchID, mover.UserID, MoveInput{From: "e2", To: "e4"}); err != nil {
		t.Fatalf("HandleMove after expired deadline returned an error: %v", err)
	}

	finalMatch, err := q.GetMatch(ctx, m.MatchID)
	if err != nil {
		t.Fatalf("get match: %v", err)
	}
	if finalMatch.Status.String != "completed" {
		t.Fatalf("match status = %q, want completed - the expired-deadline move should have finished the match instead of being silently dropped", finalMatch.Status.String)
	}
	if !finalMatch.WinnerUserID.Valid || finalMatch.WinnerUserID.Int64 != opponent.UserID {
		t.Fatalf("match winner = %+v, want opponent %d (mover should lose on time)", finalMatch.WinnerUserID, opponent.UserID)
	}

	finalOpponent, err := q.GetUserByID(ctx, opponent.UserID)
	if err != nil {
		t.Fatalf("get opponent: %v", err)
	}
	if dbutil.PGInt4ToInt(finalOpponent.TotalWins) != 1 {
		t.Fatalf("opponent total_wins = %d, want 1", dbutil.PGInt4ToInt(finalOpponent.TotalWins))
	}

	finalMover, err := q.GetUserByID(ctx, mover.UserID)
	if err != nil {
		t.Fatalf("get mover: %v", err)
	}
	if dbutil.PGInt4ToInt(finalMover.TotalLosses) != 1 {
		t.Fatalf("mover total_losses = %d, want 1", dbutil.PGInt4ToInt(finalMover.TotalLosses))
	}
}
