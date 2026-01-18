package ws

import (
	"encoding/json"
	"log"
	"sync"
)

// MessageType defines the type of WebSocket message
type MessageType string

// Host message types
const (
	MsgRoomStarted          MessageType = "room_started"
	MsgRoomEnded            MessageType = "room_ended"
	MsgPlayerJoined         MessageType = "player_joined"
	MsgPlayerLeft           MessageType = "player_left"
	MsgLeaderboardUpdate    MessageType = "leaderboard_update"
	MsgPlayerProgressUpdate MessageType = "player_progress_update"
	MsgAnalyticsUpdate      MessageType = "analytics_update"
)

// Player message types
const (
	MsgNextQuestion     MessageType = "next_question"
	MsgEvaluationResult MessageType = "evaluation_result"
	MsgError            MessageType = "error"
)

// Message is the WebSocket envelope format
type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Hub manages WebSocket connections for rooms
type Hub struct {
	// Room -> connections
	hostConns   map[string]*Connection
	playerConns map[string]map[string]*Connection // roomCode -> playerID -> conn

	mu sync.RWMutex

	// Channels for coordination
	register   chan *Connection
	unregister chan *Connection
	broadcast  chan *BroadcastMessage
}

// Connection represents a WebSocket connection
type Connection struct {
	RoomCode string
	PlayerID string // Empty for host connections
	IsHost   bool
	Send     chan []byte
	Hub      *Hub
}

// BroadcastMessage is a message to broadcast
type BroadcastMessage struct {
	RoomCode string
	ToHost   bool
	ToPlayer string // Empty means all players, specific ID means one player
	Message  *Message
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	h := &Hub{
		hostConns:   make(map[string]*Connection),
		playerConns: make(map[string]map[string]*Connection),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		broadcast:   make(chan *BroadcastMessage, 256),
	}
	go h.run()
	return h
}

func (h *Hub) run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			if conn.IsHost {
				h.hostConns[conn.RoomCode] = conn
				log.Printf("Host connected to room %s", conn.RoomCode)
			} else {
				if h.playerConns[conn.RoomCode] == nil {
					h.playerConns[conn.RoomCode] = make(map[string]*Connection)
				}
				h.playerConns[conn.RoomCode][conn.PlayerID] = conn
				log.Printf("Player %s connected to room %s", conn.PlayerID, conn.RoomCode)

				// Notify host
				h.notifyHostPlayerJoined(conn.RoomCode, conn.PlayerID)
			}
			h.mu.Unlock()

		case conn := <-h.unregister:
			h.mu.Lock()
			if conn.IsHost {
				if existing, ok := h.hostConns[conn.RoomCode]; ok && existing == conn {
					delete(h.hostConns, conn.RoomCode)
					close(conn.Send)
					log.Printf("Host disconnected from room %s", conn.RoomCode)
				}
			} else {
				if players, ok := h.playerConns[conn.RoomCode]; ok {
					if existing, ok := players[conn.PlayerID]; ok && existing == conn {
						delete(players, conn.PlayerID)
						close(conn.Send)
						log.Printf("Player %s disconnected from room %s", conn.PlayerID, conn.RoomCode)

						// Notify host
						h.notifyHostPlayerLeft(conn.RoomCode, conn.PlayerID)
					}
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			data, _ := json.Marshal(msg.Message)

			if msg.ToHost {
				if conn, ok := h.hostConns[msg.RoomCode]; ok {
					select {
					case conn.Send <- data:
					default:
						// Drop message if buffer full
					}
				}
			} else if msg.ToPlayer != "" {
				// Send to specific player
				if players, ok := h.playerConns[msg.RoomCode]; ok {
					if conn, ok := players[msg.ToPlayer]; ok {
						select {
						case conn.Send <- data:
						default:
						}
					}
				}
			} else {
				// Broadcast to all players
				if players, ok := h.playerConns[msg.RoomCode]; ok {
					for _, conn := range players {
						select {
						case conn.Send <- data:
						default:
						}
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a connection
func (h *Hub) Register(conn *Connection) {
	h.register <- conn
}

// Unregister removes a connection
func (h *Hub) Unregister(conn *Connection) {
	h.unregister <- conn
}

// BroadcastToHost sends a message to the room host (implements service.Broadcaster)
func (h *Hub) BroadcastToHost(roomCode string, msgType string, payload interface{}) {
	data, _ := json.Marshal(payload)
	h.broadcast <- &BroadcastMessage{
		RoomCode: roomCode,
		ToHost:   true,
		Message: &Message{
			Type:    MessageType(msgType),
			Payload: data,
		},
	}
}

// BroadcastToPlayer sends a message to a specific player (implements service.Broadcaster)
func (h *Hub) BroadcastToPlayer(roomCode, playerID string, msgType string, payload interface{}) {
	data, _ := json.Marshal(payload)
	h.broadcast <- &BroadcastMessage{
		RoomCode: roomCode,
		ToPlayer: playerID,
		Message: &Message{
			Type:    MessageType(msgType),
			Payload: data,
		},
	}
}

// BroadcastToAllPlayers sends a message to all players in a room (implements service.Broadcaster)
func (h *Hub) BroadcastToAllPlayers(roomCode string, msgType string, payload interface{}) {
	data, _ := json.Marshal(payload)
	h.broadcast <- &BroadcastMessage{
		RoomCode: roomCode,
		ToPlayer: "", // Empty means all
		Message: &Message{
			Type:    MessageType(msgType),
			Payload: data,
		},
	}
}

func (h *Hub) notifyHostPlayerJoined(roomCode, playerID string) {
	if conn, ok := h.hostConns[roomCode]; ok {
		data, _ := json.Marshal(&Message{
			Type:    MsgPlayerJoined,
			Payload: json.RawMessage(`{"playerId":"` + playerID + `"}`),
		})
		select {
		case conn.Send <- data:
		default:
		}
	}
}

func (h *Hub) notifyHostPlayerLeft(roomCode, playerID string) {
	if conn, ok := h.hostConns[roomCode]; ok {
		data, _ := json.Marshal(&Message{
			Type:    MsgPlayerLeft,
			Payload: json.RawMessage(`{"playerId":"` + playerID + `"}`),
		})
		select {
		case conn.Send <- data:
		default:
		}
	}
}
