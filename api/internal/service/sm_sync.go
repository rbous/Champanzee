package service

import (
	"2026champs/internal/model"
	"2026champs/internal/repository"
	"context"
	"fmt"
	"log"
	"strconv"
	"time"
)

// SMSyncService handles SurveyMonkey data synchronization
type SMSyncService struct {
	client *SMClient
	repo   repository.SMRepo
	// Question mappings: questionID -> internal_key
	questionMappings map[string]string
	// Choice mappings: choiceID -> internal_value
	choiceMappings map[string]string
}

// NewSMSyncService creates a new sync service
func NewSMSyncService(client *SMClient, repo repository.SMRepo) *SMSyncService {
	return &SMSyncService{
		client:           client,
		repo:             repo,
		questionMappings: make(map[string]string),
		choiceMappings:   make(map[string]string),
	}
}

// ConfigureMappings sets the question and choice mappings
// Call this during initialization with your specific survey mappings
func (s *SMSyncService) ConfigureMappings(questions map[string]string, choices map[string]string) {
	s.questionMappings = questions
	s.choiceMappings = choices
}

// CreateCollector creates a weblink collector for survey
func (s *SMSyncService) CreateCollector(ctx context.Context, surveyID, name string) (*model.SMCollector, error) {
	if !s.client.IsConfigured() {
		return nil, fmt.Errorf("SM_ACCESS_TOKEN not configured")
	}

	// Call SM API
	resp, err := s.client.CreateCollector(surveyID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create collector: %w", err)
	}

	// Store in DB
	collector := &model.SMCollector{
		SurveyID:    surveyID,
		CollectorID: resp.ID,
		Name:        resp.Name,
		Type:        resp.Type,
		WebLinkURL:  resp.URL,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.UpsertCollector(ctx, collector); err != nil {
		return nil, fmt.Errorf("failed to store collector: %w", err)
	}

	return collector, nil
}

// Sync fetches and processes responses for a survey
func (s *SMSyncService) Sync(ctx context.Context, surveyID string) (*model.SMSyncResult, error) {
	if !s.client.IsConfigured() {
		return nil, fmt.Errorf("SM_ACCESS_TOKEN not configured")
	}

	result := &model.SMSyncResult{}

	// Get last sync timestamp (optional optimization)
	var modifiedSince *time.Time

	// Fetch bulk response list
	list, err := s.client.ListResponses(surveyID, modifiedSince)
	if err != nil {
		return nil, fmt.Errorf("failed to list responses: %w", err)
	}

	result.Fetched = len(list.Data)
	log.Printf("SM Sync: found %d responses for survey %s", result.Fetched, surveyID)

	// Process each response
	for _, bulk := range list.Data {
		// Check if we need to update (compare date_modified if stored)
		existing, _ := s.repo.GetRawResponse(ctx, bulk.ID)

		bulkModified, _ := time.Parse(time.RFC3339, bulk.DateModified)
		if existing != nil && !bulkModified.After(existing.DateModified) {
			continue // Already up to date
		}

		// Fetch full details
		details, raw, err := s.client.GetResponseDetails(surveyID, bulk.ID)
		if err != nil {
			log.Printf("Warning: failed to fetch response %s: %v", bulk.ID, err)
			continue
		}

		// Store raw response (Layer 1)
		dateCreated, _ := time.Parse(time.RFC3339, details.DateCreated)
		dateModified, _ := time.Parse(time.RFC3339, details.DateModified)

		rawResponse := &model.SMResponseRaw{
			ResponseID:    details.ID,
			SurveyID:      details.SurveyID,
			CollectorID:   details.CollectorID,
			Status:        details.ResponseStatus,
			DateCreated:   dateCreated,
			DateModified:  dateModified,
			Raw:           raw,
			SchemaVersion: 1,
		}

		if details.ResponseStatus == "completed" {
			rawResponse.SubmittedAt = &dateModified
		}

		if err := s.repo.UpsertRawResponse(ctx, rawResponse); err != nil {
			log.Printf("Warning: failed to store raw response %s: %v", bulk.ID, err)
			continue
		}
		result.InsertedRaw++

		// Parse answers (Layer 2)
		answers, err := s.parseAnswers(details, rawResponse.SubmittedAt)
		if err != nil {
			log.Printf("Warning: failed to parse answers for %s: %v", bulk.ID, err)
			continue
		}

		// Delete existing answers for idempotency
		if err := s.repo.DeleteAnswersByResponseID(ctx, details.ID); err != nil {
			log.Printf("Warning: failed to delete old answers for %s: %v", bulk.ID, err)
		}

		// Insert new answers
		if err := s.repo.InsertAnswers(ctx, answers); err != nil {
			log.Printf("Warning: failed to insert answers for %s: %v", bulk.ID, err)
		} else {
			result.ParsedAnswers += len(answers)
		}

		// Compute features (Layer 3)
		features := s.computeFeatures(details, answers)
		if err := s.repo.UpsertFeatures(ctx, features); err != nil {
			log.Printf("Warning: failed to upsert features for %s: %v", bulk.ID, err)
		} else {
			result.UpdatedFeatures++
		}
	}

	log.Printf("SM Sync complete: %+v", result)
	return result, nil
}

// parseAnswers converts response details to normalized answer cells
func (s *SMSyncService) parseAnswers(details *SMResponseDetails, submittedAt *time.Time) ([]*model.SMAnswer, error) {
	var answers []*model.SMAnswer

	submitTime := time.Now()
	if submittedAt != nil {
		submitTime = *submittedAt
	}

	for _, page := range details.Pages {
		for _, q := range page.Questions {
			for _, a := range q.Answers {
				answer := &model.SMAnswer{
					ResponseID:  details.ID,
					SurveyID:    details.SurveyID,
					CollectorID: details.CollectorID,
					QuestionID:  q.ID,
					SubmittedAt: submitTime,
				}

				// Determine answer type and extract values
				if a.ChoiceID != "" {
					answer.AnswerType = model.SMAnswerTypeChoice
					answer.ChoiceID = &a.ChoiceID
				} else if a.Text != "" {
					// Check if it's numeric
					if num, err := strconv.Atoi(a.Text); err == nil {
						answer.AnswerType = model.SMAnswerTypeNumber
						answer.NumericValue = &num
					} else {
						answer.AnswerType = model.SMAnswerTypeText
						answer.TextValue = &a.Text
					}
				}

				if a.RowID != "" {
					answer.RowID = &a.RowID
					answer.AnswerType = model.SMAnswerTypeMatrix
				}

				answers = append(answers, answer)
			}
		}
	}

	return answers, nil
}

// computeFeatures derives analytics-ready features from answers
func (s *SMSyncService) computeFeatures(details *SMResponseDetails, answers []*model.SMAnswer) *model.SMResponseFeatures {
	dateModified, _ := time.Parse(time.RFC3339, details.DateModified)

	features := &model.SMResponseFeatures{
		ResponseID:  details.ID,
		SurveyID:    details.SurveyID,
		CollectorID: details.CollectorID,
		SubmittedAt: dateModified,
		Segments:    make(map[string]interface{}),
	}

	// Map answers to features using internal keys
	for _, a := range answers {
		internalKey, ok := s.questionMappings[a.QuestionID]
		if !ok {
			continue
		}

		switch internalKey {
		case "overall_satisfaction":
			if a.NumericValue != nil {
				features.OverallSatisfaction = a.NumericValue
			}
		case "battery_rating":
			if a.NumericValue != nil {
				features.BatteryRating = a.NumericValue
			}
		case "camera_rating":
			if a.NumericValue != nil {
				features.CameraRating = a.NumericValue
			}
		case "top_feature":
			if a.ChoiceID != nil {
				if val, ok := s.choiceMappings[*a.ChoiceID]; ok {
					features.TopFeature = &val
				} else {
					features.TopFeature = a.ChoiceID
				}
			}
		case "main_issue":
			if a.TextValue != nil {
				features.MainIssueText = a.TextValue
			}
		default:
			// Store in segments for custom mappings
			if a.TextValue != nil {
				features.Segments[internalKey] = *a.TextValue
			} else if a.NumericValue != nil {
				features.Segments[internalKey] = *a.NumericValue
			} else if a.ChoiceID != nil {
				if val, ok := s.choiceMappings[*a.ChoiceID]; ok {
					features.Segments[internalKey] = val
				}
			}
		}
	}

	return features
}

// GetSummary returns analytics summary for a survey
func (s *SMSyncService) GetSummary(ctx context.Context, surveyID string) (*model.SMSurveySummary, error) {
	return s.repo.GetSurveySummary(ctx, surveyID)
}

// GetDistribution returns histogram for a metric
func (s *SMSyncService) GetDistribution(ctx context.Context, surveyID, metric string) (*model.SMDistribution, error) {
	// Validate metric
	validMetrics := map[string]bool{
		"overall_satisfaction": true,
		"battery_rating":       true,
		"camera_rating":        true,
	}

	if !validMetrics[metric] {
		return nil, fmt.Errorf("invalid metric: %s", metric)
	}

	return s.repo.GetDistribution(ctx, surveyID, metric)
}

// CreateSurveyFromInternal creates a SurveyMonkey survey from an internal survey
func (s *SMSyncService) CreateSurveyFromInternal(ctx context.Context, survey *model.Survey, extraQuestions []string) (string, string, error) {
	log.Printf("[SM Sync] Starting survey creation from internal survey: ID=%s, Title=%s", survey.ID, survey.Title)
	log.Printf("[SM Sync] Survey has %d questions + %d AI recommended questions", len(survey.Questions), len(extraQuestions))

	if !s.client.IsConfigured() {
		log.Printf("[SM Sync] ERROR: SM_ACCESS_TOKEN not configured")
		return "", "", fmt.Errorf("SM_ACCESS_TOKEN not configured")
	}

	// Create survey in SurveyMonkey
	log.Printf("[SM Sync] Creating survey in SurveyMonkey...")
	smSurvey, err := s.client.CreateSurvey(survey.Title)
	if err != nil {
		log.Printf("[SM Sync] ERROR: Failed to create SM survey: %v", err)
		return "", "", fmt.Errorf("failed to create SM survey: %w", err)
	}

	log.Printf("[SM Sync] ✓ Survey created: %s (ID: %s)", smSurvey.Title, smSurvey.ID)

	// Get the default page
	log.Printf("[SM Sync] Fetching survey pages...")
	pages, err := s.client.GetSurveyPages(smSurvey.ID)
	if err != nil {
		log.Printf("[SM Sync] ERROR: Failed to get pages: %v", err)
		return "", "", fmt.Errorf("failed to get pages: %w", err)
	}

	if len(pages) == 0 {
		log.Printf("[SM Sync] ERROR: No pages found in survey")
		return "", "", fmt.Errorf("no pages found in survey")
	}

	pageID := pages[0].ID
	log.Printf("[SM Sync] Using page ID: %s", pageID)

	position := 1
	successCount := 0

	// 1. Convert and add standard questions
	log.Printf("[SM Sync] Converting and adding %d standard questions...", len(survey.Questions))
	for i, q := range survey.Questions {
		log.Printf("[SM Sync] Question %d/%d: %s (Type: %s)", i+1, len(survey.Questions), q.Key, q.Type)
		smQuestion := s.convertQuestion(q, position)

		_, err := s.client.CreateQuestion(smSurvey.ID, pageID, smQuestion)
		if err != nil {
			log.Printf("[SM Sync] WARNING: Failed to create question %s: %v", q.Key, err)
			continue
		}

		successCount++
		position++
		log.Printf("[SM Sync] ✓ Question %d created: %s", i+1, q.Prompt)
	}

	// 2. Add AI Recommended Questions
	if len(extraQuestions) > 0 {
		log.Printf("[SM Sync] Adding %d AI recommended questions...", len(extraQuestions))
		for i, prompt := range extraQuestions {
			log.Printf("[SM Sync] AI Question %d: %s", i+1, prompt)

			// Create simple open-ended question
			req := SMQuestionCreateRequest{
				Headings: []struct {
					Heading string `json:"heading"`
				}{{Heading: prompt}},
				Family:   "open_ended",
				Subtype:  "essay",
				Position: position,
			}

			_, err := s.client.CreateQuestion(smSurvey.ID, pageID, req)
			if err != nil {
				log.Printf("[SM Sync] WARNING: Failed to create AI question: %v", err)
				continue
			}
			position++
			log.Printf("[SM Sync] ✓ AI Question created")
		}
	}

	log.Printf("[SM Sync] Questions created: %d/%d standard + AI questions successful", successCount, len(survey.Questions))

	// Auto-create weblink collector
	log.Printf("[SM Sync] Creating weblink collector...")
	collector, err := s.CreateCollector(ctx, smSurvey.ID, survey.Title+" - Weblink")
	if err != nil {
		log.Printf("[SM Sync] WARNING: Failed to create collector: %v", err)
		return smSurvey.ID, "", nil
	}

	log.Printf("[SM Sync] ✓ Weblink collector created: %s", collector.WebLinkURL)
	log.Printf("[SM Sync] ✓✓✓ Survey creation complete! SM Survey ID: %s, Weblink: %s", smSurvey.ID, collector.WebLinkURL)
	return smSurvey.ID, collector.WebLinkURL, nil
}

// convertQuestion converts internal question to SurveyMonkey format
func (s *SMSyncService) convertQuestion(q model.BaseQuestion, position int) SMQuestionCreateRequest {
	log.Printf("[SM Sync] Converting question: Key=%s, Type=%s, Position=%d", q.Key, q.Type, position)

	req := SMQuestionCreateRequest{
		Headings: []struct {
			Heading string `json:"heading"`
		}{
			{Heading: q.Prompt},
		},
		Position: position,
	}

	switch q.Type {
	case model.QuestionTypeEssay:
		log.Printf("[SM Sync] Converting ESSAY → open_ended")
		req.Family = "open_ended"
		req.Subtype = "essay"

	case model.QuestionTypeDegree:
		log.Printf("[SM Sync] Converting DEGREE → rating scale (%d-%d)", q.ScaleMin, q.ScaleMax)
		req.Family = "single_choice"
		req.Subtype = "vertical"

		// Create rating scale choices
		choices := make([]map[string]interface{}, 0)
		for i := q.ScaleMin; i <= q.ScaleMax; i++ {
			choices = append(choices, map[string]interface{}{
				"text": fmt.Sprintf("%d", i),
			})
		}
		req.Answers = map[string]interface{}{
			"choices": choices,
		}
		log.Printf("[SM Sync] Created %d rating choices", len(choices))

	case model.QuestionTypeMCQ:
		log.Printf("[SM Sync] Converting MCQ → multiple choice (%d options)", len(q.Options))
		req.Family = "single_choice"
		req.Subtype = "vertical"

		// Create multiple choice options
		choices := make([]map[string]interface{}, 0)
		for _, opt := range q.Options {
			choices = append(choices, map[string]interface{}{
				"text": opt,
			})
		}
		req.Answers = map[string]interface{}{
			"choices": choices,
		}
		log.Printf("[SM Sync] Created %d choice options", len(choices))
	}

	return req
}
