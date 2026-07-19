package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"gaming_platform/internal/auth"
	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
	"gaming_platform/internal/domain/missions"
)

// missionPeriodKey computes which progress bucket a mission is currently
// tracking for user, matching internal/domain/missions.PeriodKeyFor exactly
// so a claim always targets the same row the list endpoint just showed.
func missionPeriodKey(m db.Mission, user db.User) string {
	return missions.PeriodKeyFor(m, int(user.CurrentLoginStreak), user.LoginStreakStartedOn.Time)
}

// handleListMissions returns every active mission along with the caller's
// current-period progress, auto-creating a zeroed progress row the first
// time a period is seen.
func (s *Server) handleListMissions(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	q := db.New(s.pool)

	user, err := q.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load missions")
		return
	}

	all, err := q.ListActiveMissions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load missions")
		return
	}

	resp := make([]missionResponse, 0, len(all))
	for _, m := range all {
		progress, err := q.GetOrCreateUserMissionProgress(r.Context(), db.GetOrCreateUserMissionProgressParams{
			UserID:    userID,
			MissionID: m.MissionID,
			PeriodKey: missionPeriodKey(m, user),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load missions")
			return
		}
		resp = append(resp, toMissionResponse(m, progress))
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleClaimMission grants a completed mission's coin/badge reward, once,
// atomically with marking it claimed.
func (s *Server) handleClaimMission(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	missionID, err := strconv.ParseInt(chi.URLParam(r, "missionID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid mission id")
		return
	}

	q := db.New(s.pool)

	m, err := q.GetMission(r.Context(), missionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "mission not found")
		return
	}

	user, err := q.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to claim mission")
		return
	}
	periodKey := missionPeriodKey(m, user)

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to claim mission")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := db.New(s.pool).WithTx(tx)

	progress, err := qtx.ClaimMissionProgress(r.Context(), db.ClaimMissionProgressParams{
		UserID:    userID,
		MissionID: missionID,
		PeriodKey: periodKey,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusConflict, "mission is not ready to claim")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to claim mission")
		return
	}

	if m.RewardCoins > 0 {
		if _, err := qtx.AdjustUserCoinBalance(r.Context(), db.AdjustUserCoinBalanceParams{
			UserID: userID,
			Delta:  m.RewardCoins,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to claim mission")
			return
		}
		if _, err := qtx.CreateCoinTransaction(r.Context(), db.CreateCoinTransactionParams{
			UserID:          userID,
			Amount:          m.RewardCoins,
			TransactionType: "mission_reward",
			Description:     dbutil.TextToPGText("Mission reward: " + m.Title),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to claim mission")
			return
		}
	}

	if m.RewardBadgeImageKey.Valid {
		badge, err := qtx.GetCosmeticItemByImageKey(r.Context(), m.RewardBadgeImageKey.String)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to claim mission")
			return
		}
		if err := qtx.GrantCosmeticToUser(r.Context(), db.GrantCosmeticToUserParams{
			UserID:         userID,
			CosmeticItemID: badge.CosmeticItemID,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to claim mission")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to claim mission")
		return
	}

	writeJSON(w, http.StatusOK, toMissionResponse(m, progress))
}
