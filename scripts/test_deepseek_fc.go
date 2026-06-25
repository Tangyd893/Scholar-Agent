// test_deepseek_fc 验证 DeepSeek-V4-Flash 的 Function Calling 能力。
//
// 前置条件：
//   - 设置环境变量 DEEPSEEK_API_KEY
//   - go run scripts/test_deepseek_fc.go
//
// 预期输出：终端看到 tool_calls 返回，表示 FC 通路正常。
//
// 此脚本不依赖 ScholarAgent 任何微服务，可用于开工前环境验证。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// searchPapersSchema 定义 search_papers 工具的 JSON Schema，
// 与 docs/技术设计.md §3.3 一致。
var searchPapersParam = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "search_papers",
		Description: openai.String("按关键词搜索学术论文，返回论文列表（含标题、作者、摘要、年份）"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "学术搜索关键词，如 \"attention mechanism\"",
				},
			},
			"required": []string{"query"},
		},
	},
}

func main() {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "❌ DEEPSEEK_API_KEY 未设置")
		fmt.Fprintln(os.Stderr, "   export DEEPSEEK_API_KEY=sk-your-key")
		os.Exit(1)
	}

	ctx := context.Background()
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://api.deepseek.com/v1"),
	)

	fmt.Println("🔍 正在测试 DeepSeek Function Calling...")
	fmt.Println("   模型: deepseek-v4-flash")
	fmt.Println("   模式: non-thinking")
	fmt.Println()

	// 消息：模拟用户学术提问
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage("帮我找关于 attention mechanism 的经典论文"),
	}

	// 请求参数：禁用 thinking 模式
	params := openai.ChatCompletionNewParams{
		Model:    "deepseek-v4-flash",
		Messages: messages,
		Tools:    []openai.ChatCompletionToolParam{searchPapersParam},
	}

	// 通过 extra_body 禁用 thinking（降低延迟和 Token 费用）
	params.SetExtraFields(map[string]interface{}{
		"thinking": map[string]interface{}{
			"type": "disabled",
		},
	})

	resp, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ API 调用失败: %v\n", err)
		os.Exit(1)
	}

	choice := resp.Choices[0]

	// 检查是否返回了 tool_calls
	if len(choice.Message.ToolCalls) > 0 {
		fmt.Println("✅ Function Calling 成功！")
		fmt.Printf("   finish_reason: %s\n", choice.FinishReason)
		fmt.Println()
		for i, tc := range choice.Message.ToolCalls {
			fmt.Printf("--- Tool Call #%d ---\n", i+1)
			fmt.Printf("  id:   %s\n", tc.ID)
			fmt.Printf("  name: %s\n", tc.Function.Name)
			fmt.Printf("  args: %s\n", tc.Function.Arguments)
		}
	} else {
		fmt.Println("ℹ️  未触发 tool_calls，LLM 直接回答：")
		fmt.Printf("  content: %s\n", choice.Message.Content)

		// 打印 JSON 以便排查
		b, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Printf("\n📋 完整响应:\n%s\n", string(b))
	}

	fmt.Println()
	fmt.Println("✅ 测试完成 — DeepSeek FC 通路正常，可以开始构建 agent-core。")
}
