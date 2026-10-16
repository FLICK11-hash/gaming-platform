package elo

import "testing"

func TestUpdatePair_EqualRatings(t *testing.T) {
	winner, loser := UpdatePair(1200, 1200)
	if winner != 1216 {
		t.Errorf("winner rating = %d, want 1216", winner)
	}
	if loser != 1184 {
		t.Errorf("loser rating = %d, want 1184", loser)
	}
}

func TestUpdatePair_UpsetGainsMore(t *testing.T) {
	// A big underdog (1000) beating a much stronger player (1600) should
	// gain far more rating than an equal-strength win would.
	winner, loser := UpdatePair(1000, 1600)
	equalWinner, _ := UpdatePair(1200, 1200)

	if gain := winner - 1000; gain <= equalWinner-1200 {
		t.Errorf("upset winner gained %d, want more than equal-strength gain of %d", gain, equalWinner-1200)
	}
	if loser >= 1600 {
		t.Errorf("loser rating = %d, want less than 1600", loser)
	}
}

func TestUpdate_Draw(t *testing.T) {
	if got := Update(1200, 1200, Draw); got != 1200 {
		t.Errorf("Update(1200, 1200, Draw) = %d, want 1200", got)
	}
}
