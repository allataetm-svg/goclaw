package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ProviderConfig struct {
	ID      string `json:"id"`
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url,omitempty"`
}

type ChannelConfig struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"` // e.g. "telegram", "whatsapp", "cli"
	Name     string            `json:"name"`
	AgentID  string            `json:"agent_id,omitempty"` // Default agent for this channel
	Settings map[string]string `json:"settings"`           // Token, URL, etc.
}

type Config struct {
	Providers      []ProviderConfig `json:"providers"`
	Channels       []ChannelConfig  `json:"channels"`
	DefaultAgent   string           `json:"default_agent"`
	MaxTokens      int              `json:"max_tokens,omitempty"`
	PairingEnabled bool             `json:"pairing_enabled,omitempty"`
	PairingCode    string           `json:"pairing_code,omitempty"`
	AllowedUsers   []string         `json:"allowed_users,omitempty"` // Whitelist of FromIDs
}

func GetConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".goclaw")
}

func GetConfigPath() string {
	configDir := GetConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create config dir: %v\n", err)
	}
	return filepath.Join(configDir, "config.json")
}

func Load() (Config, error) {
	data, err := os.ReadFile(GetConfigPath())
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config: %w", err)
	}
	var conf Config
	if err := json.Unmarshal(data, &conf); err != nil {
		return Config{}, fmt.Errorf("failed to parse config: %w", err)
	}
	if conf.MaxTokens == 0 {
		conf.MaxTokens = 8000
	}
	return conf, nil
}

func Save(conf Config) error {
	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(GetConfigPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}
