package model

import "time"

type QuestionType string

const (
	QuestionTap       QuestionType = "tap"        // "Which screen felt more confusing?" (tap to choose)
	QuestionDrag      QuestionType = "drag"       // "Drag the frustration meter"
	QuestionPick      QuestionType = "pick"       // "Pick what slowed you down"
	QuestionPredict   QuestionType = "predict"    // "Predict what others disliked most"
	QuestionTypeWords QuestionType = "type_words" // "Type 3 words to describe the experience"
	QuestionRate      QuestionType = "rate"       // Rating scales, sliders
)

type PointsConfig struct {
	SpeedMultiplier   float64 `json:"speedMultiplier" bson:"speedMultiplier"`     // Bonus for quick responses
	ClarityMultiplier float64 `json:"clarityMultiplier" bson:"clarityMultiplier"` // Bonus for clear, detailed responses
	InsightMultiplier float64 `json:"insightMultiplier" bson:"insightMultiplier"` // Bonus for unique insights
	BasePoints        int     `json:"basePoints" bson:"basePoints"`               // Base points for participation
}

type Question struct {
	ID            string       `json:"id" bson:"_id,omitempty"`
	QuestionSetID string       `json:"questionSetId" bson:"questionSetId"`
	Type          QuestionType `json:"type" bson:"type"`
	Text          string       `json:"text" bson:"text"`
	Options       []string     `json:"options,omitempty" bson:"options,omitempty"`   // for pick/tap questions
	MinWords      int          `json:"minWords,omitempty" bson:"minWords,omitempty"` // for type_words
	MaxWords      int          `json:"maxWords,omitempty" bson:"maxWords,omitempty"` // for type_words
	ScaleMin      int          `json:"scaleMin,omitempty" bson:"scaleMin,omitempty"` // for rate questions (1-10, etc.)
	ScaleMax      int          `json:"scaleMax,omitempty" bson:"scaleMax,omitempty"` // for rate questions
	Points        PointsConfig `json:"points" bson:"points"`
	TimeLimitSec  int          `json:"timeLimitSec" bson:"timeLimitSec"`
	Category      string       `json:"category" bson:"category"`                       // e.g., "usability", "performance", "design"
	Priority      int          `json:"priority" bson:"priority"`                       // 1-10, higher = more important for analysis
	AIPrompts     []string     `json:"aiPrompts,omitempty" bson:"aiPrompts,omitempty"` // Hints for AI analysis
	CreatedAt     time.Time    `json:"createdAt" bson:"createdAt"`
}
