// Package match orchestrates a single chess match's lifecycle: starting once
// both players connect, applying moves, running the bullet clocks, resigning,
// completing on checkmate/stalemate/draw/timeout, updating Elo, and
// advancing the tournament bracket.
package match

import (
	"context"
	"errors"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
	"gaming_platform/internal/domain/chess"
	"gaming_platform/internal/domain/clock"
	"gaming_platform/internal/domain/elo"
	"gaming_platform/internal/domain/hub"
	"gaming_platform/internal/domain/tournament"
)

var (
	ErrNotParticipant = errors.New("match: user is not a participant in this match")
	ErrNotYourTurn    = errors.New("match: not your turn")
	ErrMatchNotActive = errors.New("match: match is not active")
)

type deadline struct {
	MatchID        int64
	OpponentUserID int64
	At             time.Time
}

type Service struct {
	pool   *pgxpool.Pool
	engine *chess.Engine
	hub    *hub.Hub

	mu        sync.Mutex
	deadlines map[int64]deadline
}

func NewService(pool *pgxpool.Pool, h *hub.Hub) *Service {
	return &Service{
		pool:      pool,
		engine:    chess.NewEngine(),
		hub:       h,
		deadlines: make(map[int64]deadline),
	}
}

// wsOutbound is broadcast/sent to clients over the match websocket.
type wsOutbound struct {
	Type    string         `json:"type"`
	FEN     string         `json:"fen,omitempty"`
	Turn    string         `json:"turn,omitempty"`
	Outcome string         `json:"outcome,omitempty"`
	Winner  int64          `json:"winner_user_id,omitempty"`
	Clocks  map[string]int `json:"clocks,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// Join registers a player's websocket connection to a match and starts the
// match once both players are connected.
func (s *Service) Join(ctx context.Context, matchID, userID int64, ws *websocket.Conn) error {
	q := db.New(s.pool)
	m, err := q.GetMatch(ctx, matchID)
	if err != nil {
		return err
	}
	if userID != m.Player1ID && userID != m.Player2ID {
		return ErrNotParticipant
	}
	if m.Status.String == "completed" || m.Status.String == "cancelled" {
		return ErrMatchNotActive
	}

	s.hub.Register(matchID, userID, ws)

	if m.Status.String == "waiting" && s.hub.ConnectedCount(matchID) >= 2 {
		return s.startMatch(ctx, m)
	}

	s.broadcastState(ctx, matchID)
	return nil
}

// Leave unregisters a player's websocket connection (e.g. on disconnect).
// The match and its clock continue running against a disconnected player.
func (s *Service) Leave(matchID, userID int64) {
	s.hub.Unregister(matchID, userID)
}

func (s *Service) startMatch(ctx context.Context, m db.Match) error {
	q := db.New(s.pool)
	now := time.Now()

	if _, err := q.ActivateMatch(ctx, db.ActivateMatchParams{
		MatchID:           m.MatchID,
		CurrentTurnUserID: dbutil.Int64ToPGInt8(m.Player1ID),
	}); err != nil {
		return err
	}

	nowPG := pgtype.Timestamptz{Time: now, Valid: true}
	if _, err := q.CreateMatchClock(ctx, db.CreateMatchClockParams{
		MatchID:         m.MatchID,
		UserID:          m.Player1ID,
		StartingSeconds: clock.DefaultStartingSeconds,
		IsCurrentTurn:   pgtype.Bool{Bool: true, Valid: true},
		LastStartedAt:   nowPG,
	}); err != nil {
		return err
	}
	if _, err := q.CreateMatchClock(ctx, db.CreateMatchClockParams{
		MatchID:         m.MatchID,
		UserID:          m.Player2ID,
		StartingSeconds: clock.DefaultStartingSeconds,
		IsCurrentTurn:   pgtype.Bool{Bool: false, Valid: true},
		LastStartedAt:   nowPG,
	}); err != nil {
		return err
	}

	s.engine.StartGame(m.MatchID)

	s.mu.Lock()
	s.deadlines[m.MatchID] = deadline{
		MatchID:        m.MatchID,
		OpponentUserID: m.Player2ID,
		At:             now.Add(clock.DefaultStartingSeconds * time.Second),
	}
	s.mu.Unlock()

	s.broadcastState(ctx, m.MatchID)
	return nil
}

// HandleMove validates and applies a move (e.g. from="e2", to="e4") from
// userID. promo is the promotion piece letter ("q", "r", "b", "n"), or "".
func (s *Service) HandleMove(ctx context.Context, matchID, userID int64, from, to, promo string) error {
	q := db.New(s.pool)
	m, err := q.GetMatch(ctx, matchID)
	if err != nil {
		return err
	}
	if m.Status.String != "active" {
		return ErrMatchNotActive
	}
	if userID != m.Player1ID && userID != m.Player2ID {
		return ErrNotParticipant
	}
	if dbutil.PGInt8ToInt64(m.CurrentTurnUserID) != userID {
		return ErrNotYourTurn
	}

	if s.popIfExpired(matchID) {
		return s.completeMatch(ctx, matchID)
	}

	result, err := s.engine.ApplyMove(matchID, from, to, promo)
	if err != nil {
		return err
	}

	opponentID := m.Player2ID
	if userID == m.Player2ID {
		opponentID = m.Player1ID
	}

	now := time.Now()
	moverClock, err := q.GetMatchClockForUser(ctx, db.GetMatchClockForUserParams{MatchID: matchID, UserID: userID})
	if err != nil {
		return err
	}
	moverRemaining := clock.RemainingAt(clock.State{
		RemainingSeconds: int(moverClock.RemainingSeconds),
		IsCurrentTurn:    true,
		LastStartedAt:    moverClock.LastStartedAt.Time,
	}, now)
	if _, err := q.UpdateMatchClock(ctx, db.UpdateMatchClockParams{
		MatchClockID:     moverClock.MatchClockID,
		RemainingSeconds: int32(moverRemaining),
		IsCurrentTurn:    pgtype.Bool{Bool: false, Valid: true},
		LastStartedAt:    pgtype.Timestamptz{Time: now, Valid: true},
	}); err != nil {
		return err
	}

	opponentClock, err := q.GetMatchClockForUser(ctx, db.GetMatchClockForUserParams{MatchID: matchID, UserID: opponentID})
	if err != nil {
		return err
	}
	if _, err := q.UpdateMatchClock(ctx, db.UpdateMatchClockParams{
		MatchClockID:     opponentClock.MatchClockID,
		RemainingSeconds: opponentClock.RemainingSeconds,
		IsCurrentTurn:    pgtype.Bool{Bool: true, Valid: true},
		LastStartedAt:    pgtype.Timestamptz{Time: now, Valid: true},
	}); err != nil {
		return err
	}

	if result.Outcome == chess.NoOutcome {
		if err := q.SetMatchTurn(ctx, db.SetMatchTurnParams{
			MatchID:           matchID,
			CurrentTurnUserID: dbutil.Int64ToPGInt8(opponentID),
		}); err != nil {
			return err
		}

		s.mu.Lock()
		s.deadlines[matchID] = deadline{
			MatchID:        matchID,
			OpponentUserID: userID,
			At:             now.Add(time.Duration(opponentClock.RemainingSeconds) * time.Second),
		}
		s.mu.Unlock()

		s.broadcastState(ctx, matchID)
		return nil
	}

	winnerID := int64(0)
	switch result.Outcome {
	case chess.WhiteWon:
		winnerID = m.Player1ID
	case chess.BlackWon:
		winnerID = m.Player2ID
	}

	s.mu.Lock()
	delete(s.deadlines, matchID)
	s.mu.Unlock()

	return s.finishMatch(ctx, matchID, winnerID)
}

// HandleResign ends the match with userID's opponent as the winner.
func (s *Service) HandleResign(ctx context.Context, matchID, userID int64) error {
	q := db.New(s.pool)
	m, err := q.GetMatch(ctx, matchID)
	if err != nil {
		return err
	}
	if m.Status.String != "active" {
		return ErrMatchNotActive
	}
	if userID != m.Player1ID && userID != m.Player2ID {
		return ErrNotParticipant
	}

	color := chess.White
	if userID == m.Player2ID {
		color = chess.Black
	}
	_, _ = s.engine.Resign(matchID, color)

	opponentID := m.Player2ID
	if userID == m.Player2ID {
		opponentID = m.Player1ID
	}

	s.mu.Lock()
	delete(s.deadlines, matchID)
	s.mu.Unlock()

	return s.finishMatch(ctx, matchID, opponentID)
}

// popIfExpired removes and reports whether matchID's clock has already run
// out, so a stale move can be rejected in favor of a timeout loss.
func (s *Service) popIfExpired(matchID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.deadlines[matchID]
	if !ok || time.Now().Before(d.At) {
		return false
	}
	delete(s.deadlines, matchID)
	return true
}

// RunClockWatcher periodically checks all in-flight matches for expired
// clocks and completes them as timeout losses, even absent an incoming move.
func (s *Service) RunClockWatcher(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkTimeouts(ctx)
		}
	}
}

func (s *Service) checkTimeouts(ctx context.Context) {
	now := time.Now()

	s.mu.Lock()
	var expired []int64
	for matchID, d := range s.deadlines {
		if now.After(d.At) {
			expired = append(expired, matchID)
		}
	}
	s.mu.Unlock()

	for _, matchID := range expired {
		if err := s.completeMatch(ctx, matchID); err != nil {
			log.Printf("match: timeout completion error for match %d: %v", matchID, err)
		}
	}
}

// completeMatch finishes matchID as a timeout loss for whoever's clock had
// run out, if the deadline entry is still present (idempotent against races
// with a move arriving at the same moment).
func (s *Service) completeMatch(ctx context.Context, matchID int64) error {
	s.mu.Lock()
	d, ok := s.deadlines[matchID]
	if ok {
		delete(s.deadlines, matchID)
	}
	s.mu.Unlock()
	if !ok {
		return nil
	}
	return s.finishMatch(ctx, matchID, d.OpponentUserID)
}

// finishMatch persists the match result, updates both players' Elo/stats,
// and advances the tournament bracket. winnerUserID is 0 for a draw.
func (s *Service) finishMatch(ctx context.Context, matchID, winnerUserID int64) error {
	q := db.New(s.pool)

	m, err := q.GetMatch(ctx, matchID)
	if err != nil {
		return err
	}
	if m.Status.String == "completed" || m.Status.String == "cancelled" {
		return nil // already finished by a concurrent path
	}

	var winnerPG pgtype.Int8
	if winnerUserID != 0 {
		winnerPG = dbutil.Int64ToPGInt8(winnerUserID)
	}
	if _, err := q.CompleteMatch(ctx, db.CompleteMatchParams{MatchID: matchID, WinnerUserID: winnerPG}); err != nil {
		return err
	}
	s.engine.EndGame(matchID)

	if err := s.updateElo(ctx, m.Player1ID, m.Player2ID, winnerUserID); err != nil {
		log.Printf("match: elo update error for match %d: %v", matchID, err)
	}

	s.hub.Broadcast(matchID, wsOutbound{Type: "match_over", Winner: winnerUserID})

	if err := s.advanceBracket(ctx, m.TournamentID, m, winnerUserID); err != nil {
		log.Printf("match: bracket advancement error for tournament %d: %v", m.TournamentID, err)
	}

	return nil
}

func (s *Service) updateElo(ctx context.Context, player1ID, player2ID, winnerUserID int64) error {
	q := db.New(s.pool)

	p1, err := q.GetUserByID(ctx, player1ID)
	if err != nil {
		return err
	}
	p2, err := q.GetUserByID(ctx, player2ID)
	if err != nil {
		return err
	}

	p1Elo := dbutil.PGInt4ToInt(p1.CurrentElo)
	p2Elo := dbutil.PGInt4ToInt(p2.CurrentElo)

	var newP1Elo, newP2Elo int
	var p1Wins, p1Losses, p2Wins, p2Losses int

	switch winnerUserID {
	case player1ID:
		newP1Elo, newP2Elo = elo.UpdatePair(p1Elo, p2Elo)
		p1Wins, p2Losses = 1, 1
	case player2ID:
		newP2Elo, newP1Elo = elo.UpdatePair(p2Elo, p1Elo)
		p2Wins, p1Losses = 1, 1
	default:
		newP1Elo = elo.Update(p1Elo, p2Elo, elo.Draw)
		newP2Elo = elo.Update(p2Elo, p1Elo, elo.Draw)
	}

	if _, err := q.UpdateUserEloAndStats(ctx, db.UpdateUserEloAndStatsParams{
		UserID:      player1ID,
		CurrentElo:  dbutil.IntToPGInt4(newP1Elo),
		TotalWins:   dbutil.IntToPGInt4(dbutil.PGInt4ToInt(p1.TotalWins) + p1Wins),
		TotalLosses: dbutil.IntToPGInt4(dbutil.PGInt4ToInt(p1.TotalLosses) + p1Losses),
		NumGames:    dbutil.IntToPGInt4(dbutil.PGInt4ToInt(p1.NumGames) + 1),
	}); err != nil {
		return err
	}
	if _, err := q.UpdateUserEloAndStats(ctx, db.UpdateUserEloAndStatsParams{
		UserID:      player2ID,
		CurrentElo:  dbutil.IntToPGInt4(newP2Elo),
		TotalWins:   dbutil.IntToPGInt4(dbutil.PGInt4ToInt(p2.TotalWins) + p2Wins),
		TotalLosses: dbutil.IntToPGInt4(dbutil.PGInt4ToInt(p2.TotalLosses) + p2Losses),
		NumGames:    dbutil.IntToPGInt4(dbutil.PGInt4ToInt(p2.NumGames) + 1),
	}); err != nil {
		return err
	}

	return nil
}

// advanceBracket creates the final match once both semifinals are complete,
// and completes the tournament once the final is complete.
//
// Known limitation: a drawn semifinal or final has no tiebreak mechanism
// defined in the ERD, so it will not advance automatically.
func (s *Service) advanceBracket(ctx context.Context, tournamentID int64, completed db.Match, winnerUserID int64) error {
	q := db.New(s.pool)

	if completed.Round == tournament.RoundSemifinal {
		matches, err := q.ListMatchesByTournament(ctx, tournamentID)
		if err != nil {
			return err
		}

		var sf1, sf2 *db.Match
		for i := range matches {
			mm := &matches[i]
			if mm.Round != tournament.RoundSemifinal {
				continue
			}
			if mm.MatchNumber.Int32 == 1 {
				sf1 = mm
			} else if mm.MatchNumber.Int32 == 2 {
				sf2 = mm
			}
		}
		if sf1 == nil || sf2 == nil || sf1.Status.String != "completed" || sf2.Status.String != "completed" {
			return nil
		}
		if !sf1.WinnerUserID.Valid || !sf2.WinnerUserID.Valid {
			return nil // one of the semifinals was a draw; no tiebreak defined
		}

		pairing := tournament.FinalPairing(sf1.WinnerUserID.Int64, sf2.WinnerUserID.Int64)
		_, err = q.CreateMatch(ctx, db.CreateMatchParams{
			TournamentID: tournamentID,
			Round:        pairing.Round,
			MatchNumber:  dbutil.IntToPGInt4(pairing.MatchNumber),
			Player1ID:    pairing.Player1ID,
			Player2ID:    pairing.Player2ID,
		})
		return err
	}

	if completed.Round == tournament.RoundFinal {
		// The payout (or refund) and CompleteTournament must land together:
		// crediting coins without recording the tournament as completed (or
		// vice versa) would leave the ledger and the bracket disagreeing
		// about whether the prize was paid.
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		qtx := db.New(s.pool).WithTx(tx)

		tour, err := qtx.GetTournament(ctx, tournamentID)
		if err != nil {
			return err
		}

		if winnerUserID == 0 {
			// Drawn final: no tiebreak mechanism is defined, so refund each
			// finalist's wager rather than stranding it in a tournament that
			// can never advance.
			for _, uid := range []int64{completed.Player1ID, completed.Player2ID} {
				if _, err := qtx.AdjustUserCoinBalance(ctx, db.AdjustUserCoinBalanceParams{
					UserID: uid,
					Delta:  tour.CoinWager,
				}); err != nil {
					return err
				}
				if _, err := qtx.CreateCoinTransaction(ctx, db.CreateCoinTransactionParams{
					UserID:              uid,
					Amount:              tour.CoinWager,
					TransactionType:     "refund",
					Description:         dbutil.TextToPGText("Wager refund for drawn final"),
					RelatedMatchID:      dbutil.Int64ToPGInt8(completed.MatchID),
					RelatedTournamentID: dbutil.Int64ToPGInt8(tournamentID),
				}); err != nil {
					return err
				}
			}
			if _, err := qtx.CompleteTournament(ctx, db.CompleteTournamentParams{
				TournamentID: tournamentID,
				WinnerUserID: pgtype.Int8{},
			}); err != nil {
				return err
			}
			return tx.Commit(ctx)
		}

		matches, err := qtx.ListMatchesByTournament(ctx, tournamentID)
		if err != nil {
			return err
		}

		var semifinalLosers [2]int64
		idx := 0
		for _, mm := range matches {
			if mm.Round != tournament.RoundSemifinal || !mm.WinnerUserID.Valid {
				continue
			}
			loser := mm.Player1ID
			if mm.WinnerUserID.Int64 == mm.Player1ID {
				loser = mm.Player2ID
			}
			if idx < 2 {
				semifinalLosers[idx] = loser
				idx++
			}
		}

		finalistID := completed.Player1ID
		if winnerUserID == completed.Player1ID {
			finalistID = completed.Player2ID
		}

		results := tournament.FinalResults(winnerUserID, finalistID, semifinalLosers)
		for uid, result := range results {
			if err := qtx.SetTournamentPlayerResult(ctx, db.SetTournamentPlayerResultParams{
				TournamentID: tournamentID,
				UserID:       uid,
				FinalResult:  dbutil.TextToPGText(result),
			}); err != nil {
				return err
			}
		}

		payoutType := "match_winnings"
		if tour.GameMode == "tournament" {
			payoutType = "tournament_winnings"
		}
		if _, err := qtx.AdjustUserCoinBalance(ctx, db.AdjustUserCoinBalanceParams{
			UserID: winnerUserID,
			Delta:  tour.WinnerCoinPrize,
		}); err != nil {
			return err
		}
		if _, err := qtx.CreateCoinTransaction(ctx, db.CreateCoinTransactionParams{
			UserID:              winnerUserID,
			Amount:              tour.WinnerCoinPrize,
			TransactionType:     payoutType,
			Description:         dbutil.TextToPGText("Prize payout"),
			RelatedMatchID:      dbutil.Int64ToPGInt8(completed.MatchID),
			RelatedTournamentID: dbutil.Int64ToPGInt8(tournamentID),
		}); err != nil {
			return err
		}

		if _, err := qtx.CompleteTournament(ctx, db.CompleteTournamentParams{
			TournamentID: tournamentID,
			WinnerUserID: dbutil.Int64ToPGInt8(winnerUserID),
		}); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}

	return nil
}

func (s *Service) broadcastState(ctx context.Context, matchID int64) {
	state, err := s.engine.State(matchID)
	if err != nil {
		s.hub.Broadcast(matchID, wsOutbound{Type: "waiting"})
		return
	}

	s.hub.Broadcast(matchID, wsOutbound{
		Type:    "state",
		FEN:     state.FEN,
		Turn:    colorName(state.Turn),
		Outcome: string(state.Outcome),
		Clocks:  s.clockSnapshot(ctx, matchID),
	})
}

func (s *Service) clockSnapshot(ctx context.Context, matchID int64) map[string]int {
	q := db.New(s.pool)
	rows, err := q.ListMatchClocksByMatch(ctx, matchID)
	if err != nil {
		return nil
	}

	now := time.Now()
	out := make(map[string]int, len(rows))
	for _, r := range rows {
		state := clock.State{
			RemainingSeconds: int(r.RemainingSeconds),
			IsCurrentTurn:    r.IsCurrentTurn.Valid && r.IsCurrentTurn.Bool,
			LastStartedAt:    r.LastStartedAt.Time,
		}
		out[strconv.FormatInt(r.UserID, 10)] = clock.RemainingAt(state, now)
	}
	return out
}

func colorName(c chess.Color) string {
	if c == chess.White {
		return "white"
	}
	return "black"
}
