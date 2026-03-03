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
		parts = append(parts, "## Persona")
		parts = append(parts, ws.Soul)
	}

	if ws.Agent != "" {
		parts = append(parts, "## Primary Mission")
		parts = append(parts, ws.Agent)
	}

	if ws.Instructions != "" {
		parts = append(parts, "## Operational Instructions")
		parts = append(parts, ws.Instructions)
	}

	// Tool descriptions
	if len(ws.Config.Tools) > 0 {
		parts = append(parts, "## Capabilities & Tools")
		parts = append(parts, "You have access to specialized tools. You MUST use them when needed to fulfill the request.")

		for _, toolName := range ws.Config.Tools {
			if t, ok := GetTool(toolName); ok {
				parts = append(parts, fmt.Sprintf("- **%s**: %s", t.Name(), t.Description()))
			}
		}

		// Specialized instruction for delegate_task if available
		hasDelegate := false
		for _, t := range ws.Config.Tools {
			if t == "delegate_task" {
				hasDelegate = true
				break
			}
		}

		if hasDelegate {
			parts = append(parts, "### Subagent Delegation")
			parts = append(parts, "You can delegate complex tasks to other specialized agents using the `delegate_task` tool.")
			if agents, err := ListAgents(); err == nil && len(agents) > 0 {
				parts = append(parts, "Available subagents for delegation:")
				for _, a := range agents {
					if a.ID != ws.Config.ID {
						parts = append(parts, fmt.Sprintf("- ID: `%s` | Name: %s | Model: %s", a.ID, a.Name, a.Model))
					}
				}
			}
		}

		parts = append(parts, "### Multi-Message & Feedback")
		parts = append(parts, "You can send multiple messages using the `reply` tool ONLY for long-running tasks that take time to complete.")
		parts = append(parts, "IMPORTANT: For simple responses (greetings, questions that don't require tools, one-off answers), send your response directly WITHOUT using the reply tool.")
		parts = append(parts, "CRITICAL: DO NOT use reply tool for: greetings, casual chat, or questions you can answer immediately. Use it ONLY when you need to inform the user about progress during a time-consuming operation.")
		parts = append(parts, "Example - DON'T: `CALL: reply({\"text\": \"Hi! How can I help?\"})`")
		parts = append(parts, "Example - DO: Just respond directly with your answer.")
		parts = append(parts, "Example - Long task: `CALL: reply({\"text\": \"Starting the download...\"})` -> (downloads file) -> `CALL: reply({\"text\": \"Download complete!\"})`")

		parts = append(parts, "### Tool Usage Protocol")
		parts = append(parts, "1. To use a tool, you MUST output ONLY the call format starting with 'CALL:'.")
		parts = append(parts, "2. DO NOT include any conversational filler, markdown code blocks (```), or pre-text.")
		parts = append(parts, "3. Arguments MUST be a valid JSON object inside the parentheses.")
		parts = append(parts, "4. If you need information from a tool, STOP directly after the call and wait for the response.")
		parts = append(parts, "5. DO NOT say 'I will use the tool' or 'Here is the result'. Just output the call.")
		parts = append(parts, "Correct Example: `CALL: ToolName({\"key\": \"value\"})`")
		parts = append(parts, "Incorrect Example: `Here is the file: CALL: ToolName(...)`")
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

	// Migration: Ensure 'reply' tool is enabled for all existing agents
	// since it's now essential for multi-message/async feedback.
	hasReply := false
	for _, t := range ws.Config.Tools {
		if t == "reply" {
			hasReply = true
			break
		}
	}
	if !hasReply {
		ws.Config.Tools = append(ws.Config.Tools, "reply")
		_ = SaveAgentWorkspace(ws)
	}

	// Migration: Ensure 'web_search' tool is enabled
	hasWebSearch := false
	for _, t := range ws.Config.Tools {
		if t == "web_search" {
			hasWebSearch = true
			break
		}
	}
	if !hasWebSearch {
		ws.Config.Tools = append(ws.Config.Tools, "web_search")
		_ = SaveAgentWorkspace(ws)
	}

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
			Tools: []string{"delegate_task", "read_file", "write_file", "shell", "reply", "web_search"},
		},
		Soul:  "You are a helpful and intelligent AI assistant.",
		Agent: "",
		Instructions: `# Agent Operational Instructions

## Session Management
- Start each session by understanding the user's goal
- Break complex tasks into manageable steps
- Keep the user informed of progress
- Ask for clarification when needed

## Error Handling
- When encountering errors, first try to understand the cause
- If a tool fails, check the error message and try an alternative approach
- Never repeat the same failing action twice
- Report errors clearly to the user

## Memory & Context
- Extract important user preferences and facts for long-term memory
- Use /memory commands to store important information
- Search existing memories before asking for repeat information

## Safety & Security
- Never execute commands that could harm the system
- Ask for confirmation before potentially destructive operations
- Do not reveal sensitive information in responses

## Best Practices
- Only use reply tool for LONG-RUNNING tasks (downloading, processing, etc.)
- For simple responses, directly without reply tool answer
- Provide concise, actionable responses
- Learn from user feedback and adjust behavior`,
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
