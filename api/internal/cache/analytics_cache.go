package cache

import (
	"2026champs/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// AnalyticsCache handles Redis operations for L2-L4 analytics
type AnalyticsCache interface {
	// L2: Player Profile
	GetPlayerProfile(ctx context.Context, roomCode, playerID string) (*model.PlayerProfile, error)
	SetPlayerProfile(ctx context.Context, profile *model.PlayerProfile) error

	// L3: Question Profile
	GetQuestionProfile(ctx context.Context, roomCode, questionKey string) (*model.QuestionProfile, error)
	SetQuestionProfile(ctx context.Context, profile *model.QuestionProfile) error
	IncrementQuestionStats(ctx context.Context, roomCode, questionKey string, sat, unsat, skip int) error

	// L4: Room Memory
	GetRoomMemory(ctx context.Context, roomCode string) (*model.RoomMemory, error)
	SetRoomMemory(ctx context.Context, memory *model.RoomMemory) error
}

type analyticsCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewAnalyticsCache creates a new analytics cache
func NewAnalyticsCache(client *redis.Client) AnalyticsCache {
	return &analyticsCache{
		client: client,
		ttl:    24 * time.Hour,
	}
}

// Key helpers
func (c *analyticsCache) playerProfileKey(roomCode, playerID string) string {
	return fmt.Sprintf("room:%s:p:%s:profile", roomCode, playerID)
}

func (c *analyticsCache) questionProfileKey(roomCode, questionKey string) string {
	return fmt.Sprintf("room:%s:q:%s:profile", roomCode, questionKey)
}

func (c *analyticsCache) roomMemoryKey(roomCode string) string {
	return fmt.Sprintf("room:%s:memory", roomCode)
}

// L2: Player Profile
func (c *analyticsCache) GetPlayerProfile(ctx context.Context, roomCode, playerID string) (*model.PlayerProfile, error) {
	data, err := c.client.Get(ctx, c.playerProfileKey(roomCode, playerID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var profile model.PlayerProfile
	if err := json.Unmarshal([]byte(data), &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func (c *analyticsCache) SetPlayerProfile(ctx context.Context, profile *model.PlayerProfile) error {
	profile.UpdatedAt = time.Now()
	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.playerProfileKey(profile.RoomCode, profile.PlayerID), data, c.ttl).Err()
}

// L3: Question Profile
func (c *analyticsCache) GetQuestionProfile(ctx context.Context, roomCode, questionKey string) (*model.QuestionProfile, error) {
	data, err := c.client.Get(ctx, c.questionProfileKey(roomCode, questionKey)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var profile model.QuestionProfile
	if err := json.Unmarshal([]byte(data), &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func (c *analyticsCache) SetQuestionProfile(ctx context.Context, profile *model.QuestionProfile) error {
	profile.UpdatedAt = time.Now()
	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.questionProfileKey(profile.RoomCode, profile.QuestionKey), data, c.ttl).Err()
}

func (c *analyticsCache) IncrementQuestionStats(ctx context.Context, roomCode, questionKey string, sat, unsat, skip int) error {
	profile, err := c.GetQuestionProfile(ctx, roomCode, questionKey)
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
	profile.SatCount += sat
	profile.UnsatCount += unsat
	profile.SkipCount += skip
	profile.AnswerCount += sat + unsat + skip
	return c.SetQuestionProfile(ctx, profile)
}

// L4: Room Memory
func (c *analyticsCache) GetRoomMemory(ctx context.Context, roomCode string) (*model.RoomMemory, error) {
	data, err := c.client.Get(ctx, c.roomMemoryKey(roomCode)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var memory model.RoomMemory
	if err := json.Unmarshal([]byte(data), &memory); err != nil {
		return nil, err
	}
	return &memory, nil
}

func (c *analyticsCache) SetRoomMemory(ctx context.Context, memory *model.RoomMemory) error {
	memory.UpdatedAt = time.Now()
	data, err := json.Marshal(memory)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.roomMemoryKey(memory.RoomCode), data, c.ttl).Err()
}
