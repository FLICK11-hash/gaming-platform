// Package matchhistory builds a player's profile view: Elo over time, W/L
// record, tournament wins, favorite chess opening, average game length,
// equipped cosmetics, and recent matches. It also records the Elo snapshots
// and chess first-moves that feed those stats as matches complete.
package matchhistory

import (
	"context"
	"log"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
)

// openingsByFirstMove maps a first move, given as its from+to squares (e.g.
// "e2e4"), to a human-readable opening name. This is intentionally a small,
// static lookup rather than a real opening book - anything not recognized
// falls back to "Other".
var openingsByFirstMove = map[string]string{
	"e2e4": "King's Pawn",
	"d2d4": "Queen's Pawn",
	"c2c4": "English Opening",
	"g1f3": "Réti Opening",
}

const otherOpening = "Other"

type EloPoint struct {
	Elo        int       `json:"elo"`
	RecordedAt time.Time `json:"recorded_at"`
}

type RecentMatch struct {
	MatchID          int64     `json:"match_id"`
	OpponentUsername string    `json:"opponent_username"`
	GameType         string    `json:"game_type"`
	Result           string    `json:"result"`
	CompletedAt      time.Time `json:"completed_at"`
}

type Profile struct {
	UserID                int64         `json:"user_id"`
	Username              string        `json:"username"`
	CurrentElo            int           `json:"current_elo"`
	TotalWins             int           `json:"total_wins"`
	TotalLosses           int           `json:"total_losses"`
	NumGames              int           `json:"num_games"`
	EloHistory            []EloPoint    `json:"elo_history"`
	RecentMatches         []RecentMatch `json:"recent_matches"`
	FavoriteOpening       *string       `json:"favorite_opening"`
	AvgGameLengthSeconds  *float64      `json:"avg_game_length_seconds"`
	TournamentWins        int64         `json:"tournament_wins"`
	SelectedAvatar        string        `json:"selected_avatar"`
	SelectedBadge         string        `json:"selected_badge,omitempty"`
	SelectedProfileFrames []string      `json:"selected_profile_frames"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// RecordEloSnapshot appends a point to userID's Elo history. Called right
// after a non-bot player's Elo is updated at the end of a match.
func (s *Service) RecordEloSnapshot(ctx context.Context, userID, matchID int64, eloBefore, eloAfter int) {
	q := db.New(s.pool)
	if err := q.InsertEloHistory(ctx, db.InsertEloHistoryParams{
		UserID:    userID,
		MatchID:   dbutil.Int64ToPGInt8(matchID),
		EloBefore: dbutil.IntToPGInt4(eloBefore),
		Elo:       int32(eloAfter),
	}); err != nil {
		log.Printf("matchhistory: record elo snapshot error for user %d: %v", userID, err)
	}
}

// RecordFirstMove persists the opening move of a chess match (as "e2e4"
// from+to squares), the raw material the profile's favorite-opening stat is
// computed from. Called once, the moment a match's very first move is
// applied - live game state is otherwise held only in memory (see
// internal/domain/chess/engine.go) and is gone once the match completes.
func (s *Service) RecordFirstMove(ctx context.Context, matchID int64, from, to string) {
	q := db.New(s.pool)
	if err := q.SetMatchFirstMove(ctx, db.SetMatchFirstMoveParams{
		MatchID:   matchID,
		FirstMove: dbutil.TextToPGText(from + to),
	}); err != nil {
		log.Printf("matchhistory: record first move error for match %d: %v", matchID, err)
	}
}

// GetProfile assembles userID's full profile view.
func (s *Service) GetProfile(ctx context.Context, userID int64) (Profile, error) {
	q := db.New(s.pool)

	user, err := q.GetUserByID(ctx, userID)
	if err != nil {
		return Profile{}, err
	}

	profile := Profile{
		UserID:                user.UserID,
		Username:              user.Username,
		CurrentElo:            dbutil.PGInt4ToInt(user.CurrentElo),
		TotalWins:             dbutil.PGInt4ToInt(user.TotalWins),
		TotalLosses:           dbutil.PGInt4ToInt(user.TotalLosses),
		NumGames:              dbutil.PGInt4ToInt(user.NumGames),
		SelectedAvatar:        user.SelectedAvatar,
		SelectedBadge:         dbutil.PGTextToString(user.SelectedBadge),
		SelectedProfileFrames: nonNilStrings(user.SelectedProfileFrames),
		EloHistory:            []EloPoint{},
		RecentMatches:         []RecentMatch{},
	}

	eloRows, err := q.ListEloHistoryForUser(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	for _, r := range eloRows {
		profile.EloHistory = append(profile.EloHistory, EloPoint{
			Elo:        int(r.Elo),
			RecordedAt: r.RecordedAt.Time,
		})
	}

	matchRows, err := q.ListRecentCompletedMatchesForUser(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	for _, m := range matchRows {
		profile.RecentMatches = append(profile.RecentMatches, RecentMatch{
			MatchID:          m.MatchID,
			OpponentUsername: m.OpponentUsername,
			GameType:         m.GameType,
			Result:           matchResult(m.WinnerUserID, userID),
			CompletedAt:      m.CompletedAt.Time,
		})
	}

	firstMoves, err := q.ListChessFirstMovesForUser(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	if opening := favoriteOpening(firstMoves); opening != "" {
		profile.FavoriteOpening = &opening
	}

	avgRow, err := q.AvgGameLengthSecondsForUser(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	if avgRow.Games > 0 {
		avg := avgRow.AvgSeconds
		profile.AvgGameLengthSeconds = &avg
	}

	wins, err := q.CountTournamentWinsForUser(ctx, dbutil.Int64ToPGInt8(userID))
	if err != nil {
		return Profile{}, err
	}
	profile.TournamentWins = wins

	return profile, nil
}

// matchResult reports userID's outcome ("win", "loss", or "draw") for a
// match given its (possibly null, on a draw) winner.
func matchResult(winnerUserID pgtype.Int8, userID int64) string {
	if !winnerUserID.Valid {
		return "draw"
	}
	if winnerUserID.Int64 == userID {
		return "win"
	}
	return "loss"
}

// favoriteOpening returns the most frequently occurring opening name across
// a user's recorded chess first moves, or "" if they have none yet.
func favoriteOpening(firstMoves []pgtype.Text) string {
	counts := make(map[string]int)
	for _, fm := range firstMoves {
		if !fm.Valid {
			continue
		}
		name, ok := openingsByFirstMove[fm.String]
		if !ok {
			name = otherOpening
		}
		counts[name]++
	}

	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)

	best := ""
	bestCount := 0
	for _, name := range names {
		if counts[name] > bestCount {
			best, bestCount = name, counts[name]
		}
	}
	return best
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
