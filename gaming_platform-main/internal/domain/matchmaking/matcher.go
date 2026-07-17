package matchmaking

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
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

func (m *Matcher) Run(
	ctx context.Context,
	interval time.Duration,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if err := m.tick(ctx); err != nil {
				log.Printf(
					"matchmaking: tick error: %v",
					err,
				)
			}
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

	return nil
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

	candidates := make(
		[]candidate,
		0,
		len(entries),
	)

	for _, entry := range entries {
		user, err := q.GetUserByID(
			ctx,
			entry.UserID,
		)
		if err != nil {
			return err
		}

		if user.CoinBalance < coinWager {
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

	sort.Slice(group, func(i, j int) bool {
		return group[i].Elo > group[j].Elo
	})

	totalCoinPool :=
		coinWager * int64(requiredPlayers)

	bonusCoins := int64(0)
	if gameMode == "tournament" {
		bonusCoins = tournamentBonusCoins
	}

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
	if gameMode == "tournament" {
		entryTransactionType = "tournament_entry"
	}

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
		err = m.createTournamentSemifinals(
			ctx,
			q,
			createdTournament.TournamentID,
			gameType,
			seededPlayers,
		)
		if err != nil {
			return err
		}
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
	case "tournament":
		return 4, nil
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
