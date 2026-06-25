package memory

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

// RedisStore 是 MemoryStore 的 Redis 实现。
// Key 模式与 docs/技术设计.md §3.7 完全一致。
type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisStore 创建 Redis 会话存储。
// 连接地址从 REDIS_URL 环境变量读取，默认 redis://localhost:6379。
func NewRedisStore() (*RedisStore, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}

	client := redis.NewClient(opts)

	// 验证连接
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

	return &RedisStore{client: client, ttl: ttl}, nil
}

// Create 创建新会话。
func (s *RedisStore) Create(userID, title string) (string, error) {
	ctx := context.Background()
	id := fmt.Sprintf("sess_%d", time.Now().UnixNano())

	meta := map[string]interface{}{
		"user_id":    userID,
		"title":      title,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}

	metaJSON, _ := json.Marshal(meta)

	pipe := s.client.Pipeline()
	pipe.Set(ctx, keyMeta(id), metaJSON, s.ttl)
	pipe.SAdd(ctx, keyUser(userID), id)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	return id, nil
}

// GetHistory 获取最近 lastN 条消息。
func (s *RedisStore) GetHistory(sessionID string, lastN int) ([]agent.Message, error) {
	ctx := context.Background()

	// 检查会话是否存在
	exists, err := s.client.Exists(ctx, keyMeta(sessionID)).Result()
	if err != nil {
		return nil, fmt.Errorf("check session: %w", err)
	}
	if exists == 0 {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// 读取消息列表
	var start int64 = 0
	var stop int64 = -1
	if lastN > 0 {
		start = -int64(lastN)
	}

	raw, err := s.client.LRange(ctx, keyMessages(sessionID), start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	msgs := make([]agent.Message, 0, len(raw))
	for _, r := range raw {
		var m agent.Message
		if err := json.Unmarshal([]byte(r), &m); err != nil {
			continue // 跳过损坏的消息
		}
		msgs = append(msgs, m)
	}

	// 刷新 TTL
	s.client.Expire(ctx, keyMeta(sessionID), s.ttl)
	s.client.Expire(ctx, keyMessages(sessionID), s.ttl)
	s.client.Expire(ctx, keySteps(sessionID), s.ttl)

	return msgs, nil
}

// Append 追加消息。
func (s *RedisStore) Append(sessionID string, msg agent.Message) error {
	ctx := context.Background()

	msg.Timestamp = time.Now()
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	pipe := s.client.Pipeline()
	pipe.RPush(ctx, keyMessages(sessionID), msgJSON)
	pipe.Expire(ctx, keyMessages(sessionID), s.ttl)
	pipe.Expire(ctx, keyMeta(sessionID), s.ttl)
	_, err = pipe.Exec(ctx)
	return err
}

// AppendSteps 追加推理链步骤。
func (s *RedisStore) AppendSteps(sessionID string, steps []agent.StepEvent) error {
	ctx := context.Background()

	for _, step := range steps {
		b, err := json.Marshal(step)
		if err != nil {
			continue
		}
		if err := s.client.RPush(ctx, keySteps(sessionID), b).Err(); err != nil {
			return fmt.Errorf("append step: %w", err)
		}
	}

	s.client.Expire(ctx, keySteps(sessionID), s.ttl)
	return nil
}

// Touch 刷新会话 TTL。
func (s *RedisStore) Touch(sessionID string) error {
	ctx := context.Background()
	pipe := s.client.Pipeline()
	pipe.Expire(ctx, keyMeta(sessionID), s.ttl)
	pipe.Expire(ctx, keyMessages(sessionID), s.ttl)
	pipe.Expire(ctx, keySteps(sessionID), s.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// =========================================================================
// Redis Key 命名
// =========================================================================

func keyMeta(id string) string      { return fmt.Sprintf("session:%s:meta", id) }
func keyMessages(id string) string  { return fmt.Sprintf("session:%s:messages", id) }
func keySteps(id string) string     { return fmt.Sprintf("session:%s:steps", id) }
func keyUser(id string) string      { return fmt.Sprintf("user:%s:sessions", id) }
