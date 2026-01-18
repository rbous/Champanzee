package service

import (
	"2026champs/internal/model"
	"2026champs/internal/repository"
	"context"
)

// SurveyService handles survey CRUD operations
type SurveyService struct {
	surveyRepo repository.SurveyRepo
}

// NewSurveyService creates a new survey service
func NewSurveyService(surveyRepo repository.SurveyRepo) *SurveyService {
	return &SurveyService{
		surveyRepo: surveyRepo,
	}
}

// Create creates a new survey
func (s *SurveyService) Create(ctx context.Context, survey *model.Survey) (string, error) {
	return s.surveyRepo.Create(ctx, survey)
}

// GetByID retrieves a survey by ID
func (s *SurveyService) GetByID(ctx context.Context, id string) (*model.Survey, error) {
	return s.surveyRepo.GetByID(ctx, id)
}

// GetByHostID retrieves all surveys for a host
func (s *SurveyService) GetByHostID(ctx context.Context, hostID string) ([]*model.Survey, error) {
	return s.surveyRepo.GetByHostID(ctx, hostID)
}

// Update updates an existing survey
func (s *SurveyService) Update(ctx context.Context, survey *model.Survey) error {
	return s.surveyRepo.Update(ctx, survey)
}

// Delete deletes a survey
func (s *SurveyService) Delete(ctx context.Context, id string) error {
	return s.surveyRepo.Delete(ctx, id)
}
