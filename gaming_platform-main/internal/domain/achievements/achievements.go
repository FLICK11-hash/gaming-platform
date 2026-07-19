// Package achievements grants permanent, one-time cosmetic unlocks once a
// player crosses a lifetime milestone (see migration 0020_achievements.up.sql).
// Unlike the Missions system these never reset: once granted, a reward is
// kept forever. Checks are best-effort, mirroring missions/leaderboard: a
// failure here logs and moves on rather than failing the match, daily-claim,
// or tournament flow that triggered it.
package achievements

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	db "gaming_platform/internal/db/gen"
)

const (
	winsThreshold           = 10
	eloThreshold            = 1500
	tournamentWinsThreshold = 25
	loginStreakThreshold    = 100

	bronzeKnightImageKey    = "bronze-knight"
	silverCrownImageKey     = "silver-crown"
	championBannerImageKey  = "champion-banner"
	legendaryAvatarImageKey = "legendary-avatar"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// CheckMatchAchievements checks the win-count and Elo achievements for a
// single non-bot player using the totalWins/currentElo values already
// computed by the caller after a match's stats update lands.
func (s *Service) CheckMatchAchievements(ctx context.Context, userID int64, totalWins int, currentElo int) {
	if totalWins >= winsThreshold {
		s.grantIfMissing(ctx, userID, bronzeKnightImageKey)
	}
	if currentElo >= eloThreshold {
		s.grantIfMissing(ctx, userID, silverCrownImageKey)
	}
}

// CheckLoginStreakAchievement checks the 100-day login-streak achievement
// using the streak value already computed by the daily-reward claim handler.
func (s *Service) CheckLoginStreakAchievement(ctx context.Context, userID int64, streak int) {
	if streak >= loginStreakThreshold {
		s.grantIfMissing(ctx, userID, legendaryAvatarImageKey)
	}
}

// CheckTournamentAchievement checks the tournament-win-count achievement
// using the tournamentWinCount value already computed by the caller after a
// tournament's winner_user_id/status is set to 'completed'.
func (s *Service) CheckTournamentAchievement(ctx context.Context, userID int64, tournamentWinCount int) {
	if tournamentWinCount >= tournamentWinsThreshold {
		s.grantIfMissing(ctx, userID, championBannerImageKey)
	}
}

func (s *Service) grantIfMissing(ctx context.Context, userID int64, imageKey string) {
	q := db.New(s.pool)

	item, err := q.GetCosmeticItemByImageKey(ctx, imageKey)
	if err != nil {
		log.Printf("achievements: load cosmetic item %q error: %v", imageKey, err)
		return
	}

	owned, err := q.UserOwnsCosmetic(ctx, db.UserOwnsCosmeticParams{
		UserID:         userID,
		CosmeticItemID: item.CosmeticItemID,
	})
	if err != nil {
		log.Printf("achievements: ownership check error for user %d item %q: %v", userID, imageKey, err)
		return
	}
	if owned {
		return
	}

	if err := q.GrantCosmeticToUser(ctx, db.GrantCosmeticToUserParams{
		UserID:         userID,
		CosmeticItemID: item.CosmeticItemID,
	}); err != nil {
		log.Printf("achievements: grant error for user %d item %q: %v", userID, imageKey, err)
	}
}
