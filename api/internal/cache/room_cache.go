package cache

import (
	"2026champs/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RoomCache handles Redis operations for room state
type RoomCache interface {
	SetMeta(ctx context.Context, code string, meta *model.RoomMeta) error
	GetMeta(ctx context.Context, code string) (*model.RoomMeta, error)
	SetStatus(ctx context.Context, code string, status model.RoomStatus) error
	Delete(ctx context.Context, code string) error
	Exists(ctx context.Context, code string) (bool, error)
}

type roomCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRoomCache creates a new room cache
func NewRoomCache(client *redis.Client) RoomCache {
	return &roomCache{
		client: client,
		ttl:    24 * time.Hour, // Rooms expire after 24h
	}
}

func (c *roomCache) key(code string) string {
	return fmt.Sprintf("room:%s", code)
}

func (c *roomCache) SetMeta(ctx context.Context, code string, meta *model.RoomMeta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(code), data, c.ttl).Err()
}

func (c *roomCache) GetMeta(ctx context.Context, code string) (*model.RoomMeta, error) {
	data, err := c.client.Get(ctx, c.key(code)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var meta model.RoomMeta
	if err := json.Unmarshal([]byte(data), &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (c *roomCache) SetStatus(ctx context.Context, code string, status model.RoomStatus) error {
	meta, err := c.GetMeta(ctx, code)
	if err != nil {
		return err
	}
	if meta == nil {
		return fmt.Errorf("room %s not found", code)
	}
	meta.Status = status
	return c.SetMeta(ctx, code, meta)
}

func (c *roomCache) Delete(ctx context.Context, code string) error {
	return c.client.Del(ctx, c.key(code)).Err()
}

func (c *roomCache) Exists(ctx context.Context, code string) (bool, error) {
	n, err := c.client.Exists(ctx, c.key(code)).Result()
	return n > 0, err
}
