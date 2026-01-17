package model

import "time"

type SessionStatus string

const (
	SessionWaiting SessionStatus = "waiting"
	SessionActive  SessionStatus = "active"
	SessionEnded   SessionStatus = "ended"
)

type Session struct {
	ID        string        `json:"id" bson:"_id,omitempty"`
	RoomCode  string        `json:"roomCode" bson:"roomCode"`
	Status    SessionStatus `json:"status" bson:"status"`
	StartedAt time.Time     `json:"startedAt" bson:"startedAt"`
	EndedAt   *time.Time    `json:"endedAt,omitempty" bson:"endedAt,omitempty"`
}
