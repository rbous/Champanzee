package service

import (
	"2026champs/internal/model"
	"2026champs/internal/repository"
	"context"
)

type InsightService struct {
	roomRepo   repository.RoomRepo
	reportRepo repository.ReportRepo
	evaluator  *EvaluatorService
}

func NewInsightService(roomRepo repository.RoomRepo, reportRepo repository.ReportRepo, evaluator *EvaluatorService) *InsightService {
	return &InsightService{
		roomRepo:   roomRepo,
		reportRepo: reportRepo,
		evaluator:  evaluator,
	}
}

func (s *InsightService) GenerateQuestionsFromInsights(ctx context.Context, hostID string, intent string) ([]model.BaseQuestion, error) {
	// 1. Get Room Codes for Host
	rooms, err := s.roomRepo.GetByHostID(ctx, hostID)
	if err != nil {
		return nil, err
	}

	// 2. Aggregate Probes
	var probes []string
	for _, room := range rooms {
		report, err := s.reportRepo.GetAIReport(ctx, room.Code)
		if err != nil || report == nil {
			continue
		}

		probes = append(probes, report.RecommendedQuestions...)
		for _, pq := range report.PerQuestionInsights {
			probes = append(probes, pq.BestFollowUps...)
		}
	}

	// If no probes found (fresh host), return mock/empty?
	// The evaluator will return mock if config is disabled, but if probes are empty it might produce poor results.
	// But let's pass it to evaluator anyway, it might hallucinate something useful or use the intent.
	if len(probes) == 0 {
		probes = append(probes, "General follow-up about satisfaction", "Specific details about feature usage")
	}

	// 3. Call Evaluator
	return s.evaluator.CondenseProbes(ctx, probes, intent)
}
