package service

// Broadcaster interface for WebSocket broadcasting (avoids import cycle)
type Broadcaster interface {
	BroadcastToHost(roomCode string, msgType string, payload interface{})
	BroadcastToPlayer(roomCode, playerID string, msgType string, payload interface{})
	BroadcastToAllPlayers(roomCode string, msgType string, payload interface{})
	DisconnectRoom(roomCode string)
}
