package memory

import (
	"fmt"
	"strings"

	"github.com/allataetm-svg/goclaw/internal/provider"
)

type CompactionConfig struct {
	Enabled            bool
	MinStepsForSummary int
	SummaryMaxTokens   int
}

var DefaultCompactionConfig = CompactionConfig{
	Enabled:            true,
	MinStepsForSummary: 10,
	SummaryMaxTokens:   500,
}

func GenerateSessionSummary(s *SessionState) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Session Summary: %s\n\n", s.Task))

	if s.Goal != "" {
		sb.WriteString(fmt.Sprintf("**Goal**: %s\n\n", s.Goal))
	}

	if len(s.Entities) > 0 {
		sb.WriteString("### Key Entities:\n")
		for k, v := range s.Entities {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
		}
		sb.WriteString("\n")
	}

	if len(s.Steps) > 0 {
		successCount := 0
		failCount := 0
		for _, step := range s.Steps {
			if step.Success {
				successCount++
			} else {
				failCount++
			}
		}
		sb.WriteString(fmt.Sprintf("### Progress: %d steps (%d successful, %d failed)\n\n",
			len(s.Steps), successCount, failCount))
	}

	return sb.String()
}

type ContextCompactor struct {
	config CompactionConfig
}

func NewContextCompactor(config CompactionConfig) *ContextCompactor {
	if !config.Enabled {
		config = DefaultCompactionConfig
	}
	return &ContextCompactor{config: config}
}

func (cc *ContextCompactor) ShouldCompact(ephemeral *Ephemeral, session *SessionState) bool {
	if !cc.config.Enabled {
		return false
	}

	msgCount := ephemeral.CountMessages()
	if msgCount < 20 {
		return false
	}

	if session != nil && len(session.Steps) >= cc.config.MinStepsForSummary {
		return true
	}

	if ephemeral.EstimateTokens() > ephemeral.maxTokens*80/100 {
		return true
	}

	return false
}

func (cc *ContextCompactor) Compact(ephemeral *Ephemeral, session *SessionState, summaryPrompt string) string {
	if session != nil && len(session.Steps) >= cc.config.MinStepsForSummary {
		summary := GenerateSessionSummary(session)

		var recentMessages []provider.ChatMessage
		msgCount := len(ephemeral.messages)
		if msgCount > 10 {
			recentMessages = ephemeral.messages[msgCount-10:]
		} else {
			recentMessages = ephemeral.messages
		}

		ephemeral.messages = []provider.ChatMessage{
			{Role: "system", Content: ephemeral.systemMsg},
			{Role: "system", Content: summary},
		}
		ephemeral.messages = append(ephemeral.messages, recentMessages...)

		return "Context compacted with session summary."
	}

	ephemeral.TrimToFit()
	return "Context trimmed to fit token limit."
}

func CompactLongHistory(messages []provider.ChatMessage, maxTokens int) ([]provider.ChatMessage, string) {
	if len(messages) <= 2 {
		return messages, ""
	}

	totalTokens := EstimateHistoryTokens(messages)
	if totalTokens <= maxTokens {
		return messages, ""
	}

	var systemMsgs []provider.ChatMessage
	var conversation []provider.ChatMessage

	for _, msg := range messages {
		if msg.Role == "system" {
			systemMsgs = append(systemMsgs, msg)
		} else {
			conversation = append(conversation, msg)
		}
	}

	if len(conversation) == 0 {
		return messages, ""
	}

	keepCount := len(conversation) / 2
	compacted := conversation[len(conversation)-keepCount:]

	summary := fmt.Sprintf("[Previous conversation with %d messages summarized]", len(conversation)-keepCount)
	compacted = append([]provider.ChatMessage{{Role: "system", Content: summary}}, compacted...)

	result := append(systemMsgs, compacted...)

	return result, fmt.Sprintf("Compacted from %d to %d messages", len(messages), len(result))
}
