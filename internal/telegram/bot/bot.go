package bot

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/futig/agent-backend/internal/config"
	"github.com/futig/agent-backend/internal/telegram/handlers"
	"github.com/futig/agent-backend/internal/telegram/keyboard"
	"github.com/futig/agent-backend/internal/telegram/middleware"
	"github.com/futig/agent-backend/internal/telegram/render"
	"github.com/futig/agent-backend/internal/telegram/state"
	"github.com/futig/agent-backend/internal/usecase/project"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// Bot represents the Telegram bot
type Bot struct {
	api          *tgbotapi.BotAPI
	cfg          *config.TelegramConfig
	stateManager *state.Manager
	handlers     map[string]handlers.Handler
	sessionUC    handlers.SessionUsecase
	projectUC    *project.ProjectUsecase
	contextQ     []string
	keyboard     *keyboard.Builder
	logger       *zap.Logger
	loggingMW    *middleware.LoggingMiddleware
	recoveryMW   *middleware.RecoveryMiddleware
	rateLimitMW  *middleware.RateLimiterMiddleware
	updatesChan  tgbotapi.UpdatesChannel
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// New creates a new Telegram bot
func New(
	cfg *config.TelegramConfig,
	stateManager *state.Manager,
	sessionUC handlers.SessionUsecase,
	projectUC *project.ProjectUsecase,
	contextQuestions []string,
	logger *zap.Logger,
) (*Bot, error) {
	// Create bot API instance
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("create bot API: %w", err)
	}

	// Set debug mode in development
	api.Debug = false

	logger.Info("telegram bot authorized",
		zap.String("username", api.Self.UserName),
		zap.Int64("id", api.Self.ID),
	)

	bot := &Bot{
		api:          api,
		cfg:          cfg,
		stateManager: stateManager,
		sessionUC:    sessionUC,
		projectUC:    projectUC,
		contextQ:     contextQuestions,
		keyboard:     keyboard.NewBuilder(),
		logger:       logger,
		handlers:     make(map[string]handlers.Handler),
		stopChan:     make(chan struct{}),
	}

	// Initialize middleware
	bot.loggingMW = middleware.NewLoggingMiddleware(logger)
	bot.recoveryMW = middleware.NewRecoveryMiddleware(logger, api)
	bot.rateLimitMW = middleware.NewRateLimiterMiddleware(
		cfg.RateLimitPerMinute,
		cfg.RateLimitBurst,
		logger,
		api,
	)

	// Register handlers (will be implemented)
	// bot.registerHandlers()

	return bot, nil
}

// Start starts the bot
func (b *Bot) Start(ctx context.Context) error {
	b.logger.Info("starting telegram bot")

	// Configure updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = b.cfg.UpdateTimeout

	// Get updates channel
	updates := b.api.GetUpdatesChan(u)
	b.updatesChan = updates

	// Add logger to context for processUpdates
	ctx = ctxzap.ToContext(ctx, b.logger)

	// Start update processing loop
	go b.processUpdates(ctx)

	b.logger.Info("telegram bot started successfully")
	return nil
}

// Stop stops the bot gracefully with timeout
func (b *Bot) Stop() error {
	b.logger.Info("stopping telegram bot")

	// Signal to stop receiving new updates
	close(b.stopChan)
	b.api.StopReceivingUpdates()

	// Wait for all active handlers to complete
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	// Wait with timeout
	shutdownTimeout := time.Duration(b.cfg.ShutdownTimeout) * time.Second
	select {
	case <-done:
		b.logger.Info("all handlers completed gracefully")
	case <-time.After(shutdownTimeout):
		b.logger.Warn("shutdown timeout exceeded, some handlers may not have completed",
			zap.Duration("timeout", shutdownTimeout),
		)
		return fmt.Errorf("shutdown timeout exceeded")
	}

	b.logger.Info("telegram bot stopped successfully")
	return nil
}

// processUpdates processes incoming updates
func (b *Bot) processUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			ctxzap.Info(ctx, "context cancelled, stopping update processing")
			return
		case <-b.stopChan:
			ctxzap.Info(ctx, "stop signal received, stopping update processing")
			return
		case update := <-b.updatesChan:
			// Process update with middleware in separate goroutine
			b.wg.Add(1)
			go func(u tgbotapi.Update) {
				defer b.wg.Done()
				b.handleUpdateWithMiddleware(u)
			}(update)
		}
	}
}

// handleUpdateWithMiddleware processes update through middleware chain
func (b *Bot) handleUpdateWithMiddleware(update tgbotapi.Update) {
	// Rate limiter middleware (first to check)
	b.rateLimitMW.Handle(update, func(u tgbotapi.Update) {
		// Logging middleware
		b.loggingMW.Handle(u, func(u2 tgbotapi.Update) {
			// Recovery middleware
			b.recoveryMW.Handle(u2, func(u3 tgbotapi.Update) {
				// Actual handler
				b.handleUpdate(u3)
			})
		})
	})
}

// handleUpdate routes update to appropriate handler
func (b *Bot) handleUpdate(update tgbotapi.Update) {
	// Create context with logger
	ctx := ctxzap.ToContext(context.Background(), b.logger)

	// Handle callback queries
	if update.CallbackQuery != nil {
		b.handleCallbackQuery(ctx, update.CallbackQuery)
		return
	}

	// Handle messages
	if update.Message != nil {
		b.handleMessage(ctx, update.Message)
		return
	}
}

// handleMessage handles incoming messages
func (b *Bot) handleMessage(ctx context.Context, message *tgbotapi.Message) {
	// Handle commands
	if message.IsCommand() {
		b.handleCommand(ctx, message)
		return
	}

	// Get telegram session with joined session data (single query)
	userID := message.From.ID
	sessionData, err := b.stateManager.GetSessionWithSession(ctx, userID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get telegram session",
			zap.Error(err),
			zap.Int64("user_id", userID),
		)
		b.sendError(message.Chat.ID, render.ErrGeneric)
		return
	}

	// Check if user has active session
	if sessionData.SessionID == "" {
		ctxzap.Warn(ctx, "no active session for user",
			zap.Int64("user_id", userID),
		)
		b.sendError(message.Chat.ID, "Нет активной сессии. Используйте /start")
		return
	}

	// Load StateData once and attach to context for request-scoped caching
	stateData, err := b.stateManager.GetStateData(ctx, userID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get state data",
			zap.Error(err),
			zap.Int64("user_id", userID),
		)
		b.sendError(message.Chat.ID, render.ErrGeneric)
		return
	}
	ctx = state.ContextWithStateData(ctx, stateData)

	// Route to state-specific handler based on session status
	handler, exists := b.handlers[sessionData.SessionStatus]
	if !exists {
		ctxzap.Warn(ctx, "no handler for state",
			zap.String("state", sessionData.SessionStatus),
			zap.Int64("user_id", userID),
		)
		b.sendError(message.Chat.ID, render.ErrInvalidState)
		return
	}

	// Create normalized message
	msg := &handlers.Message{
		ChatID:    message.Chat.ID,
		UserID:    message.From.ID,
		MessageID: message.MessageID,
		Text:      message.Text,
		Voice:     message.Voice,
		Document:  message.Document,
	}

	// Handle message
	if err := handler.Handle(ctx, msg); err != nil {
		ctxzap.Error(ctx, "handler error",
			zap.Error(err),
			zap.String("state", sessionData.SessionStatus),
			zap.Int64("user_id", userID),
		)
		b.sendError(message.Chat.ID, render.ErrGeneric)
	}
}

// handleCommand handles bot commands
func (b *Bot) handleCommand(ctx context.Context, message *tgbotapi.Message) {
	command := message.Command()

	ctxzap.Info(ctx, "command received",
		zap.String("command", command),
		zap.Int64("user_id", message.From.ID),
	)

	switch command {
	case "start":
		b.handleStartCommand(ctx, message)
	case "help":
		b.handleHelpCommand(ctx, message)
	case "cancel":
		b.handleCancelCommand(ctx, message)
	default:
		b.sendError(message.Chat.ID, "❌ Неизвестная команда. Используйте /start")
	}
}

// handleStartCommand handles /start command
func (b *Bot) handleStartCommand(ctx context.Context, message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Show welcome message with "start session" button.
	if _, err := b.sendMessage(chatID, render.MsgWelcome, b.keyboard.StartKeyboard()); err != nil {
		ctxzap.Error(ctx, "failed to send welcome message",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
	}
}

// handleHelpCommand handles /help command
func (b *Bot) handleHelpCommand(ctx context.Context, message *tgbotapi.Message) {
	helpText := `🤖 **Команды бота:**

/start - Начать новую сессию
/help - Показать эту справку
/cancel - Отменить текущую сессию

**Как это работает:**
1. Опиши цель проекта
2. Выбери существующий проект или создай контекст вручную
3. Выбери режим: Интервью или Драфт
4. Ответь на вопросы или пришли материалы
5. Получи готовые бизнес-требования

Начни с /start`

	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ParseMode = "Markdown"
	if _, err := b.api.Send(msg); err != nil {
		ctxzap.Error(ctx, "failed to send help message",
			zap.Error(err),
		)
	}
}

// handleCancelCommand handles /cancel command
func (b *Bot) handleCancelCommand(ctx context.Context, message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID

	// Get telegram session
	telegramSession, err := b.stateManager.GetSession(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "Нет активной сессии. Используйте /start", nil)
		return
	}

	if telegramSession.SessionID == "" {
		b.sendMessage(chatID, "Нет активной сессии. Используйте /start", nil)
		return
	}

	// Get state data to check if confirmation is pending
	stateData, err := b.stateManager.GetStateData(ctx, userID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get state data", zap.Error(err))
		stateData = &state.StateData{}
	}

	// Check if already confirmed
	if stateData.PendingConfirmation != "cancel" {
		// First time - ask for confirmation
		stateData.PendingConfirmation = "cancel"
		b.stateManager.UpdateStateData(ctx, userID, stateData)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Да, завершить", "confirm:cancel"),
				tgbotapi.NewInlineKeyboardButtonData("❌ Нет, продолжить", "confirm:continue"),
			),
		)
		b.sendMessage(chatID, "⚠️ Вы уверены? Весь прогресс будет потерян.", keyboard)
		return
	}

	// Already confirmed - perform cancellation
	performCancellation(ctx, b, telegramSession.SessionID, userID, chatID)
}

func performCancellation(ctx context.Context, b *Bot, sessionID string, userID int64, chatID int64) {
	// Cancel session if exists
	if sessionID != "" {
		if err := b.sessionUC.CancelSession(ctx, sessionID); err != nil {
			ctxzap.Error(ctx, "failed to cancel session",
				zap.Error(err),
				zap.String("session_id", sessionID),
			)
		}
	}

	// Delete telegram session
	if err := b.stateManager.DeleteSession(ctx, userID); err != nil {
		ctxzap.Error(ctx, "failed to delete telegram session",
			zap.Error(err),
			zap.Int64("user_id", userID),
		)
	}

	b.sendMessage(chatID, render.MsgSessionFinished, nil)
}

// handleCallbackQuery handles callback button clicks
func (b *Bot) handleCallbackQuery(ctx context.Context, query *tgbotapi.CallbackQuery) {
	// Parse callback data
	callbackData, err := keyboard.ParseCallback(query.Data)
	if err != nil {
		ctxzap.Error(ctx, "invalid callback data",
			zap.Error(err),
			zap.String("data", query.Data),
		)
		// Быстрый ответ на некорректный callback
		b.answerCallback(query.ID, "❌ Неверные данные")
		return
	}

	ctxzap.Info(ctx, "callback query received",
		zap.String("action", callbackData.Action),
		zap.String("value", callbackData.Value),
		zap.Int64("user_id", query.From.ID),
	)

	// Route callback to handler
	// This will be implemented in callback handler
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	// Create normalized message
	msg := &handlers.Message{
		ChatID:       chatID,
		UserID:       userID,
		MessageID:    query.Message.MessageID,
		CallbackData: query.Data,
		CallbackID:   query.ID,
	}

	// Parse callback data to determine action
	callbackData, parseErr := keyboard.ParseCallback(query.Data)
	if parseErr != nil {
		// If parsing fails, try to load StateData anyway (safer)
		stateData, err := b.stateManager.GetStateData(ctx, userID)
		if err != nil {
			ctxzap.Error(ctx, "failed to get state data",
				zap.Error(err),
				zap.Int64("user_id", userID),
			)
			b.answerCallback(query.ID, "❌ Ошибка")
			return
		}
		ctx = state.ContextWithStateData(ctx, stateData)
	} else if !(callbackData.Action == "action" && callbackData.Value == "start") {
		// For "action:start" callback, we don't need existing StateData (creating new session)
		// For other actions, load StateData
		// Load StateData once and attach to context for request-scoped caching
		stateData, err := b.stateManager.GetStateData(ctx, userID)
		if err != nil {
			ctxzap.Error(ctx, "failed to get state data",
				zap.Error(err),
				zap.Int64("user_id", userID),
			)
			b.answerCallback(query.ID, "❌ Ошибка")
			return
		}
		ctx = state.ContextWithStateData(ctx, stateData)
	}

	// Get callback handler
	handler, exists := b.handlers["CALLBACK"]
	if !exists {
		ctxzap.Warn(ctx, "callback handler not registered")
		b.answerCallback(query.ID, "❌ Обработчик не найден")
		return
	}

	// Сразу отвечаем на callback, чтобы Telegram не считал запрос "устаревшим"
	b.answerCallback(query.ID, "⏳ Обрабатываю запрос...")

	// Дальнейшая тяжёлая обработка выполняется асинхронно,
	// а результаты/ошибки отправляются как обычные сообщения в чат.
	go func(ctx context.Context, m *handlers.Message, uid, cid int64) {
		if err := handler.Handle(ctx, m); err != nil {
			ctxzap.Error(ctx, "callback handler error",
				zap.Error(err),
				zap.Int64("user_id", uid),
			)
			// Сообщаем об ошибке в чат, чтобы пользователь видел результат
			b.sendError(cid, render.ErrGeneric)
		}
	}(ctx, msg, userID, chatID)
}

// sendMessage sends a message to chat
func (b *Bot) sendMessage(chatID int64, text string, replyMarkup interface{}) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	if replyMarkup != nil {
		msg.ReplyMarkup = replyMarkup
	}
	return b.api.Send(msg)
}

// sendError sends an error message
func (b *Bot) sendError(chatID int64, text string) {
	if _, err := b.sendMessage(chatID, text, nil); err != nil {
		b.logger.Error("failed to send error message",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
	}
}

// sendDocument sends a document
func (b *Bot) SendDocument(chatID int64, filename string, data []byte) error {
	doc := tgbotapi.FileBytes{
		Name:  filename,
		Bytes: data,
	}

	msg := tgbotapi.NewDocument(chatID, doc)
	if _, err := b.api.Send(msg); err != nil {
		return fmt.Errorf("send document: %w", err)
	}

	return nil
}

// answerCallback answers a callback query
func (b *Bot) answerCallback(callbackID string, text string) {
	callback := tgbotapi.NewCallback(callbackID, text)
	if _, err := b.api.Request(callback); err != nil {
		b.logger.Error("failed to answer callback",
			zap.Error(err),
			zap.String("callback_id", callbackID),
		)
	}
}

// RegisterHandler registers a handler for a state
func (b *Bot) RegisterHandler(handler handlers.Handler) {
	state := handler.GetState()

	// Validate state
	if !handlers.IsValidState(state) {
		b.logger.Fatal("invalid handler state",
			zap.String("state", state),
		)
	}

	b.handlers[state] = handler
	b.logger.Info("handler registered",
		zap.String("state", state),
	)
}

// GetAPI returns the bot API instance (for handlers)
func (b *Bot) GetAPI() *tgbotapi.BotAPI {
	return b.api
}

// GetStateManager returns the state manager (for handlers)
func (b *Bot) GetStateManager() *state.Manager {
	return b.stateManager
}

// GetKeyboard returns the keyboard builder (for handlers)
func (b *Bot) GetKeyboard() *keyboard.Builder {
	return b.keyboard
}

// GetSessionUsecase returns the session usecase (for handlers)
func (b *Bot) GetSessionUsecase() handlers.SessionUsecase {
	return b.sessionUC
}

// GetProjectUsecase returns the project usecase (for handlers)
func (b *Bot) GetProjectUsecase() *project.ProjectUsecase {
	return b.projectUC
}

// GetConfig returns the bot config (for handlers)
func (b *Bot) GetConfig() *config.TelegramConfig {
	return b.cfg
}

// GetContextQuestions returns preloaded context questions for Telegram flow
func (b *Bot) GetContextQuestions() []string {
	return b.contextQ
}
