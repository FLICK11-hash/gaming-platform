package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"gaming_platform/internal/auth"
	"gaming_platform/internal/config"
	"gaming_platform/internal/domain/achievements"
	"gaming_platform/internal/domain/hub"
	"gaming_platform/internal/domain/leaderboard"
	"gaming_platform/internal/domain/match"
	"gaming_platform/internal/domain/matchmaking"
	"gaming_platform/internal/domain/missions"
	"gaming_platform/internal/httpapi"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := runMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	issuer := auth.NewTokenIssuer(cfg.JWTSecret)
	connHub := hub.New()
	missionsSvc := missions.NewService(pool)
	leaderboardSvc := leaderboard.NewService(pool)
	achievementsSvc := achievements.NewService(pool)
	matchSvc := match.NewService(pool, connHub, missionsSvc, leaderboardSvc, achievementsSvc)
	matcher := matchmaking.New(pool)

	go matcher.Run(ctx, 2*time.Second)
	go matchSvc.RunClockWatcher(ctx)
	go leaderboardSvc.RunWeeklyRewards(ctx)

	server := httpapi.NewServer(pool, issuer, matchSvc, missionsSvc, leaderboardSvc, achievementsSvc)
	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: server.Router(),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on :%s", cfg.Port)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server: %v", err)
	}
}

// runMigrations applies internal/db/migrations against DATABASE_URL on
// startup. The path can be overridden with MIGRATIONS_PATH for deployments
// that don't run from the repo root.
func runMigrations(databaseURL string) error {
	path := os.Getenv("MIGRATIONS_PATH")
	if path == "" {
		path = "internal/db/migrations"
	}

	// golang-migrate's pgx/v5 driver registers itself under the "pgx5" scheme.
	migrationURL := databaseURL
	if idx := strings.Index(migrationURL, "://"); idx != -1 {
		migrationURL = "pgx5" + migrationURL[idx:]
	}

	m, err := migrate.New("file://"+path, migrationURL)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
