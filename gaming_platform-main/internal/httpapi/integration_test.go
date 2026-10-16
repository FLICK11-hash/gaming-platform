package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

	"gaming_platform/internal/auth"
	"gaming_platform/internal/domain/achievements"
	"gaming_platform/internal/domain/hub"
	"gaming_platform/internal/domain/leaderboard"
	"gaming_platform/internal/domain/match"
	"gaming_platform/internal/domain/matchhistory"
	"gaming_platform/internal/domain/matchmaking"
	"gaming_platform/internal/domain/missions"
	"gaming_platform/internal/domain/research"
)

// TestEndToEnd_ChessMatchToCheckmate spins up the real server against a
// live Postgres database (already migrated), registers two users, queues
// both for a 1v1 chess match, plays a real game to checkmate over an actual
// websocket connection, and confirms the winner's coin balance increased.
// This exercises the exact path a real player goes through - registration,
// matchmaking, live moves, Elo/coin settlement - in one test, rather than
// only the isolated unit-level functions the rest of the suite covers.
//
// Requires DATABASE_URL to point at an already-migrated Postgres database;
// skips otherwise, since this is an integration test, not a unit test.
func TestEndToEnd_ChessMatchToCheckmate(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping end-to-end integration test")
	}

	ctx := t.Context()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	issuer := auth.NewTokenIssuer("integration-test-secret")
	connHub := hub.New()
	missionsSvc := missions.NewService(pool)
	leaderboardSvc := leaderboard.NewService(pool)
	matchHistorySvc := matchhistory.NewService(pool)
	achievementsSvc := achievements.NewService(pool)
	matchSvc := match.NewService(pool, connHub, missionsSvc, leaderboardSvc, matchHistorySvc, achievementsSvc)
	matcher := matchmaking.New(pool)
	researchRunner := research.NewRunner(research.DefaultGamesPerOpening, "")

	matchCtx, cancelMatch := context.WithCancel(ctx)
	defer cancelMatch()
	go matcher.Run(matchCtx, 2*time.Second)
	go matchSvc.RunClockWatcher(matchCtx)

	server := NewServer(pool, issuer, matchSvc, missionsSvc, leaderboardSvc, matchHistorySvc, achievementsSvc, researchRunner, "conradflick11")
	httpSrv := httptest.NewServer(server.Router())
	defer httpSrv.Close()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	white := registerTestUser(t, httpSrv.URL, "e2e_white_"+suffix)
	black := registerTestUser(t, httpSrv.URL, "e2e_black_"+suffix)

	joinQueue(t, httpSrv.URL, white.Token)
	joinQueue(t, httpSrv.URL, black.Token)

	whiteMatch := waitForMatch(t, httpSrv.URL, white.Token, 10*time.Second)
	blackMatch := waitForMatch(t, httpSrv.URL, black.Token, 10*time.Second)
	if whiteMatch.MatchID != blackMatch.MatchID {
		t.Fatalf("players matched into different matches: %d vs %d", whiteMatch.MatchID, blackMatch.MatchID)
	}
	matchID := whiteMatch.MatchID

	// Player 1 always moves first; figure out which of our two registered
	// users matchmaking seated as player 1, regardless of registration order.
	var firstMover, secondMover *testPlayer
	if whiteMatch.Player1ID == white.UserID {
		firstMover, secondMover = white, black
	} else {
		firstMover, secondMover = black, white
	}

	wsFirst := dialMatchWS(t, httpSrv.URL, matchID, firstMover.Token)
	defer wsFirst.Close()
	wsSecond := dialMatchWS(t, httpSrv.URL, matchID, secondMover.Token)
	defer wsSecond.Close()

	// Broadcast delivers every "state"/"match_over" message to BOTH
	// connections, so confirming progress by alternating which socket we
	// read from is unreliable: each side accumulates the same sequence of
	// broadcasts, and a naive read-until-type-matches can return an
	// already-queued, stale message from an earlier move instead of the
	// fresh one just triggered. Reading every confirmation off a single
	// connection (wsFirst), deduped by FEN change, sidesteps that
	// entirely - moves are still sent from whichever connection is
	// actually authorized to move.
	lastFEN := ""
	waitForNewState := func(timeout time.Duration) map[string]any {
		t.Helper()
		_ = wsFirst.SetReadDeadline(time.Now().Add(timeout))
		for {
			var msg map[string]any
			if err := wsFirst.ReadJSON(&msg); err != nil {
				t.Fatalf("waiting for state update: %v", err)
			}
			if errMsg, ok := msg["error"]; ok {
				t.Fatalf("server returned error waiting for state update: %v", errMsg)
			}
			if msg["type"] != "state" {
				continue
			}
			fen, _ := msg["fen"].(string)
			if fen != lastFEN {
				lastFEN = fen
				return msg
			}
		}
	}
	waitForMatchOver := func(timeout time.Duration) map[string]any {
		t.Helper()
		_ = wsFirst.SetReadDeadline(time.Now().Add(timeout))
		for {
			var msg map[string]any
			if err := wsFirst.ReadJSON(&msg); err != nil {
				t.Fatalf("waiting for match_over: %v", err)
			}
			if errMsg, ok := msg["error"]; ok {
				t.Fatalf("server returned error waiting for match_over: %v", errMsg)
			}
			if msg["type"] == "match_over" {
				return msg
			}
		}
	}

	// Both players are connected: wait out the pre-match countdown before
	// the match accepts moves (HandleMove rejects anything before status
	// flips to "active", which happens after match.PreMatchCountdown).
	waitForNewState(8 * time.Second)

	// Fool's Mate: 1. f3 e5 2. g4 Qh4# - the fastest possible checkmate,
	// letting this test reach a real decisive result deterministically.
	sendMove(t, wsFirst, "f2", "f3")
	waitForNewState(3 * time.Second)

	sendMove(t, wsSecond, "e7", "e5")
	waitForNewState(3 * time.Second)

	sendMove(t, wsFirst, "g2", "g4")
	waitForNewState(3 * time.Second)

	sendMove(t, wsSecond, "d8", "h4")
	over := waitForMatchOver(3 * time.Second)

	winnerUserID := int64(over["winner_user_id"].(float64))
	if winnerUserID != secondMover.UserID {
		t.Fatalf("expected black (user %d) to win by checkmate, server reported winner %d", secondMover.UserID, winnerUserID)
	}

	// Coin settlement happens synchronously inside finishMatch before
	// match_over is broadcast, but allow a brief moment for the response
	// to land regardless.
	time.Sleep(200 * time.Millisecond)

	finalBalance := getCoinBalance(t, httpSrv.URL, secondMover.Token)
	if finalBalance <= secondMover.StartingBalance {
		t.Fatalf("winner's coin balance did not increase: started at %d, now %d", secondMover.StartingBalance, finalBalance)
	}
	t.Logf("winner's coin balance: %d -> %d", secondMover.StartingBalance, finalBalance)

	loserBalance := getCoinBalance(t, httpSrv.URL, firstMover.Token)
	if loserBalance >= firstMover.StartingBalance {
		t.Fatalf("loser's coin balance did not decrease: started at %d, now %d", firstMover.StartingBalance, loserBalance)
	}
}

type testPlayer struct {
	Username        string
	Token           string
	UserID          int64
	StartingBalance int64
}

func registerTestUser(t *testing.T, baseURL, username string) *testPlayer {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"email":    username + "@example.com",
		"password": "password123",
	})
	resp, err := http.Post(baseURL+"/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("register %s: %v", username, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("register %s: unexpected status %d", username, resp.StatusCode)
	}
	var parsed struct {
		Token string `json:"token"`
		User  struct {
			UserID      int64 `json:"user_id"`
			CoinBalance int64 `json:"coin_balance"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		t.Fatalf("decode register response for %s: %v", username, err)
	}
	return &testPlayer{
		Username:        username,
		Token:           parsed.Token,
		UserID:          parsed.User.UserID,
		StartingBalance: parsed.User.CoinBalance,
	}
}

func joinQueue(t *testing.T, baseURL, token string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"game_type": "chess",
		"game_mode": "one_vs_one",
		"min_elo":   800,
		"max_elo":   2000,
	})
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/matchmaking/queue", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("join queue: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("join queue: unexpected status %d", resp.StatusCode)
	}
}

type matchInfo struct {
	MatchID   int64
	Player1ID int64
}

func waitForMatch(t *testing.T, baseURL, token string, timeout time.Duration) matchInfo {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/matches/mine", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("poll /matches/mine: %v", err)
		}
		if resp.StatusCode == http.StatusOK {
			var parsed struct {
				MatchID   int64 `json:"match_id"`
				Player1ID int64 `json:"player_1_id"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
				resp.Body.Close()
				t.Fatalf("decode /matches/mine: %v", err)
			}
			resp.Body.Close()
			return matchInfo{MatchID: parsed.MatchID, Player1ID: parsed.Player1ID}
		}
		resp.Body.Close()
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("no match found for player within %s", timeout)
	return matchInfo{}
}

func dialMatchWS(t *testing.T, baseURL string, matchID int64, token string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(baseURL, "http") + fmt.Sprintf("/matches/%d/ws?token=%s", matchID, token)
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial match websocket: %v", err)
	}
	if resp != nil {
		resp.Body.Close()
	}
	return conn
}

func sendMove(t *testing.T, conn *websocket.Conn, from, to string) {
	t.Helper()
	if err := conn.WriteJSON(map[string]string{"type": "move", "from": from, "to": to}); err != nil {
		t.Fatalf("send move %s-%s: %v", from, to, err)
	}
}

func getCoinBalance(t *testing.T, baseURL, token string) int64 {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get /me: %v", err)
	}
	defer resp.Body.Close()
	var parsed struct {
		CoinBalance int64 `json:"coin_balance"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		t.Fatalf("decode /me: %v", err)
	}
	return parsed.CoinBalance
}
