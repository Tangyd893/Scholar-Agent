package memory_test

import (
	"testing"

	"github.com/Tangyd893/Scholar-Agent/agent-core/internal/memory"
	pkgagent "github.com/Tangyd893/Scholar-Agent/pkg/agent"
)

func TestInMemoryStore(t *testing.T) {
	store := memory.NewInMemoryStore()

	// Create session
	sid, err := store.Create("test-user", "test session")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if sid == "" {
		t.Fatal("expected non-empty session ID")
	}

	// Append message
	msg := pkgagent.Message{Role: "user", Content: "hello"}
	if err := store.Append(sid, msg); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Get history
	history, err := store.GetHistory(sid, 10)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 message, got %d", len(history))
	}
	if history[0].Content != "hello" {
		t.Errorf("expected 'hello', got '%s'", history[0].Content)
	}

	// Non-existent session
	_, err = store.GetHistory("nonexistent", 10)
	if err == nil {
		t.Error("expected error for non-existent session")
	}
}
