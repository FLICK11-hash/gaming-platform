package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

	"gaming_platform/internal/auth"
	"gaming_platform/internal/domain/leaderboard"
	"gaming_platform/internal/domain/match"
	"gaming_platform/internal/domain/matchhistory"
	"gaming_platform/internal/domain/missions"
)

type Server struct {
	pool            *pgxpool.Pool
	issuer          *auth.TokenIssuer
	matchSvc        *match.Service
	missionsSvc     *missions.Service
	leaderboardSvc  *leaderboard.Service
	matchHistorySvc *matchhistory.Service
	upgrader        websocket.Upgrader
}

func NewServer(pool *pgxpool.Pool, issuer *auth.TokenIssuer, matchSvc *match.Service, missionsSvc *missions.Service, leaderboardSvc *leaderboard.Service, matchHistorySvc *matchhistory.Service) *Server {
	return &Server{
		pool:            pool,
		issuer:          issuer,
		matchSvc:        matchSvc,
		missionsSvc:     missionsSvc,
		leaderboardSvc:  leaderboardSvc,
		matchHistorySvc: matchHistorySvc,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Post("/register", s.handleRegister)
	r.Post("/login", s.handleLogin)
	r.Get("/public/landing-stats", s.handleGetLandingStats)

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(s.issuer))

		r.Get("/me", s.handleMe)
		r.Post("/me/daily-reward", s.handleClaimDailyReward)
		r.Post("/me/avatar", s.handleEquipAvatar)
		r.Post("/me/badge", s.handleEquipBadge)
		r.Post("/me/profile-frames/toggle", s.handleToggleProfileFrame)

		r.Get("/cosmetics", s.handleListCosmetics)
		r.Post("/cosmetics/{cosmeticItemID}/purchase", s.handleBuyCosmetic)

		r.Get("/missions", s.handleListMissions)
		r.Post("/missions/{missionID}/claim", s.handleClaimMission)

		r.Get("/leaderboards", s.handleGetLeaderboards)

		r.Post("/matchmaking/queue", s.handleJoinQueue)
		r.Delete("/matchmaking/queue/{queueID}", s.handleCancelQueue)

		r.Get("/tournaments/{tournamentID}", s.handleGetTournament)

		r.Get("/matches/mine", s.handleGetMyActiveMatch)
		r.Get("/matches/{matchID}", s.handleGetMatch)
		r.Post("/matches/{matchID}/resign", s.handleResign)
		r.Get("/matches/{matchID}/ws", s.handleMatchWS)

		r.Post("/fraud-reports", s.handleCreateFraudReport)
		r.Get("/fraud-reports", s.handleListFraudReports)
		r.Get("/fraud-reports/{fraudReportID}", s.handleGetFraudReport)
		r.Patch("/fraud-reports/{fraudReportID}", s.handleUpdateFraudReportStatus)

		r.Post("/follows/{userID}", s.handleFollow)
		r.Delete("/follows/{userID}", s.handleUnfollow)
		r.Get("/users/{userID}/followers", s.handleListFollowers)
		r.Get("/users/{userID}/following", s.handleListFollowing)
		r.Get("/users/{userID}/profile", s.handleGetUserProfile)

		r.Post("/posts", s.handleCreatePost)
		r.Get("/posts/{postID}", s.handleGetPost)
		r.Get("/users/{userID}/posts", s.handleListPostsByUser)
		r.Patch("/posts/{postID}", s.handleUpdatePostStatus)
	})

	return r
}
