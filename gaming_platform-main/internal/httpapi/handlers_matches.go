package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"gaming_platform/internal/auth"
	db "gaming_platform/internal/db/gen"
)

// handleGetMyActiveMatch returns the caller's current waiting/active match,
// so the frontend can find its way from "joined queue" to a live board
// without needing to know a match id up front.
func (s *Server) handleGetMyActiveMatch(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserID(r.Context())

	q := db.New(s.pool)
	m, err := q.GetActiveMatchForUser(r.Context(), userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, "no active match")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to look up active match")
		return
	}

	writeJSON(w, http.StatusOK, toMatchResponse(m))
}

func (s *Server) handleGetMatch(w http.ResponseWriter, r *http.Request) {
	matchID, err := strconv.ParseInt(chi.URLParam(r, "matchID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid match id")
		return
	}

	q := db.New(s.pool)
	m, err := q.GetMatch(r.Context(), matchID)
	if err != nil {
		writeError(w, http.StatusNotFound, "match not found")
		return
	}

	writeJSON(w, http.StatusOK, toMatchResponse(m))
}

func (s *Server) handleResign(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserID(r.Context())

	matchID, err := strconv.ParseInt(chi.URLParam(r, "matchID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid match id")
		return
	}

	if err := s.matchSvc.HandleResign(r.Context(), matchID, userID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
