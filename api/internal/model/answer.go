package model

import "time"

type ResponseData struct {
	Text            string   `json:"text,omitempty" bson:"text,omitempty"`                       // For typed responses
	SelectedOption  string   `json:"selectedOption,omitempty" bson:"selectedOption,omitempty"`   // For pick/tap
	SelectedOptions []string `json:"selectedOptions,omitempty" bson:"selectedOptions,omitempty"` // For multiple selections
	Rating          int      `json:"rating,omitempty" bson:"rating,omitempty"`                   // For rate questions
	DragPosition    float64  `json:"dragPosition,omitempty" bson:"dragPosition,omitempty"`       // For drag questions (0-1 scale)
	Words           []string `json:"words,omitempty" bson:"words,omitempty"`                     // For type_words
}

type PointsBreakdown struct {
	SpeedPoints   int `json:"speedPoints" bson:"speedPoints"`     // Bonus for quick responses
	ClarityPoints int `json:"clarityPoints" bson:"clarityPoints"` // Bonus for clear responses
	InsightPoints int `json:"insightPoints" bson:"insightPoints"` // Bonus for unique insights
	StreakBonus   int `json:"streakBonus" bson:"streakBonus"`     // Bonus for consecutive responses
	TotalPoints   int `json:"totalPoints" bson:"totalPoints"`     // Sum of all points
}

type SentimentAnalysis struct {
	Sentiment  string   `json:"sentiment" bson:"sentiment"`   // "positive", "negative", "neutral"
	Confidence float64  `json:"confidence" bson:"confidence"` // 0-1 confidence score
	KeyThemes  []string `json:"keyThemes" bson:"keyThemes"`   // Extracted themes
	Emotion    string   `json:"emotion" bson:"emotion"`       // "frustrated", "confused", "satisfied", etc.
	Intensity  float64  `json:"intensity" bson:"intensity"`   // 0-1 intensity of emotion
}

type Answer struct {
	ID             string            `json:"id" bson:"_id,omitempty"`
	SessionID      string            `json:"sessionId" bson:"sessionId"`
	PlayerID       string            `json:"playerId" bson:"playerId"`
	QuestionID     string            `json:"questionId" bson:"questionId"`
	RoundNumber    int               `json:"roundNumber" bson:"roundNumber"`
	Response       ResponseData      `json:"response" bson:"response"`
	Points         PointsBreakdown   `json:"points" bson:"points"`
	Sentiment      SentimentAnalysis `json:"sentiment,omitempty" bson:"sentiment,omitempty"`
	TimeTakenSec   int               `json:"timeTakenSec" bson:"timeTakenSec"`
	IsSkipped      bool              `json:"isSkipped" bson:"isSkipped"`
	AnsweredAt     time.Time         `json:"answeredAt" bson:"answeredAt"`
	AIAnalysisUsed bool              `json:"aiAnalysisUsed" bson:"aiAnalysisUsed"` // Whether AI processed this response
}

// Helper methods for points calculation
func (a *Answer) CalculateTotalPoints() int {
	a.Points.TotalPoints = a.Points.SpeedPoints + a.Points.ClarityPoints + a.Points.InsightPoints + a.Points.StreakBonus
	return a.Points.TotalPoints
}

func (a *Answer) HasResponse() bool {
	return !a.IsSkipped && (a.Response.Text != "" || a.Response.SelectedOption != "" || len(a.Response.SelectedOptions) > 0 || a.Response.Rating > 0 || a.Response.DragPosition > 0 || len(a.Response.Words) > 0)
}
