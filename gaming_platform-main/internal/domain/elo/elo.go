// Package elo implements the standard Elo rating update formula.
package elo

import "math"

const kFactor = 32.0

// Result is the match outcome from the perspective of a single player.
type Result float64

const (
	Loss Result = 0.0
	Draw Result = 0.5
	Win  Result = 1.0
)

// expectedScore returns the probability that a player rated `rating` beats
// an opponent rated `opponentRating`.
func expectedScore(rating, opponentRating int) float64 {
	return 1.0 / (1.0 + math.Pow(10, float64(opponentRating-rating)/400.0))
}

// Update returns a player's new rating given their rating, their opponent's
// rating, and the match result.
func Update(rating, opponentRating int, result Result) int {
	expected := expectedScore(rating, opponentRating)
	newRating := float64(rating) + kFactor*(float64(result)-expected)
	return int(math.Round(newRating))
}

// UpdatePair computes both players' new ratings for a decisive (non-draw)
// match between winner and loser.
func UpdatePair(winnerRating, loserRating int) (newWinnerRating, newLoserRating int) {
	return Update(winnerRating, loserRating, Win), Update(loserRating, winnerRating, Loss)
}
