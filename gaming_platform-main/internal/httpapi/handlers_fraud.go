package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"gaming_platform/internal/auth"
	db "gaming_platform/internal/db/gen"
	"gaming_platform/internal/dbutil"
)

func (s *Server) handleCreateFraudReport(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserID(r.Context())

	var req createFraudReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ReportedUserID == 0 || req.Reason == "" {
		writeError(w, http.StatusBadRequest, "reported_user_id and reason are required")
		return
	}
	if req.ReportedUserID == userID {
		writeError(w, http.StatusBadRequest, "cannot report yourself")
		return
	}

	var tournamentID, matchID pgtype.Int8
	if req.TournamentID != nil {
		tournamentID = dbutil.Int64ToPGInt8(*req.TournamentID)
	}
	if req.MatchID != nil {
		matchID = dbutil.Int64ToPGInt8(*req.MatchID)
	}

	q := db.New(s.pool)
	report, err := q.CreateFraudReport(r.Context(), db.CreateFraudReportParams{
		ReporterUserID: userID,
		ReportedUserID: req.ReportedUserID,
		TournamentID:   tournamentID,
		MatchID:        matchID,
		Reason:         dbutil.TextToPGText(req.Reason),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create fraud report")
		return
	}

	writeJSON(w, http.StatusCreated, toFraudReportResponse(report))
}

func (s *Server) handleGetFraudReport(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "fraudReportID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid fraud report id")
		return
	}

	q := db.New(s.pool)
	report, err := q.GetFraudReport(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "fraud report not found")
		return
	}

	writeJSON(w, http.StatusOK, toFraudReportResponse(report))
}

func (s *Server) handleListFraudReports(w http.ResponseWriter, r *http.Request) {
	var status pgtype.Text
	if v := r.URL.Query().Get("status"); v != "" {
		status = dbutil.TextToPGText(v)
	}

	q := db.New(s.pool)
	reports, err := q.ListFraudReports(r.Context(), status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list fraud reports")
		return
	}

	resp := make([]fraudReportResponse, 0, len(reports))
	for _, rpt := range reports {
		resp = append(resp, toFraudReportResponse(rpt))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleUpdateFraudReportStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "fraudReportID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid fraud report id")
		return
	}

	var req updateFraudReportStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}

	q := db.New(s.pool)
	report, err := q.UpdateFraudReportStatus(r.Context(), db.UpdateFraudReportStatusParams{
		FraudReportID: id,
		Status:        dbutil.TextToPGText(req.Status),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update fraud report")
		return
	}

	writeJSON(w, http.StatusOK, toFraudReportResponse(report))
}
