// Package server 实现 ToolService gRPC 接口。
// Phase 2 新增 rag_query 和 generate_citation 工具。
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Tangyd893/Scholar-Agent/pkg/arxiv"
	"github.com/Tangyd893/Scholar-Agent/pkg/embedding"
	"github.com/Tangyd893/Scholar-Agent/pkg/paper"
	"github.com/Tangyd893/Scholar-Agent/pkg/qdrant"
	pb "github.com/Tangyd893/Scholar-Agent/proto/gen/tool"
)

const (
	qdrantCollection = "papers"
	embeddingDim     = 1536 // text-embedding-3-small
	defaultTopK      = 5
)

// ToolServer 实现 proto ToolServiceServer 接口。
type ToolServer struct {
	pb.UnimplementedToolServiceServer
	arxiv   *arxiv.Client
	embed   *embedding.Client
	qdrant  *qdrant.Client
	mqConn  *amqp.Connection
	mqCh    *amqp.Channel
}

const mqQueue = "pdf.parse"

// New 创建 ToolService gRPC 服务端。
func New() *ToolServer {
	s := &ToolServer{
		arxiv:  arxiv.NewClient(),
		embed:  nil,
		qdrant: qdrant.NewClient(qdrantCollection),
	}

	// 尝试连接 RabbitMQ（非致命失败）
	s.connectMQ()
	return s
}

func (s *ToolServer) connectMQ() {
	mqURL := os.Getenv("RABBITMQ_URL")
	if mqURL == "" {
		mqURL = "amqp://guest:guest@localhost:5672/"
	}

	conn, err := amqp.Dial(mqURL)
	if err != nil {
		slog.Warn("tool-service: RabbitMQ not available, IngestPDF disabled", "error", err)
		return
	}

	ch, err := conn.Channel()
	if err != nil {
		slog.Warn("tool-service: failed to open channel", "error", err)
		return
	}

	// 声明队列
	_, err = ch.QueueDeclare(mqQueue, true, false, false, false, nil)
	if err != nil {
		slog.Warn("tool-service: failed to declare queue", "error", err)
		return
	}

	s.mqConn = conn
	s.mqCh = ch
	slog.Info("tool-service: RabbitMQ connected", "queue", mqQueue)
}

// initEmbed 延迟初始化 embedding 客户端。
func (s *ToolServer) initEmbed() error {
	if s.embed != nil {
		return nil
	}
	c, err := embedding.NewClient()
	if err != nil {
		return err
	}
	s.embed = c

	// 确保 Qdrant 集合存在
	return s.qdrant.EnsureCollection(context.Background(), embeddingDim)
}

// Execute 执行同步工具调用。
func (s *ToolServer) Execute(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
	slog.Info("tool-service: Execute", "tool", req.ToolName)

	switch req.ToolName {
	case "search_papers":
		return s.searchPapers(ctx, req.ArgumentsJson)
	case "get_abstract":
		return s.getAbstract(ctx, req.ArgumentsJson)
	case "rag_query":
		return s.ragQuery(ctx, req.ArgumentsJson)
	case "generate_citation":
		return s.generateCitation(ctx, req.ArgumentsJson)
	case "parse_pdf":
		return s.parsePDF(ctx, req.ArgumentsJson)
	default:
		return &pb.ExecuteResponse{Error: fmt.Sprintf("unknown tool: %s", req.ToolName)}, nil
	}
}

// IngestPDF 提交 PDF 解析任务，发布到 RabbitMQ。
func (s *ToolServer) IngestPDF(ctx context.Context, req *pb.IngestRequest) (*pb.IngestResponse, error) {
	if s.mqCh == nil {
		return &pb.IngestResponse{JobId: ""}, fmt.Errorf("RabbitMQ not available")
	}

	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())

	msg := map[string]interface{}{
		"job_id":     jobID,
		"file_id":    req.FileId,
		"session_id": req.SessionId,
	}
	body, _ := json.Marshal(msg)

	err := s.mqCh.PublishWithContext(ctx, "", mqQueue, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		return &pb.IngestResponse{JobId: ""}, fmt.Errorf("publish to MQ: %w", err)
	}

	slog.Info("tool-service: IngestPDF published", "job_id", jobID)
	return &pb.IngestResponse{JobId: jobID}, nil
}

// parsePDF 工具（CLI 触发 PDF 解析）。
func (s *ToolServer) parsePDF(ctx context.Context, argsJSON string) (*pb.ExecuteResponse, error) {
	var params struct {
		FileID    string `json:"file_id"`
		SessionID string `json:"session_id"`
	}
	json.Unmarshal([]byte(argsJSON), &params)

	resp, err := s.IngestPDF(ctx, &pb.IngestRequest{
		FileId:    params.FileID,
		SessionId: params.SessionID,
	})
	if err != nil {
		return &pb.ExecuteResponse{Error: err.Error()}, nil
	}

	b, _ := json.Marshal(map[string]string{"job_id": resp.JobId, "status": "pending"})
	return &pb.ExecuteResponse{Result: string(b)}, nil
}

// =========================================================================
// search_papers
// =========================================================================

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

// =========================================================================
// get_abstract
// =========================================================================

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

// =========================================================================
// rag_query（Phase 2）
// =========================================================================

func (s *ToolServer) ragQuery(ctx context.Context, argsJSON string) (*pb.ExecuteResponse, error) {
	if err := s.initEmbed(); err != nil {
		return &pb.ExecuteResponse{Error: fmt.Sprintf("embedding init: %v", err)}, nil
	}

	var params struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
		return &pb.ExecuteResponse{Error: fmt.Sprintf("parse args: %v", err)}, nil
	}
	if params.Query == "" {
		return &pb.ExecuteResponse{Error: "query is required"}, nil
	}
	if params.TopK <= 0 {
		params.TopK = defaultTopK
	}

	// 向量化查询
	vec, err := s.embed.Embed(ctx, params.Query)
	if err != nil {
		return &pb.ExecuteResponse{Error: fmt.Sprintf("embed query: %v", err)}, nil
	}

	// Qdrant 检索
	results, err := s.qdrant.Search(ctx, vec, params.TopK)
	if err != nil {
		return &pb.ExecuteResponse{Error: fmt.Sprintf("qdrant search: %v", err)}, nil
	}

	b, _ := json.Marshal(map[string]interface{}{
		"query":   params.Query,
		"results": results,
		"total":   len(results),
	})
	return &pb.ExecuteResponse{Result: string(b)}, nil
}

// =========================================================================
// generate_citation（Phase 2）
// =========================================================================

func (s *ToolServer) generateCitation(ctx context.Context, argsJSON string) (*pb.ExecuteResponse, error) {
	var params struct {
		PaperID string `json:"paper_id"`
		Format  string `json:"format"` // "bibtex"（默认）
	}
	if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
		return &pb.ExecuteResponse{Error: fmt.Sprintf("parse args: %v", err)}, nil
	}
	if params.PaperID == "" {
		return &pb.ExecuteResponse{Error: "paper_id is required"}, nil
	}
	if params.Format == "" {
		params.Format = "bibtex"
	}

	// 先从 arXiv 获取论文元数据
	paper, err := s.arxiv.GetAbstract(ctx, params.PaperID)
	if err != nil {
		return &pb.ExecuteResponse{Error: fmt.Sprintf("get paper: %v", err)}, nil
	}

	citation := buildBibTeX(paper)
	return &pb.ExecuteResponse{Result: citation}, nil
}

// buildBibTeX 根据论文元数据构造 BibTeX 条目。
func buildBibTeX(p *paper.Paper) string {
	// 构造 cite key: 第一作者姓氏 + 年份 + 标题首词
	citeKey := fmt.Sprintf("%s%d%s",
		safeAuthorLastName(p.Authors),
		p.Year,
		safeFirstWord(p.Title),
	)

	return fmt.Sprintf(`@article{%s,
  author = {%s},
  title = {%s},
  journal = {arXiv preprint},
  year = {%d},
  note = {arXiv:%s},
  url = {%s},
}`, citeKey, joinAuthors(p.Authors), p.Title, p.Year, p.PaperID, p.URL)
}

func safeAuthorLastName(authors []string) string {
	if len(authors) == 0 {
		return "Unknown"
	}
	// 取第一作者逗号前的部分
	name := authors[0]
	for i, c := range name {
		if c == ',' {
			return name[:i]
		}
	}
	return name
}

func safeFirstWord(title string) string {
	for i, c := range title {
		if c == ' ' {
			return title[:i]
		}
	}
	if len(title) > 10 {
		return title[:10]
	}
	return title
}

func joinAuthors(authors []string) string {
	s := ""
	for i, a := range authors {
		if i > 0 {
			s += " and "
		}
		s += a
	}
	return s
}
