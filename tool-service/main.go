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

import "fmt"

func main() {
	fmt.Println("tool-service starting gRPC on :50052...")
	// TODO: 注册工具、初始化 arXiv client、启动 gRPC server
}
