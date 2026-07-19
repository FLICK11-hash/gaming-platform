package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"gaming_platform/internal/auth"
	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
)

// dailyLoginRewardCoins is the flat amount granted once per rolling 24h
// window by handleClaimDailyReward.
const dailyLoginRewardCoins = 100

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	q := db.New(s.pool)
	user, err := q.CreateUser(r.Context(), db.CreateUserParams{
		Username: req.Username,
		Email:    req.Email,
		Password: hashed,
	})
	if err != nil {
		writeError(w, http.StatusConflict, "username or email already in use")
		return
	}

	token, err := s.issuer.Issue(user.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: toUserResponse(user)})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	q := db.New(s.pool)
	user, err := q.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to look up user")
		return
	}

	if !auth.CheckPassword(user.Password, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	// Bot accounts (see migration 0009_bot_users.up.sql) never authenticate
	// over HTTP; this is defense-in-depth on top of the fact that nobody
	// knows their password.
	if user.IsBot {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := s.issuer.Issue(user.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{Token: token, User: toUserResponse(user)})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	q := db.New(s.pool)
	user, err := q.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// handleClaimDailyReward grants dailyLoginRewardCoins once per rolling 24h
// window, tracked by users.last_daily_reward_at.
func (s *Server) handleClaimDailyReward(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to claim daily reward")
		return
	}
	defer tx.Rollback(r.Context())
	q := db.New(s.pool).WithTx(tx)

	before, err := q.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to claim daily reward")
		return
	}

	if _, err := q.ClaimDailyLoginReward(r.Context(), db.ClaimDailyLoginRewardParams{
		UserID:       userID,
		RewardAmount: dailyLoginRewardCoins,
	}); err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusConflict, "daily reward already claimed; try again later")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to claim daily reward")
		return
	}

	if _, err := q.CreateCoinTransaction(r.Context(), db.CreateCoinTransactionParams{
		UserID:          userID,
		Amount:          dailyLoginRewardCoins,
		TransactionType: "daily_login",
		Description:     dbutil.TextToPGText("Daily login reward"),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to claim daily reward")
		return
	}

	// A claim continues the login streak if the previous one was within the
	// last 48h (one day's gap plus slack, since the only real gate is the
	// 24h cooldown above rather than a fixed wall-clock day boundary);
	// otherwise it starts a fresh streak.
	now := time.Now()
	streak := 1
	streakStart := now
	if before.LastDailyRewardAt.Valid && now.Sub(before.LastDailyRewardAt.Time) < 48*time.Hour {
		streak = int(before.CurrentLoginStreak) + 1
		if before.LoginStreakStartedOn.Valid {
			streakStart = before.LoginStreakStartedOn.Time
		}
	}

	updated, err := q.SetUserLoginStreak(r.Context(), db.SetUserLoginStreakParams{
		UserID:               userID,
		CurrentLoginStreak:   int32(streak),
		LoginStreakStartedOn: pgtype.Date{Time: streakStart, Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to claim daily reward")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to claim daily reward")
		return
	}

	if s.missionsSvc != nil {
		s.missionsSvc.RecordDailyClaim(r.Context(), userID, streak, streakStart)
	}

	writeJSON(w, http.StatusOK, toUserResponse(updated))
}
