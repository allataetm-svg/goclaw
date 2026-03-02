package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/provider"
)

type Conversation struct {
	ID        string                 `json:"id"`
	AgentID   string                 `json:"agent_id"`
	AgentName string                 `json:"agent_name"`
	Messages  []provider.ChatMessage `json:"messages"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

func getHistoryDir() string {
	return filepath.Join(config.GetConfigDir(), "history")
}

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

	sort.Slice(convs, func(i, j int) bool {
		return convs[i].UpdatedAt.After(convs[j].UpdatedAt)
	})

	return convs, nil
}

func DeleteConversation(id string) error {
	path := filepath.Join(getHistoryDir(), id+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete conversation %s: %w", id, err)
	}
	return nil
}
