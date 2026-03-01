package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

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

type Router struct {
	config        config.Config
	channels      map[string]Channel
	sessions      map[string]string                 // Mapping of user/chat IDs to agent IDs
	histories     map[string][]provider.ChatMessage // In-memory history for current gateway session
	activeTasks   map[string]context.CancelFunc     // Active processing contexts per user
	pairingCounts map[string]int                    // Track pairing attempts (max 3)
	mu            sync.RWMutex
}

func NewRouter(conf config.Config) *Router {
	return &Router{
		config:        conf,
		channels:      make(map[string]Channel),
		sessions:      make(map[string]string),
		histories:     make(map[string][]provider.ChatMessage),
		activeTasks:   make(map[string]context.CancelFunc),
		pairingCounts: make(map[string]int),
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
	// 1. Check for pairing commands for ALL users
	if r.handlePairing(msg) {
		return
	}

	// 2. Access Control
	if !r.isUserAllowed(msg.FromID) {
		r.Reply(msg, "🔐 This GoClaw instance is locked. If you are the owner, please use `/pair <your_code>` to authorize this session.")
		return
	}

	// 3. Check for routing commands for authorized users
	if r.handleCommands(msg) {
		return
	}

	// 2. Interrupt existing task if any
	r.mu.Lock()
	if cancel, exists := r.activeTasks[msg.FromID]; exists {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.activeTasks[msg.FromID] = cancel
	r.mu.Unlock()

	go r.processMessage(ctx, msg)
}

func (r *Router) processMessage(ctx context.Context, msg Message) {
	defer func() {
		r.mu.Lock()
		delete(r.activeTasks, msg.FromID)
		r.mu.Unlock()
	}()

	// 1. Identify target agent
	r.mu.RLock()
	agentID, exists := r.sessions[msg.FromID]
	r.mu.RUnlock()

	if !exists {
		agentID = r.config.DefaultAgent
	}

	// 2. Load agent workspace
	ws, prov, mod, err := agent.LoadAgent(r.config, agentID)
	if err != nil {
		r.Reply(msg, fmt.Sprintf("Error loading agent %s: %v", agentID, err))
		return
	}

	// 3. Update and prepare history
	r.mu.Lock()
	history := r.histories[msg.FromID]

	chatCount := 0
	for _, h := range history {
		if h.Role != "system" {
			chatCount++
		}
	}
	if len(history) == 0 || (chatCount > 0 && chatCount%40 == 0) {
		history = append(history, provider.ChatMessage{
			Role:    "system",
			Content: agent.BuildSystemPrompt(ws),
		})
	}
	history = append(history, provider.ChatMessage{Role: "user", Content: msg.Text})
	r.histories[msg.FromID] = history
	r.mu.Unlock()

	// 4. Agent Loop (multi-turn tool calling)
	var latestSent string // Cache of the last significant text piece sent to UI
	isFirstTurn := true
	for i := 0; i < 5; i++ { // Limit to 5 iterations for safety
		select {
		case <-ctx.Done():
			return
		default:
		}

		resp, err := prov.Query(ctx, mod, history)
		if err != nil {
			r.Reply(msg, fmt.Sprintf("Error: %v", err))
			return
		}

		// Update history with assistant message
		r.mu.Lock()
		history = append(r.histories[msg.FromID], provider.ChatMessage{Role: "assistant", Content: resp})
		r.histories[msg.FromID] = history
		r.mu.Unlock()

		// Check for tool call
		matches := toolCallRegex.FindAllStringSubmatch(resp, -1)
		if len(matches) > 0 {
			// Execute tool
			match := matches[0]
			toolName := match[1]
			argsJSON := match[2]

			callIdx := strings.Index(resp, "CALL:")
			prefix := strings.TrimSpace(resp[:callIdx])

			// IMPROVED: If prefix starts with latestSent (case-insensitive), strip it to avoid repetition
			if latestSent != "" {
				lsLower := strings.ToLower(strings.TrimSpace(latestSent))
				pLower := strings.ToLower(strings.TrimSpace(prefix))
				if strings.HasPrefix(pLower, lsLower) {
					prefix = strings.TrimSpace(prefix[len(lsLower):])
				} else {
					// Also strip common greetings if NOT first turn
					if !isFirstTurn {
						prefix = stripGreetings(prefix)
					}
				}
				prefix = strings.TrimSpace(strings.TrimPrefix(prefix, "."))
				prefix = strings.TrimSpace(strings.TrimPrefix(prefix, ","))
				prefix = strings.TrimSpace(strings.TrimPrefix(prefix, "!"))
			}

			// Send prefix if it's not a reply tool OR if it's different from the reply text
			if prefix != "" {
				if toolName != "reply" {
					r.Reply(msg, prefix)
					latestSent = prefix
				} else {
					// For 'reply', only send prefix if it doesn't match the reply text
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(argsJSON), &args); err == nil {
						txt, _ := args["text"].(string)
						if strings.TrimSpace(prefix) != strings.TrimSpace(txt) {
							r.Reply(msg, prefix)
							latestSent = prefix
						}
					}
				}
			}

			// Explicitly handle 'reply' tool to send messages during the loop
			if toolName == "reply" {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(argsJSON), &args); err == nil {
					if txt, ok := args["text"].(string); ok && txt != "" {
						r.Reply(msg, txt)
						latestSent = txt
					}
				}
			}

			toolResp, ok := r.executeTool(ctx, ws, toolName, argsJSON)
			if ok {
				// Feed tool output back and continue loop
				r.mu.Lock()
				history = append(r.histories[msg.FromID], provider.ChatMessage{Role: "user", Content: toolResp})
				r.histories[msg.FromID] = history
				r.mu.Unlock()
				isFirstTurn = false
				continue
			}
		}

		// No more tools, just reply with the response (or the remaining part after tool call)
		// Usually if it's the last iteration or no tool call, we send the whole thing.
		// If we already sent a prefix, we might want to send the suffix too, but models usually stop after a call.
		if len(matches) == 0 {
			toSend := resp
			if !isFirstTurn {
				toSend = stripGreetings(toSend)
			}
			trimmedLatest := strings.ToLower(strings.TrimSpace(latestSent))
			trimmedResp := strings.ToLower(strings.TrimSpace(toSend))

			if trimmedLatest != "" && strings.HasPrefix(trimmedResp, trimmedLatest) {
				// Strip the repetition from the original resp ( preserving case for the rest)
				startIdx := strings.Index(strings.ToLower(toSend), trimmedLatest)
				if startIdx != -1 {
					toSend = strings.TrimSpace(toSend[startIdx+len(trimmedLatest):])
					toSend = strings.TrimSpace(strings.TrimPrefix(toSend, "."))
					toSend = strings.TrimSpace(strings.TrimPrefix(toSend, "!"))
					toSend = strings.TrimSpace(strings.TrimPrefix(toSend, "?"))
				}
			}

			if toSend != "" {
				r.Reply(msg, toSend)
			}
		}
		break // Exit loop if no tool call
	}
}

func stripGreetings(text string) string {
	greetings := []string{
		"selam", "merhaba", "hos geldin", "hoş geldin", "nasılsın", "nasilsin",
		"hello", "hi", "hey", "greetings", "how can i help", "yardımcı olabilirim",
	}
	lower := strings.ToLower(text)
	found := true
	for found {
		found = false
		for _, g := range greetings {
			if strings.HasPrefix(lower, g) {
				text = strings.TrimSpace(text[len(g):])
				// Clean punctuation
				text = strings.TrimSpace(strings.TrimLeft(text, "!,.-?👋😊"))
				lower = strings.ToLower(text)
				found = true
				break
			}
		}
	}
	return text
}

var toolCallRegex = regexp.MustCompile(`(?s)CALL:\s*(\w+)\s*\((.*?)\)`)

func (r *Router) executeTool(ctx context.Context, ws agent.AgentWorkspace, toolName, argsJSON string) (string, bool) {
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

	result, err := t.Execute(ctx, args, r.config)
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

	// Handle case-insensitivity and bot suffixes like /agent@mybot
	command := strings.ToLower(parts[0])
	if strings.Contains(command, "@") {
		command = strings.Split(command, "@")[0]
	}

	switch command {
	case "/help":
		helpText := "🦞 GoClaw Gateway Commands:\n"
		helpText += "- /agent list: Show all installed agents\n"
		helpText += "- /agent switch <id>: Switch to a different agent\n"
		helpText += "- /clear: Reset current chat history\n"
		helpText += "- /history: Show current chat history entries\n"
		helpText += "- /tokens: (TODO) Show token estimation\n"
		helpText += "- /help: Show this message\n"
		r.Reply(msg, helpText)
		return true
	case "/clear":
		r.mu.Lock()
		delete(r.histories, msg.FromID)
		r.mu.Unlock()
		r.Reply(msg, "✅ Chat history cleared.")
		return true
	case "/history":
		r.mu.RLock()
		h := r.histories[msg.FromID]
		r.mu.RUnlock()
		if len(h) == 0 {
			r.Reply(msg, "History is empty.")
			return true
		}
		text := fmt.Sprintf("History (%d messages):\n", len(h))
		for i, m := range h {
			text += fmt.Sprintf("%d. [%s]: %s\n", i+1, m.Role, strings.Split(m.Content, "\n")[0])
		}
		r.Reply(msg, text)
		return true
	case "/tokens":
		r.mu.RLock()
		h := r.histories[msg.FromID]
		r.mu.RUnlock()
		chars := 0
		for _, m := range h {
			chars += len(m.Content)
		}
		r.Reply(msg, fmt.Sprintf("Estimated tokens: ~%d (based on character count/4)", chars/4))
		return true
	case "/agent":
		if len(parts) < 2 {
			r.Reply(msg, "Usage: /agent list|switch <id>")
			return true
		}
		switch strings.ToLower(parts[1]) {
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

func (r *Router) isUserAllowed(userID string) bool {
	if !r.config.PairingEnabled {
		return true
	}

	// Dynamic check: attempt to reload config to pick up new approvals from CLI
	if updated, err := config.Load(); err == nil {
		r.mu.Lock()
		r.config = updated
		r.mu.Unlock()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.config.AllowedUsers {
		if u == userID {
			return true
		}
	}
	return false
}

func (r *Router) handlePairing(msg Message) bool {
	if !r.config.PairingEnabled || r.isUserAllowed(msg.FromID) {
		return false
	}

	r.mu.Lock()
	count := r.pairingCounts[msg.FromID]
	if count >= 3 {
		r.mu.Unlock()
		r.Reply(msg, "⛔ Maximum pairing attempts reached (3/3). Please contact the administrator.")
		return true
	}
	r.pairingCounts[msg.FromID]++
	r.mu.Unlock()

	// Create a 6-digit random code
	src := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(src)
	code := fmt.Sprintf("%06d", rnd.Intn(1000000))

	// Find the channel info
	r.mu.RLock()
	ch, _ := r.channels[msg.ChannelID]
	r.mu.RUnlock()
	chType := "Unknown"
	if ch != nil {
		chType = strings.Title(ch.Type())
	}

	// Save to pending
	pairings, _ := config.LoadPendingPairings()
	pairings = append(pairings, config.PendingPairing{
		ChannelID: msg.ChannelID,
		UserID:    msg.FromID,
		Code:      code,
	})
	_ = config.SavePendingPairings(pairings)

	// LOG THE COMMAND TO CONSOLE
	fmt.Printf("\n[SECURITY] 🔓 Pairing required for %s user: %s (Attempt %d/3)\n", chType, msg.FromID, count+1)
	fmt.Printf("[SECURITY] Run this command to authorize:\n")
	cmd := fmt.Sprintf("goclaw pairing approve \"%s\" \"%s\" \"%s\"", chType, msg.FromID, code)
	fmt.Printf("   %s\n\n", cmd)

	// Inform user AND SEND CODE TO TELEGRAM
	replyText := fmt.Sprintf("🔐 This GoClaw instance is locked.\n\nYour Pairing Code: `%s`\n(Attempt %d/3)\n\nTo authorize, the owner should run:\n`%s`", code, count+1, cmd)
	r.Reply(msg, replyText)
	return true
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
	// Clear history when switching agent to maintain context integrity
	delete(r.histories, userID)
}
