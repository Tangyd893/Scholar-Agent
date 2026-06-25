// Package arxiv 提供 arXiv API 的纯函数客户端，
// 供 agent-core（本地工具）和 tool-service（gRPC 服务）共用。
package arxiv

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Tangyd893/Scholar-Agent/pkg/paper"
)

const baseURL = "http://export.arxiv.org/api/query"

// Client 是 arXiv API 的 HTTP 客户端。
type Client struct {
	http    *http.Client
	minWait time.Duration
	lastReq time.Time
}

// NewClient 创建 arXiv 客户端。
func NewClient() *Client {
	return &Client{
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
		minWait: 3 * time.Second,
	}
}

// Search 按关键词搜索论文。
func (c *Client) Search(ctx context.Context, query string) (*paper.SearchResult, error) {
	c.waitRateLimit()

	u, _ := url.Parse(baseURL)
	u.RawQuery = url.Values{
		"search_query": {fmt.Sprintf("all:%s", query)},
		"start":        {"0"},
		"max_results":  {"10"},
		"sortBy":       {"relevance"},
	}.Encode()

	body, err := c.doGet(ctx, u.String())
	if err != nil {
		return nil, err
	}

	papers, err := parseAtom(body)
	if err != nil {
		return nil, err
	}

	return &paper.SearchResult{
		Query:  query,
		Papers: papers,
		Total:  len(papers),
		Source: "arxiv",
	}, nil
}

// GetAbstract 按 arXiv ID 获取单篇论文摘要。
func (c *Client) GetAbstract(ctx context.Context, paperID string) (*paper.Paper, error) {
	c.waitRateLimit()

	u, _ := url.Parse(baseURL)
	u.RawQuery = url.Values{
		"id_list":     {paperID},
		"max_results": {"1"},
	}.Encode()

	body, err := c.doGet(ctx, u.String())
	if err != nil {
		return nil, err
	}

	papers, err := parseAtom(body)
	if err != nil {
		return nil, err
	}

	if len(papers) == 0 {
		return nil, fmt.Errorf("paper not found: %s", paperID)
	}

	return &papers[0], nil
}

func (c *Client) doGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) waitRateLimit() {
	if c.lastReq.IsZero() {
		c.lastReq = time.Now()
		return
	}
	if elapsed := time.Since(c.lastReq); elapsed < c.minWait {
		time.Sleep(c.minWait - elapsed)
	}
	c.lastReq = time.Now()
}

// =========================================================================
// Atom XML 解析
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

func parseAtom(body []byte) ([]paper.Paper, error) {
	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}

	papers := make([]paper.Paper, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		p := paper.Paper{
			PaperID:  extractID(entry.ID),
			Title:    strings.TrimSpace(entry.Title),
			Abstract: strings.TrimSpace(entry.Summary),
			URL:      extractLink(entry.Link),
		}
		if len(entry.Published) >= 4 {
			fmt.Sscanf(entry.Published[:4], "%d", &p.Year)
		}
		for _, a := range entry.Authors {
			p.Authors = append(p.Authors, a.Name)
		}
		papers = append(papers, p)
	}
	return papers, nil
}

func extractID(raw string) string {
	if idx := strings.LastIndex(raw, "/"); idx >= 0 {
		raw = raw[idx+1:]
	}
	if idx := strings.LastIndex(raw, "v"); idx > 0 {
		if len(raw) > idx+1 && raw[idx+1] >= '0' && raw[idx+1] <= '9' {
			raw = raw[:idx]
		}
	}
	return raw
}

func extractLink(links []atomLink) string {
	for _, l := range links {
		if l.Rel == "alternate" && l.HRef != "" {
			return l.HRef
		}
	}
	return ""
}
