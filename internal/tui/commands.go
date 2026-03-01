package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/allataetm-svg/goclaw/internal/agent"
	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/memory"
	"github.com/allataetm-svg/goclaw/internal/provider"
)

func (m Model) processSlashCommand(cmd string) (Model, tea.Cmd) {
	parts := strings.Fields(cmd)
	baseCmd := parts[0]

	switch baseCmd {
	case "/help":
		helpText := `Commands:
  /agent list              — List all agents
  /agent switch <id>       — Switch to another agent
  /agent add <name> <prov:model> — Add new agent
  /agent delete <id>       — Delete an agent
  /agent prompt <id> <text>— Edit agent prompt
  /model [provider:model]  — View or change model
  /history list            — List saved conversations
  /history load <id>       — Load a conversation
  /history delete <id>     — Delete a conversation
  /clear                   — Clear chat memory
  /tokens                  — Show token usage
  /exit or /quit           — Exit app`
		return m.appendSystem(helpText), nil

	case "/tokens":
		tokens := memory.EstimateHistoryTokens(m.chatHistory)
		maxTokens := m.config.MaxTokens
		pct := float64(tokens) / float64(maxTokens) * 100
		return m.appendSystem(fmt.Sprintf("📊 Token Usage: ~%d / %d (%.1f%%)", tokens, maxTokens, pct)), nil

	case "/model":
		if len(parts) > 1 {
			target := parts[1]
			tParts := strings.SplitN(target, ":", 2)
			if len(tParts) == 2 {
				provID := tParts[0]
				modName := tParts[1]

				var pc config.ProviderConfig
				for _, p := range m.config.Providers {
					if p.ID == provID {
						pc = p
						break
					}
				}
				m.currentMod = modName
				m.currentProv = provider.MakeProvider(pc)
				m.currentAgent.Model = target

				for i, a := range m.config.Agents {
					if a.ID == m.currentAgent.ID {
						m.config.Agents[i].Model = target
						break
					}
				}
				if err := config.Save(m.config); err != nil {
					return m.appendSystem("⚠️ Model changed but config save failed: " + err.Error()), nil
				}
				return m.appendSystem(fmt.Sprintf("✅ Model changed to: %s (%s)", m.currentMod, m.currentProv.Name())), nil
			}
			return m.appendSystem("❌ Invalid format. Example: /model openai:gpt-4o"), nil
		}
		return m.appendSystem(fmt.Sprintf("Current Model: %s:%s", m.currentProv.ID(), m.currentMod)), nil

	case "/clear":
		m.chatHistory = []provider.ChatMessage{
			{Role: "system", Content: agent.BuildSystemPrompt(m.currentAgent)},
		}
		return m.appendSystem("✅ Memory cleared. The agent forgot your previous conversation."), nil

	case "/agent":
		return m.handleAgentCommand(parts)

	case "/history":
		return m.handleHistoryCommand(parts)

	case "/exit", "/quit":
		m.conversation.Messages = m.chatHistory
		_ = memory.SaveConversation(m.conversation)
		return m, tea.Quit

	default:
		return m.appendSystem("Unknown command. Type /help to see available commands."), nil
	}
}

func (m Model) handleAgentCommand(parts []string) (Model, tea.Cmd) {
	if len(parts) == 1 {
		return m.appendSystem(fmt.Sprintf("Current Agent: %s (%s)", m.currentAgent.Name, m.currentAgent.ID)), nil
	}

	subCmd := parts[1]

	switch subCmd {
	case "list":
		var list string
		for _, a := range m.config.Agents {
			marker := "  "
			if a.ID == m.currentAgent.ID {
				marker = "▸ "
			}
			list += fmt.Sprintf("%s%s (ID: %s, Model: %s)\n", marker, a.Name, a.ID, a.Model)
		}
		return m.appendSystem("Installed Agents:\n" + list), nil

	case "switch":
		if len(parts) < 3 {
			return m.appendSystem("❌ Usage: /agent switch <id>"), nil
		}
		targetID := parts[2]
		ag, prov, modName, err := agent.LoadAgent(m.config, targetID)
		if err != nil {
			return m.appendSystem("❌ Error: " + err.Error()), nil
		}

		m.currentAgent = ag
		m.currentProv = prov
		m.currentMod = modName

		// Reset memory with new agent identity
		m.chatHistory = []provider.ChatMessage{
			{Role: "system", Content: agent.BuildSystemPrompt(ag)},
		}

		m.config.DefaultAgent = ag.ID
		if err := config.Save(m.config); err != nil {
			return m.appendSystem(fmt.Sprintf("⚠️ Switched to %s but config save failed: %s", ag.Name, err.Error())), nil
		}
		return m.appendSystem(fmt.Sprintf("✅ Switched to: %s (%s / %s)\nChat history has been reset.", ag.Name, prov.Name(), modName)), nil

	case "add":
		if len(parts) < 4 {
			return m.appendSystem("❌ Usage: /agent add <name> <provider:model>\nExample: /agent add Coder openai:gpt-4o"), nil
		}
		name := parts[2]
		model := parts[3]

		ag, err := agent.AddAgent(&m.config, name, "", model)
		if err != nil {
			return m.appendSystem("❌ Error: " + err.Error()), nil
		}
		return m.appendSystem(fmt.Sprintf("✅ Agent created: %s (ID: %s, Model: %s)", ag.Name, ag.ID, ag.Model)), nil

	case "delete":
		if len(parts) < 3 {
			return m.appendSystem("❌ Usage: /agent delete <id>"), nil
		}
		targetID := parts[2]

		if targetID == m.currentAgent.ID {
			return m.appendSystem("❌ Cannot delete the currently active agent. Switch first."), nil
		}

		if err := agent.DeleteAgent(&m.config, targetID); err != nil {
			return m.appendSystem("❌ Error: " + err.Error()), nil
		}
		return m.appendSystem(fmt.Sprintf("✅ Agent '%s' deleted.", targetID)), nil

	case "prompt":
		if len(parts) < 4 {
			return m.appendSystem("❌ Usage: /agent prompt <id> <new prompt text>"), nil
		}
		targetID := parts[2]
		newPrompt := strings.Join(parts[3:], " ")

		if err := agent.EditAgentPrompt(&m.config, targetID, newPrompt); err != nil {
			return m.appendSystem("❌ Error: " + err.Error()), nil
		}

		// If editing current agent, update in-memory state
		if targetID == m.currentAgent.ID {
			m.currentAgent.SystemPrompt = newPrompt
			// Update system message in history
			if len(m.chatHistory) > 0 && m.chatHistory[0].Role == "system" {
				m.chatHistory[0].Content = agent.BuildSystemPrompt(m.currentAgent)
			}
		}
		return m.appendSystem(fmt.Sprintf("✅ Prompt updated for agent '%s'.", targetID)), nil

	default:
		return m.appendSystem("❌ Unknown agent command. Usage: /agent list|switch|add|delete|prompt"), nil
	}
}

func (m Model) handleHistoryCommand(parts []string) (Model, tea.Cmd) {
	if len(parts) == 1 {
		return m.appendSystem("Usage: /history list|load|delete"), nil
	}

	subCmd := parts[1]

	switch subCmd {
	case "list":
		convs, err := memory.ListConversations()
		if err != nil {
			return m.appendSystem("❌ Error listing history: " + err.Error()), nil
		}
		if len(convs) == 0 {
			return m.appendSystem("📭 No saved conversations found."), nil
		}

		var list string
		maxShow := 10
		for i, c := range convs {
			if i >= maxShow {
				list += fmt.Sprintf("  ... and %d more\n", len(convs)-maxShow)
				break
			}
			msgCount := 0
			for _, msg := range c.Messages {
				if msg.Role != "system" {
					msgCount++
				}
			}
			list += fmt.Sprintf("  %s — Agent: %s, Msgs: %d, Updated: %s\n",
				c.ID, c.AgentName, msgCount, c.UpdatedAt.Format("2006-01-02 15:04"))
		}
		return m.appendSystem("📚 Saved Conversations:\n" + list), nil

	case "load":
		if len(parts) < 3 {
			return m.appendSystem("❌ Usage: /history load <id>"), nil
		}
		convID := parts[2]
		conv, err := memory.LoadConversation(convID)
		if err != nil {
			return m.appendSystem("❌ Error: " + err.Error()), nil
		}

		m.chatHistory = conv.Messages
		m.conversation = conv

		// Rebuild display messages
		m.messages = []string{fmt.Sprintf("📂 Loaded conversation %s (Agent: %s)", conv.ID, conv.AgentName)}
		for _, msg := range conv.Messages {
			switch msg.Role {
			case "user":
				m.messages = append(m.messages, m.styles.Sender.Render("You: ")+msg.Content)
			case "assistant":
				rendered := m.renderMarkdown(msg.Content)
				m.messages = append(m.messages, m.styles.Bot.Render(m.currentAgent.Name+": ")+"\n"+rendered)
			}
		}
		m = m.updateViewport()
		m.textarea.Reset()
		return m, nil

	case "delete":
		if len(parts) < 3 {
			return m.appendSystem("❌ Usage: /history delete <id>"), nil
		}
		convID := parts[2]
		if err := memory.DeleteConversation(convID); err != nil {
			return m.appendSystem("❌ Error: " + err.Error()), nil
		}
		return m.appendSystem(fmt.Sprintf("✅ Conversation '%s' deleted.", convID)), nil

	default:
		return m.appendSystem("❌ Unknown history command. Usage: /history list|load|delete"), nil
	}
}
