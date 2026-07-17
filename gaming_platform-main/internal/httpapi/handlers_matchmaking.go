package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"gaming_platform/internal/auth"
	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
)

// FixedCoinWager is the flat virtual-coin wager charged for every match and
// tournament entry.
const FixedCoinWager int64 = 200

func (s *Server) handleJoinQueue(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req joinQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.GameType != "chess" && req.GameType != "connect_four" {
		writeError(
			w,
			http.StatusBadRequest,
			"game_type must be chess or connect_four",
		)
		return
	}

	if req.GameMode != "one_vs_one" && req.GameMode != "tournament" {
		writeError(
			w,
			http.StatusBadRequest,
			"game_mode must be one_vs_one or tournament",
		)
		return
	}

	req.CoinWager = FixedCoinWager

	q := db.New(s.pool)

	user, err := q.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	if user.CoinBalance < req.CoinWager {
		writeError(
			w,
			http.StatusBadRequest,
			"insufficient coin balance",
		)
		return
	}

	_, err = q.GetWaitingQueueEntryForUser(r.Context(), userID)
	if err == nil {
		writeError(
			w,
			http.StatusConflict,
			"already waiting in a matchmaking queue",
		)
		return
	}

	if err != pgx.ErrNoRows {
		writeError(
			w,
			http.StatusInternalServerError,
			"failed to check existing queue entry",
		)
		return
	}

	_, err = q.GetActiveMatchForUser(r.Context(), userID)
	if err == nil {
		writeError(
			w,
			http.StatusConflict,
			"already in an active match",
		)
		return
	}

	if err != pgx.ErrNoRows {
		writeError(
			w,
			http.StatusInternalServerError,
			"failed to check existing match",
		)
		return
	}

	var minElo pgtype.Int4
	var maxElo pgtype.Int4

	if req.MinElo != nil {
		minElo = dbutil.IntToPGInt4(*req.MinElo)
	}

	if req.MaxElo != nil {
		maxElo = dbutil.IntToPGInt4(*req.MaxElo)
	}

	entry, err := q.CreateQueueEntry(
		r.Context(),
		db.CreateQueueEntryParams{
			UserID:    userID,
			GameType:  req.GameType,
			GameMode:  req.GameMode,
			CoinWager: req.CoinWager,
			MinElo:    minElo,
			MaxElo:    maxElo,
		},
	)
	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"failed to join matchmaking queue",
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		toQueueResponse(entry),
	)
}

func (s *Server) handleCancelQueue(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	queueID, err := strconv.ParseInt(
		chi.URLParam(r, "queueID"),
		10,
		64,
	)
	if err != nil || queueID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid queue ID")
		return
	}

	q := db.New(s.pool)

	err = q.CancelQueueEntry(
		r.Context(),
		db.CancelQueueEntryParams{
			QueueID: queueID,
			UserID:  userID,
		},
	)
	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"failed to cancel queue entry",
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
