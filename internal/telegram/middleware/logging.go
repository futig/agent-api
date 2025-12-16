package middleware

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// LoggingMiddleware logs all incoming updates
type LoggingMiddleware struct {
	logger *zap.Logger
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(logger *zap.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// Handle logs the update
func (m *LoggingMiddleware) Handle(update tgbotapi.Update, next func(tgbotapi.Update)) {
	start := time.Now()

	// Extract update info
	var userID int64
	var chatID int64
	var messageType string

	if update.Message != nil {
		userID = update.Message.From.ID
		chatID = update.Message.Chat.ID

		if update.Message.Voice != nil {
			messageType = "voice"
		} else if update.Message.Document != nil {
			messageType = "document"
		} else if update.Message.Text != "" {
			messageType = "text"
		} else {
			messageType = "other"
		}
	} else if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
		chatID = update.CallbackQuery.Message.Chat.ID
		messageType = "callback"
	}

	m.logger.Info("telegram update received",
		zap.Int64("user_id", userID),
		zap.Int64("chat_id", chatID),
		zap.String("type", messageType),
		zap.Int("update_id", update.UpdateID),
	)

	// Call next handler
	next(update)

	// Log completion
	duration := time.Since(start)
	m.logger.Info("telegram update processed",
		zap.Int64("user_id", userID),
		zap.Int64("chat_id", chatID),
		zap.Duration("duration", duration),
	)
}
