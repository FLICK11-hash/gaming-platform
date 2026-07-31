// Package bot chooses moves for computer-controlled opponents seated by
// matchmaking when a human waits too long for a real opponent. It supports
// both game types the platform offers - chess and Connect Four - each with
// an "easy" and a "hard" difficulty tier that map to the two fixed bot
// accounts' Elo (1200 and 2100 respectively; see migration
// 0009_bot_users.up.sql).
//
// Chess moves for real matches (ChooseChessMoveEngine) come from a real
// Stockfish engine when one's configured via ConfigureStockfish, with the
// original hand-rolled evaluation in chess.go as an automatic fallback if
// Stockfish isn't available or a call to it fails. ChooseChessMove (no
// "Engine" suffix) always uses the hand-rolled logic directly and never
// touches Stockfish - that's deliberate: the Monte Carlo research harness
// (internal/research/simulate) calls thousands of these per batch, and a
// Stockfish process round trip per move would turn a two-minute batch into
// well over an hour.
package bot

import (
	"log"
	"time"

	"gaming_platform/internal/engine/stockfish"
)

// RookieBotUCIElo and GrandmasterBotUCIElo are the Stockfish UCI_Elo
// targets for the Easy and Hard tiers. RookieBotUCIElo is
// stockfish.MinUCIElo, not the bot account's own displayed Elo (1200,
// migration 0009_bot_users.up.sql) - Stockfish's own strength limiter
// can't go below MinUCIElo (1320 as of Stockfish 16), so it's the closest
// this engine can get on its own; easyExtraBlunderChance below pulls the
// effective strength down further to approximate 1200 more closely.
const (
	RookieBotUCIElo      = stockfish.MinUCIElo
	GrandmasterBotUCIElo = 2100
)

// enginePoolSize is how many persistent Stockfish processes back each
// difficulty tier. More than 1 lets that many bot moves resolve
// concurrently instead of queueing behind a single process; this doesn't
// need to be large since a human opponent's own thinking time between
// moves is normally much longer than one Stockfish search.
const enginePoolSize = 2

var (
	rookieEngine      *stockfish.Engine
	grandmasterEngine *stockfish.Engine
)

// ConfigureStockfish starts the Stockfish process pools ChooseChessMoveEngine
// prefers. Call it once at startup; if it's never called (or the binary at
// path can't be found/started), ChooseChessMoveEngine transparently falls
// back to the same hand-rolled logic ChooseChessMove always uses - a
// missing or broken Stockfish binary degrades bot strength, it doesn't
// take the server down.
func ConfigureStockfish(path string) error {
	rookie, err := stockfish.NewEngine(path, RookieBotUCIElo, enginePoolSize)
	if err != nil {
		return err
	}
	grandmaster, err := stockfish.NewEngine(path, GrandmasterBotUCIElo, enginePoolSize)
	if err != nil {
		rookie.Close()
		return err
	}
	rookieEngine = rookie
	grandmasterEngine = grandmaster
	log.Printf("bot: Stockfish configured (rookie=%d elo, grandmaster=%d elo, pool size %d each)",
		RookieBotUCIElo, GrandmasterBotUCIElo, enginePoolSize)
	return nil
}

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
