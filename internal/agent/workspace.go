package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/allataetm-svg/goclaw/internal/config"
)

type AgentType string

const (
	AgentTypeMain AgentType = "main"
	AgentTypeSub  AgentType = "sub"
)

type AgentConfig struct {
	ID    string    `json:"id"`
	Type  AgentType `json:"type"`
	Name  string    `json:"name"`
	Model string    `json:"model"`
	Tools []string  `json:"tools,omitempty"`
}

type AgentWorkspace struct {
	Config       AgentConfig
	Soul         string
	Agent        string
	Instructions string
	Capabilities string
}

func getAgentsDir() string {
	return filepath.Join(config.GetConfigDir(), "agents")
}

func getAgentDir(id string) string {
	return filepath.Join(getAgentsDir(), id)
}

func ListAgents() ([]AgentConfig, error) {
	dir := getAgentsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []AgentConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read agents dir: %w", err)
	}

	var agents []AgentConfig
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		configPath := filepath.Join(dir, entry.Name(), "config.json")
		data, err := os.ReadFile(configPath)
		if err != nil {
			continue // Skip invalid directories silently for now
		}

		var cfg AgentConfig
		if err := json.Unmarshal(data, &cfg); err == nil {
			agents = append(agents, cfg)
		}
	}
	return agents, nil
}

func LoadAgentWorkspace(id string) (AgentWorkspace, error) {
	dir := getAgentDir(id)

	configData, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return AgentWorkspace{}, fmt.Errorf("agent config not found: %w", err)
	}

	var cfg AgentConfig
	if err := json.Unmarshal(configData, &cfg); err != nil {
		return AgentWorkspace{}, fmt.Errorf("invalid agent config: %w", err)
	}

	soulData, _ := os.ReadFile(filepath.Join(dir, "SOUL.md"))
	agentData, _ := os.ReadFile(filepath.Join(dir, "AGENT.md"))
	instructionsData, _ := os.ReadFile(filepath.Join(dir, "INSTRUCTIONS.md"))
	capabilitiesData, _ := os.ReadFile(filepath.Join(dir, "capabilities.yaml"))

	return AgentWorkspace{
		Config:       cfg,
		Soul:         string(soulData),
		Agent:        string(agentData),
		Instructions: string(instructionsData),
		Capabilities: string(capabilitiesData),
	}, nil
}

func SaveAgentWorkspace(workspace AgentWorkspace) error {
	dir := getAgentDir(workspace.Config.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create agent dir: %w", err)
	}

	configData, err := json.MarshalIndent(workspace.Config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), configData, 0644); err != nil {
		return fmt.Errorf("failed to write agent config: %w", err)
	}

	if workspace.Soul != "" {
		if err := os.WriteFile(filepath.Join(dir, "SOUL.md"), []byte(workspace.Soul), 0644); err != nil {
			return fmt.Errorf("failed to write SOUL.md: %w", err)
		}
	} else {
		os.Remove(filepath.Join(dir, "SOUL.md"))
	}

	if workspace.Agent != "" {
		if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte(workspace.Agent), 0644); err != nil {
			return fmt.Errorf("failed to write AGENT.md: %w", err)
		}
	} else {
		os.Remove(filepath.Join(dir, "AGENT.md"))
	}

	if workspace.Instructions != "" {
		if err := os.WriteFile(filepath.Join(dir, "INSTRUCTIONS.md"), []byte(workspace.Instructions), 0644); err != nil {
			return fmt.Errorf("failed to write INSTRUCTIONS.md: %w", err)
		}
	} else {
		os.Remove(filepath.Join(dir, "INSTRUCTIONS.md"))
	}

	if workspace.Capabilities != "" {
		if err := os.WriteFile(filepath.Join(dir, "capabilities.yaml"), []byte(workspace.Capabilities), 0644); err != nil {
			return fmt.Errorf("failed to write capabilities.yaml: %w", err)
		}
	} else {
		os.Remove(filepath.Join(dir, "capabilities.yaml"))
	}

	return nil
}

func DeleteAgentWorkspace(id string) error {
	dir := getAgentDir(id)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to delete agent workspace: %w", err)
	}
	return nil
}
