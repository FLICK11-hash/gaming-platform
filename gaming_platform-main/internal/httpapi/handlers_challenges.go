package httpapi

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"gaming_platform/internal/auth"
	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
)

// handleChallengeUser directly starts a match between the caller and
// userID, skipping the matchmaking queue entirely - both players are already
// known, so there's nothing to wait on. The challenged player picks the
// match up the same way any matched player does: their existing poll of
// GET /matches/mine finds it once it exists.
func (s *Server) handleChallengeUser(w http.ResponseWriter, r *http.Request) {
	challengerID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	challengedID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if challengedID == challengerID {
		writeError(w, http.StatusBadRequest, "cannot challenge yourself")
		return
	}

	var req challengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.GameType != "chess" && req.GameType != "connect_four" {
		writeError(w, http.StatusBadRequest, "game_type must be chess or connect_four")
		return
	}

	q := db.New(s.pool)

	challenger, err := q.GetUserByID(r.Context(), challengerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load challenger")
		return
	}
	challenged, err := q.GetUserByID(r.Context(), challengedID)
	if err != nil {
		writeError(w, http.StatusNotFound, "challenged user not found")
		return
	}
	if challenged.IsBot {
		writeError(w, http.StatusBadRequest, "cannot challenge a bot directly")
		return
	}

	wager := FixedCoinWager
	if challenger.CoinBalance < wager {
		writeError(w, http.StatusBadRequest, "insufficient coin balance")
		return
	}
	if challenged.CoinBalance < wager {
		writeError(w, http.StatusBadRequest, "challenged player doesn't have enough coins for the wager")
		return
	}

	for _, uid := range []int64{challengerID, challengedID} {
		if _, err := q.GetActiveMatchForUser(r.Context(), uid); err == nil {
			writeError(w, http.StatusConflict, "one of you is already in an active match")
			return
		} else if err != pgx.ErrNoRows {
			writeError(w, http.StatusInternalServerError, "failed to check active matches")
			return
		}
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create challenge")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := db.New(s.pool).WithTx(tx)

	totalCoinPool := wager * 2
	createdTournament, err := qtx.CreateTournament(r.Context(), db.CreateTournamentParams{
		GameType:        req.GameType,
		GameMode:        "one_vs_one",
		Status:          "waiting",
		CoinWager:       wager,
		TotalCoinPool:   totalCoinPool,
		WinnerCoinPrize: totalCoinPool,
		BonusCoins:      0,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create challenge")
		return
	}

	// Who's seeded player 1 (and therefore moves first) is randomized rather
	// than always the challenger.
	player1ID, player2ID := challengerID, challengedID
	if rand.Intn(2) == 0 {
		player1ID, player2ID = player2ID, player1ID
	}

	for seed, uid := range []int64{player1ID, player2ID} {
		if _, err := qtx.CreateTournamentPlayer(r.Context(), db.CreateTournamentPlayerParams{
			TournamentID: createdTournament.TournamentID,
			UserID:       uid,
			SeedNumber:   dbutil.IntToPGInt4(seed + 1),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create challenge")
			return
		}

		if _, err := qtx.AdjustUserCoinBalance(r.Context(), db.AdjustUserCoinBalanceParams{
			UserID: uid,
			Delta:  -wager,
		}); err != nil {
			writeError(w, http.StatusBadRequest, "insufficient coin balance")
			return
		}
		if _, err := qtx.CreateCoinTransaction(r.Context(), db.CreateCoinTransactionParams{
			UserID:              uid,
			Amount:              -wager,
			TransactionType:     "match_entry",
			Description:         dbutil.TextToPGText("Entry wager for " + req.GameType + " challenge"),
			RelatedTournamentID: dbutil.Int64ToPGInt8(createdTournament.TournamentID),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create challenge")
			return
		}
	}

	match, err := qtx.CreateMatch(r.Context(), db.CreateMatchParams{
		TournamentID: createdTournament.TournamentID,
		GameType:     req.GameType,
		Round:        "final",
		MatchNumber:  dbutil.IntToPGInt4(1),
		Player1ID:    player1ID,
		Player2ID:    player2ID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create challenge")
		return
	}

	if _, err := qtx.StartTournament(r.Context(), createdTournament.TournamentID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create challenge")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create challenge")
		return
	}

	writeJSON(w, http.StatusCreated, s.matchResponseWithProfiles(r, db.New(s.pool), match))
}
