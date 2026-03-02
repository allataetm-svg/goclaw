package memory

import (
	"github.com/allataetm-svg/goclaw/internal/provider"
)

type Ephemeral struct {
	messages  []provider.ChatMessage
	systemMsg string
	maxTokens int
}

func NewEphemeral(systemPrompt string, maxTokens int) *Ephemeral {
	return &Ephemeral{
		systemMsg: systemPrompt,
		maxTokens: maxTokens,
		messages:  []provider.ChatMessage{},
	}
}

func (e *Ephemeral) SetSystemPrompt(prompt string) {
	e.systemMsg = prompt
}

func (e *Ephemeral) AddMessage(msg provider.ChatMessage) {
	e.messages = append(e.messages, msg)
}

func (e *Ephemeral) GetMessages() []provider.ChatMessage {
	var result []provider.ChatMessage
	if e.systemMsg != "" {
		result = append(result, provider.ChatMessage{
			Role:    "system",
			Content: e.systemMsg,
		})
	}
	result = append(result, e.messages...)
	return result
}

func (e *Ephemeral) GetTrimmedMessages() []provider.ChatMessage {
	msgs := e.GetMessages()
	return TrimHistory(msgs, e.maxTokens)
}

func (e *Ephemeral) Clear() {
	e.messages = []provider.ChatMessage{}
}

func (e *Ephemeral) CountMessages() int {
	return len(e.messages)
}

func (e *Ephemeral) EstimateTokens() int {
	return EstimateHistoryTokens(e.GetMessages())
}

func (e *Ephemeral) TrimToFit() {
	e.messages = TrimHistory(e.messages, e.maxTokens-len(e.systemMsg)/4)
}

func (e *Ephemeral) SetMaxTokens(max int) {
	e.maxTokens = max
}

func (e *Ephemeral) LastUserMessage() string {
	for i := len(e.messages) - 1; i >= 0; i-- {
		if e.messages[i].Role == "user" {
			return e.messages[i].Content
		}
	}
	return ""
}

func (e *Ephemeral) LastAssistantMessage() string {
	for i := len(e.messages) - 1; i >= 0; i-- {
		if e.messages[i].Role == "assistant" {
			return e.messages[i].Content
		}
	}
	return ""
}
