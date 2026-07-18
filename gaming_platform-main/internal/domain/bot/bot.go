// Package bot chooses moves for computer-controlled opponents seated by
// matchmaking when a human waits too long for a real opponent. It supports
// both game types the platform offers - chess and Connect Four - each with
// an "easy" and a "hard" difficulty tier that map to the two fixed bot
// accounts' Elo (1200 and 2100 respectively; see migration
// 0009_bot_users.up.sql).
package bot

import "time"

// Difficulty selects how strong a bot's move-choosing AI plays.
type Difficulty int

const (
	Easy Difficulty = iota
	Hard
)

// EloThreshold is the human Elo cutoff at (and above) which a waiting
// player is paired against the Hard-difficulty bot instead of the Easy one.
const EloThreshold = 1500

// Bot account usernames, seeded by migration 0009_bot_users.up.sql.
const (
	RookieBotUsername      = "RookieBot"
	GrandmasterBotUsername = "GrandmasterBot"
)

// MatchWait is how long a matchmaking queue entry must wait before it's
// paired against a bot instead of a human opponent.
const MatchWait = 15 * time.Second

// DifficultyForElo maps a bot account's own (fixed) Elo to its playing
// strength tier.
func DifficultyForElo(elo int) Difficulty {
	if elo >= EloThreshold {
		return Hard
	}
	return Easy
}

// UsernameForOpponentElo picks which bot account should be seated opposite a
// human with the given current Elo.
func UsernameForOpponentElo(humanElo int) string {
	if humanElo >= EloThreshold {
		return GrandmasterBotUsername
	}
	return RookieBotUsername
}
