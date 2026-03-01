package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func saveAgentWorkspaceAt(baseDir string, ws AgentWorkspace) error {
	dir := filepath.Join(baseDir, ws.Config.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(ws.Config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), data, 0644); err != nil {
		return err
	}

	if ws.Soul != "" {
		if err := os.WriteFile(filepath.Join(dir, "SOUL.md"), []byte(ws.Soul), 0644); err != nil {
			return err
		}
	}

	if ws.Agent != "" {
		if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte(ws.Agent), 0644); err != nil {
			return err
		}
	}

	return nil
}

func loadAgentWorkspaceAt(baseDir string, id string) (AgentWorkspace, error) {
	dir := filepath.Join(baseDir, id)

	configData, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return AgentWorkspace{}, err
	}

	var cfg AgentConfig
	if err := json.Unmarshal(configData, &cfg); err != nil {
		return AgentWorkspace{}, err
	}

	soulData, _ := os.ReadFile(filepath.Join(dir, "SOUL.md"))
	agentData, _ := os.ReadFile(filepath.Join(dir, "AGENT.md"))

	return AgentWorkspace{
		Config: cfg,
		Soul:   string(soulData),
		Agent:  string(agentData),
	}, nil
}

func listAgentsAt(baseDir string) ([]AgentConfig, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	var agents []AgentConfig
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		configData, err := os.ReadFile(filepath.Join(baseDir, entry.Name(), "config.json"))
		if err != nil {
			continue
		}
		var cfg AgentConfig
		if err := json.Unmarshal(configData, &cfg); err == nil {
			agents = append(agents, cfg)
		}
	}
	return agents, nil
}

func TestSaveAndLoadAgentWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	ws := AgentWorkspace{
		Config: AgentConfig{
			ID:    "test_agent",
			Type:  AgentTypeMain,
			Name:  "Test Agent",
			Model: "openai:gpt-4o",
			Tools: []string{"read_file", "write_file"},
		},
		Soul:  "You are a helpful assistant.",
		Agent: "You assist with coding tasks.",
	}

	if err := saveAgentWorkspaceAt(tmpDir, ws); err != nil {
		t.Fatalf("Failed to save workspace: %v", err)
	}

	loaded, err := loadAgentWorkspaceAt(tmpDir, "test_agent")
	if err != nil {
		t.Fatalf("Failed to load workspace: %v", err)
	}

	if loaded.Config.ID != "test_agent" {
		t.Errorf("Expected ID 'test_agent', got '%s'", loaded.Config.ID)
	}
	if loaded.Config.Type != AgentTypeMain {
		t.Errorf("Expected type 'main', got '%s'", loaded.Config.Type)
	}
	if loaded.Config.Name != "Test Agent" {
		t.Errorf("Expected name 'Test Agent', got '%s'", loaded.Config.Name)
	}
	if loaded.Soul != "You are a helpful assistant." {
		t.Errorf("Unexpected soul: %s", loaded.Soul)
	}
	if loaded.Agent != "You assist with coding tasks." {
		t.Errorf("Unexpected agent: %s", loaded.Agent)
	}
}

func TestListAgentsWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	ws1 := AgentWorkspace{
		Config: AgentConfig{ID: "agent1", Type: AgentTypeMain, Name: "Agent One", Model: "openai:gpt-4o"},
		Soul:   "Soul 1",
	}
	ws2 := AgentWorkspace{
		Config: AgentConfig{ID: "agent2", Type: AgentTypeSub, Name: "Agent Two", Model: "ollama:llama3"},
		Soul:   "Soul 2",
	}

	if err := saveAgentWorkspaceAt(tmpDir, ws1); err != nil {
		t.Fatalf("Failed to save ws1: %v", err)
	}
	if err := saveAgentWorkspaceAt(tmpDir, ws2); err != nil {
		t.Fatalf("Failed to save ws2: %v", err)
	}

	agents, err := listAgentsAt(tmpDir)
	if err != nil {
		t.Fatalf("Failed to list agents: %v", err)
	}

	if len(agents) != 2 {
		t.Fatalf("Expected 2 agents, got %d", len(agents))
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	ws := AgentWorkspace{
		Config: AgentConfig{ID: "test", Name: "TestBot", Type: AgentTypeMain},
		Soul:   "Be helpful and kind.",
		Agent:  "You are a Go developer.",
	}

	prompt := BuildSystemPrompt(ws)
	if prompt == "" {
		t.Error("Expected non-empty system prompt")
	}

	tests := []struct {
		name     string
		contains string
	}{
		{"agent name", "TestBot"},
		{"soul content", "Be helpful and kind."},
		{"agent content", "You are a Go developer."},
	}

	for _, tc := range tests {
		if !stringContains(prompt, tc.contains) {
			t.Errorf("Prompt should contain %s (%s)", tc.name, tc.contains)
		}
	}
}

func TestBuildSystemPromptMinimalSubagent(t *testing.T) {
	ws := AgentWorkspace{
		Config: AgentConfig{ID: "sub1", Name: "SubBot", Type: AgentTypeSub},
		Soul:   "Run unit tests.",
	}

	prompt := BuildSystemPrompt(ws)
	if !stringContains(prompt, "SubBot") {
		t.Error("Prompt should contain agent name")
	}
	if !stringContains(prompt, "Run unit tests.") {
		t.Error("Prompt should contain soul")
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
