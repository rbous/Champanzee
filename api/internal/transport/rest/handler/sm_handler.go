package handler

import (
	"2026champs/internal/service"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// SMHandler handles SurveyMonkey endpoints
type SMHandler struct {
	syncSvc   *service.SMSyncService
	surveySvc *service.SurveyService
}

// NewSMHandler creates a new SM handler
func NewSMHandler(syncSvc *service.SMSyncService, surveySvc *service.SurveyService) *SMHandler {
	return &SMHandler{
		syncSvc:   syncSvc,
		surveySvc: surveySvc,
	}
}

// CreateCollectorRequest is the request body for collector creation
type CreateCollectorRequest struct {
	Name string `json:"name,omitempty"`
}

// CreateCollector handles POST /v1/sm/surveys/{surveyId}/collectors/weblink
func (h *SMHandler) CreateCollector(w http.ResponseWriter, r *http.Request) {
	surveyID := mux.Vars(r)["surveyId"]
	if surveyID == "" {
		writeError(w, http.StatusBadRequest, "surveyId is required")
		return
	}

	var req CreateCollectorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Use default name if no body
		req.Name = "WebLink Collector"
	}

	if req.Name == "" {
		req.Name = "WebLink Collector"
	}

	collector, err := h.syncSvc.CreateCollector(r.Context(), surveyID, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"collectorId": collector.CollectorID,
		"weblinkUrl":  collector.WebLinkURL,
		"name":        collector.Name,
	})
}

// CreateSurveyFromInternalRequest is the request body for creating SM survey from internal survey
type CreateSurveyFromInternalRequest struct {
	SurveyID                 string   `json:"surveyId"`
	RecommendedNextQuestions []string `json:"recommendedNextQuestions,omitempty"` // AI-suggested questions
}

// CreateSurveyFromInternal handles POST /v1/sm/surveys/from-internal
func (h *SMHandler) CreateSurveyFromInternal(w http.ResponseWriter, r *http.Request) {
	log.Printf("[SM Handler] POST /sm/surveys/from-internal")

	var req CreateSurveyFromInternalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[SM Handler] ERROR: Invalid request body: %v", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SurveyID == "" {
		log.Printf("[SM Handler] ERROR: Missing surveyId")
		writeError(w, http.StatusBadRequest, "surveyId is required")
		return
	}

	log.Printf("[SM Handler] Creating SM survey from internal survey ID: %s", req.SurveyID)

	// Get internal survey
	survey, err := h.surveySvc.GetByID(r.Context(), req.SurveyID)
	if err != nil {
		log.Printf("[SM Handler] ERROR: Failed to get survey: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if survey == nil {
		log.Printf("[SM Handler] ERROR: Survey not found: %s", req.SurveyID)
		writeError(w, http.StatusNotFound, "survey not found")
		return
	}

	log.Printf("[SM Handler] Found survey: %s (Title: %s, %d questions)", survey.ID, survey.Title, len(survey.Questions))

	// 1. Check if already exists (Idempotency)
	if survey.SMSurveyID != "" && survey.SMWebLink != "" {
		log.Printf("[SM Handler] Returning existing SM survey: ID=%s", survey.SMSurveyID)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"smSurveyId": survey.SMSurveyID,
			"weblinkUrl": survey.SMWebLink,
			"title":      survey.Title,
			"existing":   true,
		})
		return
	}

	// 2. Create in SurveyMonkey with AI augmentation
	smSurveyID, weblinkURL, err := h.syncSvc.CreateSurveyFromInternal(r.Context(), survey, req.RecommendedNextQuestions)
	if err != nil {
		log.Printf("[SM Handler] ERROR: Failed to create SM survey: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 3. Persist back to Survey
	survey.SMSurveyID = smSurveyID
	survey.SMWebLink = weblinkURL
	if err := h.surveySvc.Update(r.Context(), survey); err != nil {
		log.Printf("[SM Handler] WARNING: Failed to persist SM meta to survey: %v", err)
		// Don't fail the request, just log warning
	}

	log.Printf("[SM Handler] SUCCESS: SM survey created - ID: %s, Weblink: %s", smSurveyID, weblinkURL)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"smSurveyId": smSurveyID,
		"weblinkUrl": weblinkURL,
		"title":      survey.Title,
	})
}

// Sync handles POST /v1/sm/surveys/{surveyId}/sync
func (h *SMHandler) Sync(w http.ResponseWriter, r *http.Request) {
	surveyID := mux.Vars(r)["surveyId"]
	if surveyID == "" {
		writeError(w, http.StatusBadRequest, "surveyId is required")
		return
	}

	result, err := h.syncSvc.Sync(r.Context(), surveyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Summary handles GET /v1/sm/surveys/{surveyId}/summary
func (h *SMHandler) Summary(w http.ResponseWriter, r *http.Request) {
	surveyID := mux.Vars(r)["surveyId"]
	if surveyID == "" {
		writeError(w, http.StatusBadRequest, "surveyId is required")
		return
	}

	summary, err := h.syncSvc.GetSummary(r.Context(), surveyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// Distribution handles GET /v1/sm/surveys/{surveyId}/distribution/{metric}
func (h *SMHandler) Distribution(w http.ResponseWriter, r *http.Request) {
	surveyID := mux.Vars(r)["surveyId"]
	metric := mux.Vars(r)["metric"]

	if surveyID == "" {
		writeError(w, http.StatusBadRequest, "surveyId is required")
		return
	}
	if metric == "" {
		writeError(w, http.StatusBadRequest, "metric is required")
		return
	}

	dist, err := h.syncSvc.GetDistribution(r.Context(), surveyID, metric)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dist)
}
