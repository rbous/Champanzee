package model

import "time"

// PlayerProfile stores L2 per-player analytics (rolling)
type PlayerProfile struct {
	PlayerID string `json:"playerId" bson:"playerId"`
	RoomCode string `json:"roomCode" bson:"roomCode"`

	// Effort tracking
	EffortTrend    float64 `json:"effortTrend" bson:"effortTrend"`       // 0-1, rolling avg of specificity+clarity
	AvgResponseLen int     `json:"avgResponseLen" bson:"avgResponseLen"` // Rolling avg word count

	// Consistency tracking
	ConsistencyScore float64  `json:"consistencyScore" bson:"consistencyScore"` // 0-1, do stances conflict?
	PreviousThemes   []string `json:"previousThemes" bson:"previousThemes"`     // Last N themes mentioned

	// Friction tracking
	FollowUpFriction float64 `json:"followupFriction" bson:"followupFriction"` // 0-1, skip/unsat rate
	SkipCount        int     `json:"skipCount" bson:"skipCount"`
	UnsatCount       int     `json:"unsatCount" bson:"unsatCount"`
	TotalAnswers     int     `json:"totalAnswers" bson:"totalAnswers"`

	// Affinity
	TopicAffinity map[string]int `json:"topicAffinity" bson:"topicAffinity"` // theme -> count

	// Style (derived, not stored)
	Style string `json:"style" bson:"style"` // "brief", "detailed", "emotional", "analytical"

	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

// QuestionProfile stores L3 per-question analytics
type QuestionProfile struct {
	RoomCode    string `json:"roomCode" bson:"roomCode"`
	QuestionKey string `json:"questionKey" bson:"questionKey"`

	// Theme tracking
	ThemeCounts   map[string]int `json:"themeCounts" bson:"themeCounts"`     // theme -> count
	MissingCounts map[string]int `json:"missingCounts" bson:"missingCounts"` // missing detail -> count

	// Misunderstandings (refreshed periodically by AI)
	Misunderstandings []string `json:"misunderstandings" bson:"misunderstandings"` // Top 3-5 bullets
	BestProbes        []string `json:"bestProbes" bson:"bestProbes"`               // AI-suggested follow-up angles

	// Resolution stats
	SatCount   int `json:"satCount" bson:"satCount"`
	UnsatCount int `json:"unsatCount" bson:"unsatCount"`
	SkipCount  int `json:"skipCount" bson:"skipCount"`

	// Follow-up effectiveness
	FollowUpTriggered int `json:"followupTriggered" bson:"followupTriggered"`
	FollowUpHelped    int `json:"followupHelped" bson:"followupHelped"` // Led to SAT or improved quality

	// Rating stats (for DEGREE type)
	RatingHist  map[int]int `json:"ratingHist" bson:"ratingHist"` // value -> count
	RatingSum   int         `json:"ratingSum" bson:"ratingSum"`
	RatingCount int         `json:"ratingCount" bson:"ratingCount"`

	// Mini-clusters (optional, for advanced analytics)
	Clusters []QuestionCluster `json:"clusters,omitempty" bson:"clusters,omitempty"`

	AnswerCount int       `json:"answerCount" bson:"answerCount"`
	UpdatedAt   time.Time `json:"updatedAt" bson:"updatedAt"`
}

// QuestionCluster is a mini-cluster of viewpoints
type QuestionCluster struct {
	Label       string   `json:"label" bson:"label"`
	Keywords    []string `json:"keywords" bson:"keywords"`
	PlayerCount int      `json:"playerCount" bson:"playerCount"`
}

// RoomMemory stores L4 room-wide analytics
type RoomMemory struct {
	RoomCode string `json:"roomCode" bson:"roomCode"`

	// Global themes
	GlobalThemesTop []ThemeCount `json:"globalThemesTop" bson:"globalThemesTop"` // Top 10 themes

	// Contrasts (axes of disagreement)
	Contrasts []Contrast `json:"contrasts" bson:"contrasts"` // 2-5 axes

	// Friction points
	FrictionPoints []FrictionPoint `json:"frictionPoints" bson:"frictionPoints"` // Questions with high skip/unsat

	// Outliers
	OutlierThemes []string `json:"outlierThemes" bson:"outlierThemes"` // Rare but interesting

	// AI suggestions
	RecommendedProbes []string `json:"recommendedProbes" bson:"recommendedProbes"` // Best next questions

	// Stats
	TotalPlayers   int     `json:"totalPlayers" bson:"totalPlayers"`
	TotalAnswers   int     `json:"totalAnswers" bson:"totalAnswers"`
	CompletionRate float64 `json:"completionRate" bson:"completionRate"` // % who finished all questions

	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

// ThemeCount is a theme with its count
type ThemeCount struct {
	Theme string `json:"theme" bson:"theme"`
	Count int    `json:"count" bson:"count"`
}

// Contrast represents an axis of disagreement
type Contrast struct {
	Axis       string `json:"axis" bson:"axis"`   // e.g., "onboarding perception"
	SideA      string `json:"sideA" bson:"sideA"` // e.g., "fast but confusing"
	SideB      string `json:"sideB" bson:"sideB"` // e.g., "slow but clear"
	SideACount int    `json:"sideACount" bson:"sideACount"`
	SideBCount int    `json:"sideBCount" bson:"sideBCount"`
}

// FrictionPoint is a question with high friction
type FrictionPoint struct {
	QuestionKey string  `json:"questionKey" bson:"questionKey"`
	SkipRate    float64 `json:"skipRate" bson:"skipRate"`
	UnsatRate   float64 `json:"unsatRate" bson:"unsatRate"`
	Reason      string  `json:"reason" bson:"reason"` // AI-hypothesized reason
}

// RoomSnapshot is the instant dashboard data (frozen on room end)
type RoomSnapshot struct {
	RoomCode string    `json:"roomCode" bson:"roomCode"`
	SurveyID string    `json:"surveyId" bson:"surveyId"`
	EndedAt  time.Time `json:"endedAt" bson:"endedAt"`

	// Final leaderboard
	Leaderboard []LeaderboardEntry `json:"leaderboard" bson:"leaderboard"`

	// Per-question profiles
	QuestionProfiles []QuestionProfile `json:"questionProfiles" bson:"questionProfiles"`

	// Room memory
	Memory RoomMemory `json:"memory" bson:"memory"`

	// Stats
	TotalPlayers    int     `json:"totalPlayers" bson:"totalPlayers"`
	CompletionRate  float64 `json:"completionRate" bson:"completionRate"`
	OverallSkipRate float64 `json:"overallSkipRate" bson:"overallSkipRate"`
}

// LeaderboardEntry for snapshot
type LeaderboardEntry struct {
	PlayerID string `json:"playerId" bson:"playerId"`
	Nickname string `json:"nickname" bson:"nickname"`
	Score    int    `json:"score" bson:"score"`
	Rank     int    `json:"rank" bson:"rank"`
}

// AIReport is the AI-generated insight report (async)
type AIReport struct {
	RoomCode string `json:"roomCode" bson:"roomCode"`
	Status   string `json:"status" bson:"status"` // "pending", "generating", "ready", "failed"

	// Report content (populated when ready)
	ExecutiveSummary     []string          `json:"executiveSummary,omitempty" bson:"executiveSummary,omitempty"`
	KeyThemes            []ThemeInsight    `json:"keyThemes,omitempty" bson:"keyThemes,omitempty"`
	Contrasts            []ContrastInsight `json:"contrasts,omitempty" bson:"contrasts,omitempty"`
	PerQuestionInsights  []QuestionInsight `json:"perQuestionInsights,omitempty" bson:"perQuestionInsights,omitempty"`
	FrictionAnalysis     []FrictionInsight `json:"frictionAnalysis,omitempty" bson:"frictionAnalysis,omitempty"`
	RecommendedQuestions []string          `json:"recommendedQuestions,omitempty" bson:"recommendedQuestions,omitempty"`
	RecommendedEdits     []QuestionEdit    `json:"recommendedEdits,omitempty" bson:"recommendedEdits,omitempty"`

	CreatedAt time.Time  `json:"createdAt" bson:"createdAt"`
	ReadyAt   *time.Time `json:"readyAt,omitempty" bson:"readyAt,omitempty"`
}

// ThemeInsight is a theme with analysis
type ThemeInsight struct {
	Name             string   `json:"name" bson:"name"`
	Meaning          string   `json:"meaning" bson:"meaning"`
	Percentage       float64  `json:"percentage" bson:"percentage"`
	EvidenceSnippets []string `json:"evidenceSnippets" bson:"evidenceSnippets"`
}

// ContrastInsight is a contrast with analysis
type ContrastInsight struct {
	Axis      string `json:"axis" bson:"axis"`
	SideA     string `json:"sideA" bson:"sideA"`
	SideB     string `json:"sideB" bson:"sideB"`
	Predictor string `json:"predictor,omitempty" bson:"predictor,omitempty"` // What predicts each side
}

// QuestionInsight is per-question analysis
type QuestionInsight struct {
	QuestionKey       string   `json:"questionKey" bson:"questionKey"`
	WhatWorked        []string `json:"whatWorked" bson:"whatWorked"`
	Misunderstandings []string `json:"misunderstandings" bson:"misunderstandings"`
	MissingDetails    []string `json:"missingDetails" bson:"missingDetails"`
	BestFollowUps     []string `json:"bestFollowUps" bson:"bestFollowUps"`
}

// FrictionInsight is friction analysis
type FrictionInsight struct {
	QuestionKey        string `json:"questionKey" bson:"questionKey"`
	IssueDescription   string `json:"issueDescription" bson:"issueDescription"`
	HypothesizedReason string `json:"hypothesizedReason" bson:"hypothesizedReason"`
}

// QuestionEdit is a recommended edit
type QuestionEdit struct {
	QuestionKey   string `json:"questionKey" bson:"questionKey"`
	CurrentText   string `json:"currentText" bson:"currentText"`
	SuggestedText string `json:"suggestedText" bson:"suggestedText"`
	Reason        string `json:"reason" bson:"reason"`
}
