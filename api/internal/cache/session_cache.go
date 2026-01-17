package cache

import (
	"2026champs/internal/model"
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionCache interface {
	Set(ctx context.Context, session *model.Session) error
	Get(ctx context.Context, id string) (*model.Session, error)
	Delete(ctx context.Context, id string) error
}

type sessionCache struct {
	client *redis.Client
}

func NewSessionCache(client *redis.Client) SessionCache {
	return &sessionCache{
		client: client,
	}
}

func (c *sessionCache) Set(ctx context.Context, session *model.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, "session:"+session.ID, data, 10*time.Minute).Err()
}

func (c *sessionCache) Get(ctx context.Context, id string) (*model.Session, error) {
	data, err := c.client.Get(ctx, "session:"+id).Result()
	if err != nil {
		return nil, err
	}
	var session model.Session
	err = json.Unmarshal([]byte(data), &session)
	return &session, err
}

func (c *sessionCache) Delete(ctx context.Context, id string) error {
	return c.client.Del(ctx, "session:"+id).Err()
}
