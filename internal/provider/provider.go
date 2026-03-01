package provider

import "context"

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StreamChunk represents a chunk of streamed response
type StreamChunk struct {
	Text  string
	Done  bool
	Error error
}

// LLMProvider defines the interface for all LLM providers
type LLMProvider interface {
	Query(ctx context.Context, model string, messages []ChatMessage) (string, error)
	QueryStream(ctx context.Context, model string, messages []ChatMessage, ch chan<- StreamChunk)
	FetchModels() ([]string, error)
	ID() string
	Name() string
}
