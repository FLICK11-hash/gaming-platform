package httpapi

import (
	"net/http"
	"time"

	"gaming_platform/internal/auth"
	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/domain/research"
	"gaming_platform/internal/research/simulate"
)

type researchOpeningReportDTO struct {
	Opening      string  `json:"opening"`
	Games        int     `json:"games"`
	Wins         int     `json:"wins"`
	Losses       int     `json:"losses"`
	Draws        int     `json:"draws"`
	WinRate      float64 `json:"win_rate"`
	AveragePlies float64 `json:"average_plies"`
}

type researchConnectFourReportDTO struct {
	Column       int     `json:"column"`
	Games        int     `json:"games"`
	Wins         int     `json:"wins"`
	Losses       int     `json:"losses"`
	Draws        int     `json:"draws"`
	WinRate      float64 `json:"win_rate"`
	AveragePlies float64 `json:"average_plies"`
}

type researchChessEloTierDTO struct {
	Label         string                     `json:"label"`
	Elo           int                        `json:"elo"`
	WhiteOpenings []researchOpeningReportDTO `json:"white_openings"`
	BlackOpenings []researchOpeningReportDTO `json:"black_openings"`
}

type researchResultDTO struct {
	StartedAt             string                         `json:"started_at"`
	FinishedAt            string                         `json:"finished_at"`
	GamesPerOpening       int                            `json:"games_per_opening"`
	ChessByElo            []researchChessEloTierDTO      `json:"chess_by_elo"`
	ConnectFourDifficulty string                         `json:"connect_four_difficulty"`
	ConnectFour           []researchConnectFourReportDTO `json:"connect_four"`
}

type researchStatusResponse struct {
	Status            string             `json:"status"`
	CanRun            bool               `json:"can_run"`
	Error             string             `json:"error,omitempty"`
	Result            *researchResultDTO `json:"result,omitempty"`
	RunningForSeconds float64            `json:"running_for_seconds,omitempty"`
}

func toOpeningReportDTOs(reports []simulate.ChessOpeningReport) []researchOpeningReportDTO {
	out := make([]researchOpeningReportDTO, len(reports))
	for i, r := range reports {
		out[i] = researchOpeningReportDTO{
			Opening:      r.Opening,
			Games:        r.Games,
			Wins:         r.Wins,
			Losses:       r.Losses,
			Draws:        r.Draws,
			WinRate:      r.WinRate,
			AveragePlies: r.AveragePlies,
		}
	}
	return out
}

func toResultDTO(res *research.Result) *researchResultDTO {
	if res == nil {
		return nil
	}

	connectFour := make([]researchConnectFourReportDTO, len(res.ConnectFour))
	for i, c := range res.ConnectFour {
		connectFour[i] = researchConnectFourReportDTO{
			Column:       c.Column,
			Games:        c.Games,
			Wins:         c.Wins,
			Losses:       c.Losses,
			Draws:        c.Draws,
			WinRate:      c.WinRate,
			AveragePlies: c.AveragePlies,
		}
	}

	chessByElo := make([]researchChessEloTierDTO, len(res.ChessByElo))
	for i, tier := range res.ChessByElo {
		chessByElo[i] = researchChessEloTierDTO{
			Label:         tier.Label,
			Elo:           tier.Elo,
			WhiteOpenings: toOpeningReportDTOs(tier.WhiteOpenings),
			BlackOpenings: toOpeningReportDTOs(tier.BlackOpenings),
		}
	}

	return &researchResultDTO{
		StartedAt:             res.StartedAt.UTC().Format(time.RFC3339),
		FinishedAt:            res.FinishedAt.UTC().Format(time.RFC3339),
		GamesPerOpening:       res.GamesPerOpening,
		ChessByElo:            chessByElo,
		ConnectFourDifficulty: res.ConnectFourDifficulty,
		ConnectFour:           connectFour,
	}
}

func (s *Server) canRunResearch(r *http.Request) bool {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		return false
	}
	user, err := db.New(s.pool).GetUserByID(r.Context(), userID)
	if err != nil {
		return false
	}
	return user.Username == s.researchAdminUsername
}

func (s *Server) handleGetResearchSimulation(w http.ResponseWriter, r *http.Request) {
	status, result, errMsg, runningFor := s.researchRunner.Snapshot()

	writeJSON(w, http.StatusOK, researchStatusResponse{
		Status:            string(status),
		CanRun:            s.canRunResearch(r),
		Error:             errMsg,
		Result:            toResultDTO(result),
		RunningForSeconds: runningFor.Seconds(),
	})
}

func (s *Server) handleRunResearchSimulation(w http.ResponseWriter, r *http.Request) {
	if !s.canRunResearch(r) {
		writeError(w, http.StatusForbidden, "only the site owner can trigger a simulation run")
		return
	}

	if err := s.researchRunner.Start(); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "running"})
}
