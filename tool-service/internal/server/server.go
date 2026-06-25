// Package server 实现 ToolService gRPC 接口。
// Phase 1 支持 search_papers 和 get_abstract 两个工具。
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Tangyd893/Scholar-Agent/pkg/arxiv"
	pb "github.com/Tangyd893/Scholar-Agent/proto/gen/tool"
)

// ToolServer 实现 proto ToolServiceServer 接口。
type ToolServer struct {
	pb.UnimplementedToolServiceServer
	arxiv *arxiv.Client
}

// New 创建 ToolService gRPC 服务端。
func New() *ToolServer {
	return &ToolServer{
		arxiv: arxiv.NewClient(),
	}
}

// Execute 执行同步工具调用。
func (s *ToolServer) Execute(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
	slog.Info("tool-service: Execute", "tool", req.ToolName)

	switch req.ToolName {
	case "search_papers":
		return s.searchPapers(ctx, req.ArgumentsJson)
	case "get_abstract":
		return s.getAbstract(ctx, req.ArgumentsJson)
	default:
		return &pb.ExecuteResponse{
			Error: fmt.Sprintf("unknown tool: %s", req.ToolName),
		}, nil
	}
}

// IngestPDF 提交 PDF 解析任务（Phase 2 实现）。
func (s *ToolServer) IngestPDF(ctx context.Context, req *pb.IngestRequest) (*pb.IngestResponse, error) {
	return &pb.IngestResponse{
		JobId: "",
	}, fmt.Errorf("IngestPDF not implemented in Phase 1")
}

// searchPapers 处理 search_papers 工具调用。
func (s *ToolServer) searchPapers(ctx context.Context, argsJSON string) (*pb.ExecuteResponse, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
		return &pb.ExecuteResponse{Error: fmt.Sprintf("parse args: %v", err)}, nil
	}
	if params.Query == "" {
		return &pb.ExecuteResponse{Error: "query is required"}, nil
	}

	result, err := s.arxiv.Search(ctx, params.Query)
	if err != nil {
		return &pb.ExecuteResponse{Error: fmt.Sprintf("arxiv search: %v", err)}, nil
	}

	b, _ := json.Marshal(result)
	return &pb.ExecuteResponse{Result: string(b)}, nil
}

// getAbstract 处理 get_abstract 工具调用。
func (s *ToolServer) getAbstract(ctx context.Context, argsJSON string) (*pb.ExecuteResponse, error) {
	var params struct {
		PaperID string `json:"paper_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
		return &pb.ExecuteResponse{Error: fmt.Sprintf("parse args: %v", err)}, nil
	}
	if params.PaperID == "" {
		return &pb.ExecuteResponse{Error: "paper_id is required"}, nil
	}

	paper, err := s.arxiv.GetAbstract(ctx, params.PaperID)
	if err != nil {
		return &pb.ExecuteResponse{Error: fmt.Sprintf("get abstract: %v", err)}, nil
	}

	b, _ := json.Marshal(paper)
	return &pb.ExecuteResponse{Result: string(b)}, nil
}
