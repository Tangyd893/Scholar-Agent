// Package paper 定义学术论文相关的共享数据结构，
// 由 tool-service 产出，agent-core 和 gateway 消费。
// 禁止各服务复制此结构体。
package paper

// Paper 表示一篇学术论文的基本信息，
// 用于 search_papers 和 get_abstract 工具的返回。
type Paper struct {
	PaperID  string   `json:"paper_id"`  // arXiv ID（如 "1706.03762"）
	Title    string   `json:"title"`
	Authors  []string `json:"authors"`
	Year     int      `json:"year"`
	Abstract string   `json:"abstract"`
	URL      string   `json:"url,omitempty"` // arXiv 页面链接
}

// PaperChunk 表示论文的一个文本片段及其向量，
// 用于 Qdrant 向量检索（Phase 2 引入）。
type PaperChunk struct {
	ID        string                 `json:"id"`
	PaperID   string                 `json:"paper_id"`
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	Embedding []float32              `json:"embedding"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// SearchResult 是 search_papers 工具的返回结构。
type SearchResult struct {
	Query   string  `json:"query"`
	Papers  []Paper `json:"papers"`
	Total   int     `json:"total"`
	Source  string  `json:"source"` // "arxiv" | "semantic_scholar"
}
