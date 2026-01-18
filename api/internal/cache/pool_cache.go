package cache

import (
	"2026champs/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// PoolCache handles Redis operations for AI-generated follow-up pools
type PoolCache interface {
	SetPool(ctx context.Context, roomCode, questionKey string, pool *model.FollowUpPool) error
	GetPool(ctx context.Context, roomCode, questionKey string) (*model.FollowUpPool, error)
	DeletePool(ctx context.Context, roomCode, questionKey string) error
}

type poolCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewPoolCache creates a new pool cache
func NewPoolCache(client *redis.Client) PoolCache {
	return &poolCache{
		client: client,
		ttl:    24 * time.Hour,
	}
}

func (c *poolCache) key(roomCode, questionKey string) string {
	return fmt.Sprintf("room:%s:q:%s:pool", roomCode, questionKey)
}

func (c *poolCache) SetPool(ctx context.Context, roomCode, questionKey string, pool *model.FollowUpPool) error {
	data, err := json.Marshal(pool)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(roomCode, questionKey), data, c.ttl).Err()
}

func (c *poolCache) GetPool(ctx context.Context, roomCode, questionKey string) (*model.FollowUpPool, error) {
	data, err := c.client.Get(ctx, c.key(roomCode, questionKey)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var pool model.FollowUpPool
	if err := json.Unmarshal([]byte(data), &pool); err != nil {
		return nil, err
	}
	return &pool, nil
}

func (c *poolCache) DeletePool(ctx context.Context, roomCode, questionKey string) error {
	return c.client.Del(ctx, c.key(roomCode, questionKey)).Err()
}
