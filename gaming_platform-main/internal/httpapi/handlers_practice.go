package httpapi

import (
	"encoding/json"
	"math/rand"
	"net/http"

	"github.com/jackc/pgx/v5"

	"gaming_platform/internal/auth"
	"gaming_platform/internal/domain/bot"
	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
	"gaming_platform/internal/research/simulate"
)

// practiceTierBotUsernames maps a practice-match request's tier string to
// the fixed bot account that plays it - the same three accounts (and the
// same Stockfish Elo behind each one, see internal/domain/bot) the Monte
// Carlo research runner tests, so a player can try the exact opponent
// strength the research report describes.
var practiceTierBotUsernames = map[string]string{
	"rookie":      bot.RookieBotUsername,
	"expert":      bot.ExpertBotUsername,
	"grandmaster": bot.GrandmasterBotUsername,
}

// handleStartPracticeMatch starts a free chess match against a chosen bot
// tier, skipping the matchmaking queue entirely - both "players" are already
// known, so there's nothing to wait on. Unlike handleChallengeUser and
// handleJoinQueue, it never takes a coin wager and the resulting match is
// registered with matchSvc as practice (see Service.RegisterPracticeMatch),
// so it never moves Elo, missions, leaderboards, or achievements either -
// this is a free, unlimited, opponent-opening-chosen-by-the-player mode, not
// a competitive one.
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

	botUsername, ok := practiceTierBotUsernames[req.Tier]
	if !ok {
		writeError(w, http.StatusBadRequest, "tier must be rookie, expert, or grandmaster")
		return
	}
	if req.Color != "white" && req.Color != "black" && req.Color != "random" {
		writeError(w, http.StatusBadRequest, "color must be white, black, or random")
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

	humanIsWhite := req.Color == "white"
	if req.Color == "random" {
		humanIsWhite = rand.Intn(2) == 0
	}

	// The opening picker always describes the computer's own first move
	// (whichever ply that ends up being, depending on who moves first),
	// taken from whichever of the two Monte Carlo opening books matches the
	// computer's color - the same books runner.go tests each tier against.
	book := simulate.BlackChessOpenings
	if !humanIsWhite {
		book = simulate.WhiteChessOpenings
	}

	forcedOpening := ""
	if req.Opening != "" && req.Opening != "random" {
		valid := false
		for _, o := range book {
			if o == req.Opening {
				valid = true
				break
			}
		}
		if !valid {
			writeError(w, http.StatusBadRequest, "opening is not one of the ten options for the computer's color")
			return
		}
		forcedOpening = req.Opening
	} else {
		forcedOpening = book[rand.Intn(len(book))]
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start practice match")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := db.New(s.pool).WithTx(tx)

	createdTournament, err := qtx.CreateTournament(r.Context(), db.CreateTournamentParams{
		GameType:        "chess",
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
	if !humanIsWhite {
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
		GameType:     "chess",
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
