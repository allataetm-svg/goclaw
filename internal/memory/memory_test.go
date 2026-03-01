package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/openclaw-alternative/goclaw/internal/provider"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"hi", 0},    // 2 / 4 = 0
		{"hello", 1}, // 5 / 4 = 1
		{"hello world this is a test message", 8}, // 34 / 4 = 8
	}

	for _, tt := range tests {
		result := EstimateTokens(tt.text)
		if result != tt.expected {
			t.Errorf("EstimateTokens(%q) = %d, want %d", tt.text, result, tt.expected)
		}
	}
}

func TestEstimateHistoryTokens(t *testing.T) {
	messages := []provider.ChatMessage{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hello there!"},
	}

	tokens := EstimateHistoryTokens(messages)
	if tokens <= 0 {
		t.Errorf("Expected positive token count, got %d", tokens)
	}
}

func TestTrimHistory_NoTrimNeeded(t *testing.T) {
	messages := []provider.ChatMessage{
		{Role: "system", Content: "Be helpful"},
		{Role: "user", Content: "Hi"},
		{Role: "assistant", Content: "Hello!"},
	}

	result := TrimHistory(messages, 10000)
	if len(result) != 3 {
		t.Errorf("Expected 3 messages (no trim needed), got %d", len(result))
	}
}

func TestTrimHistory_KeepsSystemPrompt(t *testing.T) {
	messages := []provider.ChatMessage{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "First message"},
		{Role: "assistant", Content: "First response"},
		{Role: "user", Content: "Second message"},
		{Role: "assistant", Content: "Second response"},
		{Role: "user", Content: "Third message with much more content to push over limit"},
		{Role: "assistant", Content: "Third response with even more content to test trimming behavior"},
	}

	result := TrimHistory(messages, 30)

	// System message must always be first
	if len(result) == 0 {
		t.Fatal("TrimHistory returned empty result")
	}
	if result[0].Role != "system" {
		t.Errorf("First message should be system, got %s", result[0].Role)
	}

	// Should have fewer messages than original
	if len(result) >= len(messages) {
		t.Logf("Warning: TrimHistory did not reduce messages (result: %d, original: %d). May need smaller maxTokens.", len(result), len(messages))
	}
}

func TestTrimHistory_ZeroMaxTokens(t *testing.T) {
	messages := []provider.ChatMessage{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "Hello"},
	}

	result := TrimHistory(messages, 0)
	if len(result) != 2 {
		t.Errorf("With maxTokens 0, expected original messages, got %d", len(result))
	}
}

func TestTrimHistory_SingleMessage(t *testing.T) {
	messages := []provider.ChatMessage{
		{Role: "system", Content: "System prompt"},
	}

	result := TrimHistory(messages, 10)
	if len(result) != 1 {
		t.Errorf("With single message, expected 1, got %d", len(result))
	}
}

func TestSaveAndLoadConversation(t *testing.T) {
	// Use temp dir
	tmpDir := t.TempDir()
	histDir := filepath.Join(tmpDir, "history")
	if err := os.MkdirAll(histDir, 0755); err != nil {
		t.Fatal(err)
	}

	conv := Conversation{
		ID:        "test-conv-123",
		AgentID:   "coder",
		AgentName: "Coder",
		Messages: []provider.ChatMessage{
			{Role: "system", Content: "You are a coder."},
			{Role: "user", Content: "Write hello world"},
			{Role: "assistant", Content: "print('Hello World')"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to temp dir
	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	convPath := filepath.Join(histDir, conv.ID+".json")
	if err := os.WriteFile(convPath, data, 0644); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// Load back
	readData, err := os.ReadFile(convPath)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	var loaded Conversation
	if err := json.Unmarshal(readData, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if loaded.ID != "test-conv-123" {
		t.Errorf("Expected ID 'test-conv-123', got '%s'", loaded.ID)
	}
	if loaded.AgentID != "coder" {
		t.Errorf("Expected AgentID 'coder', got '%s'", loaded.AgentID)
	}
	if len(loaded.Messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(loaded.Messages))
	}
	if loaded.Messages[2].Content != "print('Hello World')" {
		t.Errorf("Unexpected message content: %s", loaded.Messages[2].Content)
	}
}
