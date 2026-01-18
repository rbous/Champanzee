package model

import "time"

// AnswerResolution describes the outcome of an answer
type AnswerResolution string

const (
	ResolutionSat       AnswerResolution = "SAT"       // Satisfactory
	ResolutionUnsat     AnswerResolution = "UNSAT"     // Unsatisfactory, may need follow-up
	ResolutionSkipped   AnswerResolution = "SKIPPED"   // Player skipped
	ResolutionAbandoned AnswerResolution = "ABANDONED" // Player left without finishing
)

// AnswerStatus is the current state of the answer
type AnswerStatus string

const (
	AnswerStatusDraft     AnswerStatus = "DRAFT"
	AnswerStatusSubmitted AnswerStatus = "SUBMITTED"
	AnswerStatusEvaluated AnswerStatus = "EVALUATED"
)

// Answer represents a player's response to a question
type Answer struct {
	ID              string `json:"id" bson:"_id,omitempty"`
	RoomCode        string `json:"roomCode" bson:"roomCode"`
	PlayerID        string `json:"playerId" bson:"playerId"`
	QuestionKey     string `json:"questionKey" bson:"questionKey"`
	ClientAttemptID string `json:"clientAttemptId" bson:"clientAttemptId"` // For idempotency

	// Response data
	TextAnswer  string `json:"textAnswer,omitempty" bson:"textAnswer,omitempty"`   // For ESSAY
	DegreeValue int    `json:"degreeValue,omitempty" bson:"degreeValue,omitempty"` // For DEGREE
	OptionIndex *int   `json:"optionIndex,omitempty" bson:"optionIndex,omitempty"` // For MCQ

	// State
	Status     AnswerStatus     `json:"status" bson:"status"`
	Resolution AnswerResolution `json:"resolution,omitempty" bson:"resolution,omitempty"`
	Tries      int              `json:"tries" bson:"tries"`

	// Points
	PointsEarned int `json:"pointsEarned" bson:"pointsEarned"`

	// AI Evaluation
	Signals     *Signals `json:"signals,omitempty" bson:"signals,omitempty"`
	EvalSummary string   `json:"evalSummary,omitempty" bson:"evalSummary,omitempty"` // Short summary

	// Timestamps
	CreatedAt   time.Time  `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt" bson:"updatedAt"`
	EvaluatedAt *time.Time `json:"evaluatedAt,omitempty" bson:"evaluatedAt,omitempty"`
}

// AttemptState is stored in Redis per player per question
type AttemptState struct {
	DraftAnswer     string           `json:"draftAnswer,omitempty"`
	SubmittedAnswer string           `json:"submittedAnswer,omitempty"`
	Status          AnswerStatus     `json:"status"`
	Resolution      AnswerResolution `json:"resolution,omitempty"`
	Tries           int              `json:"tries"`
	EvalSummary     string           `json:"evalSummary,omitempty"`
	UpdatedAt       time.Time        `json:"updatedAt"`
}

// SubmitAnswerRequest is the request body for answer submission
type SubmitAnswerRequest struct {
	QuestionKey     string `json:"questionKey"`
	ClientAttemptID string `json:"clientAttemptId"` // Unique per submission attempt
	TextAnswer      string `json:"textAnswer,omitempty"`
	DegreeValue     int    `json:"degreeValue,omitempty"`
	OptionIndex     *int   `json:"optionIndex,omitempty"`
}

// SubmitAnswerResponse is returned after answer submission
type SubmitAnswerResponse struct {
	Status       AnswerStatus     `json:"status"`
	Resolution   AnswerResolution `json:"resolution,omitempty"`
	PointsEarned int              `json:"pointsEarned"`
	EvalSummary  string           `json:"evalSummary,omitempty"`
	NextQuestion *Question        `json:"nextQuestion,omitempty"`
	FollowUp     *Question        `json:"followUp,omitempty"` // If UNSAT and follow-up triggered
}
