// Package agent 定义 Agent 核心共享类型（StepEvent 等），
// 由 agent-core 写入，gateway 和 tool-service 消费。
// 禁止各服务复制此结构体。
package agent

import "time"

// EventType 定义 SSE / gRPC Stream 中推送的事件类型。
type EventType string

const (
	EventThought     EventType = "thought"
	EventAction      EventType = "action"
	EventObservation EventType = "observation"
	EventAnswer      EventType = "answer"
	EventError       EventType = "error"
	EventJobComplete EventType = "job_complete"
)

// StepEvent 是 ReAct 推理循环中每一步的标准化事件，
// 通过 gRPC Stream 和 SSE 推送给调用方。
type StepEvent struct {
	Type         EventType `json:"type"`
	Content      string    `json:"content"`
	Step         int32     `json:"step"`
	ToolArgsJSON string    `json:"tool_args,omitempty"` // action 事件时附带
	Timestamp    time.Time `json:"timestamp"`
}

// Message 表示会话中的一条对话消息。
type Message struct {
	Role      string      `json:"role"` // "user" | "assistant"
	Content   string      `json:"content"`
	Steps     []StepEvent `json:"steps,omitempty"` // assistant 消息可附带推理链
	Timestamp time.Time   `json:"timestamp"`
}
