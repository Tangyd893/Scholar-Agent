package tool

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Tangyd893/Scholar-Agent/pkg/paper"
)

// arxivBaseURL 是 arXiv API 的查询端点。
const arxivBaseURL = "http://export.arxiv.org/api/query"

// ArxivSearchPapers 是 search_papers 的真实 arXiv API 实现。
// Phase 1 Day 4 替换 MockSearchPapers；Day 5 迁移至 tool-service gRPC。
type ArxivSearchPapers struct {
	client  *http.Client
	minWait time.Duration // 两次请求间最小间隔
	lastReq time.Time
}

// NewArxivSearch 创建 arXiv 搜索工具。
func NewArxivSearch() *ArxivSearchPapers {
	return &ArxivSearchPapers{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		minWait: 3 * time.Second, // arXiv 限流：至少间隔 3 秒
	}
}

func (a *ArxivSearchPapers) Name() string        { return "search_papers" }
func (a *ArxivSearchPapers) Description() string { return "按关键词搜索学术论文，返回论文列表（含标题、作者、摘要、年份）" }
func (a *ArxivSearchPapers) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "学术搜索关键词（英文），如 \"attention mechanism\"",
			},
		},
		"required": []string{"query"},
	}
}

// Execute 调用 arXiv API 搜索论文。
func (a *ArxivSearchPapers) Execute(ctx context.Context, args string) (string, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("arxiv_search: parse args: %w", err)
	}

	if params.Query == "" {
		return "", fmt.Errorf("arxiv_search: query is required")
	}

	// 限流
	a.waitRateLimit()

	papers, err := a.search(ctx, params.Query)
	if err != nil {
		return "", fmt.Errorf("arxiv_search: %w", err)
	}

	result := paper.SearchResult{
		Query:  params.Query,
		Papers: papers,
		Total:  len(papers),
		Source: "arxiv",
	}

	b, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("arxiv_search: marshal: %w", err)
	}
	return string(b), nil
}

// search 执行 arXiv API 查询并解析结果。
func (a *ArxivSearchPapers) search(ctx context.Context, query string) ([]paper.Paper, error) {
	// 构造 URL
	u, _ := url.Parse(arxivBaseURL)
	u.RawQuery = url.Values{
		"search_query": {fmt.Sprintf("all:%s", query)},
		"start":        {"0"},
		"max_results":  {"10"},
		"sortBy":       {"relevance"},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return parseAtomResponse(body)
}

// waitRateLimit 确保两次 arXiv 请求之间至少间隔 minWait。
func (a *ArxivSearchPapers) waitRateLimit() {
	if a.lastReq.IsZero() {
		a.lastReq = time.Now()
		return
	}
	elapsed := time.Since(a.lastReq)
	if elapsed < a.minWait {
		time.Sleep(a.minWait - elapsed)
	}
	a.lastReq = time.Now()
}

// =========================================================================
// Atom XML 解析（arXiv API 返回格式）
// =========================================================================

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	ID        string       `xml:"id"`
	Title     string       `xml:"title"`
	Summary   string       `xml:"summary"`
	Published string       `xml:"published"`
	Authors   []atomAuthor `xml:"author"`
	Link      []atomLink   `xml:"link"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomLink struct {
	HRef string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

// parseAtomResponse 将 arXiv Atom XML 解析为 Paper 列表。
func parseAtomResponse(body []byte) ([]paper.Paper, error) {
	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}

	papers := make([]paper.Paper, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		p := paper.Paper{
			PaperID:  extractArxivID(entry.ID),
			Title:    strings.TrimSpace(entry.Title),
			Abstract: strings.TrimSpace(entry.Summary),
			URL:      extractPdfLink(entry.Link),
		}

		// 解析年份
		if len(entry.Published) >= 4 {
			fmt.Sscanf(entry.Published[:4], "%d", &p.Year)
		}

		// 解析作者
		for _, a := range entry.Authors {
			p.Authors = append(p.Authors, a.Name)
		}

		papers = append(papers, p)
	}

	return papers, nil
}

// extractArxivID 从 arXiv URL（如 http://arxiv.org/abs/1706.03762v1）提取纯 ID。
func extractArxivID(raw string) string {
	// 取最后一个 / 之后的部分
	if idx := strings.LastIndex(raw, "/"); idx >= 0 {
		raw = raw[idx+1:]
	}
	// 去掉版本号后缀（v1, v2 等）
	if idx := strings.LastIndex(raw, "v"); idx > 0 {
		// 确保 v 后面是数字
		rest := raw[idx+1:]
		if len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
			raw = raw[:idx]
		}
	}
	return raw
}

// extractPdfLink 从 link 列表中提取 PDF 链接。
func extractPdfLink(links []atomLink) string {
	for _, l := range links {
		if l.Rel == "alternate" && l.HRef != "" {
			return l.HRef
		}
	}
	return ""
}
