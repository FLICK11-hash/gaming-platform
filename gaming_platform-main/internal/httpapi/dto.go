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

type userResponse struct {
	UserID               int64     `json:"user_id"`
	Username             string    `json:"username"`
	Email                string    `json:"email"`
	CurrentElo           int       `json:"current_elo"`
	TotalWins            int       `json:"total_wins"`
	TotalLosses          int       `json:"total_losses"`
	NumGames             int       `json:"num_games"`
	CoinBalance          int64     `json:"coin_balance"`
	SelectedAvatar       string    `json:"selected_avatar"`
	SelectedBadge        string    `json:"selected_badge,omitempty"`
	SelectedProfileFrame string    `json:"selected_profile_frame,omitempty"`
	AccountStatus        string    `json:"account_status"`
	CreatedAt            time.Time `json:"created_at"`
}

func toUserResponse(u db.User) userResponse {
	return userResponse{
		UserID:               u.UserID,
		Username:             u.Username,
		Email:                u.Email,
		CurrentElo:           dbutil.PGInt4ToInt(u.CurrentElo),
		TotalWins:            dbutil.PGInt4ToInt(u.TotalWins),
		TotalLosses:          dbutil.PGInt4ToInt(u.TotalLosses),
		NumGames:             dbutil.PGInt4ToInt(u.NumGames),
		CoinBalance:          u.CoinBalance,
		SelectedAvatar:       u.SelectedAvatar,
		SelectedBadge:        dbutil.PGTextToString(u.SelectedBadge),
		SelectedProfileFrame: dbutil.PGTextToString(u.SelectedProfileFrame),
		AccountStatus:        dbutil.PGTextToString(u.AccountStatus),
		CreatedAt:            u.CreatedAt.Time,
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

type tournamentPlayerResponse struct {
	UserID      int64  `json:"user_id"`
	SeedNumber  int    `json:"seed_number"`
	FinalResult string `json:"final_result,omitempty"`
}

func toTournamentPlayerResponse(p db.TournamentPlayer) tournamentPlayerResponse {
	return tournamentPlayerResponse{
		UserID:      p.UserID,
		SeedNumber:  dbutil.PGInt4ToInt(p.SeedNumber),
		FinalResult: dbutil.PGTextToString(p.FinalResult),
	}
}

type matchResponse struct {
	MatchID           int64  `json:"match_id"`
	TournamentID      int64  `json:"tournament_id"`
	Round             string `json:"round"`
	MatchNumber       int    `json:"match_number"`
	Player1ID         int64  `json:"player_1_id"`
	Player2ID         int64  `json:"player_2_id"`
	CurrentTurnUserID *int64 `json:"current_turn_user_id,omitempty"`
	WinnerUserID      *int64 `json:"winner_user_id,omitempty"`
	Status            string `json:"status"`
}

func toMatchResponse(m db.Match) matchResponse {
	resp := matchResponse{
		MatchID:      m.MatchID,
		TournamentID: m.TournamentID,
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

type tournamentResponse struct {
	TournamentID    int64                      `json:"tournament_id"`
	GameType        string                     `json:"game_type"`
	GameMode        string                     `json:"game_mode"`
	Status          string                     `json:"status"`
	CoinWager       int64                      `json:"coin_wager"`
	TotalCoinPool   int64                      `json:"total_coin_pool"`
	WinnerCoinPrize int64                      `json:"winner_coin_prize"`
	BonusCoins      int64                      `json:"bonus_coins"`
	WinnerUserID    *int64                     `json:"winner_user_id,omitempty"`
	Players         []tournamentPlayerResponse `json:"players"`
	Matches         []matchResponse            `json:"matches"`
}

func toTournamentResponse(
	t db.Tournament,
	players []db.TournamentPlayer,
	matches []db.Match,
) tournamentResponse {
	resp := tournamentResponse{
		TournamentID:    t.TournamentID,
		GameType:        t.GameType,
		GameMode:        t.GameMode,
		Status:          t.Status,
		CoinWager:       t.CoinWager,
		TotalCoinPool:   t.TotalCoinPool,
		WinnerCoinPrize: t.WinnerCoinPrize,
		BonusCoins:      t.BonusCoins,
		Players:         make([]tournamentPlayerResponse, 0, len(players)),
		Matches:         make([]matchResponse, 0, len(matches)),
	}

	if t.WinnerUserID.Valid {
		value := t.WinnerUserID.Int64
		resp.WinnerUserID = &value
	}

	for _, player := range players {
		resp.Players = append(
			resp.Players,
			toTournamentPlayerResponse(player),
		)
	}

	for _, match := range matches {
		resp.Matches = append(
			resp.Matches,
			toMatchResponse(match),
		)
	}

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

type createFraudReportRequest struct {
	ReportedUserID int64  `json:"reported_user_id"`
	TournamentID   *int64 `json:"tournament_id"`
	MatchID        *int64 `json:"match_id"`
	Reason         string `json:"reason"`
}

type updateFraudReportStatusRequest struct {
	Status string `json:"status"`
}

type fraudReportResponse struct {
	FraudReportID  int64  `json:"fraud_report_id"`
	ReporterUserID int64  `json:"reporter_user_id"`
	ReportedUserID int64  `json:"reported_user_id"`
	TournamentID   *int64 `json:"tournament_id,omitempty"`
	MatchID        *int64 `json:"match_id,omitempty"`
	Reason         string `json:"reason"`
	Status         string `json:"status"`
}

func toFraudReportResponse(f db.FraudReport) fraudReportResponse {
	resp := fraudReportResponse{
		FraudReportID:  f.FraudReportID,
		ReporterUserID: f.ReporterUserID,
		ReportedUserID: f.ReportedUserID,
		Reason:         dbutil.PGTextToString(f.Reason),
		Status:         dbutil.PGTextToString(f.Status),
	}
	if f.TournamentID.Valid {
		v := f.TournamentID.Int64
		resp.TournamentID = &v
	}
	if f.MatchID.Valid {
		v := f.MatchID.Int64
		resp.MatchID = &v
	}
	return resp
}

type createPostRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type updatePostStatusRequest struct {
	Status string `json:"status"`
}

type postResponse struct {
	PostID    int64     `json:"post_id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func toPostResponse(p db.Post) postResponse {
	return postResponse{
		PostID:    p.PostID,
		UserID:    p.UserID,
		Title:     dbutil.PGTextToString(p.Title),
		Body:      dbutil.PGTextToString(p.Body),
		Status:    dbutil.PGTextToString(p.Status),
		CreatedAt: p.CreatedAt.Time,
	}
}

type followResponse struct {
	FollowingUserID int64     `json:"following_user_id"`
	FollowedUserID  int64     `json:"followed_user_id"`
	CreatedAt       time.Time `json:"created_at"`
}

func toFollowResponse(f db.Follow) followResponse {
	return followResponse{
		FollowingUserID: f.FollowingUserID,
		FollowedUserID:  f.FollowedUserID,
		CreatedAt:       f.CreatedAt.Time,
	}
}

type inboundWSMessage struct {
	Type      string `json:"type"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	Promotion string `json:"promotion,omitempty"`
}
