// Package tool 定义工具接口与注册表。
// 每个工具通过 Tool 接口注册，ToolRegistry 负责按名称查找与执行。
package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Tangyd893/Scholar-Agent/pkg/paper"
)

// Tool 是单个工具的执行接口。
// 工具通过 ToolRegistry 注册，由 Agent Core 在 ReAct 循环中调用。
type Tool interface {
	// Name 返回工具名（与 LLM function name 一致）。
	Name() string

	// Description 返回工具描述（发送给 LLM）。
	Description() string

	// Schema 返回工具参数的 JSON Schema。
	Schema() map[string]interface{}

	// Execute 执行工具，args 为 LLM 传入的 JSON 参数字符串。
	Execute(ctx context.Context, args string) (string, error)
}

// ToolRegistry 是线程安全的工具注册表。
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry 创建空注册表。
func NewRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register 注册一个工具。同名工具会覆盖旧实现。
func (r *ToolRegistry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Get 按名称获取工具，不存在时返回 false。
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List 返回所有已注册工具的 ToolDef 列表（供 LLM 使用）。
func (r *ToolRegistry) List() []ToolDef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			Schema:      t.Schema(),
		})
	}
	return defs
}

// Execute 按名称执行工具。
func (r *ToolRegistry) Execute(ctx context.Context, name, args string) (string, error) {
	t, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return t.Execute(ctx, args)
}

// =========================================================================
// Phase 1 Mock 工具 — search_papers
// =========================================================================

// MockSearchPapers 是 search_papers 的内存 mock 实现，
// 返回固定的论文列表，不访问 arXiv API。
// Phase 1 Day 3 使用，Day 4 替换为真实 arXiv 实现。
type MockSearchPapers struct{}

func (m *MockSearchPapers) Name() string        { return "search_papers" }
func (m *MockSearchPapers) Description() string { return "按关键词搜索学术论文，返回论文列表（含标题、作者、摘要、年份）" }
func (m *MockSearchPapers) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "学术搜索关键词，如 \"attention mechanism\"",
			},
		},
		"required": []string{"query"},
	}
}

func (m *MockSearchPapers) Execute(ctx context.Context, args string) (string, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("search_papers: parse args: %w", err)
	}

	result := paper.SearchResult{
		Query:  params.Query,
		Source: "mock",
		Papers: []paper.Paper{
			{
				PaperID:  "1706.03762",
				Title:    "Attention Is All You Need",
				Authors:  []string{"Vaswani, Ashish", "Shazeer, Noam", "Parmar, Niki"},
				Year:     2017,
				Abstract: "The dominant sequence transduction models are based on complex recurrent or convolutional neural networks...",
				URL:      "https://arxiv.org/abs/1706.03762",
			},
			{
				PaperID:  "1409.0473",
				Title:    "Neural Machine Translation by Jointly Learning to Align and Translate",
				Authors:  []string{"Bahdanau, Dzmitry", "Cho, Kyunghyun", "Bengio, Yoshua"},
				Year:     2014,
				Abstract: "Neural machine translation is a recently proposed approach to machine translation...",
				URL:      "https://arxiv.org/abs/1409.0473",
			},
			{
				PaperID:  "1508.04025",
				Title:    "Effective Approaches to Attention-based Neural Machine Translation",
				Authors:  []string{"Luong, Minh-Thang", "Pham, Hieu", "Manning, Christopher D."},
				Year:     2015,
				Abstract: "An attentional mechanism has lately been used to improve neural machine translation...",
				URL:      "https://arxiv.org/abs/1508.04025",
			},
			{
				PaperID:  "1608.05859",
				Title:    "Google's Neural Machine Translation System: Bridging the Gap between Human and Machine Translation",
				Authors:  []string{"Wu, Yonghui", "Schuster, Mike", "Chen, Zhifeng"},
				Year:     2016,
				Abstract: "Neural Machine Translation (NMT) is an end-to-end learning approach...",
				URL:      "https://arxiv.org/abs/1608.05859",
			},
			{
				PaperID:  "1810.04805",
				Title:    "BERT: Pre-training of Deep Bidirectional Transformers for Language Understanding",
				Authors:  []string{"Devlin, Jacob", "Chang, Ming-Wei", "Lee, Kenton"},
				Year:     2018,
				Abstract: "We introduce a new language representation model called BERT...",
				URL:      "https://arxiv.org/abs/1810.04805",
			},
		},
		Total: 5,
	}

	b, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("search_papers: marshal result: %w", err)
	}
	return string(b), nil
}

// ToolDef 是工具的 LLM 级描述（与 llm.ToolDef 对应但独立定义，
// 避免 tool 包直接依赖 llm 包）。
type ToolDef struct {
	Name        string
	Description string
	Schema      map[string]interface{}
}
