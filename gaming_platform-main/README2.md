# Verifying Chess Gameplay

Steps to confirm the live chess board works end to end.

## 1. Start the backend

```powershell
go run ./cmd/server
```

Leave it running on port 8080.

## 2. Start the frontend

In a separate terminal, from the project root:

```powershell
npm install
npm run dev
```

Open the printed URL (defaults to `http://localhost:5173`). The Vite dev server proxies `/api/*` (including websockets) to the Go backend.

## 3. Get four players queued

Chess matches are only created once a tournament forms, which needs 4 players in the same queue bucket (same `game_type` + `entry_fee`). To test with fewer humans, open 4 browser sessions that don't share a login (e.g. 2 normal windows + 2 incognito/private windows, or 4 different browsers) so each can hold its own token in `localStorage`.

For each of the 4 sessions:

1. Register a new account (or log in if you already made one).
2. Click **Join Queue** on the dashboard.

Once all 4 are queued, the backend's matchmaker (polls periodically) pairs them into a tournament and creates the semifinal matches.

## 4. Confirm the board appears

Within a few seconds of all 4 queuing, each session should automatically stop polling and show a **Match #... chess board** card — no manual match ID entry needed. Check that:

- The board orientation is correct (white player sees rank 1 at the bottom, black player sees it flipped).
- The connection badge shows **open**.
- It's "Waiting for your opponent to connect..." until both players in that pairing have the board open, then flips to "Your move" / "Opponent's move".

## 5. Play a move

- Click one of your own pieces (only enabled on your turn), then click a legal destination square.
- Confirm the move appears on both players' boards within about a second.
- Confirm each player's clock (60s bullet clock) only counts down on their own turn.
- Move a pawn to the back rank to confirm the promotion picker (Q/R/B/N) appears and completes the move correctly.

## 6. Confirm game-ending paths

Test at least one of:

- **Checkmate/stalemate/draw** — play it out (or manufacture a fast checkmate) and confirm both sessions show the "You won! / You lost. / Draw" screen with a **Back to Lobby** button.
- **Resign** — click Resign on one session and confirm the other session immediately sees the result screen with the correct winner.
- **Timeout** — let one player's 60s clock run out without moving; confirm the opponent is declared the winner even without an incoming move (the server's clock watcher polls every 500ms independent of moves).

## 7. Confirm bracket advancement (optional, needs both semifinals)

Play both semifinal matches to completion. The two winners should be automatically placed into a new final match, which should appear the same way (board auto-detected via polling) for those two accounts. Losers of the semifinals will have no active match and return to the "not currently queued" lobby state.

## Known environment note

`npm install` may fail with `ETIMEDOUT` if run from a sandboxed environment without registry access. Run it from your normal development machine/terminal instead.
