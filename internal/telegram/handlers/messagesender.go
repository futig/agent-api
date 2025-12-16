package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// MessageSender provides centralized message sending functionality
type MessageSender struct {
	bot    *tgbotapi.BotAPI
	logger *zap.Logger
}

// NewMessageSender creates a new MessageSender
func NewMessageSender(bot *tgbotapi.BotAPI, logger *zap.Logger) *MessageSender {
	return &MessageSender{
		bot:    bot,
		logger: logger,
	}
}

// Send sends a message to the specified chat
func (s *MessageSender) Send(chatID int64, text string, markup interface{}) error {
	msg := tgbotapi.NewMessage(chatID, text)
	if markup != nil {
		msg.ReplyMarkup = markup
	}

	_, err := s.bot.Send(msg)
	if err != nil {
		s.logger.Error("failed to send message",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
		return err
	}

	return nil
}
