package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// LeaderboardCache handles Redis ZSET operations for leaderboard
type LeaderboardCache interface {
	UpdateScore(ctx context.Context, roomCode, playerID string, score int) error
	GetTop(ctx context.Context, roomCode string, limit int) ([]LeaderboardEntry, error)
	GetRank(ctx context.Context, roomCode, playerID string) (int64, error)
}

// LeaderboardEntry represents a single leaderboard entry
type LeaderboardEntry struct {
	PlayerID string `json:"playerId"`
	Nickname string `json:"nickname"`
	Score    int    `json:"score"`
	Rank     int    `json:"rank"`
}

type leaderboardCache struct {
	client *redis.Client
}

// NewLeaderboardCache creates a new leaderboard cache
func NewLeaderboardCache(client *redis.Client) LeaderboardCache {
	return &leaderboardCache{
		client: client,
	}
}

func (c *leaderboardCache) key(roomCode string) string {
	return fmt.Sprintf("room:%s:lb", roomCode)
}

func (c *leaderboardCache) UpdateScore(ctx context.Context, roomCode, playerID string, score int) error {
	return c.client.ZAdd(ctx, c.key(roomCode), redis.Z{
		Score:  float64(score),
		Member: playerID,
	}).Err()
}

func (c *leaderboardCache) GetTop(ctx context.Context, roomCode string, limit int) ([]LeaderboardEntry, error) {
	results, err := c.client.ZRevRangeWithScores(ctx, c.key(roomCode), 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	entries := make([]LeaderboardEntry, len(results))
	for i, z := range results {
		entries[i] = LeaderboardEntry{
			PlayerID: z.Member.(string),
			Score:    int(z.Score),
			Rank:     i + 1,
		}
	}
	return entries, nil
}

func (c *leaderboardCache) GetRank(ctx context.Context, roomCode, playerID string) (int64, error) {
	rank, err := c.client.ZRevRank(ctx, c.key(roomCode), playerID).Result()
	if err == redis.Nil {
		return -1, nil
	}
	return rank + 1, err // 1-indexed
}
