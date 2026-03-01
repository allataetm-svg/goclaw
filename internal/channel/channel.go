package channel

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/allataetm-svg/goclaw/internal/agent"
	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/provider"
)

// Message represents a message from/to a channel
type Message struct {
	FromID    string
	ToID      string
	Text      string
	ChannelID string
}

// Channel defines the interface for different communication backends
type Channel interface {
	ID() string
	Type() string
	Name() string
	Start(router *Router) error
	Stop() error
	Send(toID string, text string) error
}

// Router handles incoming messages and routes them to agents
type Router struct {
	config   config.Config
	channels map[string]Channel
	sessions map[string]string // Mapping of user/chat IDs to agent IDs
	mu       sync.RWMutex
}

func NewRouter(conf config.Config) *Router {
	return &Router{
		config:   conf,
		channels: make(map[string]Channel),
		sessions: make(map[string]string),
	}
}

func (r *Router) RegisterChannel(ch Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.channels[ch.ID()] = ch
}

func (r *Router) Start() error {
	for _, ch := range r.channels {
		if err := ch.Start(r); err != nil {
			return fmt.Errorf("failed to start channel %s: %w", ch.ID(), err)
		}
	}
	return nil
}

// HandleIncoming is called by channels when a new message arrives
func (r *Router) HandleIncoming(msg Message) {
	// 1. Check for routing commands
	if r.handleCommands(msg) {
		return
	}

	// 2. Identify target agent
	r.mu.RLock()
	agentID, exists := r.sessions[msg.FromID]
	r.mu.RUnlock()

	if !exists {
		agentID = r.config.DefaultAgent
	}

	// 3. Load agent workspace
	ws, prov, mod, err := agent.LoadAgent(r.config, agentID)
	if err != nil {
		r.Reply(msg, fmt.Sprintf("Error loading agent %s: %v", agentID, err))
		return
	}

	// 4. Create prompt and query provider
	history := []provider.ChatMessage{
		{Role: "system", Content: agent.BuildSystemPrompt(ws)},
		{Role: "user", Content: msg.Text},
	}

	// Create a simple response (streaming support not implemented yet in Router)
	resp, err := prov.Query(mod, history)
	if err != nil {
		r.Reply(msg, fmt.Sprintf("Error querying provider: %v", err))
		return
	}

	// 5. Check for tool call
	if toolResp, ok := r.processToolCall(ws, resp); ok {
		// Feed tool output back to agent for final response
		history = append(history, provider.ChatMessage{Role: "assistant", Content: resp})
		history = append(history, provider.ChatMessage{Role: "user", Content: toolResp})

		finalResp, err := prov.Query(mod, history)
		if err != nil {
			r.Reply(msg, fmt.Sprintf("Error in post-tool query: %v", err))
			return
		}
		r.Reply(msg, finalResp)
		return
	}

	r.Reply(msg, resp)
}

var toolCallRegex = regexp.MustCompile(`(?s)CALL:\s*(\w+)\((.*)\)`)

func (r *Router) processToolCall(ws agent.AgentWorkspace, text string) (string, bool) {
	match := toolCallRegex.FindStringSubmatch(text)
	if len(match) != 3 {
		return "", false
	}

	toolName := match[1]
	argsJSON := match[2]

	// Verify agent has permission for this tool
	hasPermission := false
	for _, t := range ws.Config.Tools {
		if t == toolName {
			hasPermission = true
			break
		}
	}
	if !hasPermission {
		return fmt.Sprintf("Error: Agent does not have permission for tool [%s]", toolName), true
	}

	t, ok := agent.GetTool(toolName)
	if !ok {
		return fmt.Sprintf("Error: Tool [%s] not found", toolName), true
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("Error parsing tool arguments: %v", err), true
	}

	result, err := t.Execute(args, r.config)
	if err != nil {
		return fmt.Sprintf("Error executing tool [%s]: %v", toolName, err), true
	}

	return result, true
}

func (r *Router) handleCommands(msg Message) bool {
	parts := strings.Fields(msg.Text)
	if len(parts) == 0 {
		return false
	}

	switch parts[0] {
	case "/agent":
		if len(parts) < 2 {
			r.Reply(msg, "Usage: /agent list|switch <id>")
			return true
		}
		switch parts[1] {
		case "list":
			agents, _ := agent.ListAgents()
			list := "Installed Agents:\n"
			for _, a := range agents {
				list += fmt.Sprintf("- %s (ID: %s)\n", a.Name, a.ID)
			}
			r.Reply(msg, list)
			return true
		case "switch":
			if len(parts) < 3 {
				r.Reply(msg, "Usage: /agent switch <id>")
				return true
			}
			id := parts[2]
			_, _, _, err := agent.LoadAgent(r.config, id)
			if err != nil {
				r.Reply(msg, fmt.Sprintf("Error switching to agent %s: %v", id, err))
				return true
			}
			r.SetSession(msg.FromID, id)
			r.Reply(msg, fmt.Sprintf("Switched to agent: %s", id))
			return true
		}
	}
	return false
}

func (r *Router) Reply(msg Message, text string) error {
	r.mu.RLock()
	ch, exists := r.channels[msg.ChannelID]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", msg.ChannelID)
	}

	return ch.Send(msg.FromID, text)
}

func (r *Router) SetSession(userID, agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[userID] = agentID
}
