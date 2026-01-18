package model

import "time"

// Player represents a participant in a room
type Player struct {
	ID            string    `json:"id" bson:"_id,omitempty"`
	RoomCode      string    `json:"roomCode" bson:"roomCode"`
	Nickname      string    `json:"nickname" bson:"nickname"`
	Score         int       `json:"score" bson:"score"`
	CurrentKey    string    `json:"currentKey" bson:"currentKey"`       // Current question key
	FollowUpsUsed int       `json:"followUpsUsed" bson:"followUpsUsed"` // Total follow-ups seen
	LastActiveAt  time.Time `json:"lastActiveAt" bson:"lastActiveAt"`
	JoinedAt      time.Time `json:"joinedAt" bson:"joinedAt"`
}

// PlayerState is the full Redis state for a player (extends Player with queue info)
type PlayerState struct {
	Player
	Queue         []string `json:"queue"`         // Remaining question keys
	ClosedParents []string `json:"closedParents"` // Base questions whose follow-ups are closed
}

// PlayerJoinResponse is returned when a player joins a room
type PlayerJoinResponse struct {
	PlayerID      string    `json:"playerId"`
	Token         string    `json:"token"`
	RoomMeta      *RoomMeta `json:"roomMeta"`
	FirstQuestion *Question `json:"firstQuestion,omitempty"`
}
