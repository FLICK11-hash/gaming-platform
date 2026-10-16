package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"gaming_platform/internal/auth"
	"gaming_platform/internal/domain/bot"
	"gaming_platform/internal/domain/connectfour"
	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
	"gaming_platform/internal/research/simulate"
)

// practiceTierBotUsernames maps a practice-match request's tier string to
// the fixed bot account that plays it - the same three accounts (and the
// same Stockfish Elo behind each one, see internal/domain/bot) the Monte
// Carlo research runner tests, so a chess player can try the exact opponent
// strength the research report describes. The same three accounts also
// back Connect Four practice matches, just relabeled Easy/Intermediate/
// Expert in the UI since there's no chess-flavored Elo to show there.
var practiceTierBotUsernames = map[string]string{
	"rookie":      bot.RookieBotUsername,
	"expert":      bot.ExpertBotUsername,
	"grandmaster": bot.GrandmasterBotUsername,
}

// handleStartPracticeMatch starts a free match (chess or Connect Four)
// against a chosen bot tier, skipping the matchmaking queue entirely - both
// "players" are already known, so there's nothing to wait on. Unlike
// handleChallengeUser and handleJoinQueue, it never takes a coin wager and
// the resulting match is registered with matchSvc as practice (see
// Service.RegisterPracticeMatch), so it never moves Elo, missions,
// leaderboards, or achievements either - this is a free, unlimited,
// opponent-opening-chosen-by-the-player mode, not a competitive one.
func (s *Server) handleStartPracticeMatch(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req startPracticeMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.GameType != "chess" && req.GameType != "connect_four" {
		writeError(w, http.StatusBadRequest, "game_type must be chess or connect_four")
		return
	}
	botUsername, ok := practiceTierBotUsernames[req.Tier]
	if !ok {
		writeError(w, http.StatusBadRequest, "tier must be rookie, expert, or grandmaster")
		return
	}
	if req.Order != "first" && req.Order != "second" && req.Order != "random" {
		writeError(w, http.StatusBadRequest, "order must be first, second, or random")
		return
	}

	q := db.New(s.pool)

	if _, err := q.GetActiveMatchForUser(r.Context(), userID); err == nil {
		writeError(w, http.StatusConflict, "already in an active match")
		return
	} else if err != pgx.ErrNoRows {
		writeError(w, http.StatusInternalServerError, "failed to check active matches")
		return
	}

	human, err := q.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}
	botUser, err := q.GetUserByUsername(r.Context(), botUsername)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "bot account not found")
		return
	}

	humanFirst := req.Order == "first"
	if req.Order == "random" {
		humanFirst = rand.Intn(2) == 0
	}

	var forcedOpening string
	if req.GameType == "connect_four" {
		forcedOpening, err = practiceConnectFourOpening(req.Opening)
	} else {
		forcedOpening, err = practiceChessOpening(req.Opening, humanFirst)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start practice match")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := db.New(s.pool).WithTx(tx)

	createdTournament, err := qtx.CreateTournament(r.Context(), db.CreateTournamentParams{
		GameType:        req.GameType,
		GameMode:        "one_vs_one",
		Status:          "waiting",
		CoinWager:       0,
		TotalCoinPool:   0,
		WinnerCoinPrize: 0,
		BonusCoins:      0,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start practice match")
		return
	}

	player1ID, player2ID := human.UserID, botUser.UserID
	if !humanFirst {
		player1ID, player2ID = botUser.UserID, human.UserID
	}

	for seed, uid := range []int64{player1ID, player2ID} {
		if _, err := qtx.CreateTournamentPlayer(r.Context(), db.CreateTournamentPlayerParams{
			TournamentID: createdTournament.TournamentID,
			UserID:       uid,
			SeedNumber:   dbutil.IntToPGInt4(seed + 1),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to start practice match")
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
		writeError(w, http.StatusInternalServerError, "failed to start practice match")
		return
	}

	if _, err := qtx.StartTournament(r.Context(), createdTournament.TournamentID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start practice match")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start practice match")
		return
	}

	s.matchSvc.RegisterPracticeMatch(match.MatchID, forcedOpening)

	writeJSON(w, http.StatusCreated, s.matchResponseWithProfiles(r, db.New(s.pool), match))
}

// practiceChessOpening validates (or, for "" / "random", picks) the
// computer's forced first move in a chess practice match: a move in
// from+to square form (e.g. "e2e4") taken from whichever of the two Monte
// Carlo opening books - see internal/research/simulate/openings.go -
// matches the computer's color. The computer is Black exactly when the
// human plays first.
func practiceChessOpening(requested string, humanFirst bool) (string, error) {
	book := simulate.WhiteChessOpenings
	if humanFirst {
		book = simulate.BlackChessOpenings
	}
	if requested == "" || requested == "random" {
		return book[rand.Intn(len(book))], nil
	}
	for _, o := range book {
		if o == requested {
			return requested, nil
		}
	}
	return "", errors.New("opening is not one of the ten options for the computer's color")
}

// practiceConnectFourOpening validates (or, for "" / "random", picks) the
// computer's forced first drop in a Connect Four practice match: a column
// numbered 1-7 the way the UI shows it, converted to the engine's 0-based
// index for storage (see connectfour.Columns and match.Service's
// popForcedOpening).
func practiceConnectFourOpening(requested string) (string, error) {
	if requested == "" || requested == "random" {
		return strconv.Itoa(rand.Intn(connectfour.Columns)), nil
	}
	n, err := strconv.Atoi(requested)
	if err != nil || n < 1 || n > connectfour.Columns {
		return "", fmt.Errorf("opening must be a column number from 1 to %d, or random", connectfour.Columns)
	}
	return strconv.Itoa(n - 1), nil
}
