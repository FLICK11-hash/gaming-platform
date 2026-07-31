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
}

func Load() (Config, error) {
	cfg := Config{
		Port:                  getEnv("PORT", "8080"),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		ResearchAdminUsername: getEnv("RESEARCH_ADMIN_USERNAME", "conradflick11"),
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
