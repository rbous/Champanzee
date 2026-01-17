package model

import "time"

type Player struct {
	ID       string    `json:"id" bson:"_id,omitempty"`
	RoomCode string    `json:"roomCode" bson:"roomCode"`
	Nickname string    `json:"nickname" bson:"nickname"`
	Score    int       `json:"score" bson:"score"`
	Rating   int       `json:"rating" bson:"rating"`
	JoinedAt time.Time `json:"joinedAt" bson:"joinedAt"`
}
