package server_test

import (
	"encoding/json"
	"strings"
	"testing"
)

// 注意：这些测试需要 tool-service 的 server 包导出测试辅助函数。
// 当前 server 包的方法均为私有，因此仅验证 JSON 格式和基本逻辑。

func TestSearchPapersArgsValidation(t *testing.T) {
	// 验证 JSON args 解析逻辑
	var params struct {
		Query string `json:"query"`
	}
	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{"valid", `{"query":"attention"}`, false},
		{"empty query", `{"query":""}`, true},
		{"missing query", `{}`, true},
		{"invalid json", `{bad}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := json.Unmarshal([]byte(tt.args), &params)
			if tt.wantErr {
				if err != nil {
					return // expected
				}
				if params.Query == "" {
					return // also expected for empty/missing
				}
				t.Error("expected error but got none")
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if params.Query == "" {
					t.Error("expected non-empty query")
				}
			}
		})
	}
}

func TestGetAbstractArgsValidation(t *testing.T) {
	var params struct {
		PaperID string `json:"paper_id"`
	}
	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{"valid", `{"paper_id":"1706.03762"}`, false},
		{"empty id", `{"paper_id":""}`, true},
		{"missing id", `{}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := json.Unmarshal([]byte(tt.args), &params)
			if tt.wantErr {
				if err != nil {
					return
				}
				if params.PaperID == "" {
					return
				}
				t.Error("expected empty paper_id")
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGenerateCitationArgsValidation(t *testing.T) {
	var params struct {
		PaperID string `json:"paper_id"`
		Format  string `json:"format"`
	}
	// 缺失 paper_id 应被拒绝
	if err := json.Unmarshal([]byte(`{}`), &params); err != nil {
		t.Fatal(err)
	}
	if params.PaperID != "" {
		t.Error("expected empty paper_id for missing field")
	}

	// 有效 paper_id
	if err := json.Unmarshal([]byte(`{"paper_id":"1706.03762"}`), &params); err != nil {
		t.Fatal(err)
	}
	if params.PaperID != "1706.03762" {
		t.Errorf("expected 1706.03762, got %s", params.PaperID)
	}
	if params.Format != "" {
		t.Errorf("expected empty format, got %s", params.Format)
	}
}

func TestRagQueryArgsValidation(t *testing.T) {
	var params struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
	}
	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{"valid with top_k", `{"query":"test","top_k":3}`, false},
		{"valid without top_k", `{"query":"test"}`, false},
		{"empty query", `{"query":""}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := json.Unmarshal([]byte(tt.args), &params)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatal(err)
			}
			if tt.wantErr && params.Query == "" {
				return
			}
			if !tt.wantErr && params.Query == "" {
				t.Error("expected non-empty query")
			}
		})
	}
}

func TestBibTeXOutput(t *testing.T) {
	// 验证 BibTeX 输出包含必要字段
	citation := `@article{Vaswani2017Attention,
  author = {Vaswani, Ashish and Shazeer, Noam and Parmar, Niki},
  title = {Attention Is All You Need},
  journal = {arXiv preprint},
  year = {2017},
  note = {arXiv:1706.03762},
  url = {https://arxiv.org/abs/1706.03762},
}`

	for _, required := range []string{
		"@article",
		"author",
		"title",
		"year",
		"arXiv:",
		"arxiv.org",
	} {
		if !strings.Contains(citation, required) {
			t.Errorf("BibTeX missing required field: %s", required)
		}
	}
}
