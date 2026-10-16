package httpapi

import (
	"time"

	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
)

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

// MaxSelectedProfileFrames is how many profile frames a player may equip at
// once.
const MaxSelectedProfileFrames = 5

type userResponse struct {
	UserID                int64     `json:"user_id"`
	Username              string    `json:"username"`
	Email                 string    `json:"email"`
	CurrentElo            int       `json:"current_elo"`
	TotalWins             int       `json:"total_wins"`
	TotalLosses           int       `json:"total_losses"`
	NumGames              int       `json:"num_games"`
	CoinBalance           int64     `json:"coin_balance"`
	SelectedAvatar        string    `json:"selected_avatar"`
	SelectedBadge         string    `json:"selected_badge,omitempty"`
	SelectedProfileFrames []string  `json:"selected_profile_frames"`
	AccountStatus         string    `json:"account_status"`
	CreatedAt             time.Time `json:"created_at"`
}

func toUserResponse(u db.User) userResponse {
	return userResponse{
		UserID:                u.UserID,
		Username:              u.Username,
		Email:                 u.Email,
		CurrentElo:            dbutil.PGInt4ToInt(u.CurrentElo),
		TotalWins:             dbutil.PGInt4ToInt(u.TotalWins),
		TotalLosses:           dbutil.PGInt4ToInt(u.TotalLosses),
		NumGames:              dbutil.PGInt4ToInt(u.NumGames),
		CoinBalance:           u.CoinBalance,
		SelectedAvatar:        u.SelectedAvatar,
		SelectedBadge:         dbutil.PGTextToString(u.SelectedBadge),
		SelectedProfileFrames: nonNilStrings(u.SelectedProfileFrames),
		AccountStatus:         dbutil.PGTextToString(u.AccountStatus),
		CreatedAt:             u.CreatedAt.Time,
	}
}

type joinQueueRequest struct {
	GameType string `json:"game_type"`
	GameMode string `json:"game_mode"`
	CoinWager int64 `json:"coin_wager"`
	MinElo   *int   `json:"min_elo"`
	MaxElo   *int   `json:"max_elo"`
}

type queueResponse struct {
	QueueID   int64  `json:"queue_id"`
	GameType  string `json:"game_type"`
	GameMode  string `json:"game_mode"`
	CoinWager int64  `json:"coin_wager"`
	Status    string `json:"status"`
}

func toQueueResponse(q db.MatchmakingQueue) queueResponse {
	return queueResponse{
		QueueID:   q.QueueID,
		GameType:  q.GameType,
		GameMode:  q.GameMode,
		CoinWager: q.CoinWager,
		Status:    dbutil.PGTextToString(q.Status),
	}
}

type challengeRequest struct {
	GameType string `json:"game_type"`
}

// startPracticeMatchRequest requests a free match (chess or Connect Four)
// against one of the three fixed bot tiers (also used, for chess, by the
// Monte Carlo research runner - see internal/domain/research/runner.go's
// chessEloTiers). Order is "first", "second", or "random" - does the human
// move first? Opening's format depends on GameType: for chess, a move in
// from+to square form (e.g. "e2e4") taken from whichever of the two Monte
// Carlo opening books matches the computer's color; for connect_four, a
// column number "1".."7". Either way "random" (or empty) lets the server
// pick one - see handleStartPracticeMatch.
type startPracticeMatchRequest struct {
	GameType string `json:"game_type"`
	Tier     string `json:"tier"`
	Order    string `json:"order"`
	Opening  string `json:"opening"`
}

type matchResponse struct {
	MatchID              int64    `json:"match_id"`
	TournamentID         int64    `json:"tournament_id"`
	GameType             string   `json:"game_type"`
	Round                string   `json:"round"`
	MatchNumber          int      `json:"match_number"`
	Player1ID            int64    `json:"player_1_id"`
	Player2ID            int64    `json:"player_2_id"`
	Player1Username      string   `json:"player_1_username,omitempty"`
	Player2Username      string   `json:"player_2_username,omitempty"`
	Player1Badge         string   `json:"player_1_badge,omitempty"`
	Player2Badge         string   `json:"player_2_badge,omitempty"`
	Player1IsBot         bool     `json:"player_1_is_bot,omitempty"`
	Player2IsBot         bool     `json:"player_2_is_bot,omitempty"`
	CurrentTurnUserID    *int64   `json:"current_turn_user_id,omitempty"`
	WinnerUserID         *int64   `json:"winner_user_id,omitempty"`
	Status               string   `json:"status"`
	Player1ProfileFrames []string `json:"player_1_profile_frames,omitempty"`
	Player2ProfileFrames []string `json:"player_2_profile_frames,omitempty"`
}

func toMatchResponse(m db.Match) matchResponse {
	resp := matchResponse{
		MatchID:      m.MatchID,
		TournamentID: m.TournamentID,
		GameType:     m.GameType,
		Round:        m.Round,
		MatchNumber:  dbutil.PGInt4ToInt(m.MatchNumber),
		Player1ID:    m.Player1ID,
		Player2ID:    m.Player2ID,
		Status:       dbutil.PGTextToString(m.Status),
	}
	if m.CurrentTurnUserID.Valid {
		v := m.CurrentTurnUserID.Int64
		resp.CurrentTurnUserID = &v
	}
	if m.WinnerUserID.Valid {
		v := m.WinnerUserID.Int64
		resp.WinnerUserID = &v
	}
	return resp
}

// withPlayerProfiles attaches each player's username, equipped badge, and
// equipped profile frames to a match response, so both the in-battle
// identity and an opponent's pins are visible in the match room.
func withPlayerProfiles(resp matchResponse, player1, player2 db.User) matchResponse {
	resp.Player1Username = player1.Username
	resp.Player2Username = player2.Username
	resp.Player1Badge = dbutil.PGTextToString(player1.SelectedBadge)
	resp.Player2Badge = dbutil.PGTextToString(player2.SelectedBadge)
	resp.Player1ProfileFrames = nonNilStrings(player1.SelectedProfileFrames)
	resp.Player2ProfileFrames = nonNilStrings(player2.SelectedProfileFrames)
	resp.Player1IsBot = player1.IsBot
	resp.Player2IsBot = player2.IsBot
	return resp
}

type cosmeticItemResponse struct {
	CosmeticItemID int64  `json:"cosmetic_item_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	ItemType       string `json:"item_type"`
	ImageKey       string `json:"image_key"`
	CoinPrice      int64  `json:"coin_price"`
	Owned          bool   `json:"owned"`
}

func toCosmeticItemResponse(item db.CosmeticItem, owned bool) cosmeticItemResponse {
	return cosmeticItemResponse{
		CosmeticItemID: item.CosmeticItemID,
		Name:           item.Name,
		Description:    dbutil.PGTextToString(item.Description),
		ItemType:       item.ItemType,
		ImageKey:       item.ImageKey,
		CoinPrice:      item.CoinPrice,
		Owned:          owned,
	}
}

type equipCosmeticRequest struct {
	CosmeticItemID int64 `json:"cosmetic_item_id"`
}

type inboundWSMessage struct {
	Type      string `json:"type"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	Promotion string `json:"promotion,omitempty"`
	Column    *int   `json:"column,omitempty"`
	Premove   bool   `json:"premove,omitempty"`
}

// nonNilStrings ensures a nil slice serializes as [] instead of null.
func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

type missionResponse struct {
	MissionID           int64  `json:"mission_id"`
	Key                 string `json:"key"`
	ResetType           string `json:"reset_type"`
	Title               string `json:"title"`
	Description         string `json:"description"`
	GoalType            string `json:"goal_type"`
	GoalCount           int    `json:"goal_count"`
	RewardCoins         int64  `json:"reward_coins"`
	RewardBadgeImageKey string `json:"reward_badge_image_key,omitempty"`
	Progress            int    `json:"progress"`
	Completed           bool   `json:"completed"`
	Claimed             bool   `json:"claimed"`
}

func toMissionResponse(m db.Mission, p db.UserMissionProgress) missionResponse {
	progress := int(p.ProgressCount)
	if progress > int(m.GoalCount) {
		progress = int(m.GoalCount)
	}
	return missionResponse{
		MissionID:           m.MissionID,
		Key:                 m.Key,
		ResetType:           m.ResetType,
		Title:               m.Title,
		Description:         m.Description,
		GoalType:            m.GoalType,
		GoalCount:           int(m.GoalCount),
		RewardCoins:         m.RewardCoins,
		RewardBadgeImageKey: dbutil.PGTextToString(m.RewardBadgeImageKey),
		Progress:            progress,
		Completed:           p.CompletedAt.Valid,
		Claimed:             p.ClaimedAt.Valid,
	}
}
