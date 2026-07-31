// Command simulate runs a batch of bot-vs-bot games entirely in memory - no
// database, no server - to answer the two questions behind this project's
// research pivot: how much does moving first matter, and how much does the
// opening line matter? It reuses the platform's own chess/connectfour
// engines and bot AI (internal/research/simulate wraps them), so results
// describe the platform's actual bots.
//
// Usage:
//
//	go run ./cmd/simulate -game chess -games 2000
//	go run ./cmd/simulate -game connectfour -games 2000 -difficulty hard
//	go run ./cmd/simulate -game chess -games 2000 -openings=false
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"gaming_platform/internal/domain/bot"
	"gaming_platform/internal/research/simulate"
)

func main() {
	game := flag.String("game", "chess", `which game to simulate: "chess" or "connectfour"`)
	games := flag.Int("games", 1000, "number of games to simulate")
	difficultyName := flag.String("difficulty", "easy", `bot difficulty for both sides: "easy" or "hard" (both sides always play the same difficulty, isolating move-order as the only asymmetry)`)
	randomizeOpenings := flag.Bool("openings", true, "sample the opening move/column at random instead of letting the bot choose it, and break results down per opening")
	asJSON := flag.Bool("json", false, "print the full result as JSON instead of a summary table")
	flag.Parse()

	var difficulty bot.Difficulty
	switch *difficultyName {
	case "easy":
		difficulty = bot.Easy
	case "hard":
		difficulty = bot.Hard
	default:
		log.Fatalf("unknown -difficulty %q: want \"easy\" or \"hard\"", *difficultyName)
	}

	var stats simulate.BatchStats
	var err error
	switch *game {
	case "chess":
		stats, err = simulate.RunChessBatch(*games, difficulty, *randomizeOpenings)
	case "connectfour":
		stats, err = simulate.RunConnectFourBatch(*games, difficulty, *randomizeOpenings)
	default:
		log.Fatalf("unknown -game %q: want \"chess\" or \"connectfour\"", *game)
	}
	if err != nil {
		log.Fatalf("simulation error: %v", err)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(stats); err != nil {
			log.Fatalf("encode result: %v", err)
		}
		return
	}

	printSummary(*game, *difficultyName, stats)
}

func printSummary(game, difficulty string, stats simulate.BatchStats) {
	fmt.Printf("%s, %s difficulty, %d games\n", game, difficulty, stats.Games)
	fmt.Printf("  first mover won:  %5d (%.1f%%)\n", stats.FirstMoverWins, stats.FirstMoverWinRate*100)
	fmt.Printf("  second mover won: %5d (%.1f%%)\n", stats.SecondMoverWins, stats.SecondMoverWinRate*100)
	fmt.Printf("  draws:            %5d (%.1f%%)\n", stats.Draws, stats.DrawRate*100)
	fmt.Printf("  average length:   %.1f plies\n", stats.AveragePlies)

	if len(stats.ByOpening) == 0 {
		return
	}
	fmt.Printf("\n  %-14s %8s %14s %8s\n", "opening", "games", "first-mover win%", "draw%")
	for _, o := range stats.SortedOpenings() {
		drawRate := 0.0
		if o.Stat.Games > 0 {
			drawRate = float64(o.Stat.Draws) / float64(o.Stat.Games) * 100
		}
		fmt.Printf("  %-14s %8d %13.1f%% %7.1f%%\n", o.Opening, o.Stat.Games, o.Stat.WinRate*100, drawRate)
	}
}
