// Agent Core — ReAct 推理引擎 gRPC 服务
//
// 职责：
//   - AgentCore.Run gRPC 流式接口
//   - 真实 ReAct 推理（DeepSeek LLM + gRPC Tool Service + Redis）
//   - LLM 降级（DeepSeek → MockLLM）
package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"

	coreagent "github.com/Tangyd893/Scholar-Agent/agent-core/internal/agent"
	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/llm"
	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/memory"
	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/tool"
	"github.com/Tangyd893/Scholar-Agent/pkg/metrics"
	pb "github.com/Tangyd893/Scholar-Agent/proto/gen/agent"
)

type agentServer struct {
	pb.UnimplementedAgentCoreServer
	agent *coreagent.Agent
}

// Run 处理推理请求，返回流式 StepEvent。
func (s *agentServer) Run(req *pb.RunRequest, stream pb.AgentCore_RunServer) error {
	slog.Info("agent-core: Run", "session_id", req.SessionId, "query", req.Query)

	events, err := s.agent.Run(stream.Context(), req.SessionId, req.Query)
	if err != nil {
		return err
	}

	stepCount := int32(0)
	for event := range events {
		stepCount++
		if err := stream.Send(&pb.StepEvent{
			Type:         string(event.Type),
			Content:      event.Content,
			Step:         event.Step,
			ToolArgsJson: event.ToolArgsJSON,
			Timestamp:    timestamppb.New(event.Timestamp),
		}); err != nil {
			return err
		}
	}

	metrics.AgentStepsPerQuery.Observe(float64(stepCount))
	return nil
}

func main() {
	port := os.Getenv("AGENT_CORE_GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	// =========================================================================
	// 初始化 LLM
	// =========================================================================
	var llmClient llm.LLMClient
	ds, err := llm.NewDeepSeek()
	if err != nil {
		slog.Warn("agent-core: DeepSeek unavailable, using MockLLM", "error", err)
		llmClient = &coreagent.MockLLM{
			ToolName:    "search_papers",
			ToolArgs:    `{"query":"attention mechanism"}`,
			FinalAnswer: "Agent Core 使用 MockLLM 模式运行。设置 DEEPSEEK_API_KEY 以启用真实推理。",
		}
	} else {
		llmClient = ds
		slog.Info("agent-core: using DeepSeek", "model", ds.Model())
	}

	// =========================================================================
	// 初始化 MemoryStore
	// =========================================================================
	var mem memory.MemoryStore
	rs, err := memory.NewRedisStore()
	if err != nil {
		slog.Warn("agent-core: Redis unavailable, using InMemoryStore", "error", err)
		mem = memory.NewInMemoryStore()
	} else {
		mem = rs
		slog.Info("agent-core: using Redis session store")
	}

	// =========================================================================
	// 初始化 Agent
	// =========================================================================
	ag := coreagent.New(llmClient, mem)

	// 尝试连接 tool-service gRPC
	grpcReg, err := tool.NewGrpcRegistry("")
	if err != nil {
		slog.Warn("agent-core: tool-service gRPC unavailable, using local mock", "error", err)
		ag.RegisterTool(&tool.MockSearchPapers{})
	} else {
		defer grpcReg.Close()
		registerGRPCTools(grpcReg)
		ag.SetToolExecutor(grpcReg)
		slog.Info("agent-core: using gRPC tool-service")
	}

	// =========================================================================
	// Metrics + gRPC
	// =========================================================================
	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9091"
	}
	go func() {
		http.Handle("/metrics", metrics.Handler())
		slog.Info("agent-core metrics", "port", metricsPort)
		http.ListenAndServe(":"+metricsPort, nil)
	}()

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("agent-core: listen failed", "error", err)
		os.Exit(1)
	}

	s := grpc.NewServer()
	pb.RegisterAgentCoreServer(s, &agentServer{agent: ag})
	reflection.Register(s)

	fmt.Printf("agent-core gRPC listening on :%s\n", port)
	slog.Info("agent-core started", "port", port)

	if err := s.Serve(lis); err != nil {
		slog.Error("agent-core: serve failed", "error", err)
		os.Exit(1)
	}
}

func registerGRPCTools(reg *tool.GrpcRegistry) {
	s := &tool.MockSearchPapers{}
	reg.RegisterMeta(s.Name(), s.Description(), s.Schema())
	reg.RegisterMeta("get_abstract", "按 arXiv paper_id 获取论文完整摘要", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"paper_id": map[string]interface{}{"type": "string", "description": "arXiv 论文 ID"},
		},
		"required": []string{"paper_id"},
	})
	reg.RegisterMeta("rag_query", "查询本地知识库中的论文片段", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{"type": "string", "description": "查询问题"},
			"top_k": map[string]interface{}{"type": "integer", "description": "返回数量，默认 5"},
		},
		"required": []string{"query"},
	})
	reg.RegisterMeta("generate_citation", "为指定论文生成 BibTeX 引用", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"paper_id": map[string]interface{}{"type": "string", "description": "arXiv 论文 ID"},
		},
		"required": []string{"paper_id"},
	})
	reg.RegisterMeta("parse_pdf", "提交 PDF 文件进行异步解析", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_id": map[string]interface{}{"type": "string", "description": "PDF 文件标识"},
		},
		"required": []string{"file_id"},
	})
}
