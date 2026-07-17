package tournament

import "testing"

func TestSemifinals_SeedsPairCorrectly(t *testing.T) {
	players := []SeededPlayer{
		{UserID: 10, Seed: 1},
		{UserID: 20, Seed: 2},
		{UserID: 30, Seed: 3},
		{UserID: 40, Seed: 4},
	}

	pairings, err := Semifinals(players)
	if err != nil {
		t.Fatalf("Semifinals returned error: %v", err)
	}
	if len(pairings) != 2 {
		t.Fatalf("got %d pairings, want 2", len(pairings))
	}

	sf1, sf2 := pairings[0], pairings[1]
	if sf1.Player1ID != 10 || sf1.Player2ID != 40 {
		t.Errorf("semifinal 1 = %d vs %d, want 10 vs 40 (seed 1 vs seed 4)", sf1.Player1ID, sf1.Player2ID)
	}
	if sf2.Player1ID != 20 || sf2.Player2ID != 30 {
		t.Errorf("semifinal 2 = %d vs %d, want 20 vs 30 (seed 2 vs seed 3)", sf2.Player1ID, sf2.Player2ID)
	}
	if sf1.MatchNumber != 1 || sf2.MatchNumber != 2 {
		t.Errorf("match numbers = %d, %d, want 1, 2", sf1.MatchNumber, sf2.MatchNumber)
	}
}

func TestSemifinals_RequiresExactlyFourPlayers(t *testing.T) {
	_, err := Semifinals([]SeededPlayer{{UserID: 1, Seed: 1}})
	if err != ErrNeedFourPlayers {
		t.Errorf("err = %v, want ErrNeedFourPlayers", err)
	}
}

func TestSemifinals_RequiresAllFourSeeds(t *testing.T) {
	players := []SeededPlayer{
		{UserID: 10, Seed: 1},
		{UserID: 20, Seed: 2},
		{UserID: 30, Seed: 3},
		{UserID: 40, Seed: 3}, // duplicate seed, missing seed 4
	}
	_, err := Semifinals(players)
	if err != ErrNeedFourPlayers {
		t.Errorf("err = %v, want ErrNeedFourPlayers", err)
	}
}

func TestFinalPairing(t *testing.T) {
	p := FinalPairing(100, 200)
	if p.Round != RoundFinal || p.MatchNumber != 3 {
		t.Errorf("final pairing round/number = %s/%d, want final/3", p.Round, p.MatchNumber)
	}
	if p.Player1ID != 100 || p.Player2ID != 200 {
		t.Errorf("final pairing players = %d vs %d, want 100 vs 200", p.Player1ID, p.Player2ID)
	}
}

func TestFinalResults(t *testing.T) {
	results := FinalResults(1, 2, [2]int64{3, 4})

	want := map[int64]string{
		1: ResultChampion,
		2: ResultFinalist,
		3: ResultSemifinalLoss,
		4: ResultSemifinalLoss,
	}
	for uid, wantResult := range want {
		if got := results[uid]; got != wantResult {
			t.Errorf("results[%d] = %s, want %s", uid, got, wantResult)
		}
	}
}
