package handler

import (
	"2026champs/internal/cache"
	"2026champs/internal/model"
	"2026champs/internal/service"
	"2026champs/internal/transport/rest/middleware"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// RoomHandler handles room endpoints
type RoomHandler struct {
	roomSvc     *service.RoomService
	playerSvc   *service.PlayerService
	leaderboard cache.LeaderboardCache
}

// NewRoomHandler creates a new room handler
func NewRoomHandler(roomSvc *service.RoomService, playerSvc *service.PlayerService, leaderboard cache.LeaderboardCache) *RoomHandler {
	return &RoomHandler{
		roomSvc:     roomSvc,
		playerSvc:   playerSvc,
		leaderboard: leaderboard,
	}
}

// CreateRoomRequest is the request body for creating a room
type CreateRoomRequest struct {
	SurveyID         string              `json:"surveyId"`
	SettingsOverride *model.RoomSettings `json:"settingsOverride,omitempty"`
}

// Create handles POST /v1/rooms
func (h *RoomHandler) Create(w http.ResponseWriter, r *http.Request) {
	hostID := middleware.GetHostID(r.Context())
	if hostID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	settings := &model.RoomSettings{}
	if req.SettingsOverride != nil {
		settings = req.SettingsOverride
	}

	room, err := h.roomSvc.CreateRoom(r.Context(), req.SurveyID, hostID, settings)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"roomCode": room.Code,
		"roomId":   room.Code, // Using code as ID for simplicity
	})
}

// Get handles GET /v1/rooms/{code}
func (h *RoomHandler) Get(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]

	room, err := h.roomSvc.GetRoom(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if room == nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}

	writeJSON(w, http.StatusOK, room)
}

// Start handles POST /v1/rooms/{code}/start
func (h *RoomHandler) Start(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]
	hostID := middleware.GetHostID(r.Context())

	if err := h.roomSvc.StartRoom(r.Context(), code, hostID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ACTIVE"})
}

// End handles POST /v1/rooms/{code}/end
func (h *RoomHandler) End(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]
	hostID := middleware.GetHostID(r.Context())

	if err := h.roomSvc.EndRoom(r.Context(), code, hostID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ENDED"})
}

// JoinRequest is the request body for joining a room
type JoinRequest struct {
	Nickname string `json:"nickname"`
}

// Join handles POST /v1/rooms/{code}/join
func (h *RoomHandler) Join(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]

	var req JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Nickname == "" {
		writeError(w, http.StatusBadRequest, "nickname is required")
		return
	}

	resp, err := h.playerSvc.JoinRoom(r.Context(), code, req.Nickname)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Leaderboard handles GET /v1/rooms/{code}/leaderboard
func (h *RoomHandler) Leaderboard(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]

	topStr := r.URL.Query().Get("top")
	top := 20
	if topStr != "" {
		if n, err := strconv.Atoi(topStr); err == nil && n > 0 {
			top = n
		}
	}

	entries, err := h.playerSvc.GetLeaderboard(r.Context(), code, top)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"leaderboard": entries})
}
