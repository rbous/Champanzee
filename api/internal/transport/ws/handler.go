package ws

import (
	"2026champs/internal/service"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev
	},
}

// Handler handles WebSocket connections
type Handler struct {
	hub     *Hub
	authSvc *service.AuthService
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, authSvc *service.AuthService) *Handler {
	return &Handler{
		hub:     hub,
		authSvc: authSvc,
	}
}

// HostWS handles GET /v1/ws/rooms/{code}/host
func (h *Handler) HostWS(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]
	token := r.URL.Query().Get("token")

	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	claims, err := h.authSvc.ValidateHostToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	conn := &Connection{
		RoomCode: code,
		IsHost:   true,
		Send:     make(chan []byte, 256),
		Hub:      h.hub,
	}

	h.hub.Register(conn)

	log.Printf("Host %s connected to room %s via WebSocket", claims.HostID, code)

	go h.writePump(wsConn, conn)
	go h.readPump(wsConn, conn)
}

// PlayerWS handles GET /v1/ws/rooms/{code}/player
func (h *Handler) PlayerWS(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]
	token := r.URL.Query().Get("token")

	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	claims, err := h.authSvc.ValidatePlayerToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	if claims.RoomCode != code {
		http.Error(w, "token not valid for this room", http.StatusForbidden)
		return
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	conn := &Connection{
		RoomCode: code,
		PlayerID: claims.PlayerID,
		IsHost:   false,
		Send:     make(chan []byte, 256),
		Hub:      h.hub,
	}

	h.hub.Register(conn)

	log.Printf("Player %s connected to room %s via WebSocket", claims.PlayerID, code)

	go h.writePump(wsConn, conn)
	go h.readPump(wsConn, conn)
}

func (h *Handler) readPump(wsConn *websocket.Conn, conn *Connection) {
	defer func() {
		h.hub.Unregister(conn)
		wsConn.Close()
	}()

	wsConn.SetReadLimit(maxMessageSize)
	wsConn.SetReadDeadline(time.Now().Add(pongWait))
	wsConn.SetPongHandler(func(string) error {
		wsConn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		// We don't process incoming messages for now
		// Future: handle client-side events like heartbeats
	}
}

func (h *Handler) writePump(wsConn *websocket.Conn, conn *Connection) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		wsConn.Close()
	}()

	for {
		select {
		case message, ok := <-conn.Send:
			wsConn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				wsConn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := wsConn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			wsConn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := wsConn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
