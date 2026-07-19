package httpapi

import (
	"net/http"

	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/domain/leaderboard"
)

const landingLeaderboardPreviewSize = 5

type landingPlayerEntry struct {
	Username string `json:"username"`
	Value    int64  `json:"value"`
}

type landingStatsResponse struct {
	TotalPlayers        int64                `json:"total_players"`
	MatchesPlayed       int64                `json:"matches_played"`
	TopPlayers          []landingPlayerEntry `json:"top_players"`
	TournamentChampions []landingPlayerEntry `json:"tournament_champions"`
}

// handleGetLandingStats returns the honest, real-data stats shown on the
// logged-out landing page: total human players, completed match count, a
// top-5 Elo preview, and a top-5 tournament champions preview (which may be
// empty since 4-player tournaments are currently disabled elsewhere). No
// auth is required since this is meant to be shown before login.
func (s *Server) handleGetLandingStats(w http.ResponseWriter, r *http.Request) {
	q := db.New(s.pool)

	totalPlayers, err := q.CountHumanPlayers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load stats")
		return
	}

	matchesPlayed, err := q.CountCompletedMatches(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load stats")
		return
	}

	board, err := s.leaderboardSvc.GetBoard(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load stats")
		return
	}

	writeJSON(w, http.StatusOK, landingStatsResponse{
		TotalPlayers:        totalPlayers,
		MatchesPlayed:       matchesPlayed,
		TopPlayers:          landingPreview(board.Elo),
		TournamentChampions: landingPreview(board.TournamentChampions),
	})
}

func landingPreview(entries []leaderboard.Entry) []landingPlayerEntry {
	limit := len(entries)
	if limit > landingLeaderboardPreviewSize {
		limit = landingLeaderboardPreviewSize
	}

	out := make([]landingPlayerEntry, 0, limit)
	for _, e := range entries[:limit] {
		out = append(out, landingPlayerEntry{Username: e.Username, Value: e.Value})
	}
	return out
}
