package llm

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// FallbackLLM 实现 LLMClient 接口，按顺序尝试多个 LLM 提供商。
// 当一个调用失败时，自动切换到下一个。
type FallbackLLM struct {
	providers []LLMClient
	names     []string
}

// NewFallback 创建降级链 LLM 客户端。
// 按 DeepSeek → MiMo → MiniMax 顺序尝试。
func NewFallback() (LLMClient, error) {
	var providers []LLMClient
	var names []string

	// 1. DeepSeek（主力）
	ds, err := NewDeepSeek()
	if err != nil {
		slog.Warn("llm: DeepSeek unavailable", "error", err)
	} else {
		providers = append(providers, ds)
		names = append(names, "deepseek-v4-flash")
	}

	// 2. MiMo V2.5 Pro（第一降级）
	if key := os.Getenv("MIMO_API_KEY"); key != "" {
		mimo := NewOpenAICompat(
			key,
			os.Getenv("MIMO_MODEL"),
			os.Getenv("MIMO_BASE_URL"),
			"mimo",
		)
		if mimo != nil {
			providers = append(providers, mimo)
			names = append(names, "mimo-v2.5-pro")
		}
	}

	// 3. MiniMax-M3（第二降级）
	if key := os.Getenv("MINIMAX_API_KEY"); key != "" {
		mm := NewOpenAICompat(
			key,
			os.Getenv("MINIMAX_MODEL"),
			os.Getenv("MINIMAX_BASE_URL"),
			"minimax",
		)
		if mm != nil {
			providers = append(providers, mm)
			names = append(names, "minimax-m3")
		}
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no LLM provider available (set DEEPSEEK_API_KEY, MIMO_API_KEY, or MINIMAX_API_KEY)")
	}

	slog.Info("llm: fallback chain", "providers", names)
	return &FallbackLLM{providers: providers, names: names}, nil
}

func (f *FallbackLLM) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	for i, p := range f.providers {
		resp, err := p.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}
		slog.Warn("llm: provider failed, trying next", "provider", f.names[i], "error", err)
	}
	return nil, fmt.Errorf("all LLM providers exhausted")
}

// OpenAICompatProvider 是通用 OpenAI 兼容 LLM 客户端（MiMo、MiniMax 等）。
type OpenAICompatProvider struct {
	*DeepSeekProvider
}

// NewOpenAICompat 创建 OpenAI 兼容客户端。
func NewOpenAICompat(apiKey, model, baseURL, name string) *OpenAICompatProvider {
	if apiKey == "" {
		return nil
	}
	if model == "" {
		model = "default"
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	// 复用 DeepSeekProvider（本质是 OpenAI 兼容客户端）
	dp := &DeepSeekProvider{
		client: newOpenAIClient(apiKey, baseURL),
		model:  model,
	}
	return &OpenAICompatProvider{dp}
}

func newOpenAIClient(apiKey, baseURL string) openai.Client {
	return openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)
}
