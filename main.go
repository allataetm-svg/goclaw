package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/allataetm-svg/goclaw/internal/agent"
	"github.com/allataetm-svg/goclaw/internal/channel"
	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/manage"
	"github.com/allataetm-svg/goclaw/internal/memory"
	"github.com/allataetm-svg/goclaw/internal/onboard"
	"github.com/allataetm-svg/goclaw/internal/provider"
	"github.com/allataetm-svg/goclaw/internal/tui"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	switch command {
	case "onboard":
		onboard.Run()
	case "tui":
		runTUI()
	case "cli":
		runCLI()
	case "gateway":
		gateway()
	case "manage":
		manage.Run()
	case "pairing":
		handlePairingCommand()
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`🦞 GoClaw - Personal AI Assistant (Lightweight & Local-first)

Available Commands:
  onboard - Starts the setup wizard to configure providers and agents.
  tui     - Starts the Terminal User Interface (Chat).
  cli     - Starts the Command Line Interface (no TTY required).
  gateway - Starts the multi-channel gateway (Telegram, Console, etc.).
  manage  - Opens the interactive agent/channel management dashboard.
  pairing - Manages user authorizations (approve <channel> <userID> <code>).
  help    - Shows this help message.

Example Usage:
  ./goclaw onboard
  ./goclaw cli
  ./goclaw gateway
  ./goclaw pairing approve Telegram 123456 000000`)
}

func runCLI() {
	conf, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	agents, err := agent.ListAgents()
	if err != nil || len(agents) == 0 {
		fmt.Println("No agents found. Please run 'goclaw onboard' first.")
		os.Exit(1)
	}

	agentID := conf.DefaultAgent
	if agentID == "" {
		agentID = agents[0].ID
	}

	ws, prov, modName, err := agent.LoadAgent(conf, agentID)
	if err != nil {
		fmt.Printf("Error loading agent: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🦞 GoClaw CLI [%s] — Model: %s\n", ws.Config.Name, modName)
	fmt.Println("Type /help for commands, /exit to quit.\n")

	history := []provider.ChatMessage{
		{Role: "system", Content: agent.BuildSystemPrompt(ws)},
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if strings.HasPrefix(input, "/") {
			response := handleCLICommand(input, &history, ws, conf, agentID)
			fmt.Printf("%s\n", response)
			continue
		}

		history = append(history, provider.ChatMessage{Role: "user", Content: input})

		fmt.Printf("%s: ", ws.Config.Name)

		ch := make(chan provider.StreamChunk)
		go func() {
			for chunk := range ch {
				if chunk.Error != nil {
					fmt.Printf("\nError: %v\n", chunk.Error)
					return
				}
				fmt.Print(chunk.Text)
				if chunk.Done {
					break
				}
			}
		}()

		prov.QueryStream(context.Background(), modName, history, ch)

		if err != nil {
			fmt.Printf("\n❌ Error: %v\n", err)
			history = history[:len(history)-1]
			continue
		}

		lastMsg := history[len(history)-1]
		history = append(history, provider.ChatMessage{Role: "assistant", Content: lastMsg.Content})
		fmt.Println()
	}
}

func handleCLICommand(cmd string, history *[]provider.ChatMessage, ws agent.AgentWorkspace, conf config.Config, agentID string) string {
	parts := strings.Fields(cmd)
	baseCmd := parts[0]

	switch baseCmd {
	case "/help":
		return `Commands:
  /memory store <key> <value>   - Store user memory
  /memory recall <query>        - Search memories
  /memory list                  - List all memories
  /knowledge add <content>      - Add knowledge
  /knowledge search <query>     - Search knowledge
  /knowledge list               - List knowledge
  /history list                 - List conversations
  /clear                        - Clear chat memory
  /tokens                       - Show token usage
  /exit                         - Exit`

	case "/exit", "/quit":
		conv := memory.Conversation{
			ID:        fmt.Sprintf("cli_%d", os.Getpid()),
			AgentID:   agentID,
			AgentName: ws.Config.Name,
			Messages:  *history,
		}
		_ = memory.SaveConversation(conv)
		fmt.Println("Conversation saved.")
		os.Exit(0)
		return ""

	case "/clear":
		*history = []provider.ChatMessage{
			{Role: "system", Content: agent.BuildSystemPrompt(ws)},
		}
		return "Memory cleared."

	case "/tokens":
		tokens := memory.EstimateHistoryTokens(*history)
		pct := float64(tokens) / float64(conf.MaxTokens) * 100
		return fmt.Sprintf("Token Usage: ~%d / %d (%.1f%%)", tokens, conf.MaxTokens, pct)

	case "/memory":
		return handleMemoryCLI(parts, agentID)

	case "/knowledge":
		return handleKnowledgeCLI(parts, agentID)

	case "/history":
		return handleHistoryCLI(parts)

	default:
		return fmt.Sprintf("Unknown command: %s", cmd)
	}
}

func handleMemoryCLI(parts []string, agentID string) string {
	if len(parts) < 2 {
		return "Usage: /memory store <key> <value> | /memory recall <query> | /memory list"
	}

	subCmd := parts[1]
	memStore := memory.NewUserMemoryStore(agentID)
	_ = memStore.Load()

	switch subCmd {
	case "list":
		mems := memStore.List()
		if len(mems) == 0 {
			return "No memories stored."
		}
		var result []string
		result = append(result, "Stored Memories:")
		for _, mem := range mems {
			result = append(result, fmt.Sprintf("  [%s] %s: %s", mem.Type, mem.Key, mem.Value))
		}
		return strings.Join(result, "\n")

	case "store":
		if len(parts) < 4 {
			return "Usage: /memory store <key> <value>"
		}
		key := parts[2]
		value := strings.Join(parts[3:], " ")
		err := memStore.Store(memory.UserMemory{
			Type:  memory.MemoryTypePreference,
			Key:   key,
			Value: value,
		})
		if err != nil {
			return "Error: " + err.Error()
		}
		return fmt.Sprintf("Memory stored: %s = %s", key, value)

	case "recall":
		if len(parts) < 3 {
			return "Usage: /memory recall <query>"
		}
		query := strings.Join(parts[2:], " ")
		results := memStore.Search(query)
		if len(results) == 0 {
			return fmt.Sprintf("No memories found for: %s", query)
		}
		var result []string
		result = append(result, "Found Memories:")
		for _, mem := range results {
			result = append(result, fmt.Sprintf("  [%s] %s: %s", mem.Type, mem.Key, mem.Value))
		}
		return strings.Join(result, "\n")

	case "delete":
		if len(parts) < 3 {
			return "Usage: /memory delete <id>"
		}
		memID := parts[2]
		err := memStore.Delete(memID)
		if err != nil {
			return "Error: " + err.Error()
		}
		return fmt.Sprintf("Memory '%s' deleted.", memID)

	default:
		return "Unknown memory command. Use store|recall|list|delete"
	}
}

func handleKnowledgeCLI(parts []string, agentID string) string {
	if len(parts) < 2 {
		return "Usage: /knowledge add <content> | /knowledge search <query> | /knowledge list"
	}

	subCmd := parts[1]
	ks := memory.NewKnowledgeStore(agentID)
	_ = ks.Load()

	switch subCmd {
	case "list":
		docs := ks.List()
		if len(docs) == 0 {
			return "No knowledge documents."
		}
		var result []string
		result = append(result, "Knowledge Documents:")
		for _, doc := range docs {
			preview := doc.Content
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			result = append(result, fmt.Sprintf("  [%s] %s", doc.Source, preview))
		}
		return strings.Join(result, "\n")

	case "add":
		if len(parts) < 3 {
			return "Usage: /knowledge add <content>"
		}
		content := strings.Join(parts[2:], " ")
		err := ks.AddDocument(memory.Document{
			Content: content,
			Source:  "manual",
		})
		if err != nil {
			return "Error: " + err.Error()
		}
		return "Knowledge document added."

	case "search":
		if len(parts) < 3 {
			return "Usage: /knowledge search <query>"
		}
		query := strings.Join(parts[2:], " ")
		results := ks.Search(query, 5)
		if len(results) == 0 {
			return fmt.Sprintf("No knowledge found for: %s", query)
		}
		var result []string
		result = append(result, "Found Knowledge:")
		for _, doc := range results {
			preview := doc.Content
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			result = append(result, fmt.Sprintf("  [%s] %s", doc.Source, preview))
		}
		return strings.Join(result, "\n")

	default:
		return "Unknown knowledge command. Use add|search|list"
	}
}

func handleHistoryCLI(parts []string) string {
	if len(parts) < 2 {
		return "Usage: /history list"
	}

	subCmd := parts[1]

	switch subCmd {
	case "list":
		convs, err := memory.ListConversations()
		if err != nil {
			return "Error: " + err.Error()
		}
		if len(convs) == 0 {
			return "No saved conversations."
		}
		var result []string
		result = append(result, "Saved Conversations:")
		for _, conv := range convs {
			result = append(result, fmt.Sprintf("  [%s] %s - %s", conv.ID, conv.AgentName, conv.UpdatedAt.Format("2006-01-02 15:04")))
		}
		return strings.Join(result, "\n")

	default:
		return "Unknown history command. Use list"
	}
}

func runTUI() {
	tui.Run()
}

func gateway() {
	conf, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	router := channel.NewRouter(conf)

	console := channel.NewConsoleChannel("cli", "Main Console", conf.DefaultAgent)
	router.RegisterChannel(console)

	for _, cc := range conf.Channels {
		var ch channel.Channel
		switch cc.Type {
		case "telegram":
			token := cc.Settings["token"]
			if token == "" {
				fmt.Printf("Warning: Skipping channel %s, token not found\n", cc.ID)
				continue
			}
			ch = channel.NewTelegramChannel(cc.ID, cc.Name, token, cc.AgentID)
		default:
			fmt.Printf("Warning: Unknown channel type %s for %s\n", cc.Type, cc.ID)
			continue
		}

		router.RegisterChannel(ch)
		fmt.Printf("Registered channel: %s (%s)\n", cc.Name, cc.Type)
	}

	fmt.Println("🚀 GoClaw Gateway Started. Listening for messages...")
	if err := router.Start(); err != nil {
		fmt.Printf("Gateway error: %v\n", err)
		return
	}

	select {}
}

func handlePairingCommand() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: goclaw pairing approve <channel> <userID> <code>")
		return
	}

	sub := os.Args[2]
	if sub != "approve" {
		fmt.Printf("Unknown pairing subcommand: %s\n", sub)
		return
	}

	userID := os.Args[4]
	code := os.Args[5]

	if err := config.ApprovePairing(userID, code); err != nil {
		fmt.Printf("❌ Approval failed: %v\n", err)
	} else {
		fmt.Printf("✅ User %s successfully approved and added to whitelist.\n", userID)
	}
}
