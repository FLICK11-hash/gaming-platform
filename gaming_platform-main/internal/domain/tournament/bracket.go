// Package tournament implements bracket seeding and advancement for the
// 4-player, single-elimination format (2 semifinals + 1 final) used by
// every game type on the platform.
package tournament

import "errors"

const (
	RoundSemifinal = "semifinal"
	RoundFinal     = "final"

	ResultChampion      = "champion"
	ResultFinalist      = "finalist"
	ResultSemifinalLoss = "semifinal_loss"
)

var ErrNeedFourPlayers = errors.New("tournament: bracket requires exactly 4 seeded players")

// SeededPlayer is a player entering the bracket with their seed (1-4).
type SeededPlayer struct {
	UserID int64
	Seed   int
}

// Pairing is one match's two participants, identified by round and match
// number (1, 2 = semifinals; 3 = final).
type Pairing struct {
	Round       string
	MatchNumber int
	Player1ID   int64
	Player2ID   int64
}

// Semifinals returns the two semifinal pairings for 4 seeded players:
// seed 1 vs seed 4, and seed 2 vs seed 3.
func Semifinals(players []SeededPlayer) ([]Pairing, error) {
	if len(players) != 4 {
		return nil, ErrNeedFourPlayers
	}

	bySeed := make(map[int]int64, 4)
	for _, p := range players {
		bySeed[p.Seed] = p.UserID
	}
	for s := 1; s <= 4; s++ {
		if _, ok := bySeed[s]; !ok {
			return nil, ErrNeedFourPlayers
		}
	}

	return []Pairing{
		{Round: RoundSemifinal, MatchNumber: 1, Player1ID: bySeed[1], Player2ID: bySeed[4]},
		{Round: RoundSemifinal, MatchNumber: 2, Player1ID: bySeed[2], Player2ID: bySeed[3]},
	}, nil
}

// FinalPairing builds the final match once both semifinal winners are known.
func FinalPairing(semifinal1WinnerID, semifinal2WinnerID int64) Pairing {
	return Pairing{
		Round:       RoundFinal,
		MatchNumber: 3,
		Player1ID:   semifinal1WinnerID,
		Player2ID:   semifinal2WinnerID,
	}
}

// FinalResults computes each of the 4 players' final_result once the
// tournament (final match) is complete.
func FinalResults(championID, finalistID int64, semifinalLoserIDs [2]int64) map[int64]string {
	results := map[int64]string{
		championID: ResultChampion,
		finalistID: ResultFinalist,
	}
	for _, loserID := range semifinalLoserIDs {
		results[loserID] = ResultSemifinalLoss
	}
	return results
}
