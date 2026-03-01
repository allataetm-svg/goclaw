package agent

import (
	"fmt"
	"strings"

	"github.com/openclaw-alternative/goclaw/internal/config"
	"github.com/openclaw-alternative/goclaw/internal/provider"
)

// BuildSystemPrompt creates the full system prompt for an agent
func BuildSystemPrompt(agent config.AgentConfig) string {
	base := fmt.Sprintf("Your name is %s.", agent.Name)
	if agent.SystemPrompt != "" {
		return base + "\n" + agent.SystemPrompt
	}
	return base
}

// LoadAgent finds an agent from config and returns its provider and model name
func LoadAgent(conf config.Config, agentID string) (config.AgentConfig, provider.LLMProvider, string, error) {
	var ag config.AgentConfig
	found := false
	for _, a := range conf.Agents {
		if a.ID == agentID {
			ag = a
			found = true
			break
		}
	}
	if !found {
		return config.AgentConfig{}, nil, "", fmt.Errorf("agent not found: %s", agentID)
	}

	parts := strings.SplitN(ag.Model, ":", 2)
	if len(parts) != 2 {
		return config.AgentConfig{}, nil, "", fmt.Errorf("invalid agent model format: %s (expected provider:model)", ag.Model)
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
	return ag, prov, modName, nil
}

// AddAgent adds a new agent to the config
func AddAgent(conf *config.Config, name, systemPrompt, model string) (config.AgentConfig, error) {
	if name == "" {
		return config.AgentConfig{}, fmt.Errorf("agent name cannot be empty")
	}

	id := strings.ToLower(strings.ReplaceAll(name, " ", "_"))

	// Check for duplicate ID
	for _, a := range conf.Agents {
		if a.ID == id {
			return config.AgentConfig{}, fmt.Errorf("agent with ID '%s' already exists", id)
		}
	}

	if systemPrompt == "" {
		systemPrompt = "You are a helpful and intelligent AI assistant."
	}

	ag := config.AgentConfig{
		ID:           id,
		Name:         name,
		SystemPrompt: systemPrompt,
		Model:        model,
	}

	conf.Agents = append(conf.Agents, ag)
	if err := config.Save(*conf); err != nil {
		return config.AgentConfig{}, fmt.Errorf("failed to save config after adding agent: %w", err)
	}
	return ag, nil
}

// DeleteAgent removes an agent from the config
func DeleteAgent(conf *config.Config, agentID string) error {
	if agentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}

	found := false
	newAgents := make([]config.AgentConfig, 0, len(conf.Agents))
	for _, a := range conf.Agents {
		if a.ID == agentID {
			found = true
			continue
		}
		newAgents = append(newAgents, a)
	}

	if !found {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	if len(newAgents) == 0 {
		return fmt.Errorf("cannot delete the last remaining agent")
	}

	conf.Agents = newAgents

	// If deleted agent was the default, switch to first available
	if conf.DefaultAgent == agentID {
		conf.DefaultAgent = conf.Agents[0].ID
	}

	if err := config.Save(*conf); err != nil {
		return fmt.Errorf("failed to save config after deleting agent: %w", err)
	}
	return nil
}

// EditAgentPrompt updates the system prompt for an agent
func EditAgentPrompt(conf *config.Config, agentID, newPrompt string) error {
	for i, a := range conf.Agents {
		if a.ID == agentID {
			conf.Agents[i].SystemPrompt = newPrompt
			return config.Save(*conf)
		}
	}
	return fmt.Errorf("agent not found: %s", agentID)
}

// EditAgentModel updates the model for an agent
func EditAgentModel(conf *config.Config, agentID, newModel string) error {
	parts := strings.SplitN(newModel, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid model format: %s (expected provider:model)", newModel)
	}

	for i, a := range conf.Agents {
		if a.ID == agentID {
			conf.Agents[i].Model = newModel
			return config.Save(*conf)
		}
	}
	return fmt.Errorf("agent not found: %s", agentID)
}
