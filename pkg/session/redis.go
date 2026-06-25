// Package session 提供网关使用的轻量 Redis 会话存储。
// 与 agent-core/internal/memory 共享相同的 Key 模式。
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Tangyd893/Scholar-Agent/pkg/agent"
)

// Store 是 Redis 会话存储，实现与 agent-core/internal/memory 兼容的接口。
type Store struct {
	client *redis.Client
	ttl    time.Duration
}

// NewStore 创建 Redis 会话存储。
func NewStore() (*Store, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	ttl := 7 * 24 * time.Hour
	if v := os.Getenv("SESSION_TTL_DAYS"); v != "" {
		if days, err := strconv.Atoi(v); err == nil && days > 0 {
			ttl = time.Duration(days) * 24 * time.Hour
		}
	}
	return &Store{client: client, ttl: ttl}, nil
}

func (s *Store) Create(userID, title string) (string, error) {
	ctx := context.Background()
	id := fmt.Sprintf("sess_%d", time.Now().UnixNano())
	meta, _ := json.Marshal(map[string]interface{}{
		"user_id": userID, "title": title,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	})
	pipe := s.client.Pipeline()
	pipe.Set(ctx, fmt.Sprintf("session:%s:meta", id), meta, s.ttl)
	pipe.SAdd(ctx, fmt.Sprintf("user:%s:sessions", userID), id)
	_, err := pipe.Exec(ctx)
	return id, err
}

func (s *Store) GetMessages(sessionID string, lastN int) ([]agent.Message, error) {
	ctx := context.Background()
	var start int64 = 0
	var stop int64 = -1
	if lastN > 0 {
		start = -int64(lastN)
	}
	raw, err := s.client.LRange(ctx, fmt.Sprintf("session:%s:messages", sessionID), start, stop).Result()
	if err != nil {
		return nil, err
	}
	msgs := make([]agent.Message, 0, len(raw))
	for _, r := range raw {
		var m agent.Message
		if err := json.Unmarshal([]byte(r), &m); err != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}
