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

type researchOpeningStatDTO struct {
	Opening        string  `json:"opening"`
	Games          int     `json:"games"`
	FirstMoverWins int     `json:"first_mover_wins"`
	Draws          int     `json:"draws"`
	WinRate        float64 `json:"win_rate"`
}

type researchConnectFourDTO struct {
	Games              int                      `json:"games"`
	FirstMoverWins     int                      `json:"first_mover_wins"`
	SecondMoverWins    int                      `json:"second_mover_wins"`
	Draws              int                      `json:"draws"`
	FirstMoverWinRate  float64                  `json:"first_mover_win_rate"`
	SecondMoverWinRate float64                  `json:"second_mover_win_rate"`
	DrawRate           float64                  `json:"draw_rate"`
	AveragePlies       float64                  `json:"average_plies"`
	ByOpening          []researchOpeningStatDTO `json:"by_opening"`
}

type researchResultDTO struct {
	StartedAt       string                     `json:"started_at"`
	FinishedAt      string                     `json:"finished_at"`
	GamesPerOpening int                        `json:"games_per_opening"`
	WhiteOpenings   []researchOpeningReportDTO `json:"white_openings"`
	BlackOpenings   []researchOpeningReportDTO `json:"black_openings"`
	ConnectFour     researchConnectFourDTO     `json:"connect_four"`
}

type researchStatusResponse struct {
	Status  string              `json:"status"`
	CanRun  bool                `json:"can_run"`
	Error   string              `json:"error,omitempty"`
	Result  *researchResultDTO  `json:"result,omitempty"`
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

	byOpening := make([]researchOpeningStatDTO, 0, len(res.ConnectFour.ByOpening))
	for _, entry := range res.ConnectFour.SortedOpenings() {
		byOpening = append(byOpening, researchOpeningStatDTO{
			Opening:        entry.Opening,
			Games:          entry.Stat.Games,
			FirstMoverWins: entry.Stat.FirstMoverWins,
			Draws:          entry.Stat.Draws,
			WinRate:        entry.Stat.WinRate,
		})
	}

	return &researchResultDTO{
		StartedAt:       res.StartedAt.UTC().Format(time.RFC3339),
		FinishedAt:      res.FinishedAt.UTC().Format(time.RFC3339),
		GamesPerOpening: res.GamesPerOpening,
		WhiteOpenings:   toOpeningReportDTOs(res.WhiteOpenings),
		BlackOpenings:   toOpeningReportDTOs(res.BlackOpenings),
		ConnectFour: researchConnectFourDTO{
			Games:              res.ConnectFour.Games,
			FirstMoverWins:     res.ConnectFour.FirstMoverWins,
			SecondMoverWins:    res.ConnectFour.SecondMoverWins,
			Draws:              res.ConnectFour.Draws,
			FirstMoverWinRate:  res.ConnectFour.FirstMoverWinRate,
			SecondMoverWinRate: res.ConnectFour.SecondMoverWinRate,
			DrawRate:           res.ConnectFour.DrawRate,
			AveragePlies:       res.ConnectFour.AveragePlies,
			ByOpening:          byOpening,
		},
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
	status, result, errMsg := s.researchRunner.Snapshot()

	writeJSON(w, http.StatusOK, researchStatusResponse{
		Status: string(status),
		CanRun: s.canRunResearch(r),
		Error:  errMsg,
		Result: toResultDTO(result),
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
