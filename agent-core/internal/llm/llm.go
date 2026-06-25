// Package llm 定义 LLM 客户端抽象接口与通用类型。
// 所有 LLM 提供商（DeepSeek、MiMo、MiniMax）均实现 LLMClient 接口，
// 以便 Agent Core 在运行时进行降级切换。
package llm

import "context"

// LLMClient 是 LLM 调用的统一抽象，封装不同提供商的 API。
// 当前 Phase 1 仅实现 Chat；ChatStream 留待 SSE 阶段。
type LLMClient interface {
	// Chat 发送对话请求并返回完整响应。
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}

// Role 定义消息角色。
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message 表示一条对话消息。
type Message struct {
	Role       Role   `json:"role"`
	Content    string `json:"content,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"` // tool 消息时使用
	Name       string `json:"name,omitempty"`         // tool 消息时使用
}

// ToolDef 描述一个工具的函数签名，发送给 LLM 用于 Function Calling。
type ToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"` // JSON Schema
}

// ToolCall 表示 LLM 返回的一次工具调用请求。
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ChatRequest 是一次 LLM 调用的请求参数。
type ChatRequest struct {
	Messages    []Message `json:"messages"`
	Tools       []ToolDef `json:"tools,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// ChatResponse 是 LLM 调用的完整响应。
type ChatResponse struct {
	Content   string     `json:"content,omitempty"`   // 纯文本回答（无 tool call 时）
	ToolCalls []ToolCall `json:"tool_calls,omitempty"` // 工具调用请求
	Model     string     `json:"model"`
}
