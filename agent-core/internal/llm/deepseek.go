package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// DeepSeekProvider 基于 OpenAI 兼容接口的 DeepSeek LLM 客户端。
// 默认使用 deepseek-v4-flash 模型 + non-thinking 模式。
type DeepSeekProvider struct {
	client openai.Client
	model  string
}

// NewDeepSeek 从环境变量创建 DeepSeek 客户端。
//
// 环境变量：
//   - DEEPSEEK_API_KEY（必填）
//   - DEEPSEEK_MODEL（默认 deepseek-v4-flash）
func NewDeepSeek() (*DeepSeekProvider, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY 未设置")
	}

	model := os.Getenv("DEEPSEEK_MODEL")
	if model == "" {
		model = "deepseek-v4-flash"
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://api.deepseek.com/v1"),
	)

	return &DeepSeekProvider{
		client: client,
		model:  model,
	}, nil
}

// Model 返回当前使用的模型 ID。
func (d *DeepSeekProvider) Model() string { return d.model }

// Chat 实现 LLMClient 接口，调用 DeepSeek Chat Completions API。
//
// 关键设计：
//   - 使用 extra_body {"thinking":{"type":"disabled"}} 禁用思考模式
//   - 当 LLM 返回 finish_reason="tool_calls" 时，解析 ToolCalls 字段
//   - 当 LLM 返回 finish_reason="stop" 时，直接返回 Content
func (d *DeepSeekProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 转换消息格式
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, m := range req.Messages {
		switch m.Role {
		case RoleSystem:
			messages = append(messages, openai.SystemMessage(m.Content))
		case RoleUser:
			messages = append(messages, openai.UserMessage(m.Content))
		case RoleAssistant:
			if m.Content != "" {
				messages = append(messages, openai.AssistantMessage(m.Content))
			}
		case RoleTool:
			messages = append(messages, openai.ToolMessage(m.Content, m.ToolCallID))
		}
	}

	// 转换工具定义
	tools := make([]openai.ChatCompletionToolParam, 0, len(req.Tools))
	for _, t := range req.Tools {
		tools = append(tools, openai.ChatCompletionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        t.Name,
				Description: openai.String(t.Description),
				Parameters:  openai.FunctionParameters(t.Parameters.(map[string]interface{})),
			},
		})
	}

	params := openai.ChatCompletionNewParams{
		Model:    d.model,
		Messages: messages,
		Tools:    tools,
	}

	// 禁用 thinking 模式以降低延迟和 Token 费用
	params.SetExtraFields(map[string]interface{}{
		"thinking": map[string]interface{}{
			"type": "disabled",
		},
	})

	resp, err := d.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("deepseek chat: %w", err)
	}

	choice := resp.Choices[0]
	chatResp := &ChatResponse{
		Model: resp.Model,
	}

	// 根据 finish_reason 判断是 tool call 还是纯文本
	if choice.FinishReason == "tool_calls" && len(choice.Message.ToolCalls) > 0 {
		for _, tc := range choice.Message.ToolCalls {
			chatResp.ToolCalls = append(chatResp.ToolCalls, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
	} else {
		chatResp.Content = choice.Message.Content
	}

	return chatResp, nil
}

// toolCallToJSON 将 LLM 返回的 tool call arguments 解析为 map。
func toolCallToJSON(args string) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(args), &m); err != nil {
		return nil, fmt.Errorf("parse tool args: %w", err)
	}
	return m, nil
}
