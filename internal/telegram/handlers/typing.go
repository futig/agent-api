package handlers

import (
	"context"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// TypingNotifier sends periodic "typing" actions to show bot activity
type TypingNotifier struct {
	bot     *tgbotapi.BotAPI
	chatID  int64
	ticker  *time.Ticker
	done    chan struct{}
	logger  *zap.Logger
	started bool
}

// NewTypingNotifier creates a new typing indicator
func NewTypingNotifier(bot *tgbotapi.BotAPI, chatID int64, logger *zap.Logger) *TypingNotifier {
	return &TypingNotifier{
		bot:    bot,
		chatID: chatID,
		done:   make(chan struct{}),
		logger: logger,
	}
}

// Start begins sending typing indicators every 4 seconds
// Telegram typing action expires after 5 seconds, so we send every 4 seconds
func (t *TypingNotifier) Start(ctx context.Context) {
	if t.started {
		return
	}

	t.started = true
	t.ticker = time.NewTicker(4 * time.Second)

	// Send initial typing action immediately
	action := tgbotapi.NewChatAction(t.chatID, tgbotapi.ChatTyping)
	if _, err := t.bot.Request(action); err != nil {
		t.logger.Warn("failed to send initial typing action",
			zap.Error(err),
			zap.Int64("chat_id", t.chatID),
		)
	}

	go func() {
		for {
			select {
			case <-t.ticker.C:
				action := tgbotapi.NewChatAction(t.chatID, tgbotapi.ChatTyping)
				if _, err := t.bot.Request(action); err != nil {
					t.logger.Warn("failed to send typing action",
						zap.Error(err),
						zap.Int64("chat_id", t.chatID),
					)
				}
			case <-t.done:
				t.ticker.Stop()
				return
			case <-ctx.Done():
				t.ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops sending typing indicators
func (t *TypingNotifier) Stop() {
	if !t.started {
		return
	}

	close(t.done)
	t.started = false
}
