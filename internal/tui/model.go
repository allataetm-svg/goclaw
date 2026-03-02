package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/allataetm-svg/goclaw/internal/agent"
	"github.com/allataetm-svg/goclaw/internal/config"
	"github.com/allataetm-svg/goclaw/internal/memory"
	"github.com/allataetm-svg/goclaw/internal/provider"
)

// Model is the main TUI application model
type Model struct {
	config       config.Config
	currentAgent agent.AgentWorkspace
	currentProv  provider.LLMProvider
	currentMod   string

	viewport viewport.Model
	textarea textarea.Model
	styles   Styles

	chatHistory []provider.ChatMessage // Raw history sent to API
	messages    []string               // Formatted display messages

	conversation memory.Conversation // Persistent conversation

	streamCh      chan provider.StreamChunk
	streamingText string
	isThinking    bool
	err           error

	mdRenderer *glamour.TermRenderer
}

// Tea messages
type streamStartMsg struct {
	ch chan provider.StreamChunk
}

type streamChunkMsg struct {
	text string
	done bool
	err  error
}

func waitForStreamChunk(ch chan provider.StreamChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-ch
		if !ok {
			return streamChunkMsg{done: true}
		}
		return streamChunkMsg{text: chunk.Text, done: chunk.Done, err: chunk.Error}
	}
}

// NewModel creates a new TUI model
func NewModel() Model {
	conf, err := config.Load()
	if err != nil {
		fmt.Println("Config not found. Please run 'goclaw onboard' first.")
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
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	ta := textarea.New()
	ta.Placeholder = "Type a message... (/help for commands)"
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 2000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20)
	initMsg := fmt.Sprintf("🦞 GoClaw [%s] — Model: %s\nType /help for commands.", ws.Config.Name, modName)
	vp.SetContent(lipgloss.NewStyle().Width(80).Render(initMsg))

	history := []provider.ChatMessage{
		{Role: "system", Content: agent.BuildSystemPrompt(ws)},
	}

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(76),
	)

	conv := memory.Conversation{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		AgentID:   ws.Config.ID,
		AgentName: ws.Config.Name,
		Messages:  history,
		CreatedAt: time.Now(),
	}

	return Model{
		config:       conf,
		currentAgent: ws,
		currentProv:  prov,
		currentMod:   modName,
		textarea:     ta,
		chatHistory:  history,
		messages:     []string{initMsg},
		viewport:     vp,
		styles:       DefaultStyles(),
		mdRenderer:   renderer,
		conversation: conv,
	}
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// renderMarkdown renders markdown text for terminal display
func (m Model) renderMarkdown(text string) string {
	if m.mdRenderer == nil {
		return text
	}
	rendered, err := m.mdRenderer.Render(text)
	if err != nil {
		return text
	}
	return strings.TrimSpace(rendered)
}

func (m Model) appendSystem(msg string) Model {
	sysMsg := m.styles.System.Render("System: ") + msg
	m.messages = append(m.messages, sysMsg)
	m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n\n")))
	m.textarea.Reset()
	m.viewport.GotoBottom()
	return m
}

func (m Model) updateViewport() Model {
	if m.viewport.Width > 0 && m.viewport.Height > 0 {
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n\n")))
		m.viewport.GotoBottom()
	}
	return m
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 6
		m.textarea.SetWidth(msg.Width)
		m = m.updateViewport()
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			// Save conversation before exit
			m.conversation.Messages = m.chatHistory
			_ = memory.SaveConversation(m.conversation)
			return m, tea.Quit

		case tea.KeyEnter:
			if msg.Alt {
				return m, tiCmd
			}

			v := strings.TrimSpace(m.textarea.Value())
			if v == "" {
				return m, nil
			}

			if strings.HasPrefix(v, "/") {
				return m.processSlashCommand(v)
			}

			// Show user message
			userMsg := m.styles.Sender.Render("You: ") + v
			m.messages = append(m.messages, userMsg)
			m = m.updateViewport()
			m.textarea.Reset()

			// Push to chat history
			m.chatHistory = append(m.chatHistory, provider.ChatMessage{Role: "user", Content: v})

			// Token limiting (context window management)
			m.chatHistory = memory.TrimHistory(m.chatHistory, m.config.MaxTokens)

			m.isThinking = true
			m.streamingText = ""

			currentHistory := make([]provider.ChatMessage, len(m.chatHistory))
			copy(currentHistory, m.chatHistory)

			return m, func() tea.Msg {
				ch := make(chan provider.StreamChunk, 100)
				go m.currentProv.QueryStream(context.Background(), m.currentMod, currentHistory, ch)
				return streamStartMsg{ch: ch}
			}
		}

	case streamStartMsg:
		m.streamCh = msg.ch
		return m, waitForStreamChunk(m.streamCh)

	case streamChunkMsg:
		if msg.err != nil {
			m.isThinking = false
			errText := m.styles.Error.Render("Error: ") + msg.err.Error()
			m.messages = append(m.messages, errText)
			m = m.updateViewport()
			return m, nil
		}

		if msg.done {
			m.isThinking = false

			// Add complete response to chat history
			m.chatHistory = append(m.chatHistory, provider.ChatMessage{Role: "assistant", Content: m.streamingText})

			// Save conversation persistently
			m.conversation.Messages = m.chatHistory
			_ = memory.SaveConversation(m.conversation)

			// Render final markdown
			renderedText := m.renderMarkdown(m.streamingText)
			botPrefix := m.styles.Bot.Render(m.currentAgent.Config.Name + ": ")

			// Replace streaming display with final rendered version
			if len(m.messages) > 0 {
				last := m.messages[len(m.messages)-1]
				if strings.Contains(last, "▊") || strings.HasPrefix(last, botPrefix) {
					m.messages[len(m.messages)-1] = botPrefix + "\n" + renderedText
				} else {
					m.messages = append(m.messages, botPrefix+"\n"+renderedText)
				}
			} else {
				m.messages = append(m.messages, botPrefix+"\n"+renderedText)
			}

			m.streamingText = ""
			m = m.updateViewport()
			return m, nil
		}

		// Append streaming chunk
		m.streamingText += msg.text

		botPrefix := m.styles.Bot.Render(m.currentAgent.Config.Name + ": ")
		streamDisplay := botPrefix + m.streamingText + "▊"

		// Replace or add streaming message in display
		if len(m.messages) > 0 {
			last := m.messages[len(m.messages)-1]
			if strings.Contains(last, "▊") || (m.isThinking && strings.HasPrefix(last, botPrefix)) {
				m.messages[len(m.messages)-1] = streamDisplay
			} else {
				m.messages = append(m.messages, streamDisplay)
			}
		} else {
			m.messages = append(m.messages, streamDisplay)
		}

		m = m.updateViewport()
		return m, waitForStreamChunk(m.streamCh)
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m Model) View() string {
	thinking := ""
	if m.isThinking && m.streamingText == "" {
		thinking = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(
			fmt.Sprintf("\n\n%s (%s / %s) is thinking...", m.currentAgent.Config.Name, m.currentProv.Name(), m.currentMod))
	}

	return fmt.Sprintf(
		"%s%s\n\n%s",
		m.viewport.View(),
		thinking,
		m.textarea.View(),
	) + "\n\n"
}

// Run starts the TUI application
func Run() {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Failed to start TUI: %v\n", err)
		os.Exit(1)
	}
}
