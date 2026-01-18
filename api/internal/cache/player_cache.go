package cache

import (
	"2026champs/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// PlayerCache handles Redis operations for player state
type PlayerCache interface {
	// Player info
	SetPlayer(ctx context.Context, roomCode, playerID string, player *model.Player) error
	GetPlayer(ctx context.Context, roomCode, playerID string) (*model.Player, error)
	GetAllPlayers(ctx context.Context, roomCode string) (map[string]*model.Player, error)
	UpdateScore(ctx context.Context, roomCode, playerID string, score int) error

	// Queue operations
	SetQueue(ctx context.Context, roomCode, playerID string, questions []string) error
	GetQueue(ctx context.Context, roomCode, playerID string) ([]string, error)
	PopQueue(ctx context.Context, roomCode, playerID string) (string, error)
	InsertInQueue(ctx context.Context, roomCode, playerID string, afterKey string, newKeys ...string) error

	// Current question
	SetCurrent(ctx context.Context, roomCode, playerID, questionKey string) error
	GetCurrent(ctx context.Context, roomCode, playerID string) (string, error)

	// Question map (stores Question JSON for follow-ups/overrides)
	SetQuestionMap(ctx context.Context, roomCode, playerID, key string, q *model.Question) error
	GetQuestionMap(ctx context.Context, roomCode, playerID, key string) (*model.Question, error)

	// Closed parents (for skip chains)
	AddClosedParent(ctx context.Context, roomCode, playerID, parentKey string) error
	IsParentClosed(ctx context.Context, roomCode, playerID, parentKey string) (bool, error)

	// Attempt state
	SetAttempt(ctx context.Context, roomCode, playerID, questionKey string, state *model.AttemptState) error
	GetAttempt(ctx context.Context, roomCode, playerID, questionKey string) (*model.AttemptState, error)
}

type playerCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewPlayerCache creates a new player cache
func NewPlayerCache(client *redis.Client) PlayerCache {
	return &playerCache{
		client: client,
		ttl:    24 * time.Hour,
	}
}

// Key helpers
func (c *playerCache) playersKey(roomCode string) string {
	return fmt.Sprintf("room:%s:players", roomCode)
}

func (c *playerCache) queueKey(roomCode, playerID string) string {
	return fmt.Sprintf("room:%s:p:%s:q", roomCode, playerID)
}

func (c *playerCache) currentKey(roomCode, playerID string) string {
	return fmt.Sprintf("room:%s:p:%s:current", roomCode, playerID)
}

func (c *playerCache) qmapKey(roomCode, playerID string) string {
	return fmt.Sprintf("room:%s:p:%s:qmap", roomCode, playerID)
}

func (c *playerCache) closedKey(roomCode, playerID string) string {
	return fmt.Sprintf("room:%s:p:%s:closedParents", roomCode, playerID)
}

func (c *playerCache) attemptKey(roomCode, playerID, questionKey string) string {
	return fmt.Sprintf("room:%s:p:%s:attempt:%s", roomCode, playerID, questionKey)
}

// Player operations
func (c *playerCache) SetPlayer(ctx context.Context, roomCode, playerID string, player *model.Player) error {
	data, err := json.Marshal(player)
	if err != nil {
		return err
	}
	return c.client.HSet(ctx, c.playersKey(roomCode), playerID, data).Err()
}

func (c *playerCache) GetPlayer(ctx context.Context, roomCode, playerID string) (*model.Player, error) {
	data, err := c.client.HGet(ctx, c.playersKey(roomCode), playerID).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var player model.Player
	if err := json.Unmarshal([]byte(data), &player); err != nil {
		return nil, err
	}
	return &player, nil
}

func (c *playerCache) GetAllPlayers(ctx context.Context, roomCode string) (map[string]*model.Player, error) {
	data, err := c.client.HGetAll(ctx, c.playersKey(roomCode)).Result()
	if err != nil {
		return nil, err
	}
	players := make(map[string]*model.Player)
	for id, jsonStr := range data {
		var p model.Player
		if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
			continue
		}
		players[id] = &p
	}
	return players, nil
}

func (c *playerCache) UpdateScore(ctx context.Context, roomCode, playerID string, score int) error {
	player, err := c.GetPlayer(ctx, roomCode, playerID)
	if err != nil || player == nil {
		return err
	}
	player.Score = score
	return c.SetPlayer(ctx, roomCode, playerID, player)
}

// Queue operations
func (c *playerCache) SetQueue(ctx context.Context, roomCode, playerID string, questions []string) error {
	key := c.queueKey(roomCode, playerID)
	c.client.Del(ctx, key)
	if len(questions) == 0 {
		return nil
	}
	args := make([]interface{}, len(questions))
	for i, q := range questions {
		args[i] = q
	}
	return c.client.RPush(ctx, key, args...).Err()
}

func (c *playerCache) GetQueue(ctx context.Context, roomCode, playerID string) ([]string, error) {
	return c.client.LRange(ctx, c.queueKey(roomCode, playerID), 0, -1).Result()
}

func (c *playerCache) PopQueue(ctx context.Context, roomCode, playerID string) (string, error) {
	val, err := c.client.LPop(ctx, c.queueKey(roomCode, playerID)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (c *playerCache) InsertInQueue(ctx context.Context, roomCode, playerID string, afterKey string, newKeys ...string) error {
	// Get current queue
	queue, err := c.GetQueue(ctx, roomCode, playerID)
	if err != nil {
		return err
	}

	// Filter out keys that already exist in the queue
	exists := make(map[string]bool)
	for _, k := range queue {
		exists[k] = true
	}

	uniqueNewKeys := make([]string, 0)
	for _, nk := range newKeys {
		if !exists[nk] {
			uniqueNewKeys = append(uniqueNewKeys, nk)
		}
	}

	if len(uniqueNewKeys) == 0 {
		return nil // Nothing to insert
	}

	// Find position and insert
	newQueue := make([]string, 0, len(queue)+len(uniqueNewKeys))
	inserted := false
	for _, k := range queue {
		newQueue = append(newQueue, k)
		if k == afterKey && !inserted {
			newQueue = append(newQueue, uniqueNewKeys...)
			inserted = true
		}
	}
	if !inserted {
		// Append at end if afterKey not found
		newQueue = append(newQueue, uniqueNewKeys...)
	}

	return c.SetQueue(ctx, roomCode, playerID, newQueue)
}

// Current question
func (c *playerCache) SetCurrent(ctx context.Context, roomCode, playerID, questionKey string) error {
	return c.client.Set(ctx, c.currentKey(roomCode, playerID), questionKey, c.ttl).Err()
}

func (c *playerCache) GetCurrent(ctx context.Context, roomCode, playerID string) (string, error) {
	val, err := c.client.Get(ctx, c.currentKey(roomCode, playerID)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// Question map
func (c *playerCache) SetQuestionMap(ctx context.Context, roomCode, playerID, key string, q *model.Question) error {
	data, err := json.Marshal(q)
	if err != nil {
		return err
	}
	return c.client.HSet(ctx, c.qmapKey(roomCode, playerID), key, data).Err()
}

func (c *playerCache) GetQuestionMap(ctx context.Context, roomCode, playerID, key string) (*model.Question, error) {
	data, err := c.client.HGet(ctx, c.qmapKey(roomCode, playerID), key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var q model.Question
	if err := json.Unmarshal([]byte(data), &q); err != nil {
		return nil, err
	}
	return &q, nil
}

// Closed parents
func (c *playerCache) AddClosedParent(ctx context.Context, roomCode, playerID, parentKey string) error {
	return c.client.SAdd(ctx, c.closedKey(roomCode, playerID), parentKey).Err()
}

func (c *playerCache) IsParentClosed(ctx context.Context, roomCode, playerID, parentKey string) (bool, error) {
	return c.client.SIsMember(ctx, c.closedKey(roomCode, playerID), parentKey).Result()
}

// Attempt state
func (c *playerCache) SetAttempt(ctx context.Context, roomCode, playerID, questionKey string, state *model.AttemptState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.attemptKey(roomCode, playerID, questionKey), data, c.ttl).Err()
}

func (c *playerCache) GetAttempt(ctx context.Context, roomCode, playerID, questionKey string) (*model.AttemptState, error) {
	data, err := c.client.Get(ctx, c.attemptKey(roomCode, playerID, questionKey)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var state model.AttemptState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, err
	}
	return &state, nil
}
