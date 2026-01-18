package model

// QuestionType defines the type of question
type QuestionType string

const (
	QuestionTypeEssay  QuestionType = "ESSAY"  // Free text, AI-evaluated, can gate
	QuestionTypeDegree QuestionType = "DEGREE" // Rating/slider, never gates
	QuestionTypeMCQ    QuestionType = "MCQ"    // Multiple choice, Kahoot-style, never gates
)

// Question is a runtime question instance (base or follow-up)
type Question struct {
	Key       string       `json:"key"`                 // e.g., "Q1", "Q1.1", "Q1.2"
	ParentKey string       `json:"parentKey,omitempty"` // For follow-ups, points to base question
	Type      QuestionType `json:"type"`
	Prompt    string       `json:"prompt"`
	Rubric    string       `json:"rubric,omitempty"` // Grading guidance for AI
	PointsMax int          `json:"pointsMax"`
	Threshold float64      `json:"threshold,omitempty"` // ESSAY: satisfactory threshold
	ScaleMin  int          `json:"scaleMin,omitempty"`  // DEGREE only
	ScaleMax  int          `json:"scaleMax,omitempty"`  // DEGREE only
	Options   []string     `json:"options,omitempty"`   // MCQ only
}

// FollowUpMode describes the type of follow-up
type FollowUpMode string

const (
	FollowUpClarify   FollowUpMode = "clarify"   // Missing details
	FollowUpDeepen    FollowUpMode = "deepen"    // Examples, specifics
	FollowUpBranch    FollowUpMode = "branch"    // Related angles within scope
	FollowUpChallenge FollowUpMode = "challenge" // Inconsistencies
	FollowUpCompare   FollowUpMode = "compare"   // Choose between options
)

// FollowUpPool holds pre-generated follow-up questions organized by mode
type FollowUpPool struct {
	Clarify   []Question `json:"clarify,omitempty"`
	Deepen    []Question `json:"deepen,omitempty"`
	Branch    []Question `json:"branch,omitempty"`
	Challenge []Question `json:"challenge,omitempty"`
	Compare   []Question `json:"compare,omitempty"`
}
