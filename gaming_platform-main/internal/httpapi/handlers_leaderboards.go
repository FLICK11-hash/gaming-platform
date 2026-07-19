package httpapi

import "net/http"

// handleGetLeaderboards returns the top 10 for every leaderboard tab (Elo,
// weekly wins, tournament championships, best win streak, coin balance).
func (s *Server) handleGetLeaderboards(w http.ResponseWriter, r *http.Request) {
	board, err := s.leaderboardSvc.GetBoard(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load leaderboards")
		return
	}
	writeJSON(w, http.StatusOK, board)
}
