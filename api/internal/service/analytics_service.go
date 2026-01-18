package service

import (
	"2026champs/internal/cache"
	"2026champs/internal/model"
	"context"
)

// AnalyticsService manages L2-L4 analytics updates
type AnalyticsService struct {
	analyticsCache cache.AnalyticsCache
	evaluator      *EvaluatorService
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(analyticsCache cache.AnalyticsCache, evaluator *EvaluatorService) *AnalyticsService {
	return &AnalyticsService{
		analyticsCache: analyticsCache,
		evaluator:      evaluator,
	}
}

// UpdatePlayerProfile updates L2 analytics after an answer
func (s *AnalyticsService) UpdatePlayerProfile(ctx context.Context, roomCode, playerID string, signals *model.Signals, resolution model.AnswerResolution) error {
	profile, err := s.analyticsCache.GetPlayerProfile(ctx, roomCode, playerID)
	if err != nil {
		return err
	}
	if profile == nil {
		profile = &model.PlayerProfile{
			PlayerID:      playerID,
			RoomCode:      roomCode,
			TopicAffinity: make(map[string]int),
		}
	}

	profile.TotalAnswers++

	// Update effort trend (rolling avg of specificity + clarity)
	if signals != nil {
		effortScore := (signals.Specificity + signals.Clarity) / 2.0
		if profile.TotalAnswers == 1 {
			profile.EffortTrend = effortScore
		} else {
			// Exponential moving average
			profile.EffortTrend = 0.7*profile.EffortTrend + 0.3*effortScore
		}

		// Update topic affinity
		for _, theme := range signals.Themes {
			profile.TopicAffinity[theme]++
		}

		// Update previous themes (keep last 10)
		profile.PreviousThemes = append(signals.Themes, profile.PreviousThemes...)
		if len(profile.PreviousThemes) > 10 {
			profile.PreviousThemes = profile.PreviousThemes[:10]
		}
	}

	// Update friction tracking
	switch resolution {
	case model.ResolutionSkipped:
		profile.SkipCount++
	case model.ResolutionUnsat:
		profile.UnsatCount++
	}
	profile.FollowUpFriction = float64(profile.SkipCount+profile.UnsatCount) / float64(profile.TotalAnswers)

	// Derive style (simple heuristic)
	if profile.EffortTrend < 0.3 {
		profile.Style = "brief"
	} else if profile.EffortTrend > 0.7 {
		profile.Style = "detailed"
	} else {
		profile.Style = "balanced"
	}

	return s.analyticsCache.SetPlayerProfile(ctx, profile)
}

// UpdateQuestionProfile updates L3 analytics after an answer
func (s *AnalyticsService) UpdateQuestionProfile(ctx context.Context, roomCode, questionKey string, signals *model.Signals, resolution model.AnswerResolution, degreeValue int) error {
	profile, err := s.analyticsCache.GetQuestionProfile(ctx, roomCode, questionKey)
	if err != nil {
		return err
	}
	if profile == nil {
		profile = &model.QuestionProfile{
			RoomCode:      roomCode,
			QuestionKey:   questionKey,
			ThemeCounts:   make(map[string]int),
			MissingCounts: make(map[string]int),
			RatingHist:    make(map[int]int),
		}
	}

	profile.AnswerCount++

	// Update resolution stats
	switch resolution {
	case model.ResolutionSat:
		profile.SatCount++
	case model.ResolutionUnsat:
		profile.UnsatCount++
	case model.ResolutionSkipped:
		profile.SkipCount++
	}

	// Update ratings (for DEGREE type)
	if degreeValue > 0 {
		profile.RatingHist[degreeValue]++
		profile.RatingSum += degreeValue
		profile.RatingCount++
	}

	// Update theme and missing counts from signals
	if signals != nil {
		for _, theme := range signals.Themes {
			profile.ThemeCounts[theme]++
		}
		for _, missing := range signals.Missing {
			profile.MissingCounts[missing]++
		}
	}

	return s.analyticsCache.SetQuestionProfile(ctx, profile)
}

// UpdateRoomMemory updates L4 analytics
func (s *AnalyticsService) UpdateRoomMemory(ctx context.Context, roomCode string, signals *model.Signals) error {
	memory, err := s.analyticsCache.GetRoomMemory(ctx, roomCode)
	if err != nil {
		return err
	}
	if memory == nil {
		memory = &model.RoomMemory{
			RoomCode: roomCode,
		}
	}

	memory.TotalAnswers++

	// Update global themes (simplified - would need proper aggregation)
	if signals != nil {
		for _, theme := range signals.Themes {
			found := false
			for i := range memory.GlobalThemesTop {
				if memory.GlobalThemesTop[i].Theme == theme {
					memory.GlobalThemesTop[i].Count++
					found = true
					break
				}
			}
			if !found {
				memory.GlobalThemesTop = append(memory.GlobalThemesTop, model.ThemeCount{Theme: theme, Count: 1})
			}
		}

		// Keep only top 10, sorted
		if len(memory.GlobalThemesTop) > 10 {
			// Simple bubble sort for small list
			for i := 0; i < len(memory.GlobalThemesTop)-1; i++ {
				for j := i + 1; j < len(memory.GlobalThemesTop); j++ {
					if memory.GlobalThemesTop[j].Count > memory.GlobalThemesTop[i].Count {
						memory.GlobalThemesTop[i], memory.GlobalThemesTop[j] = memory.GlobalThemesTop[j], memory.GlobalThemesTop[i]
					}
				}
			}
			memory.GlobalThemesTop = memory.GlobalThemesTop[:10]
		}
	}

	return s.analyticsCache.SetRoomMemory(ctx, memory)
}

// GetQuestionProfile returns L3 profile for a question
func (s *AnalyticsService) GetQuestionProfile(ctx context.Context, roomCode, questionKey string) (*model.QuestionProfile, error) {
	return s.analyticsCache.GetQuestionProfile(ctx, roomCode, questionKey)
}

// GetRoomMemory returns L4 room memory
func (s *AnalyticsService) GetRoomMemory(ctx context.Context, roomCode string) (*model.RoomMemory, error) {
	return s.analyticsCache.GetRoomMemory(ctx, roomCode)
}

// RefreshL3 refreshes misunderstandings for a question (call periodically)
func (s *AnalyticsService) RefreshL3(ctx context.Context, roomCode, questionKey string, recentSummaries []string) error {
	profile, err := s.analyticsCache.GetQuestionProfile(ctx, roomCode, questionKey)
	if err != nil || profile == nil {
		return err
	}

	// Only refresh if we have enough answers
	if profile.AnswerCount < 5 {
		return nil
	}

	updated, err := s.evaluator.RefreshQuestionProfile(ctx, profile, recentSummaries)
	if err != nil {
		return err
	}

	return s.analyticsCache.SetQuestionProfile(ctx, updated)
}
