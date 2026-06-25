package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/agent"
	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/memory"
	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/tool"
	pkgagent "github.com/Tangyd893/Scholar-Agent/pkg/agent"
)

func TestAgentMockRun(t *testing.T) {
	mockLLM := &agent.MockLLM{
		ToolName:    "search_papers",
		ToolArgs:    `{"query":"test"}`,
		FinalAnswer: "Mock answer for test",
	}

	ag := agent.New(mockLLM, memory.NewInMemoryStore())
	ag.RegisterTool(&tool.MockSearchPapers{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events, err := ag.Run(ctx, "sess_test", "test query")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	var thoughts, actions, observations, answers int
	for e := range events {
		switch e.Type {
		case pkgagent.EventThought:
			thoughts++
		case pkgagent.EventAction:
			actions++
		case pkgagent.EventObservation:
			observations++
		case pkgagent.EventAnswer:
			answers++
		case pkgagent.EventError:
			t.Errorf("unexpected error: %s", e.Content)
		}
	}

	if thoughts == 0 {
		t.Error("expected at least 1 thought event")
	}
	if actions == 0 {
		t.Error("expected at least 1 action event")
	}
	if observations == 0 {
		t.Error("expected at least 1 observation event")
	}
	if answers == 0 {
		t.Error("expected at least 1 answer event")
	}
}

func TestAgentMaxSteps(t *testing.T) {
	// MockLLM that always returns tool calls (never answers)
	infiniteMock := &agent.MockLLM{
		ToolName:    "search_papers",
		ToolArgs:    `{"query":"test"}`,
		FinalAnswer: "", // empty → returns tool call every time
	}

	ag := agent.New(infiniteMock, memory.NewInMemoryStore())
	ag.RegisterTool(&tool.MockSearchPapers{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events, err := ag.Run(ctx, "sess_maxsteps", "test")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	hasError := false
	stepCount := int32(0)
	for e := range events {
		if e.Type == pkgagent.EventError {
			hasError = true
		}
		if e.Step > stepCount {
			stepCount = e.Step
		}
	}
	if !hasError {
		t.Error("expected error event when max steps reached")
	}
	if stepCount > 5 {
		t.Errorf("expected max 5 steps, got %d", stepCount)
	}
}

func TestMockLLM(t *testing.T) {
	mock := &agent.MockLLM{
		ToolName:    "search_papers",
		ToolArgs:    `{"query":"test"}`,
		FinalAnswer: "Final answer",
	}

	// First call: tool call
	resp1, err := mock.Chat(context.Background(), nil)
	if err != nil {
		t.Fatalf("Chat 1 failed: %v", err)
	}
	if len(resp1.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(resp1.ToolCalls))
	}
	if resp1.ToolCalls[0].Name != "search_papers" {
		t.Errorf("expected search_papers, got %s", resp1.ToolCalls[0].Name)
	}

	// Second call: answer
	resp2, err := mock.Chat(context.Background(), nil)
	if err != nil {
		t.Fatalf("Chat 2 failed: %v", err)
	}
	if resp2.Content != "Final answer" {
		t.Errorf("expected 'Final answer', got '%s'", resp2.Content)
	}
}
