package matchmaking

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
	"gaming_platform/internal/domain/bot"
	"gaming_platform/internal/domain/tournament"
)

const (
	searchWindow        = 12
	tournamentBonusCoins int64 = 100
)

type Matcher struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Matcher {
	return &Matcher{pool: pool}
}

// jitterUpTo returns a random duration in [0, max), used to spread ticks out
// so concurrent runs (or a slow database) don't pile every tick up at the
// exact same instant.
func jitterUpTo(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(max)))
}

func (m *Matcher) Run(
	ctx context.Context,
	interval time.Duration,
) {
	const maxJitter = 500 * time.Millisecond

	timer := time.NewTimer(interval + jitterUpTo(maxJitter))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-timer.C:
			if err := m.tick(ctx); err != nil {
				log.Printf(
					"matchmaking: tick error: %v",
					err,
				)
			}
			timer.Reset(interval + jitterUpTo(maxJitter))
		}
	}
}

func (m *Matcher) tick(ctx context.Context) error {
	q := db.New(m.pool)

	buckets, err := q.ListDistinctWaitingBuckets(ctx)
	if err != nil {
		return err
	}

	for _, bucket := range buckets {
		err := m.matchBucket(
			ctx,
			bucket.GameType,
			bucket.GameMode,
			bucket.CoinWager,
		)
		if err != nil {
			log.Printf(
				"matchmaking: bucket %s/%s/%d error: %v",
				bucket.GameType,
				bucket.GameMode,
				bucket.CoinWager,
				err,
			)
		}
	}

	// A second, independent pass: any entry still waiting after
	// bot.MatchWait gets seated against a bot instead of a human, so nobody
	// waits forever for an opponent that never shows up. This runs after the
	// human-pairing pass above so a real opponent is always preferred over a
	// bot when one's available in the same tick.
	if err := m.matchStaleEntriesWithBots(ctx); err != nil {
		log.Printf("matchmaking: bot-matching error: %v", err)
	}

	return nil
}

// matchStaleEntriesWithBots seats every queue entry that's been waiting at
// least bot.MatchWait against a bot opponent, one entry per transaction, in
// a loop until none are left.
func (m *Matcher) matchStaleEntriesWithBots(ctx context.Context) error {
	cutoff := time.Now().Add(-bot.MatchWait)
	for {
		matched, err := m.matchOneStaleEntryWithBot(ctx, cutoff)
		if err != nil {
			return err
		}
		if !matched {
			return nil
		}
	}
}

// matchOneStaleEntryWithBot locks and seats a single stale waiting queue
// entry against a bot opponent chosen by the human's current Elo, mirroring
// the tournament/match shape createOneVsOneMatch produces for two humans.
// It reports whether an entry was found (and processed) at all, so the
// caller can loop until the queue is drained rather than assuming exactly
// one stale entry exists per tick.
func (m *Matcher) matchOneStaleEntryWithBot(ctx context.Context, cutoff time.Time) (bool, error) {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	q := db.New(m.pool).WithTx(tx)

	entry, err := q.LockOldestStaleWaitingQueueEntry(
		ctx,
		pgtype.Timestamptz{Time: cutoff, Valid: true},
	)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	human, err := q.GetUserByID(ctx, entry.UserID)
	if err != nil {
		return false, err
	}

	// The human's balance may have dropped below the wager since they
	// joined the queue (e.g. spent on a cosmetic); rather than fail the
	// whole tick, just cancel the stale entry and move on to the next one.
	if human.CoinBalance < entry.CoinWager {
		if err := q.CancelQueueEntry(ctx, db.CancelQueueEntryParams{
			QueueID: entry.QueueID,
			UserID:  entry.UserID,
		}); err != nil {
			return false, err
		}
		return true, tx.Commit(ctx)
	}

	botUsername := bot.UsernameForOpponentElo(dbutil.PGInt4ToInt(human.CurrentElo))
	botUser, err := q.GetUserByUsername(ctx, botUsername)
	if err != nil {
		return false, fmt.Errorf("look up bot account %s: %w", botUsername, err)
	}

	totalCoinPool := entry.CoinWager * 2

	createdTournament, err := q.CreateTournament(ctx, db.CreateTournamentParams{
		GameType:        entry.GameType,
		GameMode:        entry.GameMode,
		Status:          "waiting",
		CoinWager:       entry.CoinWager,
		TotalCoinPool:   totalCoinPool,
		WinnerCoinPrize: totalCoinPool,
		BonusCoins:      0,
	})
	if err != nil {
		return false, err
	}

	if _, err := q.CreateTournamentPlayer(ctx, db.CreateTournamentPlayerParams{
		TournamentID: createdTournament.TournamentID,
		UserID:       human.UserID,
		SeedNumber:   dbutil.IntToPGInt4(1),
	}); err != nil {
		return false, err
	}
	if _, err := q.CreateTournamentPlayer(ctx, db.CreateTournamentPlayerParams{
		TournamentID: createdTournament.TournamentID,
		UserID:       botUser.UserID,
		SeedNumber:   dbutil.IntToPGInt4(2),
	}); err != nil {
		return false, err
	}

	if err := q.MarkQueueEntryMatched(ctx, entry.QueueID); err != nil {
		return false, err
	}

	for _, uid := range []int64{human.UserID, botUser.UserID} {
		if _, err := q.AdjustUserCoinBalance(ctx, db.AdjustUserCoinBalanceParams{
			UserID: uid,
			Delta:  -entry.CoinWager,
		}); err != nil {
			return false, fmt.Errorf("debit wager for user %d: %w", uid, err)
		}
		if _, err := q.CreateCoinTransaction(ctx, db.CreateCoinTransactionParams{
			UserID:              uid,
			Amount:              -entry.CoinWager,
			TransactionType:     "match_entry",
			Description:         dbutil.TextToPGText("Entry wager for " + entry.GameType + " vs bot"),
			RelatedTournamentID: dbutil.Int64ToPGInt8(createdTournament.TournamentID),
		}); err != nil {
			return false, err
		}
	}

	// Who's seated player 1 (and therefore moves first) is randomized; the
	// match service triggers the bot's own moves autonomously based on whose
	// turn it currently is, not which player slot it occupies.
	player1ID, player2ID := human.UserID, botUser.UserID
	if rand.Intn(2) == 0 {
		player1ID, player2ID = player2ID, player1ID
	}
	if _, err := q.CreateMatch(ctx, db.CreateMatchParams{
		TournamentID: createdTournament.TournamentID,
		GameType:     entry.GameType,
		Round:        "final",
		MatchNumber:  dbutil.IntToPGInt4(1),
		Player1ID:    player1ID,
		Player2ID:    player2ID,
	}); err != nil {
		return false, err
	}

	if _, err := q.StartTournament(ctx, createdTournament.TournamentID); err != nil {
		return false, err
	}

	return true, tx.Commit(ctx)
}

type candidate struct {
	Queue db.MatchmakingQueue
	Elo   int
}

func (m *Matcher) matchBucket(
	ctx context.Context,
	gameType string,
	gameMode string,
	coinWager int64,
) error {
	requiredPlayers, err := playersRequired(gameMode)
	if err != nil {
		return err
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := db.New(m.pool).WithTx(tx)

	entries, err := q.ListWaitingQueueEntries(
		ctx,
		db.ListWaitingQueueEntriesParams{
			GameType:  gameType,
			GameMode:  gameMode,
			CoinWager: coinWager,
		},
	)
	if err != nil {
		return err
	}

	if len(entries) < requiredPlayers {
		return nil
	}

	// One query for every waiting entry's user instead of one per entry -
	// this tick runs every ~2s for every distinct bucket, so a per-entry
	// round trip here scales badly with queue size.
	userIDs := make([]int64, len(entries))
	for i, entry := range entries {
		userIDs[i] = entry.UserID
	}
	users, err := q.GetUsersByIDs(ctx, userIDs)
	if err != nil {
		return err
	}
	usersByID := make(map[int64]db.User, len(users))
	for _, u := range users {
		usersByID[u.UserID] = u
	}

	candidates := make(
		[]candidate,
		0,
		len(entries),
	)

	for _, entry := range entries {
		user, ok := usersByID[entry.UserID]
		if !ok || user.CoinBalance < coinWager {
			continue
		}

		candidates = append(
			candidates,
			candidate{
				Queue: entry,
				Elo: dbutil.PGInt4ToInt(
					user.CurrentElo,
				),
			},
		)
	}

	group, found := findCompatibleGroup(
		candidates,
		requiredPlayers,
	)
	if !found {
		return nil
	}

	// Randomize seating order so who's seeded player 1 (and therefore moves
	// first) is random rather than always the higher-Elo player.
	rand.Shuffle(len(group), func(i, j int) {
		group[i], group[j] = group[j], group[i]
	})

	totalCoinPool :=
		coinWager * int64(requiredPlayers)

	bonusCoins := int64(0)
	// 4-player tournament mode is disabled for now (too much work to keep
	// fully working right now) - may come back in the future. Left here,
	// commented out, since handleJoinQueue no longer accepts
	// game_mode=tournament.
	//
	// if gameMode == "tournament" {
	// 	bonusCoins = tournamentBonusCoins
	// }

	winnerCoinPrize :=
		totalCoinPool + bonusCoins

	createdTournament, err := q.CreateTournament(
		ctx,
		db.CreateTournamentParams{
			GameType:        gameType,
			GameMode:        gameMode,
			Status:          "waiting",
			CoinWager:       coinWager,
			TotalCoinPool:   totalCoinPool,
			WinnerCoinPrize: winnerCoinPrize,
			BonusCoins:      bonusCoins,
		},
	)
	if err != nil {
		return err
	}

	seededPlayers := make(
		[]tournament.SeededPlayer,
		requiredPlayers,
	)

	entryTransactionType := "match_entry"
	// Tournament mode disabled, see above.
	// if gameMode == "tournament" {
	// 	entryTransactionType = "tournament_entry"
	// }

	for index, player := range group {
		seed := index + 1

		_, err := q.CreateTournamentPlayer(
			ctx,
			db.CreateTournamentPlayerParams{
				TournamentID:
					createdTournament.TournamentID,
				UserID: player.Queue.UserID,
				SeedNumber:
					dbutil.IntToPGInt4(seed),
			},
		)
		if err != nil {
			return err
		}

		seededPlayers[index] =
			tournament.SeededPlayer{
				UserID: player.Queue.UserID,
				Seed:   seed,
			}

		err = q.MarkQueueEntryMatched(
			ctx,
			player.Queue.QueueID,
		)
		if err != nil {
			return err
		}

		// Debit the wager now, inside the same transaction as the rest of
		// match creation, so a player can never be seated at a table they
		// didn't actually pay into. A concurrent spend that dropped their
		// balance since the candidate scan above makes this match zero rows
		// (pgx.ErrNoRows), which aborts and rolls back the whole tick.
		if _, err := q.AdjustUserCoinBalance(
			ctx,
			db.AdjustUserCoinBalanceParams{
				UserID: player.Queue.UserID,
				Delta:  -coinWager,
			},
		); err != nil {
			return fmt.Errorf(
				"debit wager for user %d: %w",
				player.Queue.UserID,
				err,
			)
		}

		if _, err := q.CreateCoinTransaction(
			ctx,
			db.CreateCoinTransactionParams{
				UserID:          player.Queue.UserID,
				Amount:          -coinWager,
				TransactionType: entryTransactionType,
				Description:     dbutil.TextToPGText("Entry wager for " + gameType + " " + gameMode),
				RelatedTournamentID: dbutil.Int64ToPGInt8(
					createdTournament.TournamentID,
				),
			},
		); err != nil {
			return err
		}
	}

	if gameMode == "one_vs_one" {
		err = m.createOneVsOneMatch(
			ctx,
			q,
			createdTournament.TournamentID,
			gameType,
			seededPlayers,
		)
		if err != nil {
			return err
		}
	} else {
		// 4-player tournament mode is disabled for now (too much work to
		// keep fully working right now) - may come back in the future.
		// The implementation is left in place, just unreachable, since
		// handleJoinQueue no longer accepts game_mode=tournament.
		//
		// err = m.createTournamentSemifinals(
		// 	ctx,
		// 	q,
		// 	createdTournament.TournamentID,
		// 	gameType,
		// 	seededPlayers,
		// )
		// if err != nil {
		// 	return err
		// }
		return fmt.Errorf("matchmaking: tournament mode is disabled")
	}

	_, err = q.StartTournament(
		ctx,
		createdTournament.TournamentID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func playersRequired(gameMode string) (int, error) {
	switch gameMode {
	case "one_vs_one":
		return 2, nil
	// 4-player tournament mode is disabled for now, see handleJoinQueue.
	// case "tournament":
	// 	return 4, nil
	default:
		return 0, fmt.Errorf(
			"unsupported game mode: %s",
			gameMode,
		)
	}
}

func (m *Matcher) createOneVsOneMatch(
	ctx context.Context,
	q *db.Queries,
	tournamentID int64,
	gameType string,
	players []tournament.SeededPlayer,
) error {
	if len(players) != 2 {
		return fmt.Errorf(
			"one_vs_one requires exactly 2 players",
		)
	}

	_, err := q.CreateMatch(
		ctx,
		db.CreateMatchParams{
			TournamentID: tournamentID,
			GameType:     gameType,
			Round:        "final",
			MatchNumber:  dbutil.IntToPGInt4(1),
			Player1ID:    players[0].UserID,
			Player2ID:    players[1].UserID,
		},
	)

	return err
}

func (m *Matcher) createTournamentSemifinals(
	ctx context.Context,
	q *db.Queries,
	tournamentID int64,
	gameType string,
	players []tournament.SeededPlayer,
) error {
	pairings, err := tournament.Semifinals(players)
	if err != nil {
		return err
	}

	for _, pairing := range pairings {
		_, err := q.CreateMatch(
			ctx,
			db.CreateMatchParams{
				TournamentID: tournamentID,
				GameType:     gameType,
				Round:        pairing.Round,
				MatchNumber:
					dbutil.IntToPGInt4(
						pairing.MatchNumber,
					),
				Player1ID: pairing.Player1ID,
				Player2ID: pairing.Player2ID,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func withinRange(
	player candidate,
	otherElo int,
) bool {
	if player.Queue.MinElo.Valid &&
		otherElo < int(player.Queue.MinElo.Int32) {
		return false
	}

	if player.Queue.MaxElo.Valid &&
		otherElo > int(player.Queue.MaxElo.Int32) {
		return false
	}

	return true
}

func mutuallyCompatible(
	group []candidate,
) bool {
	for _, first := range group {
		for _, second := range group {
			if first.Queue.QueueID ==
				second.Queue.QueueID {
				continue
			}

			if !withinRange(
				first,
				second.Elo,
			) {
				return false
			}
		}
	}

	return true
}

func findCompatibleGroup(
	candidates []candidate,
	groupSize int,
) ([]candidate, bool) {
	if len(candidates) < groupSize {
		return nil, false
	}

	limit := len(candidates)
	if limit > searchWindow {
		limit = searchWindow
	}

	pool := candidates[:limit]
	indices := make([]int, groupSize)

	var result []candidate
	found := false

	var choose func(start int, depth int)

	choose = func(start int, depth int) {
		if found {
			return
		}

		if depth == groupSize {
			group := make(
				[]candidate,
				groupSize,
			)

			for index, poolIndex :=
				range indices {
				group[index] =
					pool[poolIndex]
			}

			if mutuallyCompatible(group) {
				result = group
				found = true
			}

			return
		}

		remainingNeeded :=
			groupSize - depth

		for index := start;
			index <= len(pool)-remainingNeeded;
			index++ {
			indices[depth] = index
			choose(index+1, depth+1)

			if found {
				return
			}
		}
	}

	choose(0, 0)

	return result, found
}
