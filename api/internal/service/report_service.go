package service

import (
	"2026champs/internal/cache"
	"2026champs/internal/model"
	"2026champs/internal/repository"
	"context"
	"time"
)

// ReportService handles post-room report generation
type ReportService struct {
	roomRepo       repository.RoomRepo
	answerRepo     repository.AnswerRepo
	reportRepo     repository.ReportRepo
	surveyRepo     repository.SurveyRepo
	analyticsCache cache.AnalyticsCache
	leaderboard    cache.LeaderboardCache
	evaluator      *EvaluatorService
}

// NewReportService creates a new report service
func NewReportService(
	roomRepo repository.RoomRepo,
	answerRepo repository.AnswerRepo,
	reportRepo repository.ReportRepo,
	surveyRepo repository.SurveyRepo,
	analyticsCache cache.AnalyticsCache,
	leaderboard cache.LeaderboardCache,
	evaluator *EvaluatorService,
) *ReportService {
	return &ReportService{
		roomRepo:       roomRepo,
		answerRepo:     answerRepo,
		reportRepo:     reportRepo,
		surveyRepo:     surveyRepo,
		analyticsCache: analyticsCache,
		leaderboard:    leaderboard,
		evaluator:      evaluator,
	}
}

// CreateSnapshot creates the instant dashboard snapshot on room end
func (s *ReportService) CreateSnapshot(ctx context.Context, roomCode string, questionKeys []string) (*model.RoomSnapshot, error) {
	// Get room info
	room, err := s.roomRepo.GetByCode(ctx, roomCode)
	if err != nil {
		return nil, err
	}

	// Get leaderboard
	entries, err := s.leaderboard.GetTop(ctx, roomCode, 100)
	if err != nil {
		return nil, err
	}

	leaderboard := []model.LeaderboardEntry{}
	for i, e := range entries {
		leaderboard = append(leaderboard, model.LeaderboardEntry{
			PlayerID: e.PlayerID,
			Score:    int(e.Score),
			Rank:     i + 1,
		})
	}

	// Get question profiles
	profiles := []model.QuestionProfile{}
	for _, qKey := range questionKeys {
		profile, err := s.analyticsCache.GetQuestionProfile(ctx, roomCode, qKey)
		if err == nil && profile != nil {
			profiles = append(profiles, *profile)
		}
	}

	// Get room memory
	memory, _ := s.analyticsCache.GetRoomMemory(ctx, roomCode)
	if memory == nil {
		memory = &model.RoomMemory{RoomCode: roomCode}
	}

	// Calculate stats
	totalSkips := 0
	totalAnswers := 0
	for _, p := range profiles {
		totalSkips += p.SkipCount
		totalAnswers += p.AnswerCount
	}
	skipRate := 0.0
	if totalAnswers > 0 {
		skipRate = float64(totalSkips) / float64(totalAnswers)
	}

	snapshot := &model.RoomSnapshot{
		RoomCode:         roomCode,
		SurveyID:         room.SurveyID,
		EndedAt:          time.Now(),
		Leaderboard:      leaderboard,
		QuestionProfiles: profiles,
		Memory:           *memory,
		TotalPlayers:     len(leaderboard),
		OverallSkipRate:  skipRate,
	}

	// Save snapshot
	if err := s.reportRepo.SaveSnapshot(ctx, snapshot); err != nil {
		return nil, err
	}

	return snapshot, nil
}

// GetSnapshot retrieves the instant dashboard snapshot
func (s *ReportService) GetSnapshot(ctx context.Context, roomCode string) (*model.RoomSnapshot, error) {
	snapshot, err := s.reportRepo.GetSnapshot(ctx, roomCode)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, nil
	}

	// Patch with latest Survey details (SM Survey ID link)
	if snapshot.SurveyID != "" {
		survey, err := s.surveyRepo.GetByID(ctx, snapshot.SurveyID)
		if err == nil && survey != nil {
			snapshot.SMSurveyID = survey.SMSurveyID
			snapshot.SMWebLink = survey.SMWebLink
		}
	}

	return snapshot, nil
}

// TriggerAIReport starts async AI report generation
func (s *ReportService) TriggerAIReport(ctx context.Context, roomCode string) error {
	// Create pending report
	report := &model.AIReport{
		RoomCode:  roomCode,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	return s.reportRepo.SaveAIReport(ctx, report)
}

// GenerateAIReport generates the full AI report (call async)
func (s *ReportService) GenerateAIReport(ctx context.Context, roomCode string) (*model.AIReport, error) {
	// Get snapshot
	snapshot, err := s.reportRepo.GetSnapshot(ctx, roomCode)
	if err != nil || snapshot == nil {
		return nil, err
	}

	// Sample evidence from answers (simplified - just get summaries from signals)
	evidenceSamples := make(map[string][]string)
	answers, err := s.answerRepo.GetByRoomCode(ctx, roomCode)
	if err == nil {
		for _, ans := range answers {
			if ans.Signals != nil && ans.Signals.Summary != "" {
				if evidenceSamples[ans.QuestionKey] == nil {
					evidenceSamples[ans.QuestionKey] = []string{}
				}
				if len(evidenceSamples[ans.QuestionKey]) < 5 {
					evidenceSamples[ans.QuestionKey] = append(evidenceSamples[ans.QuestionKey], ans.Signals.Summary)
				}
			}
		}
	}

	// Generate AI report
	report, err := s.evaluator.GenerateAIReport(ctx, snapshot, evidenceSamples)
	if err != nil {
		return nil, err
	}

	// Save report
	if err := s.reportRepo.SaveAIReport(ctx, report); err != nil {
		return nil, err
	}

	return report, nil
}

// GetAIReport retrieves the AI report
func (s *ReportService) GetAIReport(ctx context.Context, roomCode string) (*model.AIReport, error) {
	return s.reportRepo.GetAIReport(ctx, roomCode)
}
