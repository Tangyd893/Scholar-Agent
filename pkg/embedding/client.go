// Package embedding 提供文本向量化服务。
// 默认使用 OpenAI 兼容 API（text-embedding-3-small，1536 维）。
package embedding

import (
	"context"
	"fmt"
	"os"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Client 是文本向量化客户端。
type Client struct {
	api     openai.Client
	model   string
	baseURL string
}

// NewClient 从环境变量创建 Embedding 客户端。
//
// 环境变量：
//   - EMBEDDING_API_KEY（默认复用 DEEPSEEK_API_KEY）
//   - EMBEDDING_MODEL（默认 text-embedding-3-small）
//   - EMBEDDING_BASE_URL（DeepSeek 兼容则用 https://api.deepseek.com/v1）
func NewClient() (*Client, error) {
	apiKey := os.Getenv("EMBEDDING_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("EMBEDDING_API_KEY or DEEPSEEK_API_KEY not set")
	}

	model := os.Getenv("EMBEDDING_MODEL")
	if model == "" {
		model = "text-embedding-3-small"
	}

	baseURL := os.Getenv("EMBEDDING_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)

	return &Client{api: client, model: model, baseURL: baseURL}, nil
}

// Embed 将文本转换为向量。
func (c *Client) Embed(ctx context.Context, text string) ([]float64, error) {
	params := openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(text),
		},
		Model: c.model,
	}

	resp, err := c.api.Embeddings.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("embed: empty response")
	}

	// 转换 float32 → float64
	vec := make([]float64, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		vec[i] = float64(v)
	}
	return vec, nil
}
