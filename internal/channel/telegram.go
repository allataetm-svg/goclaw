package channel

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramChannel implements Channel for Telegram interaction
type TelegramChannel struct {
	id      string
	name    string
	token   string
	agentID string
	router  *Router
	bot     *tgbotapi.BotAPI
}

func NewTelegramChannel(id, name, token, agentID string) *TelegramChannel {
	return &TelegramChannel{
		id:      id,
		name:    name,
		token:   token,
		agentID: agentID,
	}
}

func (c *TelegramChannel) ID() string   { return c.id }
func (c *TelegramChannel) Type() string { return "telegram" }
func (c *TelegramChannel) Name() string { return c.name }

func (c *TelegramChannel) Start(router *Router) error {
	c.router = router
	bot, err := tgbotapi.NewBotAPI(c.token)
	if err != nil {
		return fmt.Errorf("failed to create telegram bot: %w", err)
	}
	c.bot = bot

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	go func() {
		for update := range updates {
			if update.Message == nil {
				continue
			}

			// Use Chat ID as FromID to maintain session per chat
			fromID := fmt.Sprintf("%d", update.Message.Chat.ID)

			// If message is from a person, also use their name in logs if possible
			log.Printf("[%s] %s: %s", c.name, update.Message.From.UserName, update.Message.Text)

			c.router.HandleIncoming(Message{
				FromID:    fromID,
				Text:      update.Message.Text,
				ChannelID: c.id,
			})
		}
	}()

	return nil
}

func (c *TelegramChannel) Stop() error {
	if c.bot != nil {
		c.bot.StopReceivingUpdates()
	}
	return nil
}

func (c *TelegramChannel) Send(toID string, text string) error {
	var chatID int64
	_, err := fmt.Sscanf(toID, "%d", &chatID)
	if err != nil {
		return fmt.Errorf("invalid chat id: %w", err)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	_, err = c.bot.Send(msg)
	return err
}
