// Package agent 实现 ReAct（Reasoning + Acting）推理循环引擎。
//
// 核心流程：
//  1. 加载会话历史
//  2. 循环（最多 maxSteps 次）：
//     a. 调用 LLM（带工具定义，non-thinking 模式）
//     b. 无 Tool Call → 推送 answer 事件并结束
//     c. 有 Tool Call → 推送 thought/action，执行工具，推送 observation
//     d. 将工具结果追加到消息历史
//  3. 达到 maxSteps → 推送 error 事件
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/llm"
	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/memory"
	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/tool"
	"github.com/Tangyd893/Scholar-Agent/pkg/agent"
)

const (
	defaultMaxSteps     = 5
	defaultDeviceID     = "cli-user"
	defaultSessionTitle = "CLI 问答"
)

// systemPrompt 是 Agent 的系统提示词。
const systemPrompt = `你是一个学术研究助手 ScholarAgent，帮助用户检索、理解和总结学术论文。

你可以使用以下策略：
1. 当用户询问论文时，使用 search_papers 工具搜索相关论文
2. 根据搜索结果，筛选最相关的论文进行推荐
3. 用中文回答，但论文标题和作者保留原文

重要规则：
- 每次只调用一个工具
- 搜索时使用英文关键词以获得更好的结果
- 回答时引用论文标题和年份
- 如果工具返回空结果，建议用户更换搜索词`

// Agent 是 ReAct 推理引擎的核心结构。
type Agent struct {
	llm      llm.LLMClient
	tools    *tool.ToolRegistry
	memory   memory.MemoryStore
	maxSteps int
}

// New 创建一个新的 Agent 实例。
// 参数从环境变量读取；llm 和 memory 为 nil 时使用默认实现。
func New(llmClient llm.LLMClient, mem memory.MemoryStore) *Agent {
	maxSteps := defaultMaxSteps
	if v := os.Getenv("AGENT_MAX_STEPS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxSteps = n
		}
	}

	if mem == nil {
		mem = memory.NewInMemoryStore()
	}

	return &Agent{
		llm:      llmClient,
		tools:    tool.NewRegistry(),
		memory:   mem,
		maxSteps: maxSteps,
	}
}

// RegisterTool 向 Agent 注册一个工具。
func (a *Agent) RegisterTool(t tool.Tool) {
	a.tools.Register(t)
}

// Run 执行一次 ReAct 推理，通过 channel 推送 StepEvent。
//
// 调用方通过消费返回的 channel 获取实时推理事件。
// channel 在推理完成或出错时关闭。
func (a *Agent) Run(ctx context.Context, sessionID, query string) (<-chan agent.StepEvent, error) {
	// 确保会话存在
	history, err := a.memory.GetHistory(sessionID, 0)
	if err != nil {
		// 会话不存在时自动创建
		sessionID, err = a.memory.Create(defaultDeviceID, defaultSessionTitle)
		if err != nil {
			return nil, fmt.Errorf("create session: %w", err)
		}
		slog.Info("auto-created session", "session_id", sessionID)
		history = nil
	}

	// 追加用户消息
	userMsg := agent.Message{
		Role:    "user",
		Content: query,
	}
	if err := a.memory.Append(sessionID, userMsg); err != nil {
		return nil, fmt.Errorf("append user message: %w", err)
	}

	events := make(chan agent.StepEvent, 20)

	go func() {
		defer close(events)
		a.runLoop(ctx, sessionID, query, history, events)
	}()

	return events, nil
}

// runLoop 是 ReAct 主循环的实现，运行在独立 goroutine 中。
func (a *Agent) runLoop(ctx context.Context, sessionID, query string, history []agent.Message, events chan<- agent.StepEvent) {
	// 构建初始消息列表
	messages := a.buildMessages(history, query)
	toolDefs := a.buildToolDefs()

	var allSteps []agent.StepEvent

	for step := 1; step <= a.maxSteps; step++ {
		select {
		case <-ctx.Done():
			emit(events, agent.EventError, fmt.Sprintf("请求被取消: %v", ctx.Err()), int32(step))
			return
		default:
		}

		// ---------- 调用 LLM ----------
		slog.Info("agent: calling LLM", "step", step, "msgCount", len(messages), "toolCount", len(toolDefs))

		resp, err := a.llm.Chat(ctx, &llm.ChatRequest{
			Messages: messages,
			Tools:    toolDefs,
		})
		if err != nil {
			slog.Error("agent: LLM call failed", "step", step, "error", err)
			emit(events, agent.EventError, fmt.Sprintf("LLM 调用失败: %v", err), int32(step))
			return
		}

		// ---------- 无 Tool Call → 最终回答 ----------
		if len(resp.ToolCalls) == 0 {
			slog.Info("agent: got answer", "step", step)

			answerEvent := agent.StepEvent{
				Type:      agent.EventAnswer,
				Content:   resp.Content,
				Step:      int32(step),
				Timestamp: time.Now(),
			}
			allSteps = append(allSteps, answerEvent)
			events <- answerEvent

			// 保存 assistant 消息 + 推理链
			assistantMsg := agent.Message{
				Role:    "assistant",
				Content: resp.Content,
				Steps:   allSteps,
			}
			if err := a.memory.Append(sessionID, assistantMsg); err != nil {
				slog.Warn("agent: failed to save assistant message", "error", err)
			}

			return
		}

		// ---------- 有 Tool Call ----------
		for _, tc := range resp.ToolCalls {
			// 发布 thought 事件
			thought := agent.StepEvent{
				Type:      agent.EventThought,
				Content:   fmt.Sprintf("需要调用工具 %s 获取更多信息", tc.Name),
				Step:      int32(step),
				Timestamp: time.Now(),
			}
			allSteps = append(allSteps, thought)
			events <- thought

			// 发布 action 事件
			action := agent.StepEvent{
				Type:         agent.EventAction,
				Content:      tc.Name,
				Step:         int32(step),
				ToolArgsJSON: tc.Arguments,
				Timestamp:    time.Now(),
			}
			allSteps = append(allSteps, action)
			events <- action

			slog.Info("agent: executing tool", "step", step, "tool", tc.Name)

			// 执行工具
			result, err := a.tools.Execute(ctx, tc.Name, tc.Arguments)

			// 构建 observation 内容
			var obsContent string
			if err != nil {
				obsContent = fmt.Sprintf("工具 %s 执行失败: %v", tc.Name, err)
			} else {
				obsContent = result
			}

			// 发布 observation 事件
			observation := agent.StepEvent{
				Type:      agent.EventObservation,
				Content:   obsContent,
				Step:      int32(step),
				Timestamp: time.Now(),
			}
			allSteps = append(allSteps, observation)
			events <- observation

			// 将 assistant 的 tool call 消息追加到消息历史（供 LLM 上下文）
			messages = append(messages, llm.Message{
				Role:    llm.RoleAssistant,
				Content: "", // tool call 时 content 为空
			})

			// 将 tool 执行结果追加到消息历史
			messages = append(messages, llm.Message{
				Role:       llm.RoleTool,
				Content:    obsContent,
				ToolCallID: tc.ID,
				Name:       tc.Name,
			})
		}
	}

	// ---------- 达到 maxSteps ----------
	slog.Warn("agent: max steps reached", "maxSteps", a.maxSteps)
	emit(events, agent.EventError,
		fmt.Sprintf("推理步数已达上限（%d 步），请简化问题或开启新会话", a.maxSteps),
		int32(a.maxSteps))
}

// buildMessages 构建发送给 LLM 的消息列表。
func (a *Agent) buildMessages(history []agent.Message, query string) []llm.Message {
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: systemPrompt},
	}

	// 追加历史消息（排除之前的推理链细节，只传角色+内容）
	for _, m := range history {
		msgs = append(msgs, llm.Message{
			Role:    llm.Role(m.Role),
			Content: m.Content,
		})
	}

	return msgs
}

// buildToolDefs 将注册表中的工具转换为 LLM 可理解的 ToolDef 列表。
func (a *Agent) buildToolDefs() []llm.ToolDef {
	list := a.tools.List()
	defs := make([]llm.ToolDef, 0, len(list))
	for _, t := range list {
		defs = append(defs, llm.ToolDef{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Schema,
		})
	}
	return defs
}

// emit 是向 channel 发送事件的辅助函数。
func emit(events chan<- agent.StepEvent, typ agent.EventType, content string, step int32) {
	events <- agent.StepEvent{
		Type:      typ,
		Content:   content,
		Step:      step,
		Timestamp: time.Now(),
	}
}

// =========================================================================
// MockLLM — 开发期使用的 mock LLM，不依赖外部 API
// =========================================================================

// MockLLM 实现 LLMClient 接口，返回固定的 tool call。
// 用于在没有 API Key 的情况下验证 ReAct 流程。
type MockLLM struct {
	mu          sync.Mutex
	callCount   int
	ToolName    string
	ToolArgs    string
	FinalAnswer string
}

func (m *MockLLM) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	m.mu.Lock()
	m.callCount++
	count := m.callCount
	m.mu.Unlock()

	// 第一次调用：返回 tool call
	if count == 1 {
		toolName := m.ToolName
		if toolName == "" {
			toolName = "search_papers"
		}
		toolArgs := m.ToolArgs
		if toolArgs == "" {
			toolArgs = `{"query":"attention mechanism"}`
		}
		return &llm.ChatResponse{
			ToolCalls: []llm.ToolCall{
				{
					ID:        "mock_call_1",
					Name:      toolName,
					Arguments: toolArgs,
				},
			},
			Model: "mock",
		}, nil
	}

	// 第二次调用：返回最终答案
	answer := m.FinalAnswer
	if answer == "" {
		answer = "根据搜索结果，推荐以下经典论文：\n1. Attention Is All You Need (Vaswani et al., 2017)\n2. BERT (Devlin et al., 2018)"
	}
	return &llm.ChatResponse{
		Content: answer,
		Model:   "mock",
	}, nil
}

// Ensure interfaces are satisfied
var _ llm.LLMClient = (*MockLLM)(nil)
var _ llm.LLMClient = (*llm.DeepSeekProvider)(nil)

// toolCallToJSON 将 LLM 返回的 tool call arguments 解析为 map。
func toolCallToJSON(args string) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(args), &m); err != nil {
		return nil, fmt.Errorf("parse tool args: %w", err)
	}
	return m, nil
}
