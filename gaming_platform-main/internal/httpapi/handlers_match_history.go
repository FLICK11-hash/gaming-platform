package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// handleGetUserProfile returns userID's profile: Elo history, W/L record,
// tournament wins, favorite opening, average game length, equipped
// cosmetics, and recent matches. Any authenticated user can view any other
// user's profile - it's a public-within-the-app page, not owner-only.
func (s *Server) handleGetUserProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	profile, err := s.matchHistorySvc.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load profile")
		return
	}

	writeJSON(w, http.StatusOK, profile)
}
