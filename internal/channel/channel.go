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
	"github.com/allataetm-svg/goclaw/internal/memory"
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

// UsageMode controls per-response usage footer
type UsageMode string

const (
	UsageOff    UsageMode = "off"
	UsageTokens UsageMode = "tokens"
	UsageFull   UsageMode = "full"
)

type Router struct {
	config        config.Config
	channels      map[string]Channel
	sessions      map[string]string                  // user/chat ID -> agent ID
	histories     map[string][]provider.ChatMessage  // In-memory history
	memStores     map[string]*memory.UserMemoryStore // user/agent -> memory store
	activeTasks   map[string]context.CancelFunc
	pairingCounts map[string]int
	usageModes    map[string]UsageMode // per-user usage display mode
	mu            sync.RWMutex
}

func NewRouter(conf config.Config) *Router {
	return &Router{
		config:        conf,
		channels:      make(map[string]Channel),
		sessions:      make(map[string]string),
		histories:     make(map[string][]provider.ChatMessage),
		memStores:     make(map[string]*memory.UserMemoryStore),
		activeTasks:   make(map[string]context.CancelFunc),
		pairingCounts: make(map[string]int),
		usageModes:    make(map[string]UsageMode),
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

	// 3. Check for routing/chat commands
	if r.handleCommands(msg) {
		return
	}

	// 4. Interrupt existing task if any
	r.mu.Lock()
	if cancel, exists := r.activeTasks[msg.FromID]; exists {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.activeTasks[msg.FromID] = cancel
	r.mu.Unlock()

	go r.processMessage(ctx, msg)
}

func (r *Router) getAgentID(userID string) string {
	r.mu.RLock()
	agentID, exists := r.sessions[userID]
	r.mu.RUnlock()
	if !exists {
		agentID = r.config.DefaultAgent
	}
	return agentID
}

func (r *Router) getMemoryStore(userID, agentID string) *memory.UserMemoryStore {
	key := userID + ":" + agentID
	r.mu.Lock()
	defer r.mu.Unlock()
	if ms, ok := r.memStores[key]; ok {
		return ms
	}
	ms := memory.NewUserMemoryStore(agentID)
	_ = ms.Load()
	r.memStores[key] = ms
	return ms
}

func (r *Router) enhanceWithMemory(msg Message, memStore *memory.UserMemoryStore) string {
	var memContext []string

	userInfo := fmt.Sprintf("# USER.md\nUser ID: %s", msg.FromID)
	memContext = append(memContext, userInfo)

	mems := memStore.List()
	if len(mems) > 0 {
		seen := make(map[string]bool)
		memContext = append(memContext, "\n# MEMORY.md\nLearned facts about this user:")
		for _, mem := range mems {
			if !seen[mem.Key] {
				memContext = append(memContext, fmt.Sprintf("- %s: %s", mem.Key, mem.Value))
				seen[mem.Key] = true
			}
		}
	}

	return strings.Join(memContext, "\n") + "\n\n" + msg.Text
}

func (r *Router) processMessage(ctx context.Context, msg Message) {
	defer func() {
		r.mu.Lock()
		delete(r.activeTasks, msg.FromID)
		r.mu.Unlock()
	}()

	agentID := r.getAgentID(msg.FromID)

	// Load agent workspace
	ws, prov, mod, err := agent.LoadAgent(r.config, agentID)
	if err != nil {
		r.Reply(msg, fmt.Sprintf("Error loading agent %s: %v", agentID, err))
		return
	}

	// Load memory and enhance input
	memStore := r.getMemoryStore(msg.FromID, agentID)
	enhancedInput := r.enhanceWithMemory(msg, memStore)

	// Prepare history
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
	history = append(history, provider.ChatMessage{Role: "user", Content: enhancedInput})

	// Auto-compact if approaching token limit
	history, compactMsg := memory.CompactLongHistory(history, r.config.MaxTokens)
	if compactMsg != "" {
		fmt.Printf("[Memory] %s for user %s\n", compactMsg, msg.FromID)
	}

	r.histories[msg.FromID] = history
	r.mu.Unlock()

	// Agent Loop (multi-turn tool calling) — up to 10 iterations like OpenClaw
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		resp, err := prov.Query(ctx, mod, history)
		if err != nil {
			if strings.Contains(err.Error(), "context canceled") || err == context.Canceled {
				return // User interrupted the task, silently abort
			}
			r.Reply(msg, fmt.Sprintf("Error: %v", err))
			return
		}

		// Update history with assistant message
		r.mu.Lock()
		history = append(r.histories[msg.FromID], provider.ChatMessage{Role: "assistant", Content: resp})
		r.histories[msg.FromID] = history
		r.mu.Unlock()

		// Check for tool call
		matches := toolCallRegex.FindStringSubmatch(resp)
		matchIdx := toolCallRegex.FindStringIndex(resp)

		if len(matches) > 0 && len(matchIdx) > 0 {
			toolName := matches[1]
			argsJSON := matches[2]

			// Send any text before the tool call as an intermediate message
			if matchIdx[0] > 0 {
				preText := strings.TrimSpace(resp[:matchIdx[0]])
				if preText != "" {
					r.Reply(msg, preText)
				}
			}

			toolResp, ok := r.executeTool(ctx, ws, toolName, argsJSON)
			if ok {
				// Feed tool result back as context
				r.mu.Lock()
				toolResultMsg := fmt.Sprintf("[Tool Result: %s]\n%s", toolName, toolResp)
				history = append(r.histories[msg.FromID], provider.ChatMessage{Role: "user", Content: toolResultMsg})
				r.histories[msg.FromID] = history
				r.mu.Unlock()
				continue
			}
		}

		// No tool call — send the response directly
		if len(matches) == 0 && resp != "" {
			r.Reply(msg, resp)
			r.appendUsageFooter(msg)
		}
		break
	}
}

func (r *Router) appendUsageFooter(msg Message) {
	r.mu.RLock()
	mode := r.usageModes[msg.FromID]
	h := r.histories[msg.FromID]
	r.mu.RUnlock()

	if mode == "" || mode == UsageOff {
		return
	}

	chars := 0
	for _, m := range h {
		chars += len(m.Content)
	}
	tokens := chars / 4

	switch mode {
	case UsageTokens:
		r.Reply(msg, fmt.Sprintf("📊 ~%d tokens", tokens))
	case UsageFull:
		pct := float64(tokens) / float64(r.config.MaxTokens) * 100
		r.Reply(msg, fmt.Sprintf("📊 ~%d / %d tokens (%.1f%%) | %d messages", tokens, r.config.MaxTokens, pct, len(h)))
	}
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
		helpText := `🦞 GoClaw Commands:
- /agent list — list installed agents
- /agent switch <id> — switch agent
- /status — session status (model + tokens)
- /new, /reset — reset session
- /clear — clear chat history
- /compact — compact context
- /usage off|tokens|full — usage footer
- /tokens — token estimate
- /history — show history
- /help — this message`
		r.Reply(msg, helpText)
		return true

	case "/clear", "/new", "/reset":
		r.mu.Lock()
		delete(r.histories, msg.FromID)
		r.mu.Unlock()
		r.Reply(msg, "✅ Session reset.")
		return true

	case "/status":
		agentID := r.getAgentID(msg.FromID)
		ws, _, mod, err := agent.LoadAgent(r.config, agentID)
		if err != nil {
			r.Reply(msg, fmt.Sprintf("Error: %v", err))
			return true
		}
		r.mu.RLock()
		h := r.histories[msg.FromID]
		r.mu.RUnlock()
		chars := 0
		for _, m := range h {
			chars += len(m.Content)
		}
		tokens := chars / 4
		pct := float64(tokens) / float64(r.config.MaxTokens) * 100
		status := fmt.Sprintf("🦞 **%s**\nModel: `%s`\nTokens: ~%d / %d (%.1f%%)\nMessages: %d",
			ws.Config.Name, mod, tokens, r.config.MaxTokens, pct, len(h))
		r.Reply(msg, status)
		return true

	case "/compact":
		r.mu.Lock()
		h := r.histories[msg.FromID]
		compacted, info := memory.CompactLongHistory(h, r.config.MaxTokens/2)
		r.histories[msg.FromID] = compacted
		r.mu.Unlock()
		if info != "" {
			r.Reply(msg, fmt.Sprintf("✅ %s", info))
		} else {
			r.Reply(msg, "✅ Context compacted.")
		}
		return true

	case "/usage":
		if len(parts) < 2 {
			r.Reply(msg, "Usage: /usage off|tokens|full")
			return true
		}
		mode := strings.ToLower(parts[1])
		switch mode {
		case "off":
			r.mu.Lock()
			r.usageModes[msg.FromID] = UsageOff
			r.mu.Unlock()
			r.Reply(msg, "Usage footer disabled.")
		case "tokens":
			r.mu.Lock()
			r.usageModes[msg.FromID] = UsageTokens
			r.mu.Unlock()
			r.Reply(msg, "Usage footer: tokens only.")
		case "full":
			r.mu.Lock()
			r.usageModes[msg.FromID] = UsageFull
			r.mu.Unlock()
			r.Reply(msg, "Usage footer: full details.")
		default:
			r.Reply(msg, "Usage: /usage off|tokens|full")
		}
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
			firstLine := strings.Split(m.Content, "\n")[0]
			if len(firstLine) > 80 {
				firstLine = firstLine[:80] + "..."
			}
			text += fmt.Sprintf("%d. [%s]: %s\n", i+1, m.Role, firstLine)
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
		tokens := chars / 4
		pct := float64(tokens) / float64(r.config.MaxTokens) * 100
		r.Reply(msg, fmt.Sprintf("📊 Tokens: ~%d / %d (%.1f%%)", tokens, r.config.MaxTokens, pct))
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
			currentAgent := r.getAgentID(msg.FromID)
			for _, a := range agents {
				marker := ""
				if a.ID == currentAgent {
					marker = " ← active"
				}
				list += fmt.Sprintf("- %s (ID: %s)%s\n", a.Name, a.ID, marker)
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

	// Dynamic reload to pick up CLI approvals
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
		chType = capitalizeFirst(ch.Type())
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

// capitalizeFirst replaces deprecated strings.Title
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
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
	// Clear history when switching agent
	delete(r.histories, userID)
}
