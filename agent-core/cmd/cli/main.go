// ScholarAgent CLI — Phase 1 端到端验收入口
//
// 用法：
//   go run ./agent-core/cmd/cli --query "帮我找 attention 相关的经典论文"          （DeepSeek + mock 工具）
//   go run ./agent-core/cmd/cli --query "测试" --mock                             （MockLLM + mock 工具，零 API 调用）
//   go run ./agent-core/cmd/cli --query "attention mechanism" --mock --arxiv     （MockLLM + 真实 arXiv）
//   go run ./agent-core/cmd/cli --query "attention mechanism" --arxiv            （DeepSeek + 真实 arXiv 本地）
//   go run ./agent-core/cmd/cli --query "attention mechanism" --arxiv --grpc     （DeepSeek + arXiv via gRPC）
//   go run ./agent-core/cmd/cli --query "继续" --mock --redis                    （MockLLM + Redis 持久化）
//   go run ./agent-core/cmd/cli --query "测试" --mock --redis --session sess_xxx （恢复指定会话）
//
// 环境变量：
//   DEEPSEEK_API_KEY — 设置后自动使用真实 DeepSeek（否则回退 MockLLM）
//   REDIS_URL         — Redis 连接串（默认 redis://localhost:6379）
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
	useRedis := flag.Bool("redis", false, "使用 Redis 持久化会话（需先 docker compose up redis）")
	restoreSession := flag.String("session", "", "指定会话 ID 以恢复历史（需 --redis）")
	flag.Parse()

	if *query == "" {
		fmt.Fprintln(os.Stderr, "用法: go run ./agent-core/cmd/cli --query \"你的问题\" [--mock] [--arxiv] [--grpc] [--redis]")
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
		ds, err := llm.NewDeepSeek()
		if err != nil {
			fmt.Printf("⚠️  DeepSeek 初始化失败: %v\n", err)
			fmt.Println("   自动切换为 MockLLM 模式（使用 --mock 可跳过此提示）\n")
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
	// 2. 初始化会话存储
	// =========================================================================
	var mem memory.MemoryStore
	var existingSession string

	if *useRedis {
		rs, err := memory.NewRedisStore()
		if err != nil {
			fmt.Printf("⚠️  Redis 连接失败: %v\n", err)
			fmt.Println("   自动降级为内存存储（会话不会持久化）")
			fmt.Println("   提示: docker compose -f deploy/docker-compose.yml up -d redis\n")
			mem = memory.NewInMemoryStore()
		} else {
			mem = rs
			fmt.Println("💾 使用 Redis 持久化会话\n")
			existingSession = *restoreSession
		}
	} else {
		mem = memory.NewInMemoryStore()
	}

	// =========================================================================
	// 3. 初始化 Agent + 注册工具
	// =========================================================================
	ag := agent.New(llmClient, mem)

	if *useGrpc {
		grpcReg, err := tool.NewGrpcRegistry("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ gRPC 连接失败: %v\n", err)
			fmt.Fprintln(os.Stderr, "   请先启动 tool-service: go run ./tool-service")
			os.Exit(1)
		}
		defer grpcReg.Close()

		s := &tool.MockSearchPapers{}
		grpcReg.RegisterMeta(s.Name(), s.Description(), s.Schema())
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
		grpcReg.RegisterMeta("rag_query", "查询本地知识库中的论文片段（需先上传 PDF 入库）", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "查询问题，如 'Transformer 的注意力机制如何工作'",
				},
				"top_k": map[string]interface{}{
					"type":        "integer",
					"description": "返回结果数量，默认 5",
				},
			},
			"required": []string{"query"},
		})
		grpcReg.RegisterMeta("generate_citation", "为指定论文生成 BibTeX 引用", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"paper_id": map[string]interface{}{
					"type":        "string",
					"description": "arXiv 论文 ID，如 1706.03762",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "引用格式，默认 bibtex",
				},
			},
			"required": []string{"paper_id"},
		})

		ag.SetToolExecutor(grpcReg)
		fmt.Printf("🔗 通过 gRPC 调用 tool-service（%d 个工具）\n\n", len(grpcReg.List()))
	} else if *useArxiv {
		ag.RegisterTool(tool.NewArxivSearch())
		fmt.Println("📡 使用真实 arXiv API（本地）\n")
	} else {
		ag.RegisterTool(&tool.MockSearchPapers{})
	}

	// =========================================================================
	// 4. 创建或恢复会话
	// =========================================================================
	var sessID string
	if existingSession != "" {
		sessID = existingSession
		fmt.Printf("📋 恢复会话: %s\n", sessID)
	} else {
		var err error
		sessID, err = mem.Create("cli-user", *query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 创建会话失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("📋 会话: %s\n", sessID)
	}

	// =========================================================================
	// 5. 运行 Agent
	// =========================================================================
	fmt.Printf("📝 问题: %s\n", *query)
	fmt.Println(strings.Repeat("─", 60))

	events, err := ag.Run(ctx, sessID, *query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Agent 启动失败: %v\n", err)
		os.Exit(1)
	}

	for event := range events {
		printEvent(event)
	}

	fmt.Println(strings.Repeat("─", 60))
	fmt.Println("✅ 推理完成")
}

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
