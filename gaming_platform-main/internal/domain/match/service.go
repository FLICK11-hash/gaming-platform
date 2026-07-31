// Package match orchestrates a single match's lifecycle: starting once both
// players connect, applying moves, running the bullet clocks, resigning,
// completing on a decisive result/draw/timeout, updating Elo, and advancing
// the tournament bracket. It hosts two game types - chess and Connect Four -
// dispatching to the matching engine per match while keeping the clock,
// disconnect-grace-period, countdown, and bracket logic shared between them.
package match

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
	"gaming_platform/internal/domain/achievements"
	"gaming_platform/internal/domain/bot"
	"gaming_platform/internal/domain/chess"
	"gaming_platform/internal/domain/clock"
	"gaming_platform/internal/domain/connectfour"
	"gaming_platform/internal/domain/elo"
	"gaming_platform/internal/domain/hub"
	"gaming_platform/internal/domain/leaderboard"
	"gaming_platform/internal/domain/matchhistory"
	"gaming_platform/internal/domain/missions"
	"gaming_platform/internal/domain/tournament"
)

var (
	ErrNotParticipant = errors.New("match: user is not a participant in this match")
	ErrNotYourTurn    = errors.New("match: not your turn")
	ErrMatchNotActive = errors.New("match: match is not active")
)

// GameTypeConnectFour identifies a Connect Four match. Anything else
// (including the empty string, for defensiveness) is treated as chess.
const GameTypeConnectFour = "connect_four"

// MoveInput carries the game-specific fields for a single move: From/To/
// Promotion for chess, Column for Connect Four.
type MoveInput struct {
	From      string
	To        string
	Promotion string
	Column    *int
	// IsPremove marks a move that was queued during the opponent's turn and
	// auto-fired the instant it became the mover's turn; it costs the mover
	// PremoveCostSeconds off their own clock.
	IsPremove bool
}

type deadline struct {
	MatchID        int64
	OpponentUserID int64
	At             time.Time
}

// DisconnectGracePeriod is how long a player has to reconnect to an active
// match's websocket before they forfeit it to their opponent.
const DisconnectGracePeriod = 5 * time.Second

// PreMatchCountdown is how long players see a "get ready" countdown after
// both have connected, before the match clocks actually start.
const PreMatchCountdown = 5 * time.Second

// PremoveCostSeconds is deducted from a player's own clock whenever they
// submit a move flagged as a pre-move (queued during the opponent's turn
// and auto-fired the instant it becomes theirs), trading a small amount of
// time for not having to react in the moment.
const PremoveCostSeconds = 0.2

// secondsToDuration converts a fractional-second clock value into a
// time.Duration without losing the fractional part to integer truncation.
func secondsToDuration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}

type disconnectDeadline struct {
	OpponentUserID int64
	At             time.Time
}

type Service struct {
	pool            *pgxpool.Pool
	chessEngine     *chess.Engine
	c4Engine        *connectfour.Engine
	hub             *hub.Hub
	missionsSvc     *missions.Service
	leaderboardSvc  *leaderboard.Service
	matchHistorySvc *matchhistory.Service
	achievementsSvc *achievements.Service

	mu           sync.Mutex
	deadlines    map[int64]deadline
	disconnects  map[int64]map[int64]disconnectDeadline // matchID -> userID -> deadline
	countingDown map[int64]bool
}

func NewService(pool *pgxpool.Pool, h *hub.Hub, missionsSvc *missions.Service, leaderboardSvc *leaderboard.Service, matchHistorySvc *matchhistory.Service, achievementsSvc *achievements.Service) *Service {
	return &Service{
		pool:            pool,
		chessEngine:     chess.NewEngine(),
		c4Engine:        connectfour.NewEngine(),
		hub:             h,
		missionsSvc:     missionsSvc,
		leaderboardSvc:  leaderboardSvc,
		matchHistorySvc: matchHistorySvc,
		achievementsSvc: achievementsSvc,
		deadlines:       make(map[int64]deadline),
		disconnects:     make(map[int64]map[int64]disconnectDeadline),
		countingDown:    make(map[int64]bool),
	}
}

// SendError delivers an error reply to a single connected player's websocket,
// going through the hub's per-connection write lock so it can never race with
// a concurrent Broadcast to the same connection.
func (s *Service) SendError(matchID, userID int64, message string) {
	_ = s.hub.SendTo(matchID, userID, map[string]string{"type": "error", "error": message})
}

// wsOutbound is broadcast/sent to clients over the match websocket.
type wsOutbound struct {
	Type          string         `json:"type"`
	FEN           string         `json:"fen,omitempty"`
	Board         [][]int        `json:"board,omitempty"`
	Turn          string         `json:"turn,omitempty"`
	Outcome       string         `json:"outcome,omitempty"`
	Winner        int64          `json:"winner_user_id,omitempty"`
	Clocks        map[string]int `json:"clocks,omitempty"`
	Seconds       int            `json:"seconds,omitempty"`
	WinningSquare string         `json:"winning_square,omitempty"`
	WinningCells  [][2]int       `json:"winning_cells,omitempty"`
	Reason        string         `json:"reason,omitempty"`
	Error         string         `json:"error,omitempty"`
}

// MatchOverReasonC4TimeTiebreak marks a match_over message where a Connect
// Four board filled up with nobody connecting four, so the win was decided
// by whoever had more clock time left rather than an actual connection.
const MatchOverReasonC4TimeTiebreak = "c4_time_tiebreak"

// moveHighlight identifies what to visually call out in the final position
// of a just-finished game: the checkmating square in chess, or the winning
// line's cells in Connect Four. Zero value means nothing to highlight.
type moveHighlight struct {
	Square string
	Cells  [][2]int
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

	if s.clearDisconnect(matchID, userID) && m.Status.String == "active" {
		s.hub.Broadcast(matchID, wsOutbound{Type: "opponent_reconnected"})
	}

	if m.Status.String == "waiting" {
		opponentID := m.Player2ID
		if userID == m.Player2ID {
			opponentID = m.Player1ID
		}

		// A bot opponent never opens a websocket, so it can never bring
		// ConnectedCount to 2; start the countdown as soon as the human
		// joins instead of waiting for a connection that will never come.
		opponentIsBot, err := s.isBot(ctx, opponentID)
		if err != nil {
			return err
		}

		if opponentIsBot || s.hub.ConnectedCount(matchID) >= 2 {
			s.beginCountdown(m)
			return nil
		}
	}

	s.broadcastState(ctx, m, moveHighlight{})
	return nil
}

// beginCountdown announces a get-ready countdown to both players once they've
// both connected, then starts the match after PreMatchCountdown elapses. It
// is idempotent per match so a reconnect during the countdown can't restart
// or duplicate it.
func (s *Service) beginCountdown(m db.Match) {
	s.mu.Lock()
	if s.countingDown[m.MatchID] {
		s.mu.Unlock()
		return
	}
	s.countingDown[m.MatchID] = true
	s.mu.Unlock()

	s.hub.Broadcast(m.MatchID, wsOutbound{
		Type:    "countdown",
		Seconds: int(PreMatchCountdown.Seconds()),
	})

	go func() {
		time.Sleep(PreMatchCountdown)

		s.mu.Lock()
		delete(s.countingDown, m.MatchID)
		s.mu.Unlock()

		if err := s.startMatch(context.Background(), m); err != nil {
			log.Printf("match: countdown start error for match %d: %v", m.MatchID, err)
		}
	}()
}

// Leave unregisters a player's websocket connection (e.g. on disconnect) and,
// if the match is still active, starts a grace period after which the
// disconnected player forfeits to their opponent if they haven't reconnected.
func (s *Service) Leave(ctx context.Context, matchID, userID int64, ws *websocket.Conn) {
	s.hub.Unregister(matchID, userID, ws)

	q := db.New(s.pool)
	m, err := q.GetMatch(ctx, matchID)
	if err != nil || m.Status.String != "active" {
		return
	}
	if userID != m.Player1ID && userID != m.Player2ID {
		return
	}

	opponentID := m.Player2ID
	if userID == m.Player2ID {
		opponentID = m.Player1ID
	}

	s.mu.Lock()
	if s.disconnects[matchID] == nil {
		s.disconnects[matchID] = make(map[int64]disconnectDeadline)
	}
	s.disconnects[matchID][userID] = disconnectDeadline{
		OpponentUserID: opponentID,
		At:             time.Now().Add(DisconnectGracePeriod),
	}
	s.mu.Unlock()

	s.hub.Broadcast(matchID, wsOutbound{Type: "opponent_disconnected"})
}

// clearDisconnect cancels any pending disconnect grace period for userID in
// matchID, reporting whether one was cancelled.
func (s *Service) clearDisconnect(matchID, userID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	users, ok := s.disconnects[matchID]
	if !ok {
		return false
	}
	if _, ok := users[userID]; !ok {
		return false
	}
	delete(users, userID)
	if len(users) == 0 {
		delete(s.disconnects, matchID)
	}
	return true
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
		MatchID:          m.MatchID,
		UserID:           m.Player1ID,
		StartingSeconds:  clock.DefaultStartingSeconds,
		RemainingSeconds: clock.DefaultStartingSeconds,
		IsCurrentTurn:    pgtype.Bool{Bool: true, Valid: true},
		LastStartedAt:    nowPG,
	}); err != nil {
		return err
	}
	if _, err := q.CreateMatchClock(ctx, db.CreateMatchClockParams{
		MatchID:          m.MatchID,
		UserID:           m.Player2ID,
		StartingSeconds:  clock.DefaultStartingSeconds,
		RemainingSeconds: clock.DefaultStartingSeconds,
		IsCurrentTurn:    pgtype.Bool{Bool: false, Valid: true},
		LastStartedAt:    nowPG,
	}); err != nil {
		return err
	}

	if m.GameType == GameTypeConnectFour {
		s.c4Engine.StartGame(m.MatchID)
	} else {
		s.chessEngine.StartGame(m.MatchID)
	}

	s.mu.Lock()
	s.deadlines[m.MatchID] = deadline{
		MatchID:        m.MatchID,
		OpponentUserID: m.Player2ID,
		At:             now.Add(clock.DefaultStartingSeconds * time.Second),
	}
	s.mu.Unlock()

	s.broadcastState(ctx, m, moveHighlight{})
	s.triggerBotMoveIfNeeded(m.MatchID)
	return nil
}

// isBot reports whether userID belongs to one of the fixed bot accounts
// seeded by migration 0009_bot_users.up.sql.
func (s *Service) isBot(ctx context.Context, userID int64) (bool, error) {
	q := db.New(s.pool)
	u, err := q.GetUserByID(ctx, userID)
	if err != nil {
		return false, err
	}
	return u.IsBot, nil
}

// triggerBotMoveIfNeeded checks whether it's currently a bot's turn in
// matchID and, if so, asynchronously plays that bot's move after a short
// "thinking" delay. It runs on its own background context rather than the
// caller's, since the caller (an HTTP/WS handler) may return and cancel its
// context long before the delay elapses.
func (s *Service) triggerBotMoveIfNeeded(matchID int64) {
	go func() {
		ctx := context.Background()
		q := db.New(s.pool)

		m, err := q.GetMatch(ctx, matchID)
		if err != nil || m.Status.String != "active" {
			return
		}

		moverID := dbutil.PGInt8ToInt64(m.CurrentTurnUserID)
		if moverID == 0 {
			return
		}

		mover, err := q.GetUserByID(ctx, moverID)
		if err != nil || !mover.IsBot {
			return
		}

		time.Sleep(botThinkDelay())

		input, err := s.chooseBotMove(m, mover)
		if err != nil {
			log.Printf("match: bot move-selection error for match %d: %v", matchID, err)
			return
		}

		if err := s.HandleMove(ctx, matchID, moverID, input); err != nil {
			log.Printf("match: bot move error for match %d: %v", matchID, err)
		}
	}()
}

// botThinkDelay returns a short randomized pause before a bot plays its
// move, so it doesn't feel like it's responding instantaneously.
func botThinkDelay() time.Duration {
	return 400*time.Millisecond + time.Duration(rand.Intn(900))*time.Millisecond
}

// chooseBotMove asks the bot package for botUser's move given m's current
// live engine state.
func (s *Service) chooseBotMove(m db.Match, botUser db.User) (MoveInput, error) {
	difficulty := bot.DifficultyForElo(dbutil.PGInt4ToInt(botUser.CurrentElo))

	if m.GameType == GameTypeConnectFour {
		state, err := s.c4Engine.State(m.MatchID)
		if err != nil {
			return MoveInput{}, err
		}
		column, err := bot.ChooseConnectFourMove(state.Board, state.Turn, difficulty)
		if err != nil {
			return MoveInput{}, err
		}
		return MoveInput{Column: &column}, nil
	}

	state, err := s.chessEngine.State(m.MatchID)
	if err != nil {
		return MoveInput{}, err
	}
	from, to, promo, err := bot.ChooseChessMoveEngine(state.FEN, difficulty)
	if err != nil {
		return MoveInput{}, err
	}
	return MoveInput{From: from, To: to, Promotion: promo}, nil
}

// HandleMove validates and applies a move from userID: From/To/Promotion for
// chess, or Column for Connect Four, depending on the match's game type.
func (s *Service) HandleMove(ctx context.Context, matchID, userID int64, input MoveInput) error {
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

	opponentID := m.Player2ID
	if userID == m.Player2ID {
		opponentID = m.Player1ID
	}

	// Capture the opening move before it's applied - the live game state
	// this reads from is discarded once the match completes (see
	// chess.Engine.EndGame), so this is the only chance to persist it for
	// the profile's favorite-opening stat.
	isFirstChessMove := false
	if m.GameType != GameTypeConnectFour {
		if state, err := s.chessEngine.State(matchID); err == nil && state.FEN == chess.StartFEN {
			isFirstChessMove = true
		}
	}

	finished, winnerID, highlight, err := s.applyMove(m, userID, input)
	if err != nil {
		return err
	}

	if isFirstChessMove && s.matchHistorySvc != nil {
		s.matchHistorySvc.RecordFirstMove(ctx, matchID, input.From, input.To)
	}

	now := time.Now()
	// One round trip for both clocks instead of a per-player read, so the
	// clock reads for a bullet game (60s total per side) don't pile up
	// database latency the way two separate GetMatchClockForUser calls did.
	clocks, err := q.ListMatchClocksByMatch(ctx, matchID)
	if err != nil {
		return err
	}
	var moverClock, opponentClock db.MatchClock
	for _, c := range clocks {
		switch c.UserID {
		case userID:
			moverClock = c
		case opponentID:
			opponentClock = c
		}
	}
	moverRemaining := clock.RemainingAt(clock.State{
		RemainingSeconds: moverClock.RemainingSeconds,
		IsCurrentTurn:    true,
		LastStartedAt:    moverClock.LastStartedAt.Time,
	}, now)
	if input.IsPremove {
		moverRemaining -= PremoveCostSeconds
		if moverRemaining < 0 {
			moverRemaining = 0
		}
	}
	// Both clocks' post-move state (mover's new remaining time, the turn
	// handoff) lands in a single write instead of two.
	if err := q.ApplyMoveClocks(ctx, db.ApplyMoveClocksParams{
		MatchID:               matchID,
		MoverUserID:           userID,
		MoverRemainingSeconds: moverRemaining,
		OpponentUserID:        opponentID,
		LastStartedAt:         pgtype.Timestamptz{Time: now, Valid: true},
	}); err != nil {
		return err
	}

	if !finished {
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
			At:             now.Add(secondsToDuration(opponentClock.RemainingSeconds)),
		}
		s.mu.Unlock()

		s.broadcastState(ctx, m, moveHighlight{})
		s.triggerBotMoveIfNeeded(matchID)
		return nil
	}

	// A Connect Four game that ends with the board full and nobody
	// connecting four isn't a true draw: whoever has more time left on
	// their clock at that moment wins, so every game reaches a winner.
	// moverRemaining and opponentClock.RemainingSeconds are both snapshots
	// as of `now`, so they're directly comparable.
	reason := ""
	if winnerID == 0 && m.GameType == GameTypeConnectFour {
		winnerID = winnerByRemainingTime(
			userID, moverRemaining,
			opponentID, opponentClock.RemainingSeconds,
		)
		reason = MatchOverReasonC4TimeTiebreak
	}

	s.mu.Lock()
	delete(s.deadlines, matchID)
	s.mu.Unlock()

	// Broadcast the final position (with the winning move/line highlighted)
	// before match_over, so both players actually see how the game ended
	// instead of jumping straight from the prior position to a bare
	// "match over" message - previously only the player who made the move
	// had already seen it applied locally; the opponent never received it.
	s.broadcastState(ctx, m, highlight)

	return s.finishMatch(ctx, matchID, winnerID, reason)
}

// applyMove dispatches a move to the correct game engine and reports whether
// the game finished and, if so, who won (0 for a draw), plus what to
// highlight in the final position if it did.
func (s *Service) applyMove(m db.Match, userID int64, input MoveInput) (finished bool, winnerUserID int64, highlight moveHighlight, err error) {
	if m.GameType == GameTypeConnectFour {
		return s.applyConnectFourMove(m, userID, input)
	}
	return s.applyChessMove(m, input)
}

func (s *Service) applyChessMove(m db.Match, input MoveInput) (bool, int64, moveHighlight, error) {
	result, err := s.chessEngine.ApplyMove(m.MatchID, input.From, input.To, input.Promotion)
	if err != nil {
		return false, 0, moveHighlight{}, err
	}
	if result.Outcome == chess.NoOutcome {
		return false, 0, moveHighlight{}, nil
	}

	winnerID := int64(0)
	switch result.Outcome {
	case chess.WhiteWon:
		winnerID = m.Player1ID
	case chess.BlackWon:
		winnerID = m.Player2ID
	}

	var highlight moveHighlight
	if winnerID != 0 {
		highlight.Square = input.To
	}
	return true, winnerID, highlight, nil
}

func (s *Service) applyConnectFourMove(m db.Match, userID int64, input MoveInput) (bool, int64, moveHighlight, error) {
	if input.Column == nil {
		return false, 0, moveHighlight{}, errors.New("match: column is required for connect four moves")
	}

	color := connectfour.Red
	if userID == m.Player2ID {
		color = connectfour.Green
	}

	result, err := s.c4Engine.DropChip(m.MatchID, *input.Column, color)
	if err != nil {
		return false, 0, moveHighlight{}, err
	}

	switch result.Outcome {
	case connectfour.NoOutcome:
		return false, 0, moveHighlight{}, nil
	case connectfour.RedWon:
		return true, m.Player1ID, moveHighlight{Cells: result.WinningCells}, nil
	case connectfour.GreenWon:
		return true, m.Player2ID, moveHighlight{Cells: result.WinningCells}, nil
	default: // Draw: the board is full with nobody connecting four.
		return true, 0, moveHighlight{}, nil
	}
}

// winnerByRemainingTime picks whichever player has strictly more time left
// on their clock, or 0 (a true draw) on an exact tie.
func winnerByRemainingTime(userA int64, remainingA float64, userB int64, remainingB float64) int64 {
	switch {
	case remainingA > remainingB:
		return userA
	case remainingB > remainingA:
		return userB
	default:
		return 0
	}
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

	if m.GameType != GameTypeConnectFour {
		color := chess.White
		if userID == m.Player2ID {
			color = chess.Black
		}
		_, _ = s.chessEngine.Resign(matchID, color)
	}

	opponentID := m.Player2ID
	if userID == m.Player2ID {
		opponentID = m.Player1ID
	}

	s.mu.Lock()
	delete(s.deadlines, matchID)
	s.mu.Unlock()

	return s.finishMatch(ctx, matchID, opponentID, "")
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
// It also checks for players who failed to reconnect within the disconnect
// grace period and forfeits the match to their opponent.
func (s *Service) RunClockWatcher(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkTimeouts(ctx)
			s.checkDisconnects(ctx)
		}
	}
}

func (s *Service) checkDisconnects(ctx context.Context) {
	now := time.Now()

	type forfeit struct {
		MatchID        int64
		OpponentUserID int64
	}
	var expired []forfeit

	s.mu.Lock()
	for matchID, users := range s.disconnects {
		for userID, d := range users {
			if now.After(d.At) {
				expired = append(expired, forfeit{MatchID: matchID, OpponentUserID: d.OpponentUserID})
				delete(users, userID)
			}
		}
		if len(users) == 0 {
			delete(s.disconnects, matchID)
		}
	}
	s.mu.Unlock()

	for _, f := range expired {
		if err := s.finishMatch(ctx, f.MatchID, f.OpponentUserID, ""); err != nil {
			log.Printf("match: disconnect forfeit error for match %d: %v", f.MatchID, err)
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
	return s.finishMatch(ctx, matchID, d.OpponentUserID, "")
}

// finishMatch persists the match result, updates both players' Elo/stats,
// and advances the tournament bracket. winnerUserID is 0 for a draw.
func (s *Service) finishMatch(ctx context.Context, matchID, winnerUserID int64, reason string) error {
	q := db.New(s.pool)

	var winnerPG pgtype.Int8
	if winnerUserID != 0 {
		winnerPG = dbutil.Int64ToPGInt8(winnerUserID)
	}
	// CompleteMatch only affects a row still in status 'active', so this
	// doubles as the concurrency guard: if a checkmate detected from a move
	// races a clock-watcher timeout (or a resign races a disconnect
	// forfeit) for the same match, only the first caller gets a row back -
	// the second gets pgx.ErrNoRows and stands down instead of running Elo
	// and prize payout a second time for the same result.
	m, err := q.CompleteMatch(ctx, db.CompleteMatchParams{MatchID: matchID, WinnerUserID: winnerPG})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // already finished by a concurrent path
		}
		return err
	}
	if m.GameType == GameTypeConnectFour {
		s.c4Engine.EndGame(matchID)
	} else {
		s.chessEngine.EndGame(matchID)
	}

	s.mu.Lock()
	delete(s.disconnects, matchID)
	s.mu.Unlock()

	if err := s.updateElo(ctx, matchID, m.Player1ID, m.Player2ID, winnerUserID); err != nil {
		log.Printf("match: elo update error for match %d: %v", matchID, err)
	}

	s.hub.Broadcast(matchID, wsOutbound{Type: "match_over", Winner: winnerUserID, Reason: reason})

	if err := s.advanceBracket(ctx, m.TournamentID, m, winnerUserID); err != nil {
		log.Printf("match: bracket advancement error for tournament %d: %v", m.TournamentID, err)
	}

	return nil
}

func (s *Service) updateElo(ctx context.Context, matchID, player1ID, player2ID, winnerUserID int64) error {
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

	// Bots' Elo is a fixed difficulty label (see migration
	// 0009_bot_users.up.sql), not a real rating: their current p1Elo/p2Elo
	// is still used as an input above so the human's rating moves correctly,
	// but the bot's own row is never written back.
	if !p1.IsBot {
		if _, err := q.UpdateUserEloAndStats(ctx, db.UpdateUserEloAndStatsParams{
			UserID:      player1ID,
			CurrentElo:  dbutil.IntToPGInt4(newP1Elo),
			TotalWins:   dbutil.IntToPGInt4(dbutil.PGInt4ToInt(p1.TotalWins) + p1Wins),
			TotalLosses: dbutil.IntToPGInt4(dbutil.PGInt4ToInt(p1.TotalLosses) + p1Losses),
			NumGames:    dbutil.IntToPGInt4(dbutil.PGInt4ToInt(p1.NumGames) + 1),
		}); err != nil {
			return err
		}
		if s.matchHistorySvc != nil {
			s.matchHistorySvc.RecordEloSnapshot(ctx, player1ID, matchID, p1Elo, newP1Elo)
		}
	}
	if !p2.IsBot {
		if _, err := q.UpdateUserEloAndStats(ctx, db.UpdateUserEloAndStatsParams{
			UserID:      player2ID,
			CurrentElo:  dbutil.IntToPGInt4(newP2Elo),
			TotalWins:   dbutil.IntToPGInt4(dbutil.PGInt4ToInt(p2.TotalWins) + p2Wins),
			TotalLosses: dbutil.IntToPGInt4(dbutil.PGInt4ToInt(p2.TotalLosses) + p2Losses),
			NumGames:    dbutil.IntToPGInt4(dbutil.PGInt4ToInt(p2.NumGames) + 1),
		}); err != nil {
			return err
		}
		if s.matchHistorySvc != nil {
			s.matchHistorySvc.RecordEloSnapshot(ctx, player2ID, matchID, p2Elo, newP2Elo)
		}
	}

	if s.missionsSvc != nil {
		s.missionsSvc.RecordMatchCompletion(ctx, player1ID, player2ID, winnerUserID, p1Elo, p2Elo)
	}

	if s.leaderboardSvc != nil {
		s.leaderboardSvc.RecordMatchResult(ctx, player1ID, player2ID, winnerUserID)
	}

	if s.achievementsSvc != nil {
		if !p1.IsBot {
			s.achievementsSvc.CheckMatchAchievements(ctx, player1ID, dbutil.PGInt4ToInt(p1.TotalWins)+p1Wins, newP1Elo)
		}
		if !p2.IsBot {
			s.achievementsSvc.CheckMatchAchievements(ctx, player2ID, dbutil.PGInt4ToInt(p2.TotalWins)+p2Wins, newP2Elo)
		}
	}

	return nil
}

// advanceBracket creates the final match once both semifinals are complete,
// and completes the tournament once the final is complete.
//
// Known limitation: a drawn semifinal or final has no tiebreak mechanism
// defined in the ERD, so it will not advance automatically.
func (s *Service) advanceBracket(ctx context.Context, tournamentID int64, completed db.Match, winnerUserID int64) error {
	// 4-player tournament mode is disabled for now (too much work to keep
	// fully working right now) - may come back in the future. No match ever
	// has Round == RoundSemifinal while it's disabled (createOneVsOneMatch
	// always creates Round "final"), so this is unreachable; left in place,
	// commented out, for an easy re-enable (re-add `q := db.New(s.pool)` too).
	//
	// if completed.Round == tournament.RoundSemifinal {
	// 	matches, err := q.ListMatchesByTournament(ctx, tournamentID)
	// 	if err != nil {
	// 		return err
	// 	}
	//
	// 	var sf1, sf2 *db.Match
	// 	for i := range matches {
	// 		mm := &matches[i]
	// 		if mm.Round != tournament.RoundSemifinal {
	// 			continue
	// 		}
	// 		if mm.MatchNumber.Int32 == 1 {
	// 			sf1 = mm
	// 		} else if mm.MatchNumber.Int32 == 2 {
	// 			sf2 = mm
	// 		}
	// 	}
	// 	if sf1 == nil || sf2 == nil || sf1.Status.String != "completed" || sf2.Status.String != "completed" {
	// 		return nil
	// 	}
	// 	if !sf1.WinnerUserID.Valid || !sf2.WinnerUserID.Valid {
	// 		return nil // one of the semifinals was a draw; no tiebreak defined
	// 	}
	//
	// 	pairing := tournament.FinalPairing(sf1.WinnerUserID.Int64, sf2.WinnerUserID.Int64)
	// 	_, err = q.CreateMatch(ctx, db.CreateMatchParams{
	// 		TournamentID: tournamentID,
	// 		GameType:     completed.GameType,
	// 		Round:        pairing.Round,
	// 		MatchNumber:  dbutil.IntToPGInt4(pairing.MatchNumber),
	// 		Player1ID:    pairing.Player1ID,
	// 		Player2ID:    pairing.Player2ID,
	// 	})
	// 	return err
	// }

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

		tournamentWins, err := qtx.CountTournamentWinsForUser(ctx, dbutil.Int64ToPGInt8(winnerUserID))
		if err != nil {
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		if s.achievementsSvc != nil {
			s.achievementsSvc.CheckTournamentAchievement(ctx, winnerUserID, int(tournamentWins))
		}
		return nil
	}

	return nil
}

func (s *Service) broadcastState(ctx context.Context, m db.Match, highlight moveHighlight) {
	if m.GameType == GameTypeConnectFour {
		state, err := s.c4Engine.State(m.MatchID)
		if err != nil {
			s.hub.Broadcast(m.MatchID, wsOutbound{Type: "waiting"})
			return
		}
		s.hub.Broadcast(m.MatchID, wsOutbound{
			Type:         "state",
			Board:        boardRows(state.Board),
			Turn:         connectFourColorName(state.Turn),
			Outcome:      connectFourOutcomeName(state.Outcome),
			Clocks:       s.clockSnapshot(ctx, m.MatchID),
			WinningCells: highlight.Cells,
		})
		return
	}

	state, err := s.chessEngine.State(m.MatchID)
	if err != nil {
		s.hub.Broadcast(m.MatchID, wsOutbound{Type: "waiting"})
		return
	}

	s.hub.Broadcast(m.MatchID, wsOutbound{
		Type:          "state",
		FEN:           state.FEN,
		Turn:          colorName(state.Turn),
		Outcome:       string(state.Outcome),
		Clocks:        s.clockSnapshot(ctx, m.MatchID),
		WinningSquare: highlight.Square,
	})
}

func boardRows(board connectfour.Board) [][]int {
	rows := make([][]int, connectfour.Rows)
	for r := range board {
		rows[r] = append([]int(nil), board[r][:]...)
	}
	return rows
}

func connectFourColorName(c connectfour.Color) string {
	if c == connectfour.Red {
		return "red"
	}
	return "green"
}

func connectFourOutcomeName(o connectfour.Outcome) string {
	switch o {
	case connectfour.RedWon:
		return "red_won"
	case connectfour.GreenWon:
		return "green_won"
	case connectfour.Draw:
		return "draw"
	default:
		return ""
	}
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
			RemainingSeconds: r.RemainingSeconds,
			IsCurrentTurn:    r.IsCurrentTurn.Valid && r.IsCurrentTurn.Bool,
			LastStartedAt:    r.LastStartedAt.Time,
		}
		// Clients display whole seconds; fractional pre-move costs still
		// affect this via accumulation and, more importantly, timeouts.
		out[strconv.FormatInt(r.UserID, 10)] = int(clock.RemainingAt(state, now))
	}
	return out
}

func colorName(c chess.Color) string {
	if c == chess.White {
		return "white"
	}
	return "black"
}
