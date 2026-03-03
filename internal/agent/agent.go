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

	parts = append(parts, fmt.Sprintf("# OVERVIEW.md\nYour name is %s.", ws.Config.Name))

	if ws.Soul != "" {
		parts = append(parts, "# SOUL.md\n"+ws.Soul)
	}

	if ws.Agent != "" {
		parts = append(parts, "# AGENT.md\n"+ws.Agent)
	}

	if ws.Instructions != "" {
		parts = append(parts, "# INSTRUCTIONS.md\n"+ws.Instructions)
	}

	// Tool descriptions
	if len(ws.Config.Tools) > 0 {
		toolList := "# CAPABILITIES.md\nYou have access to the following tools:\n\n"
		for _, toolName := range ws.Config.Tools {
			if t, ok := GetTool(toolName); ok {
				toolList += fmt.Sprintf("- **%s**: %s\n", t.Name(), t.Description())
			}
		}
		parts = append(parts, toolList)

		// Subagent info for delegate_task
		for _, t := range ws.Config.Tools {
			if t == "delegate_task" {
				if agents, err := ListAgents(); err == nil && len(agents) > 0 {
					agentList := "# AGENTS.md\nAvailable subagents:\n"
					for _, a := range agents {
						if a.ID != ws.Config.ID {
							agentList += fmt.Sprintf("- `%s` (%s, %s)\n", a.ID, a.Name, a.Model)
						}
					}
					parts = append(parts, agentList)
				}
				break
			}
		}

		parts = append(parts, `## 🤖 CORE DIRECTIVE & BEHAVIOR
You are an advanced, autonomous AI Agent operating within the GoClaw framework.
You are capable of using tools to interact with the system, internet, and users.

CRITICAL LANGUAGE RULES:
- You MUST speak naturally, fluently, and correctly in the user's language (e.g., flawless Turkish).
- NEVER produce cut-off, incomplete words (e.g. write "hoşbuldum", NEVER "hoşbul").
- Pay close attention to grammar, spelling, and character encoding.

## 🛠️ TOOL USAGE PROTOCOL
You must use your tools to accomplish tasks. To use a tool, output exactly ONE call in this format at the very end of your response:

CALL: tool_name({"arg1": "value1", "arg2": "value2"})

STRICT RULES FOR TOOLS:
1. You can freely write your thoughts or responses to the user *before* the CALL line. The user WILL see the text you write before the CALL line.
2. The CALL line MUST be the absolute last line of your output.
3. The arguments MUST be perfectly formatted, valid JSON.
4. DO NOT wrap the CALL line in markdown code blocks (triple backticks).
5. Once you output a CALL line, STOP immediately. Do NOT invent the tool's result. Wait for the system to inject the [Tool Result: ...] into the context.
6. If the tool result is long, analyze it silently and ONLY summarize the important parts when replying to the user.
7. NEVER repeat the same conversational message twice. Keep track of what you have already told the user.`)
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
			Tools: []string{"delegate_task", "read_file", "write_file", "shell", "reply", "web_search", "web_fetch", "scheduler", "heartbeat", "skills", "sessions", "secrets"},
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
- Use reply tool for all messages to users
- Send complete information in ONE reply - never split messages
- Never repeat information from previous messages
- Be concise and direct`,
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
