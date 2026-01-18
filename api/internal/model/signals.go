package model

// Signals contains AI-extracted structured data from an answer
type Signals struct {
	Themes             []string `json:"themes,omitempty"`       // Key themes mentioned
	Missing            []string `json:"missing,omitempty"`      // Missing details
	Specificity        float64  `json:"specificity"`            // 0-1
	Clarity            float64  `json:"clarity"`                // 0-1
	Sentiment          float64  `json:"sentiment"`              // -1 to 1
	ConfidenceLanguage float64  `json:"confidence_language"`    // 0-1
	Summary            string   `json:"summary,omitempty"`      // 1 sentence
	ClusterHint        string   `json:"cluster_hint,omitempty"` // Optional grouping hint
	RiskFlags          []string `json:"risk_flags,omitempty"`   // toxicity, spam, irrelevant
}

// EvaluationResult is the AI response for answer evaluation
type EvaluationResult struct {
	Resolution   string  `json:"resolution"`   // SAT, UNSAT
	QualityScore float64 `json:"qualityScore"` // 0-1
	Signals      Signals `json:"signals"`
	FollowUpHint string  `json:"followup_hint,omitempty"`  // Suggestion for follow-up type
	NotesForHost string  `json:"notes_for_host,omitempty"` // Private notes
}

// FollowUpGeneration is the AI response for follow-up generation
type FollowUpGeneration struct {
	FollowUps []GeneratedFollowUp `json:"followUps"`
}

// GeneratedFollowUp is a single AI-generated follow-up question
type GeneratedFollowUp struct {
	QuestionKey   string       `json:"questionKey"` // e.g., "Q1.1"
	ParentKey     string       `json:"parentKey"`   // e.g., "Q1"
	Type          QuestionType `json:"type"`
	Prompt        string       `json:"prompt"`
	Rubric        string       `json:"rubric,omitempty"`
	PointsMax     int          `json:"pointsMax"`
	Threshold     float64      `json:"threshold,omitempty"`
	ReasonInScope string       `json:"reason_in_scope"` // Why this is relevant to survey intent
}
