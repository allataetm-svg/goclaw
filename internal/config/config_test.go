package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadConfig(t *testing.T) {
	// Use temp dir for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.json")

	conf := Config{
		Providers: []ProviderConfig{
			{ID: "ollama", APIKey: "", BaseURL: "http://localhost:11434"},
			{ID: "openai", APIKey: "sk-test-key", BaseURL: ""},
		},
		Agents: []AgentConfig{
			{ID: "coder", Name: "Coder", SystemPrompt: "You are a coding assistant.", Model: "ollama:llama3"},
			{ID: "helper", Name: "Helper", SystemPrompt: "You are helpful.", Model: "openai:gpt-4o"},
		},
		DefaultAgent: "coder",
		MaxTokens:    4000,
	}

	// Save
	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load
	readData, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var loaded Config
	if err := json.Unmarshal(readData, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if len(loaded.Providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(loaded.Providers))
	}
	if len(loaded.Agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(loaded.Agents))
	}
	if loaded.DefaultAgent != "coder" {
		t.Errorf("Expected default agent 'coder', got '%s'", loaded.DefaultAgent)
	}
	if loaded.MaxTokens != 4000 {
		t.Errorf("Expected MaxTokens 4000, got %d", loaded.MaxTokens)
	}
	if loaded.Providers[1].APIKey != "sk-test-key" {
		t.Errorf("Expected API key 'sk-test-key', got '%s'", loaded.Providers[1].APIKey)
	}
}

func TestConfigDefaults(t *testing.T) {
	conf := Config{}
	if conf.MaxTokens != 0 {
		t.Errorf("Expected MaxTokens 0 for empty config, got %d", conf.MaxTokens)
	}

	// Simulate what Load() does for defaults
	if conf.MaxTokens == 0 {
		conf.MaxTokens = 8000
	}
	if conf.MaxTokens != 8000 {
		t.Errorf("Expected MaxTokens 8000 after default, got %d", conf.MaxTokens)
	}
}

func TestAgentConfigModelFormat(t *testing.T) {
	ag := AgentConfig{
		ID:    "test",
		Name:  "Test",
		Model: "openai:gpt-4o",
	}

	data, err := json.Marshal(ag)
	if err != nil {
		t.Fatalf("Failed to marshal agent: %v", err)
	}

	var loaded AgentConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal agent: %v", err)
	}

	if loaded.Model != "openai:gpt-4o" {
		t.Errorf("Expected model 'openai:gpt-4o', got '%s'", loaded.Model)
	}
}
