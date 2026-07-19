// Package missions tracks progress toward recurring daily and streak-based
// objectives (see migration 0017_missions.up.sql) and grants their coin/badge
// rewards once claimed. Progress updates are best-effort: a failure here
// logs and moves on rather than failing the match or daily-reward flow that
// triggered it.
package missions

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	db "gaming_platform/internal/db/gen"
)

const (
	GoalTypeWins        = "wins"
	GoalTypeGamesPlayed = "games_played"
	GoalTypeUpsetWin    = "upset_win"
	GoalTypeLoginStreak = "login_streak"

	ResetTypeDaily  = "daily"
	ResetTypeStreak = "streak"

	// UpsetEloThreshold is how far above the winner's Elo the loser's Elo
	// must have been, at the time the match started, for the win to count
	// toward an upset-win mission.
	UpsetEloThreshold = 100
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// RecordMatchCompletion updates daily mission progress for both players in a
// just-finished match: games-played for both, and wins (plus upset-win, if
// the loser was rated UpsetEloThreshold or more above the winner) for
// whoever won. winnerUserID is 0 for a draw, in which case only
// games-played progresses. player1Elo/player2Elo are each player's Elo
// immediately before this match's result was applied.
func (s *Service) RecordMatchCompletion(ctx context.Context, player1ID, player2ID, winnerUserID int64, player1Elo, player2Elo int) {
	s.incrementDaily(ctx, player1ID, GoalTypeGamesPlayed, 1)
	s.incrementDaily(ctx, player2ID, GoalTypeGamesPlayed, 1)

	if winnerUserID == 0 {
		return
	}
	s.incrementDaily(ctx, winnerUserID, GoalTypeWins, 1)

	winnerElo, loserElo := player1Elo, player2Elo
	if winnerUserID == player2ID {
		winnerElo, loserElo = player2Elo, player1Elo
	}
	if loserElo-winnerElo >= UpsetEloThreshold {
		s.incrementDaily(ctx, winnerUserID, GoalTypeUpsetWin, 1)
	}
}

// RecordDailyClaim updates login-streak mission progress after a successful
// daily-reward claim. streak is the user's current_login_streak value after
// this claim, and streakStartedOn is the date the current streak began.
func (s *Service) RecordDailyClaim(ctx context.Context, userID int64, streak int, streakStartedOn time.Time) {
	q := db.New(s.pool)
	all, err := q.ListActiveMissions(ctx)
	if err != nil {
		log.Printf("missions: list active missions error: %v", err)
		return
	}
	for _, m := range all {
		if m.GoalType != GoalTypeLoginStreak || m.ResetType != ResetTypeStreak {
			continue
		}
		chunk := (streak - 1) / int(m.GoalCount)
		periodKey := fmt.Sprintf("%s#%d", streakStartedOn.Format("2006-01-02"), chunk)
		if err := s.applyIncrement(ctx, userID, m.MissionID, periodKey, 1, int(m.GoalCount)); err != nil {
			log.Printf("missions: increment error for user %d mission %s: %v", userID, m.Key, err)
		}
	}
}

func (s *Service) incrementDaily(ctx context.Context, userID int64, goalType string, by int) {
	q := db.New(s.pool)
	all, err := q.ListActiveMissions(ctx)
	if err != nil {
		log.Printf("missions: list active missions error: %v", err)
		return
	}
	today := time.Now().UTC().Format("2006-01-02")
	for _, m := range all {
		if m.GoalType != goalType || m.ResetType != ResetTypeDaily {
			continue
		}
		if err := s.applyIncrement(ctx, userID, m.MissionID, today, by, int(m.GoalCount)); err != nil {
			log.Printf("missions: increment error for user %d mission %s: %v", userID, m.Key, err)
		}
	}
}

func (s *Service) applyIncrement(ctx context.Context, userID, missionID int64, periodKey string, by, goalCount int) error {
	q := db.New(s.pool)
	row, err := q.IncrementMissionProgress(ctx, db.IncrementMissionProgressParams{
		UserID:        userID,
		MissionID:     missionID,
		PeriodKey:     periodKey,
		ProgressCount: int32(by),
	})
	if err != nil {
		return err
	}
	if int(row.ProgressCount) >= goalCount && !row.CompletedAt.Valid {
		return q.MarkMissionProgressCompleted(ctx, db.MarkMissionProgressCompletedParams{
			UserID:    userID,
			MissionID: missionID,
			PeriodKey: periodKey,
		})
	}
	return nil
}

// PeriodKeyFor computes the current progress bucket for a mission given a
// user's login-streak state, so callers (the missions API and the daily
// reward handler) always agree on which period_key a given mission/user
// pair is currently tracking.
func PeriodKeyFor(m db.Mission, currentLoginStreak int, loginStreakStartedOn time.Time) string {
	if m.ResetType == ResetTypeStreak {
		chunk := 0
		if currentLoginStreak > 0 {
			chunk = (currentLoginStreak - 1) / int(m.GoalCount)
		}
		return fmt.Sprintf("%s#%d", loginStreakStartedOn.Format("2006-01-02"), chunk)
	}
	return time.Now().UTC().Format("2006-01-02")
}
