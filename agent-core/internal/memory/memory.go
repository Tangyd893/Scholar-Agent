// Package memory 定义会话存储接口与内存实现。
// Phase 1 使用 InMemoryStore；Phase 1 Day 6 替换为 RedisStore。
package memory

import (
	"fmt"
	"sync"
	"time"

	"github.com/Tangyd893/Scholar-Agent/pkg/agent"
)

// MemoryStore 是会话历史的持久化接口。
// Phase 1 实现：InMemoryStore；Phase 2：RedisStore。
type MemoryStore interface {
	// Create 创建新会话，返回 session ID。
	Create(userID, title string) (string, error)

	// GetHistory 获取会话最近 lastN 条消息。
	GetHistory(sessionID string, lastN int) ([]agent.Message, error)

	// Append 追加一条消息到会话历史。
	Append(sessionID string, msg agent.Message) error

	// AppendSteps 追加推理链步骤。
	AppendSteps(sessionID string, steps []agent.StepEvent) error

	// Touch 刷新会话 TTL（预留，内存实现为空操作）。
	Touch(sessionID string) error
}

// InMemoryStore 是 MemoryStore 的线程安全内存实现。
// 服务重启后数据丢失；生产环境请使用 RedisStore。
type InMemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*session
	userSess map[string][]string // userID → sessionIDs
}

type session struct {
	ID        string
	UserID    string
	Title     string
	CreatedAt time.Time
	Messages  []agent.Message
	Steps     []agent.StepEvent
}

// NewInMemoryStore 创建空的内存存储。
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		sessions: make(map[string]*session),
		userSess: make(map[string][]string),
	}
}

func (s *InMemoryStore) Create(userID, title string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("sess_%d", time.Now().UnixNano())
	sess := &session{
		ID:        id,
		UserID:    userID,
		Title:     title,
		CreatedAt: time.Now(),
	}
	s.sessions[id] = sess
	s.userSess[userID] = append(s.userSess[userID], id)
	return id, nil
}

func (s *InMemoryStore) GetHistory(sessionID string, lastN int) ([]agent.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	msgs := sess.Messages
	if lastN > 0 && len(msgs) > lastN {
		msgs = msgs[len(msgs)-lastN:]
	}

	// 返回副本
	result := make([]agent.Message, len(msgs))
	copy(result, msgs)
	return result, nil
}

func (s *InMemoryStore) Append(sessionID string, msg agent.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	msg.Timestamp = time.Now()
	sess.Messages = append(sess.Messages, msg)
	return nil
}

func (s *InMemoryStore) AppendSteps(sessionID string, steps []agent.StepEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	sess.Steps = append(sess.Steps, steps...)
	return nil
}

func (s *InMemoryStore) Touch(sessionID string) error {
	// 内存实现无需 TTL 刷新
	return nil
}
