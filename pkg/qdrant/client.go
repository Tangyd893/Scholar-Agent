// Package qdrant 提供 Qdrant 向量数据库的 REST 客户端。
package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Client 是 Qdrant 的 REST API 客户端。
type Client struct {
	baseURL    string
	http       *http.Client
	collection string
}

// Point 是 Qdrant 中的一个向量点（文档片段）。
type Point struct {
	ID      string                 `json:"id"`
	Vector  []float64              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

// SearchResult 是检索返回的匹配结果。
type SearchResult struct {
	ID      string                 `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

// NewClient 从环境变量创建 Qdrant 客户端。
func NewClient(collection string) *Client {
	baseURL := os.Getenv("QDRANT_URL")
	if baseURL == "" {
		baseURL = "http://localhost:6333"
	}

	return &Client{
		baseURL:    baseURL,
		http:       &http.Client{},
		collection: collection,
	}
}

// EnsureCollection 确保集合存在（不存在则创建）。
// vectorSize 为向量维度（text-embedding-3-small = 1536）。
func (c *Client) EnsureCollection(ctx context.Context, vectorSize int) error {
	url := fmt.Sprintf("%s/collections/%s", c.baseURL, c.collection)

	// 先检查是否存在
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("check collection: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode == 200 {
		return nil
	}

	// 创建集合
	body := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}
	b, _ := json.Marshal(body)

	req, _ = http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err = c.http.Do(req)
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create collection failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Upsert 批量插入或更新向量点。
func (c *Client) Upsert(ctx context.Context, points []Point) error {
	url := fmt.Sprintf("%s/collections/%s/points", c.baseURL, c.collection)

	body := map[string]interface{}{
		"points": points,
	}
	b, _ := json.Marshal(body)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("upsert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upsert failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Search 向量相似度检索，返回 Top-K 结果。
func (c *Client) Search(ctx context.Context, vector []float64, topK int) ([]SearchResult, error) {
	url := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, c.collection)

	body := map[string]interface{}{
		"vector": vector,
		"limit":  topK,
		"with_payload": true,
	}
	b, _ := json.Marshal(body)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []SearchResult `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode search result: %w", err)
	}

	return result.Result, nil
}
