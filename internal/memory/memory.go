package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/openclaw-alternative/goclaw/internal/config"
	"github.com/openclaw-alternative/goclaw/internal/provider"
)

// Conversation represents a persistent chat conversation
type Conversation struct {
	ID        string                 `json:"id"`
	AgentID   string                 `json:"agent_id"`
	AgentName string                 `json:"agent_name"`
	Messages  []provider.ChatMessage `json:"messages"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// EstimateTokens estimates token count from text (~4 chars per token)
func EstimateTokens(text string) int {
	return len(text) / 4
}

// EstimateHistoryTokens estimates total tokens in a message history
func EstimateHistoryTokens(messages []provider.ChatMessage) int {
	total := 0
	for _, m := range messages {
		total += EstimateTokens(m.Content) + 4 // overhead per message for role etc.
	}
	return total
}

// TrimHistory trims chat history to fit within maxTokens, keeping system prompt and most recent messages
func TrimHistory(messages []provider.ChatMessage, maxTokens int) []provider.ChatMessage {
	if maxTokens <= 0 || len(messages) <= 1 {
		return messages
	}

	// Separate system messages from chat messages
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

	// Trim oldest message pairs (user+assistant) from the beginning
	trimmed := chatMessages
	for EstimateHistoryTokens(trimmed) > remainingTokens && len(trimmed) > 2 {
		trimmed = trimmed[2:] // Remove oldest user+assistant pair
	}

	result := make([]provider.ChatMessage, 0, len(systemMessages)+len(trimmed))
	result = append(result, systemMessages...)
	result = append(result, trimmed...)
	return result
}

// --- Persistent History ---

func getHistoryDir() string {
	return filepath.Join(config.GetConfigDir(), "history")
}

// SaveConversation saves a conversation to disk
func SaveConversation(conv Conversation) error {
	dir := getHistoryDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create history dir: %w", err)
	}
	conv.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, conv.ID+".json"), data, 0644); err != nil {
		return fmt.Errorf("failed to write conversation: %w", err)
	}
	return nil
}

// LoadConversation loads a conversation from disk by ID
func LoadConversation(id string) (Conversation, error) {
	data, err := os.ReadFile(filepath.Join(getHistoryDir(), id+".json"))
	if err != nil {
		return Conversation{}, fmt.Errorf("failed to read conversation %s: %w", id, err)
	}
	var conv Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return Conversation{}, fmt.Errorf("failed to parse conversation %s: %w", id, err)
	}
	return conv, nil
}

// ListConversations lists all saved conversations, sorted by most recent
func ListConversations() ([]Conversation, error) {
	dir := getHistoryDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read history dir: %w", err)
	}

	var convs []Conversation
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var conv Conversation
		if err := json.Unmarshal(data, &conv); err != nil {
			continue
		}
		convs = append(convs, conv)
	}

	// Sort by UpdatedAt descending (most recent first)
	sort.Slice(convs, func(i, j int) bool {
		return convs[i].UpdatedAt.After(convs[j].UpdatedAt)
	})

	return convs, nil
}

// DeleteConversation deletes a saved conversation by ID
func DeleteConversation(id string) error {
	path := filepath.Join(getHistoryDir(), id+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete conversation %s: %w", id, err)
	}
	return nil
}
