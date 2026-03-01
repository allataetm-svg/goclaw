package agent

import (
	"fmt"
	"strings"
	"sync"

	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/provider"
)

// BuildSystemPrompt creates the full system prompt for an agent workspace.
func BuildSystemPrompt(ws AgentWorkspace) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Your name is %s.", ws.Config.Name))

	if ws.Soul != "" {
		parts = append(parts, ws.Soul)
	}

	if ws.Agent != "" {
		parts = append(parts, ws.Agent)
	}

	// Tool descriptions
	if len(ws.Config.Tools) > 0 {
		parts = append(parts, "## Available Tools")
		for _, toolName := range ws.Config.Tools {
			if t, ok := GetTool(toolName); ok {
				parts = append(parts, fmt.Sprintf("- **%s**: %s", t.Name(), t.Description()))
			}
		}
		parts = append(parts, "To use a tool, output strictly in this format: `CALL: ToolName({\"arg1\": \"val1\"})`.")
	}

	return strings.Join(parts, "\n\n")
}

var (
	toolRegistry = make(map[string]Tool)
	registryMu   sync.RWMutex
)

func RegisterTool(t Tool) {
	registryMu.Lock()
	defer registryMu.Unlock()
	toolRegistry[t.Name()] = t
}

func GetTool(name string) (Tool, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	t, ok := toolRegistry[name]
	return t, ok
}

// LoadAgent finds an agent workspace and returns its provider and model name.
func LoadAgent(conf config.Config, agentID string) (AgentWorkspace, provider.LLMProvider, string, error) {
	ws, err := LoadAgentWorkspace(agentID)
	if err != nil {
		return AgentWorkspace{}, nil, "", fmt.Errorf("agent not found: %s: %w", agentID, err)
	}

	parts := strings.SplitN(ws.Config.Model, ":", 2)
	if len(parts) != 2 {
		return AgentWorkspace{}, nil, "", fmt.Errorf("invalid agent model format: %s (expected provider:model)", ws.Config.Model)
	}
	provID := parts[0]
	modName := parts[1]

	var pc config.ProviderConfig
	for _, p := range conf.Providers {
		if p.ID == provID {
			pc = p
			break
		}
	}

	prov := provider.MakeProvider(pc)
	return ws, prov, modName, nil
}

// AddAgent creates a new agent workspace directory.
func AddAgent(name, model string, agentType AgentType) (AgentWorkspace, error) {
	if name == "" {
		return AgentWorkspace{}, fmt.Errorf("agent name cannot be empty")
	}

	id := strings.ToLower(strings.ReplaceAll(name, " ", "_"))

	// Check for duplicate
	existing, _ := LoadAgentWorkspace(id)
	if existing.Config.ID != "" {
		return AgentWorkspace{}, fmt.Errorf("agent with ID '%s' already exists", id)
	}

	if agentType == "" {
		agentType = AgentTypeMain
	}

	ws := AgentWorkspace{
		Config: AgentConfig{
			ID:    id,
			Type:  agentType,
			Name:  name,
			Model: model,
		},
		Soul:  "You are a helpful and intelligent AI assistant.",
		Agent: "",
	}

	if err := SaveAgentWorkspace(ws); err != nil {
		return AgentWorkspace{}, fmt.Errorf("failed to save agent workspace: %w", err)
	}
	return ws, nil
}

// DeleteAgent removes an agent workspace directory.
func DeleteAgent(agentID string) error {
	if agentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}
	return DeleteAgentWorkspace(agentID)
}

// EditAgentSoul updates the SOUL.md for an agent.
func EditAgentSoul(agentID, newSoul string) error {
	ws, err := LoadAgentWorkspace(agentID)
	if err != nil {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	ws.Soul = newSoul
	return SaveAgentWorkspace(ws)
}

// EditAgentModel updates the model for an agent.
func EditAgentModel(agentID, newModel string) error {
	parts := strings.SplitN(newModel, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid model format: %s (expected provider:model)", newModel)
	}

	ws, err := LoadAgentWorkspace(agentID)
	if err != nil {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	ws.Config.Model = newModel
	return SaveAgentWorkspace(ws)
}
