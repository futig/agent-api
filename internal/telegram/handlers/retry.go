package handlers

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

const (
	maxSendRetries    = 3
	retrySleepBase    = time.Second
	criticalRetryBase = 500 * time.Millisecond
)

// sendMessageWithRetry sends a message with retry logic for critical messages
func sendMessageWithRetry(
	bot *tgbotapi.BotAPI,
	chatID int64,
	text string,
	markup interface{},
	maxRetries int,
	logger *zap.Logger,
) error {
	msg := tgbotapi.NewMessage(chatID, text)
	if markup != nil {
		msg.ReplyMarkup = markup
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		_, err := bot.Send(msg)
		if err == nil {
			// Success
			if attempt > 0 {
				logger.Info("message sent after retry",
					zap.Int("attempt", attempt+1),
					zap.Int64("chat_id", chatID),
				)
			}
			return nil
		}

		lastErr = err

		// Don't retry on last attempt
		if attempt < maxRetries-1 {
			sleepDuration := retrySleepBase * time.Duration(attempt+1)
			logger.Warn("failed to send message, retrying",
				zap.Error(err),
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", maxRetries),
				zap.Duration("retry_in", sleepDuration),
				zap.Int64("chat_id", chatID),
			)
			time.Sleep(sleepDuration)
		}
	}

	// All retries failed
	logger.Error("failed to send message after all retries",
		zap.Error(lastErr),
		zap.Int("max_retries", maxRetries),
		zap.Int64("chat_id", chatID),
	)

	return lastErr
}

// sendCriticalMessage sends a critical message that must be delivered (e.g., confirmations)
func sendCriticalMessage(
	bot *tgbotapi.BotAPI,
	chatID int64,
	text string,
	markup interface{},
	logger *zap.Logger,
) error {
	return sendMessageWithRetry(bot, chatID, text, markup, maxSendRetries, logger)
}
