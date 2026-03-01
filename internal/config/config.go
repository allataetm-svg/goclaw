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

type PendingPairing struct {
	ChannelID string `json:"channel_id"`
	UserID    string `json:"user_id"`
	Code      string `json:"code"`
}

func GetPairingPath() string {
	return filepath.Join(GetConfigDir(), "pairing.json")
}

func LoadPendingPairings() ([]PendingPairing, error) {
	data, err := os.ReadFile(GetPairingPath())
	if err != nil {
		if os.IsNotExist(err) {
			return []PendingPairing{}, nil
		}
		return nil, err
	}
	var pairings []PendingPairing
	if err := json.Unmarshal(data, &pairings); err != nil {
		return nil, err
	}
	return pairings, nil
}

func SavePendingPairings(pairings []PendingPairing) error {
	data, err := json.MarshalIndent(pairings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(GetPairingPath(), data, 0644)
}

func ApprovePairing(userID string, code string) error {
	pairings, err := LoadPendingPairings()
	if err != nil {
		return err
	}

	foundIdx := -1
	for i, p := range pairings {
		if p.UserID == userID && p.Code == code {
			foundIdx = i
			break
		}
	}

	if foundIdx == -1 {
		return fmt.Errorf("pairing request for user %s with code %s not found", userID, code)
	}

	// Remove from pending
	pairings = append(pairings[:foundIdx], pairings[foundIdx+1:]...)
	_ = SavePendingPairings(pairings)

	// Add to allowed in main config
	conf, err := Load()
	if err != nil {
		return err
	}

	alreadyAllowed := false
	for _, u := range conf.AllowedUsers {
		if u == userID {
			alreadyAllowed = true
			break
		}
	}
	if !alreadyAllowed {
		conf.AllowedUsers = append(conf.AllowedUsers, userID)
		return Save(conf)
	}
	return nil
}
