package memory

import (
	"github.com/allataetm-svg/goclaw/internal/provider"
)

func EstimateTokens(text string) int {
	return len(text) / 4
}

func EstimateHistoryTokens(messages []provider.ChatMessage) int {
	total := 0
	for _, m := range messages {
		total += EstimateTokens(m.Content) + 4
	}
	return total
}

func TrimHistory(messages []provider.ChatMessage, maxTokens int) []provider.ChatMessage {
	if maxTokens <= 0 || len(messages) <= 1 {
		return messages
	}

	var systemMessages []provider.ChatMessage
	var chatMessages []provider.ChatMessage

	for _, m := range messages {
		if m.Role == "system" {
			systemMessages = append(systemMessages, m)
		} else {
			chatMessages = append(chatMessages, m)
		}
	}

	systemTokens := EstimateHistoryTokens(systemMessages)
	remainingTokens := maxTokens - systemTokens

	if remainingTokens <= 0 {
		return systemMessages
	}

	trimmed := chatMessages
	for EstimateHistoryTokens(trimmed) > remainingTokens && len(trimmed) > 2 {
		trimmed = trimmed[2:]
	}

	result := make([]provider.ChatMessage, 0, len(systemMessages)+len(trimmed))
	result = append(result, systemMessages...)
	result = append(result, trimmed...)
	return result
}
