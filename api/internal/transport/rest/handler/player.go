package handler

import (
	"2026champs/internal/model"
	"2026champs/internal/service"
	"2026champs/internal/transport/rest/middleware"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// PlayerHandler handles player endpoints
type PlayerHandler struct {
	playerSvc *service.PlayerService
	answerSvc *service.AnswerService
}

// NewPlayerHandler creates a new player handler
func NewPlayerHandler(playerSvc *service.PlayerService, answerSvc *service.AnswerService) *PlayerHandler {
	return &PlayerHandler{
		playerSvc: playerSvc,
		answerSvc: answerSvc,
	}
}

// GetCurrentQuestion handles GET /v1/rooms/{code}/question/current
func (h *PlayerHandler) GetCurrentQuestion(w http.ResponseWriter, r *http.Request) {
	roomCode := middleware.GetRoomCode(r.Context())
	playerID := middleware.GetPlayerID(r.Context())

	question, player, err := h.playerSvc.GetCurrentQuestion(r.Context(), roomCode, playerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"done":     false,
		"question": question,
		"player":   nil,
	}

	if player != nil {
		response["player"] = map[string]interface{}{
			"score": player.Score,
		}
	}

	if question == nil {
		response["done"] = true
	}

	writeJSON(w, http.StatusOK, response)
}

// DraftRequest is the request body for saving a draft
type DraftRequest struct {
	Draft string `json:"draft"`
}

// SaveDraft handles PUT /v1/rooms/{code}/questions/{questionKey}/draft
func (h *PlayerHandler) SaveDraft(w http.ResponseWriter, r *http.Request) {
	roomCode := middleware.GetRoomCode(r.Context())
	playerID := middleware.GetPlayerID(r.Context())
	questionKey := mux.Vars(r)["questionKey"]

	var req DraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.answerSvc.SaveDraft(r.Context(), roomCode, playerID, questionKey, req.Draft); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// SubmitAnswer handles POST /v1/rooms/{code}/answers
func (h *PlayerHandler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	roomCode := middleware.GetRoomCode(r.Context())
	playerID := middleware.GetPlayerID(r.Context())

	var req model.SubmitAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.answerSvc.SubmitAnswer(r.Context(), roomCode, playerID, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Skip handles POST /v1/rooms/{code}/questions/{questionKey}/skip
func (h *PlayerHandler) Skip(w http.ResponseWriter, r *http.Request) {
	roomCode := middleware.GetRoomCode(r.Context())
	playerID := middleware.GetPlayerID(r.Context())
	questionKey := mux.Vars(r)["questionKey"]

	nextQuestion, err := h.answerSvc.Skip(r.Context(), roomCode, playerID, questionKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if nextQuestion == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"done": true, "nextQuestion": nil})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"done": false, "nextQuestion": nextQuestion})
}
