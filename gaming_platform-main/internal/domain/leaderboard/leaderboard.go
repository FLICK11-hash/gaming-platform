// Package leaderboard computes the top-10 rankings shown on the dashboard
// (Elo, weekly wins, tournament championships, best win streak, and coin
// balance) and runs a weekly job that grants a cosmetic badge to the top 10
// players by wins for the ISO week that just ended.
package leaderboard

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
)

const weeklyRewardBadgeImageKey = "weekly-champion"

type Entry struct {
	UserID         int64  `json:"user_id"`
	Username       string `json:"username"`
	SelectedAvatar string `json:"selected_avatar"`
	SelectedBadge  string `json:"selected_badge,omitempty"`
	Value          int64  `json:"value"`
}

type Board struct {
	Elo                 []Entry `json:"elo"`
	WeeklyWins          []Entry `json:"weekly_wins"`
	TournamentChampions []Entry `json:"tournament_champions"`
	WinStreak           []Entry `json:"win_streak"`
	Coins               []Entry `json:"coins"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// startOfWeek returns midnight UTC on the Monday of t's ISO week.
func startOfWeek(t time.Time) time.Time {
	t = t.UTC()
	weekday := int(t.Weekday())
	if weekday == 0 { // time.Sunday == 0; ISO weeks start Monday
		weekday = 7
	}
	daysSinceMonday := weekday - 1
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -daysSinceMonday)
}

func weekKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// GetBoard returns the current top 10 for every leaderboard tab.
func (s *Service) GetBoard(ctx context.Context) (Board, error) {
	q := db.New(s.pool)
	var board Board

	eloRows, err := q.TopUsersByElo(ctx)
	if err != nil {
		return board, err
	}
	for _, r := range eloRows {
		board.Elo = append(board.Elo, Entry{
			UserID:         r.UserID,
			Username:       r.Username,
			SelectedAvatar: r.SelectedAvatar,
			SelectedBadge:  dbutil.PGTextToString(r.SelectedBadge),
			Value:          int64(dbutil.PGInt4ToInt(r.CurrentElo)),
		})
	}

	coinRows, err := q.TopUsersByCoins(ctx)
	if err != nil {
		return board, err
	}
	for _, r := range coinRows {
		board.Coins = append(board.Coins, Entry{
			UserID:         r.UserID,
			Username:       r.Username,
			SelectedAvatar: r.SelectedAvatar,
			SelectedBadge:  dbutil.PGTextToString(r.SelectedBadge),
			Value:          r.CoinBalance,
		})
	}

	streakRows, err := q.TopUsersByWinStreak(ctx)
	if err != nil {
		return board, err
	}
	for _, r := range streakRows {
		board.WinStreak = append(board.WinStreak, Entry{
			UserID:         r.UserID,
			Username:       r.Username,
			SelectedAvatar: r.SelectedAvatar,
			SelectedBadge:  dbutil.PGTextToString(r.SelectedBadge),
			Value:          int64(r.BestWinStreak),
		})
	}

	champRows, err := q.TopUsersByTournamentChampionships(ctx)
	if err != nil {
		return board, err
	}
	for _, r := range champRows {
		board.TournamentChampions = append(board.TournamentChampions, Entry{
			UserID:         r.UserID,
			Username:       r.Username,
			SelectedAvatar: r.SelectedAvatar,
			SelectedBadge:  dbutil.PGTextToString(r.SelectedBadge),
			Value:          r.Championships,
		})
	}

	now := time.Now().UTC()
	weeklyRows, err := q.TopUsersByWeeklyWins(ctx, db.TopUsersByWeeklyWinsParams{
		Since: pgtype.Timestamptz{Time: startOfWeek(now), Valid: true},
		Until: pgtype.Timestamptz{Time: now.Add(time.Hour), Valid: true},
	})
	if err != nil {
		return board, err
	}
	for _, r := range weeklyRows {
		board.WeeklyWins = append(board.WeeklyWins, Entry{
			UserID:         r.UserID,
			Username:       r.Username,
			SelectedAvatar: r.SelectedAvatar,
			SelectedBadge:  dbutil.PGTextToString(r.SelectedBadge),
			Value:          r.WeeklyWins,
		})
	}

	return board, nil
}

// RecordMatchResult updates each player's win streak after a match: a win
// extends the streak (and raises the all-time best if a new high), while a
// loss or draw resets it to zero. winnerUserID is 0 for a draw.
func (s *Service) RecordMatchResult(ctx context.Context, player1ID, player2ID, winnerUserID int64) {
	q := db.New(s.pool)

	for _, playerID := range []int64{player1ID, player2ID} {
		var err error
		if playerID == winnerUserID {
			_, err = q.RecordWinStreak(ctx, playerID)
		} else {
			err = q.ResetWinStreak(ctx, playerID)
		}
		if err != nil {
			log.Printf("leaderboard: win streak update error for user %d: %v", playerID, err)
		}
	}
}

// RunWeeklyRewards periodically checks whether the ISO week that just ended
// has already had its top-10 cosmetic reward granted, and grants it if not.
// Ticking hourly (rather than scheduling for exactly Monday 00:00 UTC) keeps
// this resilient to the process being down at the exact rollover moment.
func (s *Service) RunWeeklyRewards(ctx context.Context) {
	s.tryRunWeeklyReward(ctx)

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tryRunWeeklyReward(ctx)
		}
	}
}

func (s *Service) tryRunWeeklyReward(ctx context.Context) {
	q := db.New(s.pool)

	now := time.Now().UTC()
	thisWeekStart := startOfWeek(now)
	lastWeekStart := thisWeekStart.AddDate(0, 0, -7)
	key := weekKey(lastWeekStart)

	already, err := q.HasLeaderboardRewardRun(ctx, key)
	if err != nil {
		log.Printf("leaderboard: check reward run error: %v", err)
		return
	}
	if already {
		return
	}

	rows, err := q.TopUsersByWeeklyWins(ctx, db.TopUsersByWeeklyWinsParams{
		Since: pgtype.Timestamptz{Time: lastWeekStart, Valid: true},
		Until: pgtype.Timestamptz{Time: thisWeekStart, Valid: true},
	})
	if err != nil {
		log.Printf("leaderboard: list weekly winners error: %v", err)
		return
	}

	if len(rows) > 0 {
		badge, err := q.GetCosmeticItemByImageKey(ctx, weeklyRewardBadgeImageKey)
		if err != nil {
			log.Printf("leaderboard: load weekly champion badge error: %v", err)
			return
		}
		for _, r := range rows {
			if err := q.GrantCosmeticToUser(ctx, db.GrantCosmeticToUserParams{
				UserID:         r.UserID,
				CosmeticItemID: badge.CosmeticItemID,
			}); err != nil {
				log.Printf("leaderboard: grant weekly champion badge error for user %d: %v", r.UserID, err)
			}
		}
	}

	if err := q.RecordLeaderboardRewardRun(ctx, key); err != nil {
		log.Printf("leaderboard: record reward run error: %v", err)
	}
}
