package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SMResponseRaw is Layer 1: raw SurveyMonkey response storage
type SMResponseRaw struct {
	ID            primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	ResponseID    string                 `json:"responseId" bson:"response_id"`
	SurveyID      string                 `json:"surveyId" bson:"survey_id"`
	CollectorID   string                 `json:"collectorId" bson:"collector_id"`
	Status        string                 `json:"status" bson:"status"`
	DateCreated   time.Time              `json:"dateCreated" bson:"date_created"`
	DateModified  time.Time              `json:"dateModified" bson:"date_modified"`
	SubmittedAt   *time.Time             `json:"submittedAt,omitempty" bson:"submitted_at,omitempty"`
	LastSeenAt    time.Time              `json:"lastSeenAt" bson:"last_seen_at"`
	Raw           map[string]interface{} `json:"raw" bson:"raw"`
	SchemaVersion int                    `json:"schemaVersion" bson:"schema_version"`
}

// SMAnswerType defines the type of answer cell
type SMAnswerType string

const (
	SMAnswerTypeChoice SMAnswerType = "choice"
	SMAnswerTypeText   SMAnswerType = "text"
	SMAnswerTypeNumber SMAnswerType = "number"
	SMAnswerTypeMatrix SMAnswerType = "matrix"
)

// SMAnswer is Layer 2: normalized answer cell
type SMAnswer struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ResponseID   string             `json:"responseId" bson:"response_id"`
	SurveyID     string             `json:"surveyId" bson:"survey_id"`
	CollectorID  string             `json:"collectorId" bson:"collector_id"`
	QuestionID   string             `json:"questionId" bson:"question_id"`
	RowID        *string            `json:"rowId,omitempty" bson:"row_id,omitempty"`
	ChoiceID     *string            `json:"choiceId,omitempty" bson:"choice_id,omitempty"`
	TextValue    *string            `json:"textValue,omitempty" bson:"text_value,omitempty"`
	NumericValue *int               `json:"numericValue,omitempty" bson:"numeric_value,omitempty"`
	AnswerType   SMAnswerType       `json:"answerType" bson:"answer_type"`
	SubmittedAt  time.Time          `json:"submittedAt" bson:"submitted_at"`
}

// SMResponseFeatures is Layer 3: derived analytics-ready data
type SMResponseFeatures struct {
	ID                  primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	ResponseID          string                 `json:"responseId" bson:"response_id"`
	SurveyID            string                 `json:"surveyId" bson:"survey_id"`
	CollectorID         string                 `json:"collectorId" bson:"collector_id"`
	SubmittedAt         time.Time              `json:"submittedAt" bson:"submitted_at"`
	OverallSatisfaction *int                   `json:"overallSatisfaction,omitempty" bson:"overall_satisfaction,omitempty"`
	TopFeature          *string                `json:"topFeature,omitempty" bson:"top_feature,omitempty"`
	BatteryRating       *int                   `json:"batteryRating,omitempty" bson:"battery_rating,omitempty"`
	CameraRating        *int                   `json:"cameraRating,omitempty" bson:"camera_rating,omitempty"`
	MainIssueText       *string                `json:"mainIssueText,omitempty" bson:"main_issue_text,omitempty"`
	Segments            map[string]interface{} `json:"segments,omitempty" bson:"segments,omitempty"`
}

// SMCollector stores weblink collector info
type SMCollector struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	SurveyID    string             `json:"surveyId" bson:"survey_id"`
	CollectorID string             `json:"collectorId" bson:"collector_id"`
	Name        string             `json:"name" bson:"name"`
	Type        string             `json:"type" bson:"type"`
	WebLinkURL  string             `json:"webLinkUrl" bson:"weblink_url"`
	CreatedAt   time.Time          `json:"createdAt" bson:"created_at"`
}

// SMQuestionMapping maps SurveyMonkey question IDs to internal keys
type SMQuestionMapping struct {
	SurveyID    string `json:"surveyId" bson:"survey_id"`
	QuestionID  string `json:"questionId" bson:"question_id"`
	InternalKey string `json:"internalKey" bson:"internal_key"` // e.g., "battery_rating"
	Heading     string `json:"heading" bson:"heading"`
	Type        string `json:"type" bson:"type"`
}

// SMChoiceMapping maps choice IDs to internal values
type SMChoiceMapping struct {
	QuestionID    string `json:"questionId" bson:"question_id"`
	ChoiceID      string `json:"choiceId" bson:"choice_id"`
	Label         string `json:"label" bson:"label"`
	InternalValue string `json:"internalValue" bson:"internal_value"` // e.g., "Battery"
}

// SMSyncResult is returned after sync operation
type SMSyncResult struct {
	Fetched         int `json:"fetched"`
	InsertedRaw     int `json:"insertedRaw"`
	ParsedAnswers   int `json:"parsedAnswers"`
	UpdatedFeatures int `json:"updatedFeatures"`
}

// SMSurveySummary is the analytics summary response
type SMSurveySummary struct {
	TotalResponses         int              `json:"totalResponses"`
	AvgOverallSatisfaction float64          `json:"avgOverallSatisfaction"`
	TopFeatureCounts       []SMFeatureCount `json:"topFeatureCounts"`
	LatestSubmittedAt      *time.Time       `json:"latestSubmittedAt,omitempty"`
}

// SMFeatureCount for top feature ranking
type SMFeatureCount struct {
	Feature string `json:"feature"`
	Count   int    `json:"count"`
}

// SMDistribution is the histogram response
type SMDistribution struct {
	Metric     string      `json:"metric"`
	Histogram  map[int]int `json:"histogram"`
	TotalCount int         `json:"totalCount"`
}
