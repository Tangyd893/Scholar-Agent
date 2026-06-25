// Tool Service — 工具注册与执行
//
// 职责：
//   - 工具注册表（search_papers / get_abstract / rag_query / generate_citation）
//   - arXiv API 调用（主）/ Semantic Scholar（备）
//   - gRPC Server（ToolService.Execute / IngestPDF）
//   - PDF 上传 → RabbitMQ 发布（Phase 2）
//
// Phase 1 最小范围：search_papers + get_abstract
package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/Tangyd893/Scholar-Agent/proto/gen/tool"
	"github.com/Tangyd893/Scholar-Agent/tool-service/internal/server"
)

func main() {
	port := os.Getenv("TOOL_SERVICE_GRPC_PORT")
	if port == "" {
		port = "50052"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	s := grpc.NewServer()
	pb.RegisterToolServiceServer(s, server.New())

	// 注册反射服务，方便 grpcurl 调试
	reflection.Register(s)

	fmt.Printf("tool-service gRPC listening on :%s\n", port)
	slog.Info("tool-service started", "port", port)

	if err := s.Serve(lis); err != nil {
		slog.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}
