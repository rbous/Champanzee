package model

import "time"

type RoomStatus string

const (
	RoomWaiting RoomStatus = "waiting"
	RoomLive    RoomStatus = "live"
	RoomEnded   RoomStatus = "ended"
)

type RoomSettings struct {
	Mode               string `json:"mode" bson:"mode"` // "realtime" or "async"
	MaxPlayers         int    `json:"maxPlayers" bson:"maxPlayers"`
	TimePerQuestionSec int    `json:"timePerQuestionSec" bson:"timePerQuestionSec"`
	QuestionSetID      string `json:"questionSetId" bson:"questionSetId"`
}

type Room struct {
	Code            string       `json:"code" bson:"code"`
	Status          RoomStatus   `json:"status" bson:"status"`
	HostPlayerID    string       `json:"hostPlayerId" bson:"hostPlayerId"`
	ActiveSessionID string       `json:"activeSessionId" bson:"activeSessionId"`
	Settings        RoomSettings `json:"settings" bson:"settings"`
	CreatedAt       time.Time    `json:"createdAt" bson:"createdAt"`
}
