package model

import "time"

// SurveySettings configures survey behavior
type SurveySettings struct {
	SatisfactoryThreshold float64 `json:"satisfactoryThreshold" bson:"satisfactoryThreshold"` // 0-1, for ESSAY gating
	MaxFollowUps          int     `json:"maxFollowUps" bson:"maxFollowUps"`                   // per question
	DefaultPointsMax      int     `json:"defaultPointsMax" bson:"defaultPointsMax"`
	AllowSkipAfter        int     `json:"allowSkipAfter" bson:"allowSkipAfter"` // number of attempts before skip allowed
}

// Survey is a persistent template created by a host
type Survey struct {
	ID        string         `json:"id" bson:"_id,omitempty"`
	HostID    string         `json:"hostId" bson:"hostId"`
	Title     string         `json:"title" bson:"title"`
	Intent    string         `json:"intent" bson:"intent"` // Scope/purpose description
	Settings  SurveySettings `json:"settings" bson:"settings"`
	Questions []BaseQuestion `json:"questions" bson:"questions"`
	CreatedAt time.Time      `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt" bson:"updatedAt"`
}

// BaseQuestion is a question template in a survey
type BaseQuestion struct {
	Key       string       `json:"key" bson:"key"`   // e.g., "Q1", "Q2"
	Type      QuestionType `json:"type" bson:"type"` // ESSAY, DEGREE
	Prompt    string       `json:"prompt" bson:"prompt"`
	Rubric    string       `json:"rubric" bson:"rubric"`       // Grading guidance for AI
	PointsMax int          `json:"pointsMax" bson:"pointsMax"` // Max points for this question
	Threshold float64      `json:"threshold" bson:"threshold"` // Satisfactory threshold (ESSAY only)
	// For DEGREE type
	ScaleMin int `json:"scaleMin,omitempty" bson:"scaleMin,omitempty"`
	ScaleMax int `json:"scaleMax,omitempty" bson:"scaleMax,omitempty"`
}
