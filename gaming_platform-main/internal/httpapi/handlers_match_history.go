package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	db "gaming_platform/internal/db/gen"
)

// handleGetUserProfile returns username's profile: Elo history, W/L record,
// tournament wins, favorite opening, average game length, equipped
// cosmetics, and recent matches. Any authenticated user can search up and
// view any other user's profile - it's a public-within-the-app page, not
// owner-only.
func (s *Server) handleGetUserProfile(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	user, err := db.New(s.pool).GetUserByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load profile")
		return
	}

	profile, err := s.matchHistorySvc.GetProfile(r.Context(), user.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load profile")
		return
	}

	writeJSON(w, http.StatusOK, profile)
}
