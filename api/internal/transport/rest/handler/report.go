package handler

import (
	"2026champs/internal/service"
	"2026champs/internal/transport/rest/middleware"
	"net/http"

	"github.com/gorilla/mux"
)

// ReportHandler handles report endpoints
type ReportHandler struct {
	reportSvc *service.ReportService
}

// NewReportHandler creates a new report handler
func NewReportHandler(reportSvc *service.ReportService) *ReportHandler {
	return &ReportHandler{reportSvc: reportSvc}
}

// GetSnapshot handles GET /v1/reports/{roomCode}/snapshot
func (h *ReportHandler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	roomCode := mux.Vars(r)["roomCode"]
	hostID := middleware.GetHostID(r.Context())
	if hostID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	snapshot, err := h.reportSvc.GetSnapshot(r.Context(), roomCode)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if snapshot == nil {
		writeError(w, http.StatusNotFound, "snapshot not found")
		return
	}

	writeJSON(w, http.StatusOK, snapshot)
}

// GetAIReport handles GET /v1/reports/{roomCode}/ai
func (h *ReportHandler) GetAIReport(w http.ResponseWriter, r *http.Request) {
	roomCode := mux.Vars(r)["roomCode"]
	hostID := middleware.GetHostID(r.Context())
	if hostID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	report, err := h.reportSvc.GetAIReport(r.Context(), roomCode)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if report == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "not_started"})
		return
	}

	writeJSON(w, http.StatusOK, report)
}

// GenerateAIReport handles POST /v1/reports/{roomCode}/ai
func (h *ReportHandler) GenerateAIReport(w http.ResponseWriter, r *http.Request) {
	roomCode := mux.Vars(r)["roomCode"]
	hostID := middleware.GetHostID(r.Context())
	if hostID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Start async generation (in production, this would queue a job)
	go func() {
		h.reportSvc.GenerateAIReport(r.Context(), roomCode)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "generating"})
}
