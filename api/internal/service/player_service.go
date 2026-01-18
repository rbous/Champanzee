package service

import (
	"2026champs/internal/cache"
	"2026champs/internal/model"
	"2026champs/internal/repository"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PlayerService handles player join and queue operations
type PlayerService struct {
	surveyRepo  repository.SurveyRepo
	roomCache   cache.RoomCache
	playerCache cache.PlayerCache
	leaderboard cache.LeaderboardCache
	authSvc     *AuthService
	broadcaster Broadcaster
}

// NewPlayerService creates a new player service
func NewPlayerService(
	surveyRepo repository.SurveyRepo,
	roomCache cache.RoomCache,
	playerCache cache.PlayerCache,
	leaderboard cache.LeaderboardCache,
	authSvc *AuthService,
) *PlayerService {
	return &PlayerService{
		surveyRepo:  surveyRepo,
		roomCache:   roomCache,
		playerCache: playerCache,
		leaderboard: leaderboard,
		authSvc:     authSvc,
	}
}

// SetBroadcaster sets the broadcaster for WebSocket events
func (s *PlayerService) SetBroadcaster(b Broadcaster) {
	s.broadcaster = b
}

// JoinRoom handles player joining a room
func (s *PlayerService) JoinRoom(ctx context.Context, roomCode, nickname string) (*model.PlayerJoinResponse, error) {
	// Get room meta
	meta, err := s.roomCache.GetMeta(ctx, roomCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}
	if meta == nil {
		return nil, fmt.Errorf("room not found")
	}
	if meta.Status == model.RoomStatusEnded {
		return nil, fmt.Errorf("room has ended")
	}

	// Generate player ID and token
	playerID := "p_" + uuid.New().String()[:8]
	token, err := s.authSvc.GeneratePlayerToken(roomCode, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Create player
	now := time.Now()
	player := &model.Player{
		ID:            playerID,
		RoomCode:      roomCode,
		Nickname:      nickname,
		Score:         0,
		CurrentKey:    "",
		FollowUpsUsed: 0,
		LastActiveAt:  now,
		JoinedAt:      now,
	}

	// Store in Redis
	if err := s.playerCache.SetPlayer(ctx, roomCode, playerID, player); err != nil {
		return nil, fmt.Errorf("failed to save player: %w", err)
	}

	// Initialize leaderboard entry
	if err := s.leaderboard.UpdateScore(ctx, roomCode, playerID, 0); err != nil {
		return nil, fmt.Errorf("failed to init leaderboard: %w", err)
	}

	// Get survey to initialize queue
	survey, err := s.surveyRepo.GetByID(ctx, meta.SurveyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get survey: %w", err)
	}
	if survey == nil {
		return nil, fmt.Errorf("survey not found")
	}

	// Initialize player queue with base questions
	var questionKeys []string
	for _, q := range survey.Questions {
		questionKeys = append(questionKeys, q.Key)

		// Store question in player's qmap
		question := &model.Question{
			Key:       q.Key,
			Type:      q.Type,
			Prompt:    q.Prompt,
			Rubric:    q.Rubric,
			PointsMax: q.PointsMax,
			Threshold: q.Threshold,
			ScaleMin:  q.ScaleMin,
			ScaleMax:  q.ScaleMax,
		}
		if err := s.playerCache.SetQuestionMap(ctx, roomCode, playerID, q.Key, question); err != nil {
			return nil, fmt.Errorf("failed to set question map: %w", err)
		}
	}

	if err := s.playerCache.SetQueue(ctx, roomCode, playerID, questionKeys); err != nil {
		return nil, fmt.Errorf("failed to set queue: %w", err)
	}

	// Get first question (only if room is active)
	var firstQuestion *model.Question
	if meta.Status == model.RoomStatusActive && len(questionKeys) > 0 {
		firstKey := questionKeys[0]
		player.CurrentKey = firstKey
		if err := s.playerCache.SetPlayer(ctx, roomCode, playerID, player); err != nil {
			return nil, err
		}
		if err := s.playerCache.SetCurrent(ctx, roomCode, playerID, firstKey); err != nil {
			return nil, err
		}
		firstQuestion, _ = s.playerCache.GetQuestionMap(ctx, roomCode, playerID, firstKey)
	} else if len(questionKeys) > 0 {
		// Initialize current key but don't return question yet if in lobby
		firstKey := questionKeys[0]
		player.CurrentKey = firstKey
		if err := s.playerCache.SetPlayer(ctx, roomCode, playerID, player); err != nil {
			return nil, err
		}
		if err := s.playerCache.SetCurrent(ctx, roomCode, playerID, firstKey); err != nil {
			return nil, err
		}
	}

	return &model.PlayerJoinResponse{
		PlayerID:      playerID,
		Token:         token,
		RoomMeta:      meta,
		FirstQuestion: firstQuestion,
	}, nil
}

// GetCurrentQuestion retrieves the player's current question
func (s *PlayerService) GetCurrentQuestion(ctx context.Context, roomCode, playerID string) (*model.Question, error) {
	// Check room status
	meta, err := s.roomCache.GetMeta(ctx, roomCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get room meta: %w", err)
	}
	if meta == nil {
		return nil, fmt.Errorf("room not found")
	}
	if meta.Status == model.RoomStatusLobby {
		return nil, nil // Waiting for host
	}

	currentKey, err := s.playerCache.GetCurrent(ctx, roomCode, playerID)
	if err != nil {
		return nil, err
	}
	if currentKey == "" {
		return nil, nil // No more questions
	}
	return s.playerCache.GetQuestionMap(ctx, roomCode, playerID, currentKey)
}

// AdvanceToNextQuestion moves to the next question in queue
func (s *PlayerService) AdvanceToNextQuestion(ctx context.Context, roomCode, playerID string) (*model.Question, error) {
	// Pop current from queue
	_, err := s.playerCache.PopQueue(ctx, roomCode, playerID)
	if err != nil {
		return nil, err
	}

	// Get next from queue
	queue, err := s.playerCache.GetQueue(ctx, roomCode, playerID)
	if err != nil {
		return nil, err
	}

	if len(queue) == 0 {
		// No more questions
		if err := s.playerCache.SetCurrent(ctx, roomCode, playerID, ""); err != nil {
			return nil, err
		}
		return nil, nil
	}

	nextKey := queue[0]

	// Check if parent is closed (for follow-ups)
	q, err := s.playerCache.GetQuestionMap(ctx, roomCode, playerID, nextKey)
	if err != nil {
		return nil, err
	}
	if q != nil && q.ParentKey != "" {
		closed, err := s.playerCache.IsParentClosed(ctx, roomCode, playerID, q.ParentKey)
		if err != nil {
			return nil, err
		}
		if closed {
			// Skip this follow-up and try next
			return s.AdvanceToNextQuestion(ctx, roomCode, playerID)
		}
	}

	// Update current
	if err := s.playerCache.SetCurrent(ctx, roomCode, playerID, nextKey); err != nil {
		return nil, err
	}

	// Update player current key
	player, err := s.playerCache.GetPlayer(ctx, roomCode, playerID)
	if err != nil {
		return nil, err
	}
	if player != nil {
		player.CurrentKey = nextKey
		player.LastActiveAt = time.Now()
		if err := s.playerCache.SetPlayer(ctx, roomCode, playerID, player); err != nil {
			return nil, err
		}
	}

	return q, nil
}

// InsertFollowUp inserts a follow-up question after the current question
func (s *PlayerService) InsertFollowUp(ctx context.Context, roomCode, playerID string, followUp *model.Question) error {
	currentKey, err := s.playerCache.GetCurrent(ctx, roomCode, playerID)
	if err != nil {
		return err
	}

	// Store in qmap
	if err := s.playerCache.SetQuestionMap(ctx, roomCode, playerID, followUp.Key, followUp); err != nil {
		return err
	}

	// Insert after current
	return s.playerCache.InsertInQueue(ctx, roomCode, playerID, currentKey, followUp.Key)
}

// GetPlayer retrieves a player by ID
func (s *PlayerService) GetPlayer(ctx context.Context, roomCode, playerID string) (*model.Player, error) {
	return s.playerCache.GetPlayer(ctx, roomCode, playerID)
}

// UpdateScore updates a player's score and broadcasts leaderboard update
func (s *PlayerService) UpdateScore(ctx context.Context, roomCode, playerID string, pointsToAdd int) (int, error) {
	player, err := s.playerCache.GetPlayer(ctx, roomCode, playerID)
	if err != nil {
		return 0, err
	}
	if player == nil {
		return 0, fmt.Errorf("player not found")
	}

	newScore := player.Score + pointsToAdd
	player.Score = newScore

	if err := s.playerCache.SetPlayer(ctx, roomCode, playerID, player); err != nil {
		return 0, err
	}
	if err := s.leaderboard.UpdateScore(ctx, roomCode, playerID, newScore); err != nil {
		return 0, err
	}

	// Broadcast leaderboard update to host
	if s.broadcaster != nil {
		entries, _ := s.leaderboard.GetTop(ctx, roomCode, 20)
		s.broadcaster.BroadcastToHost(roomCode, "leaderboard_update", map[string]interface{}{
			"leaderboard": entries,
		})
	}

	return newScore, nil
}
