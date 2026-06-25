// ScholarAgent CLI — Phase 1 端到端验收入口
//
// 用法：
//   go run ./agent-core/cmd/cli --query "帮我找 attention 相关的经典论文"          （DeepSeek + mock 工具）
//   go run ./agent-core/cmd/cli --query "测试" --mock                             （MockLLM + mock 工具，零 API 调用）
//   go run ./agent-core/cmd/cli --query "attention mechanism" --mock --arxiv     （MockLLM + 真实 arXiv）
//   go run ./agent-core/cmd/cli --query "attention mechanism" --arxiv            （DeepSeek + 真实 arXiv 本地）
//   go run ./agent-core/cmd/cli --query "attention mechanism" --arxiv --grpc     （DeepSeek + arXiv via gRPC）
//
// 环境变量：
//   DEEPSEEK_API_KEY — 设置后自动使用真实 DeepSeek（否则回退 MockLLM）
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/agent"
	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/llm"
	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/memory"
	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/tool"
	pkgagent "github.com/Tangyd893/Scholar-Agent/pkg/agent"
)

func main() {
	query := flag.String("query", "", "向 Agent 提问的问题（必填）")
	useMock := flag.Bool("mock", false, "强制使用 Mock LLM（无需 API Key）")
	useArxiv := flag.Bool("arxiv", false, "使用真实 arXiv API（默认 mock 工具）")
	useGrpc := flag.Bool("grpc", false, "通过 gRPC 调用 tool-service（需先启动 tool-service）")
	flag.Parse()

	if *query == "" {
		fmt.Fprintln(os.Stderr, "用法: go run ./agent-core/cmd/cli --query \"你的问题\" [--mock]")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// =========================================================================
	// 1. 初始化 LLM 客户端
	// =========================================================================
	var llmClient llm.LLMClient
	usingMock := *useMock

	if !usingMock {
		// 尝试连接真实 DeepSeek
		ds, err := llm.NewDeepSeek()
		if err != nil {
			fmt.Printf("⚠️  DeepSeek 初始化失败: %v\n", err)
			fmt.Println("   自动切换为 MockLLM 模式（使用 --mock 可跳过此提示）")
			fmt.Println()
			usingMock = true
		} else {
			llmClient = ds
			fmt.Printf("🔗 使用 DeepSeek (%s)\n\n", ds.Model())
		}
	}

	if usingMock {
		llmClient = &agent.MockLLM{
			ToolName: "search_papers",
			ToolArgs: `{"query":"attention mechanism"}`,
			FinalAnswer: `根据搜索结果，为您推荐以下关于 attention mechanism 的经典论文：

1. **Attention Is All You Need** (Vaswani et al., 2017)
   - arXiv: 1706.03762
   - 提出了 Transformer 架构，完全基于注意力机制

2. **Neural Machine Translation by Jointly Learning to Align and Translate** (Bahdanau et al., 2014)
   - arXiv: 1409.0473
   - 首次将注意力机制引入神经机器翻译

3. **BERT: Pre-training of Deep Bidirectional Transformers** (Devlin et al., 2018)
   - arXiv: 1810.04805
   - 基于 Transformer 编码器的预训练语言模型`,
		}
		fmt.Println("🧪 使用 MockLLM（模拟 DeepSeek 行为）\n")
	}

	// =========================================================================
	// 2. 初始化组件
	// =========================================================================
	mem := memory.NewInMemoryStore()
	ag := agent.New(llmClient, mem)

	// 注册工具
	if *useGrpc {
		// gRPC 模式：连接远程 tool-service
		grpcReg, err := tool.NewGrpcRegistry("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ gRPC 连接失败: %v\n", err)
			fmt.Fprintln(os.Stderr, "   请先启动 tool-service: go run ./tool-service")
			os.Exit(1)
		}
		defer grpcReg.Close()

		// 注册工具元数据（Schema 等），实际执行走 gRPC
		s := &tool.MockSearchPapers{}
		grpcReg.RegisterMeta(s.Name(), s.Description(), s.Schema())
		// 注册 get_abstract（仅在 gRPC 模式下可用）
		grpcReg.RegisterMeta("get_abstract", "按 arXiv paper_id 获取论文完整摘要", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"paper_id": map[string]interface{}{
					"type":        "string",
					"description": "arXiv 论文 ID，如 1706.03762",
				},
			},
			"required": []string{"paper_id"},
		})

		ag.SetToolExecutor(grpcReg)
		fmt.Println("🔗 通过 gRPC 调用 tool-service\n")
	} else if *useArxiv {
		ag.RegisterTool(tool.NewArxivSearch())
		fmt.Println("📡 使用真实 arXiv API（本地）\n")
	} else {
		ag.RegisterTool(&tool.MockSearchPapers{})
	}

	// 创建会话
	sessionID, err := mem.Create("cli-user", *query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 创建会话失败: %v\n", err)
		os.Exit(1)
	}

	// =========================================================================
	// 3. 运行 Agent
	// =========================================================================
	fmt.Printf("📝 问题: %s\n", *query)
	fmt.Printf("📋 会话: %s\n", sessionID)
	fmt.Println(strings.Repeat("─", 60))

	events, err := ag.Run(ctx, sessionID, *query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Agent 启动失败: %v\n", err)
		os.Exit(1)
	}

	// 消费事件流，格式化打印
	for event := range events {
		printEvent(event)
	}

	fmt.Println(strings.Repeat("─", 60))
	fmt.Println("✅ 推理完成")
}

// printEvent 格式化打印一个 StepEvent。
func printEvent(e pkgagent.StepEvent) {
	switch e.Type {
	case pkgagent.EventThought:
		fmt.Printf("\n💭 [思考 %d] %s\n", e.Step, e.Content)

	case pkgagent.EventAction:
		fmt.Printf("🔧 [行动 %d] 调用工具: %s\n", e.Step, e.Content)
		if e.ToolArgsJSON != "" {
			fmt.Printf("   参数: %s\n", e.ToolArgsJSON)
		}

	case pkgagent.EventObservation:
		// 截断过长的 observation
		content := e.Content
		if len(content) > 500 {
			content = content[:500] + "...(已截断)"
		}
		fmt.Printf("📊 [观察 %d]\n%s\n", e.Step, content)

	case pkgagent.EventAnswer:
		fmt.Printf("\n🎓 [回答]\n%s\n", e.Content)

	case pkgagent.EventError:
		fmt.Printf("\n❌ [错误] %s\n", e.Content)
	}
}
