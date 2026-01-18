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
// AnswerService handles answer submission, drafts, and drafts
type AnswerService struct {
	answerRepo   repository.AnswerRepo
	surveyRepo   repository.SurveyRepo
	roomCache    cache.RoomCache // Added RoomCache
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
	surveyRepo repository.SurveyRepo,
	roomCache cache.RoomCache, // Added arg
	playerCache cache.PlayerCache,
	poolCache cache.PoolCache,
	playerSvc *PlayerService,
	evaluator *EvaluatorService,
) *AnswerService {
	return &AnswerService{
		answerRepo:  answerRepo,
		surveyRepo:  surveyRepo,
		roomCache:   roomCache,
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

// checkRoomActive helper
func (s *AnswerService) checkRoomActive(ctx context.Context, roomCode string) error {
	meta, err := s.roomCache.GetMeta(ctx, roomCode)
	if err != nil {
		return fmt.Errorf("failed to get room meta: %w", err)
	}
	if meta == nil {
		return fmt.Errorf("room not found")
	}
	if meta.Status != model.RoomStatusActive {
		return fmt.Errorf("room is not active (status: %s)", meta.Status)
	}
	return nil
}

// SaveDraft saves a draft answer
func (s *AnswerService) SaveDraft(ctx context.Context, roomCode, playerID, questionKey, draft string) error {
	if err := s.checkRoomActive(ctx, roomCode); err != nil {
		return err
	}
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
	if err := s.checkRoomActive(ctx, roomCode); err != nil {
		return nil, err
	}
	// Idempotency check
	exists, err := s.answerRepo.CheckIdempotency(ctx, roomCode, playerID, req.QuestionKey, req.ClientAttemptID)
	if err != nil {
		return nil, fmt.Errorf("idempotency check failed: %w", err)
	}
	if exists {
		// Already processed, return pending if it was recent or evaluated if done
		// For simplicity, if it exists, we assume it's being processed or done.
		// Ideally we check state.
		return &model.SubmitAnswerResponse{
			Status: model.AnswerStatusSubmitted, // Or Evaluated if we checked
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
	state.Status = model.AnswerStatusSubmitted // Mark as submitted
	state.UpdatedAt = time.Now()

	// Update attempt state immediately to indicate "Submitted"
	if err := s.playerCache.SetAttempt(ctx, roomCode, playerID, req.QuestionKey, state); err != nil {
		return nil, err
	}

	// BROADCAST IMMEDIATE ACK/THINKING
	if s.broadcaster != nil {
		// 1. Tell Host that player has submitted
		s.broadcaster.BroadcastToHost(roomCode, "player_progress_update", map[string]interface{}{
			"playerId":    playerID,
			"questionKey": req.QuestionKey,
			"status":      string(model.AnswerStatusSubmitted),
			"optionIndex": req.OptionIndex,
		})

		// 2. Tell Player that AI is thinking (The immediate feedback requested)
		s.broadcaster.BroadcastToPlayer(roomCode, playerID, "ai_thinking", map[string]interface{}{
			"questionKey": req.QuestionKey,
		})
	}

	// ASYNC PROCESSING
	// Create a detached context for the goroutine
	// Note: In production, use a proper background context with timeout or worker pool
	go func(asyncCtx context.Context, rCode, pID string, request model.SubmitAnswerRequest, q *model.Question, st *model.AttemptState) {
		// Recover from panics in goroutine
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovered from panic in SubmitAnswer async: %v\n", r)
			}
		}()

		// Create answer record shell
		answer := &model.Answer{
			RoomCode:        rCode,
			PlayerID:        pID,
			QuestionKey:     request.QuestionKey,
			ClientAttemptID: request.ClientAttemptID,
			TextAnswer:      request.TextAnswer,
			DegreeValue:     request.DegreeValue,
			Tries:           st.Tries,
			Status:          model.AnswerStatusSubmitted,
			OptionIndex:     request.OptionIndex,
		}

		var response model.SubmitAnswerResponse

		// Evaluate based on question type
		switch q.Type {
		case model.QuestionTypeEssay:
			// AI evaluation (Slow)
			evalResult, err := s.evaluator.EvaluateAnswer(asyncCtx, q, answer)
			if err != nil {
				fmt.Printf("Evaluation failed: %v\n", err)
				// Broadcast error?
				if s.broadcaster != nil {
					s.broadcaster.BroadcastToPlayer(rCode, pID, "error", map[string]string{"message": "Evaluation failed"})
				}
				return
			}

			answer.Status = model.AnswerStatusEvaluated
			answer.Resolution = model.AnswerResolution(evalResult.Resolution)
			answer.Signals = &evalResult.Signals
			answer.EvalSummary = evalResult.Signals.Summary

			// Calculate points
			points := 0
			if model.AnswerResolution(evalResult.Resolution) == model.ResolutionSat {
				points = int(evalResult.QualityScore * float64(q.PointsMax))
			}

			answer.PointsEarned = points

			st.Status = model.AnswerStatusEvaluated
			st.Resolution = answer.Resolution
			st.EvalSummary = answer.EvalSummary

			response.Status = answer.Status
			response.Resolution = answer.Resolution
			response.PointsEarned = points
			response.EvalSummary = answer.EvalSummary

			if answer.Resolution == model.ResolutionSat {
				followUp, err := s.getOrGenerateFollowUp(asyncCtx, rCode, pID, q, evalResult, answer.TextAnswer)
				if err == nil && followUp != nil {
					if err := s.playerSvc.InsertFollowUp(asyncCtx, rCode, pID, followUp); err == nil {
						response.FollowUp = followUp
					}
				}
			}

		case model.QuestionTypeDegree, model.QuestionTypeMCQ:
			// Degree and MCQ questions give fixed points (half of max)
			points := q.PointsMax / 2
			answer.PointsEarned = points
			answer.Status = model.AnswerStatusEvaluated
			answer.Resolution = model.ResolutionSat

			st.Status = model.AnswerStatusEvaluated
			st.Resolution = model.ResolutionSat

			response.Status = answer.Status
			response.Resolution = answer.Resolution
			response.PointsEarned = points
		}

		// Save attempt state
		s.playerCache.SetAttempt(asyncCtx, rCode, pID, request.QuestionKey, st)

		// Persist answer
		now := time.Now()
		answer.EvaluatedAt = &now
		s.answerRepo.Create(asyncCtx, answer)

		// Update score & Host Broadcast
		if answer.PointsEarned > 0 {
			s.playerSvc.UpdateScore(asyncCtx, rCode, pID, answer.PointsEarned)
		}

		if s.broadcaster != nil {
			// Notify Host
			s.broadcaster.BroadcastToHost(rCode, "player_progress_update", map[string]interface{}{
				"playerId":    pID,
				"questionKey": request.QuestionKey,
				"status":      string(answer.Status),
				"resolution":  string(answer.Resolution),
				"optionIndex": answer.OptionIndex,
			})

			// Notify Player (The "ACK" that work is done)

			// If satisfactory, advance
			if answer.Resolution == model.ResolutionSat {
				nextQ, _ := s.playerSvc.AdvanceToNextQuestion(asyncCtx, rCode, pID)
				response.NextQuestion = nextQ
			}

			// Broadcast Result to Player
			s.broadcaster.BroadcastToPlayer(rCode, pID, "evaluation_result", response)

			// Update Analytics (L2/L3/L4)
			if s.analyticsSvc != nil {
				s.analyticsSvc.UpdatePlayerProfile(asyncCtx, rCode, pID, answer.Signals, answer.Resolution)
				s.analyticsSvc.UpdateQuestionProfile(asyncCtx, rCode, request.QuestionKey, answer.Signals, answer.Resolution, answer.DegreeValue, answer.OptionIndex)
				s.analyticsSvc.UpdateRoomMemory(asyncCtx, rCode, answer.Signals)
			}
		}

	}(context.Background(), roomCode, playerID, *req, question, state)

	// Return immediate ACK
	return &model.SubmitAnswerResponse{
		Status: model.AnswerStatusSubmitted,
	}, nil
}

// Skip marks a question as skipped and closes its follow-up chain
func (s *AnswerService) Skip(ctx context.Context, roomCode, playerID, questionKey string) (*model.Question, error) {
	if err := s.checkRoomActive(ctx, roomCode); err != nil {
		return nil, err
	}
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
	// Depth check - don't go too deep!
	// Key format: Q1.1.1
	dots := 0
	for _, char := range question.Key {
		if char == '.' {
			dots++
		}
	}
	if dots >= 2 { // Max depth: Q1.1.1 (2 dots) -> no more children
		fmt.Printf("[FollowUp] Depth limit reached for %s (dots=%d). Stopping.\n", question.Key, dots)
		return nil, nil
	}

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

	// Fetch Survey Intent
	surveyIntent := "Gather general feedback"

	roomMeta, err := s.roomCache.GetMeta(ctx, roomCode)
	if err == nil && roomMeta != nil {
		survey, err := s.surveyRepo.GetByID(ctx, roomMeta.SurveyID)
		if err == nil && survey != nil && survey.Intent != "" {
			surveyIntent = survey.Intent
		}
	}

	// Generate on-demand
	player, _ := s.playerSvc.GetPlayer(ctx, roomCode, playerID)
	return s.evaluator.GenerateFollowUp(ctx, question, player, evalResult, answerText, qProfile, roomMemory, history, surveyIntent)
}
