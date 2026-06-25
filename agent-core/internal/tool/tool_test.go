package tool_test

import (
	"context"
	"testing"

	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/tool"
)

func TestMockSearchPapers(t *testing.T) {
	mock := &tool.MockSearchPapers{}

	if mock.Name() != "search_papers" {
		t.Errorf("expected 'search_papers', got '%s'", mock.Name())
	}

	result, err := mock.Execute(context.Background(), `{"query":"attention"}`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if len(result) < 10 {
		t.Error("result too short")
	}
}

func TestToolRegistry(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Register(&tool.MockSearchPapers{})

	// List
	defs := reg.List()
	if len(defs) != 1 {
		t.Errorf("expected 1 tool def, got %d", len(defs))
	}

	// Get
	tr, ok := reg.Get("search_papers")
	if !ok {
		t.Fatal("search_papers not found")
	}

	// Execute via registry
	result, err := reg.Execute(context.Background(), "search_papers", `{"query":"test"}`)
	if err != nil {
		t.Fatalf("Execute via registry failed: %v", err)
	}
	if result == "" {
		t.Fatal("empty result")
	}

	_ = tr // suppress unused warning
}

func TestGrpcRegistryMeta(t *testing.T) {
	// GrpcRegistry requires a running gRPC server; skip in unit test
	t.Skip("requires running tool-service gRPC server")
}
