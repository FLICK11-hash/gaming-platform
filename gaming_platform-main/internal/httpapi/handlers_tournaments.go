package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	db "gaming_platform/internal/db/gen"
)

func (s *Server) handleGetTournament(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := strconv.ParseInt(chi.URLParam(r, "tournamentID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament id")
		return
	}

	q := db.New(s.pool)

	t, err := q.GetTournament(r.Context(), tournamentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "tournament not found")
		return
	}

	players, err := q.ListTournamentPlayers(r.Context(), tournamentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load tournament players")
		return
	}

	matches, err := q.ListMatchesByTournament(r.Context(), tournamentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load tournament matches")
		return
	}

	writeJSON(w, http.StatusOK, toTournamentResponse(t, players, matches))
}
