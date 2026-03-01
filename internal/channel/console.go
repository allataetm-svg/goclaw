package channel

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConsoleChannel implements Channel for CLI interaction
type ConsoleChannel struct {
	id      string
	name    string
	agentID string
	router  *Router
	stopCh  chan struct{}
}

func NewConsoleChannel(id, name, agentID string) *ConsoleChannel {
	return &ConsoleChannel{
		id:      id,
		name:    name,
		agentID: agentID,
		stopCh:  make(chan struct{}),
	}
}

func (c *ConsoleChannel) ID() string   { return c.id }
func (c *ConsoleChannel) Type() string { return "console" }
func (c *ConsoleChannel) Name() string { return c.name }

func (c *ConsoleChannel) Start(router *Router) error {
	c.router = router
	if c.agentID != "" {
		router.SetSession("user", c.agentID)
	}

	go c.run()
	return nil
}

func (c *ConsoleChannel) Stop() error {
	close(c.stopCh)
	return nil
}

func (c *ConsoleChannel) Send(toID string, text string) error {
	fmt.Printf("\n[%s] 🦞 Assistant: %s\n> ", c.name, text)
	return nil
}

func (c *ConsoleChannel) run() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("[%s] Console Channel Started. Type your message below.\n> ", c.name)

	for {
		select {
		case <-c.stopCh:
			return
		default:
			if !scanner.Scan() {
				return
			}
			text := strings.TrimSpace(scanner.Text())
			if text == "" {
				continue
			}

			c.router.HandleIncoming(Message{
				FromID:    "user",
				Text:      text,
				ChannelID: c.id,
			})
		}
	}
}
