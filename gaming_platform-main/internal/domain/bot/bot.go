// Package bot chooses moves for computer-controlled opponents seated by
// matchmaking when a human waits too long for a real opponent. It supports
// both game types the platform offers - chess and Connect Four - each with
// an Easy/Medium/Hard difficulty tier that maps to the three fixed bot
// accounts' Elo (1320, 2000, and 2500 respectively; see migration
// 0024_rebalance_bot_elo.up.sql).
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

// RookieBotUCIElo, ExpertBotUCIElo, and GrandmasterBotUCIElo are the
// Stockfish UCI_Elo targets for the Easy, Medium, and Hard tiers.
// RookieBotUCIElo is stockfish.MinUCIElo, not quite the bot account's own
// displayed Elo (1320 vs. the ~1200 originally intended) - Stockfish's own
// strength limiter can't go any lower, so this is the closest this engine
// can get on its own; easyExtraBlunderChance below pulls the effective
// strength down further for callers that still want to approximate 1200.
const (
	RookieBotUCIElo      = stockfish.MinUCIElo
	ExpertBotUCIElo      = 2000
	GrandmasterBotUCIElo = 2500
)

// enginePoolSize is how many persistent Stockfish processes back each
// difficulty tier. More than 1 lets that many bot moves resolve
// concurrently instead of queueing behind a single process; this doesn't
// need to be large since a human opponent's own thinking time between
// moves is normally much longer than one Stockfish search. Kept at 1
// deliberately: these run for the server's entire lifetime (3 tiers x this
// many processes, all the time), and the platform's free-tier deploy target
// has very little RAM to spare.
const enginePoolSize = 1

var (
	rookieEngine      *stockfish.Engine
	expertEngine      *stockfish.Engine
	grandmasterEngine *stockfish.Engine
)

// ConfigureStockfish starts the Stockfish process pools ChooseChessMoveEngine
// prefers, one per difficulty tier, each searching for
// stockfish.RealMatchMoveTime per move. Call it once at startup; if it's
// never called (or the binary at path can't be found/started),
// ChooseChessMoveEngine transparently falls back to the same hand-rolled
// logic ChooseChessMove always uses - a missing or broken Stockfish binary
// degrades bot strength, it doesn't take the server down.
func ConfigureStockfish(path string) error {
	rookie, err := stockfish.NewEngine(path, RookieBotUCIElo, enginePoolSize, stockfish.RealMatchMoveTime)
	if err != nil {
		return err
	}
	expert, err := stockfish.NewEngine(path, ExpertBotUCIElo, enginePoolSize, stockfish.RealMatchMoveTime)
	if err != nil {
		rookie.Close()
		return err
	}
	grandmaster, err := stockfish.NewEngine(path, GrandmasterBotUCIElo, enginePoolSize, stockfish.RealMatchMoveTime)
	if err != nil {
		rookie.Close()
		expert.Close()
		return err
	}
	rookieEngine = rookie
	expertEngine = expert
	grandmasterEngine = grandmaster
	log.Printf("bot: Stockfish configured (rookie=%d elo, expert=%d elo, grandmaster=%d elo, pool size %d each)",
		RookieBotUCIElo, ExpertBotUCIElo, GrandmasterBotUCIElo, enginePoolSize)
	return nil
}

// Difficulty selects how strong a bot's move-choosing AI plays.
type Difficulty int

const (
	Easy Difficulty = iota
	Medium
	Hard
)

// EasyToMediumElo and MediumToHardElo are the human-Elo cutoffs used to pick
// which bot a waiting player is paired against; since each bot account's
// own Elo is fixed (see ExpertBotUCIElo etc. above), the same two cutoffs
// also decide which difficulty tier a seated bot plays at, via
// DifficultyForElo.
const (
	EasyToMediumElo = 1700
	MediumToHardElo = 2200
)

// Bot account usernames, seeded by migration 0009_bot_users.up.sql and
// 0024_rebalance_bot_elo.up.sql.
const (
	RookieBotUsername      = "RookieBot"
	ExpertBotUsername      = "ExpertBot"
	GrandmasterBotUsername = "GrandmasterBot"
)

// MatchWait is how long a matchmaking queue entry must wait before it's
// paired against a bot instead of a human opponent.
const MatchWait = 15 * time.Second

// DifficultyForElo maps a bot account's own (fixed) Elo to its playing
// strength tier.
func DifficultyForElo(elo int) Difficulty {
	switch {
	case elo >= MediumToHardElo:
		return Hard
	case elo >= EasyToMediumElo:
		return Medium
	default:
		return Easy
	}
}

// UsernameForOpponentElo picks which bot account should be seated opposite a
// human with the given current Elo.
func UsernameForOpponentElo(humanElo int) string {
	switch {
	case humanElo >= MediumToHardElo:
		return GrandmasterBotUsername
	case humanElo >= EasyToMediumElo:
		return ExpertBotUsername
	default:
		return RookieBotUsername
	}
}
