package service

import (
	"2026champs/internal/cache"
	"2026champs/internal/model"
	"2026champs/internal/repository"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"
)

// RoomService handles room lifecycle operations
type RoomService struct {
	roomRepo   repository.RoomRepo
	surveyRepo repository.SurveyRepo
	roomCache  cache.RoomCache
	authSvc    *AuthService
}

// NewRoomService creates a new room service
func NewRoomService(
	roomRepo repository.RoomRepo,
	surveyRepo repository.SurveyRepo,
	roomCache cache.RoomCache,
	authSvc *AuthService,
) *RoomService {
	return &RoomService{
		roomRepo:   roomRepo,
		surveyRepo: surveyRepo,
		roomCache:  roomCache,
		authSvc:    authSvc,
	}
}

// CreateRoom creates a new room from a survey
func (s *RoomService) CreateRoom(ctx context.Context, surveyID, hostID string, settings *model.RoomSettings) (*model.Room, error) {
	// Verify survey exists
	survey, err := s.surveyRepo.GetByID(ctx, surveyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get survey: %w", err)
	}
	if survey == nil {
		return nil, fmt.Errorf("survey not found")
	}

	// Generate unique room code
	code, err := s.generateRoomCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate room code: %w", err)
	}

	room := &model.Room{
		Code:     code,
		SurveyID: surveyID,
		HostID:   hostID,
		Status:   model.RoomStatusLobby,
		Settings: *settings,
	}

	// Persist to MongoDB
	if err := s.roomRepo.Create(ctx, room); err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	// Cache in Redis
	settingsJSON, _ := json.Marshal(settings)
	meta := &model.RoomMeta{
		SurveyID:     surveyID,
		HostID:       hostID,
		Status:       model.RoomStatusLobby,
		CreatedAt:    room.CreatedAt,
		SettingsJSON: string(settingsJSON),
	}
	if err := s.roomCache.SetMeta(ctx, code, meta); err != nil {
		return nil, fmt.Errorf("failed to cache room: %w", err)
	}

	return room, nil
}

// GetRoom retrieves a room by code
func (s *RoomService) GetRoom(ctx context.Context, code string) (*model.Room, error) {
	return s.roomRepo.GetByCode(ctx, code)
}

// GetRoomMeta retrieves room metadata from Redis
func (s *RoomService) GetRoomMeta(ctx context.Context, code string) (*model.RoomMeta, error) {
	return s.roomCache.GetMeta(ctx, code)
}

// StartRoom transitions room to ACTIVE status
func (s *RoomService) StartRoom(ctx context.Context, code, hostID string) error {
	room, err := s.roomRepo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if room == nil {
		return fmt.Errorf("room not found")
	}
	if room.HostID != hostID {
		return fmt.Errorf("unauthorized: not room host")
	}
	if room.Status != model.RoomStatusLobby {
		return fmt.Errorf("room is not in lobby status")
	}

	now := time.Now()
	room.Status = model.RoomStatusActive
	room.StartedAt = &now

	if err := s.roomRepo.Update(ctx, room); err != nil {
		return err
	}
	return s.roomCache.SetStatus(ctx, code, model.RoomStatusActive)
}

// EndRoom transitions room to ENDED status
func (s *RoomService) EndRoom(ctx context.Context, code, hostID string) error {
	room, err := s.roomRepo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if room == nil {
		return fmt.Errorf("room not found")
	}
	if room.HostID != hostID {
		return fmt.Errorf("unauthorized: not room host")
	}

	now := time.Now()
	room.Status = model.RoomStatusEnded
	room.EndedAt = &now

	if err := s.roomRepo.Update(ctx, room); err != nil {
		return err
	}
	return s.roomCache.SetStatus(ctx, code, model.RoomStatusEnded)
}

// generateRoomCode creates a 6-char alphanumeric code
func (s *RoomService) generateRoomCode(ctx context.Context) (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	const codeLen = 6

	for attempts := 0; attempts < 10; attempts++ {
		b := make([]byte, codeLen)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}

		code := make([]byte, codeLen)
		for i := range code {
			code[i] = chars[int(b[i])%len(chars)]
		}
		codeStr := string(code)

		// Check uniqueness
		exists, err := s.roomCache.Exists(ctx, codeStr)
		if err != nil {
			return "", err
		}
		if !exists {
			return codeStr, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique room code")
}
