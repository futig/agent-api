package middleware

import (
	"fmt"
	"runtime/debug"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// RecoveryMiddleware recovers from panics
type RecoveryMiddleware struct {
	logger *zap.Logger
	bot    *tgbotapi.BotAPI
}

// NewRecoveryMiddleware creates a new recovery middleware
func NewRecoveryMiddleware(logger *zap.Logger, bot *tgbotapi.BotAPI) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		logger: logger,
		bot:    bot,
	}
}

// Handle recovers from panics
func (m *RecoveryMiddleware) Handle(update tgbotapi.Update, next func(tgbotapi.Update)) {
	defer func() {
		if r := recover(); r != nil {
			// Log the panic
			m.logger.Error("panic recovered in telegram handler",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())),
				zap.Int("update_id", update.UpdateID),
			)

			// Try to send error message to user
			var chatID int64
			if update.Message != nil {
				chatID = update.Message.Chat.ID
			} else if update.CallbackQuery != nil {
				chatID = update.CallbackQuery.Message.Chat.ID
			}

			if chatID != 0 {
				msg := tgbotapi.NewMessage(chatID, "❌ Произошла ошибка. Попробуйте ещё раз или нажмите /start")
				_, err := m.bot.Send(msg)
				if err != nil {
					m.logger.Error("failed to send error message",
						zap.Error(err),
						zap.Int64("chat_id", chatID),
					)
				}
			}
		}
	}()

	next(update)
}

// RecoverWithError recovers and returns error
func RecoverWithError() error {
	if r := recover(); r != nil {
		return fmt.Errorf("panic: %v", r)
	}
	return nil
}
