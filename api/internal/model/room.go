package model

import "time"

// RoomStatus represents the lifecycle of a room
type RoomStatus string

const (
	RoomStatusLobby  RoomStatus = "LOBBY"  // Created, waiting for players
	RoomStatusActive RoomStatus = "ACTIVE" // Game in progress
	RoomStatusEnded  RoomStatus = "ENDED"  // Game finished
)

// RoomSettings can override survey settings for this specific room
type RoomSettings struct {
	SatisfactoryThreshold *float64 `json:"satisfactoryThreshold,omitempty" bson:"satisfactoryThreshold,omitempty"`
	MaxFollowUps          *int     `json:"maxFollowUps,omitempty" bson:"maxFollowUps,omitempty"`
	AllowSkipImmediately  *bool    `json:"allowSkipImmediately,omitempty" bson:"allowSkipImmediately,omitempty"`
}

// Room is a live session created from a survey (ephemeral in Redis, persisted in Mongo for history)
type Room struct {
	Code         string       `json:"code" bson:"code"` // 6-char join code
	SurveyID     string       `json:"surveyId" bson:"surveyId"`
	HostID       string       `json:"hostId" bson:"hostId"`
	Status       RoomStatus   `json:"status" bson:"status"`
	Settings     RoomSettings `json:"settings" bson:"settings"`         // Overrides from survey
	ScopeSummary string       `json:"scopeSummary" bson:"scopeSummary"` // Short AI-generated scope
	CreatedAt    time.Time    `json:"createdAt" bson:"createdAt"`
	StartedAt    *time.Time   `json:"startedAt,omitempty" bson:"startedAt,omitempty"`
	EndedAt      *time.Time   `json:"endedAt,omitempty" bson:"endedAt,omitempty"`
}

// RoomMeta is the Redis-stored room metadata
type RoomMeta struct {
	SurveyID     string     `json:"surveyId"`
	HostID       string     `json:"hostId"`
	Status       RoomStatus `json:"status"`
	CreatedAt    time.Time  `json:"createdAt"`
	SettingsJSON string     `json:"settingsJson"`
	ScopeSummary string     `json:"scopeSummary"`
}
