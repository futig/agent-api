package handlers

import (
	"context"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	progressInterval     = 10 * time.Second
	typingActionInterval = 4 * time.Second // Telegram typing expires after 5s
)

// ProgressNotifier sends periodic progress messages and typing indicators during long operations
type ProgressNotifier struct {
	bot            *tgbotapi.BotAPI
	chatID         int64
	progressTicker *time.Ticker
	typingTicker   *time.Ticker
	done           chan struct{}
	messages       []string
	index          int
	stopped        bool
}

// NewProgressNotifier creates a new progress notifier
func NewProgressNotifier(bot *tgbotapi.BotAPI, chatID int64) *ProgressNotifier {
	return &ProgressNotifier{
		bot:    bot,
		chatID: chatID,
		done:   make(chan struct{}),
		messages: []string{
			"⏳ Всё ещё обрабатываю...",
			"⏳ Это займёт ещё немного времени...",
			"⏳ Работаю над запросом...",
			"⏳ Почти готово...",
		},
	}
}

// Start begins sending periodic progress messages and typing indicators
func (pn *ProgressNotifier) Start(ctx context.Context) {
	pn.progressTicker = time.NewTicker(progressInterval)
	pn.typingTicker = time.NewTicker(typingActionInterval)

	// Send initial typing action
	pn.sendTypingAction()

	// Progress messages goroutine
	go func() {
		for {
			select {
			case <-pn.progressTicker.C:
				// Send next progress message
				message := pn.messages[pn.index%len(pn.messages)]
				pn.index++

				msg := tgbotapi.NewMessage(pn.chatID, message)
				pn.bot.Send(msg)

			case <-pn.done:
				return

			case <-ctx.Done():
				return
			}
		}
	}()

	// Typing indicator goroutine
	go func() {
		for {
			select {
			case <-pn.typingTicker.C:
				pn.sendTypingAction()

			case <-pn.done:
				return

			case <-ctx.Done():
				return
			}
		}
	}()
}

// sendTypingAction sends a "typing" action to show user the bot is working
func (pn *ProgressNotifier) sendTypingAction() {
	action := tgbotapi.NewChatAction(pn.chatID, tgbotapi.ChatTyping)
	pn.bot.Send(action)
}

// Stop stops sending progress messages and typing indicators
func (pn *ProgressNotifier) Stop() {
	if pn.stopped {
		return
	}

	pn.stopped = true

	if pn.progressTicker != nil {
		pn.progressTicker.Stop()
	}
	if pn.typingTicker != nil {
		pn.typingTicker.Stop()
	}
	close(pn.done)
}
