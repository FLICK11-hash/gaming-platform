package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	// ResearchAdminUsername is the only account allowed to trigger a new
	// Monte Carlo simulation run from the website - those runs are
	// CPU-heavy and share a core with real matches, so triggering one
	// isn't left open to every logged-in user.
	ResearchAdminUsername string
	// StockfishPath is where to find the Stockfish binary for real-match
	// bot moves (see internal/domain/bot.ConfigureStockfish). Defaults to
	// the bare command name, which resolves via PATH - true in the deployed
	// container, where the Dockerfile installs the Linux package. For local
	// development on a machine without Stockfish on PATH (e.g. Windows),
	// set this to the full path of a downloaded binary instead. If the
	// binary can't be found or started at all, the server logs a warning
	// and bot moves fall back to the platform's own built-in logic - a
	// missing engine degrades bot strength, it doesn't stop the server
	// from starting.
	StockfishPath string
}

func Load() (Config, error) {
	cfg := Config{
		Port:                  getEnv("PORT", "8080"),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		ResearchAdminUsername: getEnv("RESEARCH_ADMIN_USERNAME", "conradflick11"),
		StockfishPath:         getEnv("STOCKFISH_PATH", "stockfish"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
