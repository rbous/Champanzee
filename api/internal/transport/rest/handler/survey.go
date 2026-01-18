package handler

import (
	"2026champs/internal/model"
	"2026champs/internal/service"
	"2026champs/internal/transport/rest/middleware"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// SurveyHandler handles survey endpoints
type SurveyHandler struct {
	surveySvc *service.SurveyService
}

// NewSurveyHandler creates a new survey handler
func NewSurveyHandler(surveySvc *service.SurveyService) *SurveyHandler {
	return &SurveyHandler{surveySvc: surveySvc}
}

// CreateSurveyRequest is the request body for creating a survey
type CreateSurveyRequest struct {
	Title     string               `json:"title"`
	Intent    string               `json:"intent"`
	Settings  model.SurveySettings `json:"settings"`
	Questions []model.BaseQuestion `json:"questions"`
}

// Create handles POST /v1/surveys
func (h *SurveyHandler) Create(w http.ResponseWriter, r *http.Request) {
	hostID := middleware.GetHostID(r.Context())
	if hostID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateSurveyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Assign keys to questions if not provided
	for i := range req.Questions {
		if req.Questions[i].Key == "" {
			req.Questions[i].Key = "Q" + string(rune('1'+i))
		}
	}

	survey := &model.Survey{
		HostID:    hostID,
		Title:     req.Title,
		Intent:    req.Intent,
		Settings:  req.Settings,
		Questions: req.Questions,
	}

	id, err := h.surveySvc.Create(r.Context(), survey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"surveyId": id})
}

// Get handles GET /v1/surveys/{surveyId}
func (h *SurveyHandler) Get(w http.ResponseWriter, r *http.Request) {
	surveyID := mux.Vars(r)["surveyId"]

	survey, err := h.surveySvc.GetByID(r.Context(), surveyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if survey == nil {
		writeError(w, http.StatusNotFound, "survey not found")
		return
	}

	writeJSON(w, http.StatusOK, survey)
}

// List handles GET /v1/surveys
func (h *SurveyHandler) List(w http.ResponseWriter, r *http.Request) {
	hostID := middleware.GetHostID(r.Context())
	if hostID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	surveys, err := h.surveySvc.GetByHostID(r.Context(), hostID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"surveys": surveys})
}
