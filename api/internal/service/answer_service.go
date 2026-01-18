package service

import (
	"2026champs/internal/cache"
	"2026champs/internal/model"
	"2026champs/internal/repository"
	"context"
	"fmt"
	"time"
)

// AnswerService handles answer submission, drafts, and skips
type AnswerService struct {
	answerRepo   repository.AnswerRepo
	playerCache  cache.PlayerCache
	poolCache    cache.PoolCache
	playerSvc    *PlayerService
	evaluator    *EvaluatorService
	broadcaster  Broadcaster
	analyticsSvc *AnalyticsService
}

// NewAnswerService creates a new answer service
func NewAnswerService(
	answerRepo repository.AnswerRepo,
	playerCache cache.PlayerCache,
	poolCache cache.PoolCache,
	playerSvc *PlayerService,
	evaluator *EvaluatorService,
) *AnswerService {
	return &AnswerService{
		answerRepo:  answerRepo,
		playerCache: playerCache,
		poolCache:   poolCache,
		playerSvc:   playerSvc,
		evaluator:   evaluator,
	}
}

// SetBroadcaster sets the broadcaster for WebSocket events
func (s *AnswerService) SetBroadcaster(b Broadcaster) {
	s.broadcaster = b
}

// SetAnalyticsService sets the analytics service for L2/L3/L4 updates
func (s *AnswerService) SetAnalyticsService(svc *AnalyticsService) {
	s.analyticsSvc = svc
}

// SaveDraft saves a draft answer
func (s *AnswerService) SaveDraft(ctx context.Context, roomCode, playerID, questionKey, draft string) error {
	state, err := s.playerCache.GetAttempt(ctx, roomCode, playerID, questionKey)
	if err != nil {
		return err
	}
	if state == nil {
		state = &model.AttemptState{
			Status: model.AnswerStatusDraft,
			Tries:  0,
		}
	}
	state.DraftAnswer = draft
	state.UpdatedAt = time.Now()
	return s.playerCache.SetAttempt(ctx, roomCode, playerID, questionKey, state)
}

// SubmitAnswer handles answer submission with idempotency and evaluation
func (s *AnswerService) SubmitAnswer(ctx context.Context, roomCode, playerID string, req *model.SubmitAnswerRequest) (*model.SubmitAnswerResponse, error) {
	// Idempotency check
	exists, err := s.answerRepo.CheckIdempotency(ctx, roomCode, playerID, req.QuestionKey, req.ClientAttemptID)
	if err != nil {
		return nil, fmt.Errorf("idempotency check failed: %w", err)
	}
	if exists {
		// Already processed, return success
		return &model.SubmitAnswerResponse{
			Status: model.AnswerStatusEvaluated,
		}, nil
	}

	// Get current question
	question, err := s.playerCache.GetQuestionMap(ctx, roomCode, playerID, req.QuestionKey)
	if err != nil {
		return nil, err
	}
	if question == nil {
		return nil, fmt.Errorf("question not found")
	}

	// Get/create attempt state
	state, err := s.playerCache.GetAttempt(ctx, roomCode, playerID, req.QuestionKey)
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = &model.AttemptState{
			Status: model.AnswerStatusDraft,
			Tries:  0,
		}
	}
	state.Tries++
	state.SubmittedAnswer = req.TextAnswer
	state.UpdatedAt = time.Now()

	// Create answer record
	answer := &model.Answer{
		RoomCode:        roomCode,
		PlayerID:        playerID,
		QuestionKey:     req.QuestionKey,
		ClientAttemptID: req.ClientAttemptID,
		TextAnswer:      req.TextAnswer,
		DegreeValue:     req.DegreeValue,
		Status:          model.AnswerStatusSubmitted,
		Tries:           state.Tries,
	}

	var response model.SubmitAnswerResponse

	// Evaluate based on question type
	switch question.Type {
	case model.QuestionTypeEssay:
		// AI evaluation
		evalResult, err := s.evaluator.EvaluateAnswer(ctx, question, answer)
		if err != nil {
			return nil, fmt.Errorf("evaluation failed: %w", err)
		}

		answer.Status = model.AnswerStatusEvaluated
		answer.Resolution = model.AnswerResolution(evalResult.Resolution)
		answer.Signals = &evalResult.Signals
		answer.EvalSummary = evalResult.Signals.Summary

		// Calculate points
		points := int(evalResult.QualityScore * float64(question.PointsMax))
		answer.PointsEarned = points

		state.Status = model.AnswerStatusEvaluated
		state.Resolution = answer.Resolution
		state.EvalSummary = answer.EvalSummary

		response.Status = answer.Status
		response.Resolution = answer.Resolution
		response.PointsEarned = points
		response.EvalSummary = answer.EvalSummary

		// Handle UNSAT - may trigger follow-up
		if answer.Resolution == model.ResolutionUnsat {
			followUp, err := s.getOrGenerateFollowUp(ctx, roomCode, playerID, question, evalResult, answer.TextAnswer)
			if err == nil && followUp != nil {
				if err := s.playerSvc.InsertFollowUp(ctx, roomCode, playerID, followUp); err == nil {
					response.FollowUp = followUp
				}
			}
		}

	case model.QuestionTypeDegree:
		// Degree questions give fixed points, no gating
		points := question.PointsMax / 2 // Participation points
		answer.PointsEarned = points
		answer.Status = model.AnswerStatusEvaluated
		answer.Resolution = model.ResolutionSat

		state.Status = model.AnswerStatusEvaluated
		state.Resolution = model.ResolutionSat

		response.Status = answer.Status
		response.Resolution = answer.Resolution
		response.PointsEarned = points
	}

	// Save attempt state
	if err := s.playerCache.SetAttempt(ctx, roomCode, playerID, req.QuestionKey, state); err != nil {
		return nil, err
	}

	// Persist answer to MongoDB
	now := time.Now()
	answer.EvaluatedAt = &now
	if _, err := s.answerRepo.Create(ctx, answer); err != nil {
		return nil, err
	}

	// Update player score (this also broadcasts leaderboard update)
	if answer.PointsEarned > 0 {
		if _, err := s.playerSvc.UpdateScore(ctx, roomCode, playerID, answer.PointsEarned); err != nil {
			return nil, err
		}
	}

	// Broadcast player progress to host
	if s.broadcaster != nil {
		s.broadcaster.BroadcastToHost(roomCode, "player_progress_update", map[string]interface{}{
			"playerId":    playerID,
			"questionKey": req.QuestionKey,
			"status":      string(answer.Status),
			"resolution":  string(answer.Resolution),
		})
	}

	// If satisfactory or degree, advance to next question
	if answer.Resolution == model.ResolutionSat {
		nextQ, err := s.playerSvc.AdvanceToNextQuestion(ctx, roomCode, playerID)
		if err != nil {
			return nil, err
		}
		response.NextQuestion = nextQ
	}

	return &response, nil
}

// Skip marks a question as skipped and closes its follow-up chain
func (s *AnswerService) Skip(ctx context.Context, roomCode, playerID, questionKey string) (*model.Question, error) {
	// Get question to find parent
	question, err := s.playerCache.GetQuestionMap(ctx, roomCode, playerID, questionKey)
	if err != nil {
		return nil, err
	}

	// Determine which parent to close
	parentToClose := questionKey
	if question != nil && question.ParentKey != "" {
		parentToClose = question.ParentKey
	}

	// Close parent chain
	if err := s.playerCache.AddClosedParent(ctx, roomCode, playerID, parentToClose); err != nil {
		return nil, err
	}

	// Update attempt state
	state := &model.AttemptState{
		Status:     model.AnswerStatusEvaluated,
		Resolution: model.ResolutionSkipped,
		UpdatedAt:  time.Now(),
	}
	if err := s.playerCache.SetAttempt(ctx, roomCode, playerID, questionKey, state); err != nil {
		return nil, err
	}

	// Create skipped answer record
	answer := &model.Answer{
		RoomCode:    roomCode,
		PlayerID:    playerID,
		QuestionKey: questionKey,
		Status:      model.AnswerStatusEvaluated,
		Resolution:  model.ResolutionSkipped,
	}
	if _, err := s.answerRepo.Create(ctx, answer); err != nil {
		return nil, err
	}

	// Advance to next question
	return s.playerSvc.AdvanceToNextQuestion(ctx, roomCode, playerID)
}

// getOrGenerateFollowUp retrieves from pool or generates on-demand
func (s *AnswerService) getOrGenerateFollowUp(ctx context.Context, roomCode, playerID string, question *model.Question, evalResult *model.EvaluationResult, answerText string) (*model.Question, error) {
	// Try pool first
	pool, err := s.poolCache.GetPool(ctx, roomCode, question.Key)
	if err != nil {
		return nil, err
	}

	if pool != nil {
		// Pick based on follow-up hint
		var candidates []model.Question
		switch evalResult.FollowUpHint {
		case "clarify":
			candidates = pool.Clarify
		case "deepen":
			candidates = pool.Deepen
		default:
			candidates = pool.Clarify // Default to clarify
		}

		if len(candidates) > 0 {
			fu := candidates[0]
			// Remove used follow-up from pool
			if evalResult.FollowUpHint == "clarify" && len(pool.Clarify) > 0 {
				pool.Clarify = pool.Clarify[1:]
			} else if len(pool.Deepen) > 0 {
				pool.Deepen = pool.Deepen[1:]
			}
			s.poolCache.SetPool(ctx, roomCode, question.Key, pool)
			return &fu, nil
		}
	}

	// Fetch analytics context for personalization
	var qProfile *model.QuestionProfile
	var roomMemory *model.RoomMemory

	if s.analyticsSvc != nil {
		// Best effort fetching - don't fail if analytics are missing
		qProfile, _ = s.analyticsSvc.GetQuestionProfile(ctx, roomCode, question.Key)
		roomMemory, _ = s.analyticsSvc.GetRoomMemory(ctx, roomCode)
	}

	// Fetch conversation history
	// We want all previous answers from this player in this room to give context
	// In a real implementation, we might filter to just the current question thread (parent + siblings)
	allAnswers, err := s.answerRepo.GetByRoomAndPlayer(ctx, roomCode, playerID)
	history := []model.Answer{}
	if err == nil {
		// Simple filter: include answers related to the same parent key or the question itself
		// Ideally we traverse the tree, but for now, let's just dump the last few answers as context
		for _, ans := range allAnswers {
			// Include if it's the base question or a related follow-up
			// Heuristic: starts with the same prefix? Or just all recent answers?
			// Let's pass all recent answers for rich context
			history = append(history, *ans)
		}
	}

	// Generate on-demand
	player, _ := s.playerSvc.GetPlayer(ctx, roomCode, playerID)
	return s.evaluator.GenerateFollowUp(ctx, question, player, evalResult, answerText, qProfile, roomMemory, history)
}
