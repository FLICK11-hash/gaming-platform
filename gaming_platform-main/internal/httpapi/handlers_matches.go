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

	writeJSON(w, http.StatusOK, s.matchResponseWithProfiles(r, q, m))
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

	writeJSON(w, http.StatusOK, s.matchResponseWithProfiles(r, q, m))
}

// matchResponseWithProfiles builds a matchResponse and, best-effort, attaches
// both players' equipped profile frames so opponents can see each other's
// pins in the match room. If either player lookup fails, the frames are just
// left empty rather than failing the whole request.
func (s *Server) matchResponseWithProfiles(r *http.Request, q *db.Queries, m db.Match) matchResponse {
	resp := toMatchResponse(m)

	player1, err1 := q.GetUserByID(r.Context(), m.Player1ID)
	player2, err2 := q.GetUserByID(r.Context(), m.Player2ID)
	if err1 == nil && err2 == nil {
		resp = withPlayerProfileFrames(resp, player1, player2)
	}

	return resp
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
