// Agent Core — ReAct 推理引擎 gRPC 服务
//
// 职责：
//   - AgentCore.Run gRPC 流式接口（CLI / Gateway 通过此接口发起推理）
//   - LLM Gateway（DeepSeek FC + MiMo/MiniMax 降级）
//   - 会话管理（Redis / 内存）
//
// Phase 1 最小范围：gRPC Server 可启动，Run 返回固定事件（验证连通性）
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

	"github.com/Tangyd893/Scholar-Agent/pkg/metrics"
	pb "github.com/Tangyd893/Scholar-Agent/proto/gen/agent"
)

// agentServer 实现 proto AgentCoreServer 接口。
type agentServer struct {
	pb.UnimplementedAgentCoreServer
}

// Run 处理推理请求，返回流式 StepEvent。
// Phase 1 最小实现：返回 thought + answer 两个事件以验证 gRPC 连通性。
func (s *agentServer) Run(req *pb.RunRequest, stream pb.AgentCore_RunServer) error {
	slog.Info("agent-core: Run", "session_id", req.SessionId, "query", req.Query)

	// Thought 事件
	if err := stream.Send(&pb.StepEvent{
		Type:      "thought",
		Content:   fmt.Sprintf("收到问题: %s", req.Query),
		Step:      1,
		Timestamp: timestamppb.Now(),
	}); err != nil {
		return err
	}

	// Answer 事件
	if err := stream.Send(&pb.StepEvent{
		Type:      "answer",
		Content:   "Agent Core gRPC 服务运行正常。完整的 ReAct 推理将在后续版本中接入。",
		Step:      1,
		Timestamp: timestamppb.Now(),
	}); err != nil {
		return err
	}

	return nil
}

func main() {
	port := os.Getenv("AGENT_CORE_GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("agent-core: listen failed", "error", err)
		os.Exit(1)
	}

	// Metrics endpoint
	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9091"
	}
	go func() {
		http.Handle("/metrics", metrics.Handler())
		slog.Info("agent-core metrics", "port", metricsPort)
		http.ListenAndServe(":"+metricsPort, nil)
	}()

	s := grpc.NewServer()
	pb.RegisterAgentCoreServer(s, &agentServer{})
	reflection.Register(s)

	fmt.Printf("agent-core gRPC listening on :%s\n", port)
	slog.Info("agent-core started", "port", port)

	if err := s.Serve(lis); err != nil {
		slog.Error("agent-core: serve failed", "error", err)
		os.Exit(1)
	}
}
