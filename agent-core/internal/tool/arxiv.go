package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Tangyd893/Scholar-Agent/pkg/arxiv"
)

// ArxivSearchPapers 实现 tool.Tool 接口，委托给 pkg/arxiv.Client。
type ArxivSearchPapers struct {
	arxiv *arxiv.Client
}

func NewArxivSearch() *ArxivSearchPapers {
	return &ArxivSearchPapers{arxiv: arxiv.NewClient()}
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

func (a *ArxivSearchPapers) Execute(ctx context.Context, args string) (string, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("search_papers: parse args: %w", err)
	}
	if params.Query == "" {
		return "", fmt.Errorf("search_papers: query is required")
	}

	result, err := a.arxiv.Search(ctx, params.Query)
	if err != nil {
		return "", fmt.Errorf("search_papers: %w", err)
	}

	b, _ := json.Marshal(result)
	return string(b), nil
}
