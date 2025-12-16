package telegram

import (
	"context"
	"fmt"

	"github.com/futig/agent-backend/internal/config"
	"github.com/futig/agent-backend/internal/telegram/bot"
	"github.com/futig/agent-backend/internal/telegram/handlers"
	"github.com/futig/agent-backend/internal/telegram/state"
	"github.com/futig/agent-backend/internal/usecase/project"
	"go.uber.org/zap"
)

// Bot is the main telegram bot interface
type Bot interface {
	Start(ctx context.Context) error
	Stop() error
}

// NewBot initializes the telegram bot with all dependencies
func NewBot(
	cfg *config.TelegramConfig,
	contextQuestions []string,
	storage state.Storage,
	sessionUC handlers.SessionUsecase,
	projectUC *project.ProjectUsecase,
	logger *zap.Logger,
) (Bot, error) {
	// Create state manager
	stateManager := state.NewManager(storage)

	// Create bot instance
	b, err := bot.New(cfg, stateManager, sessionUC, projectUC, contextQuestions, logger)
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}

	// Register handlers
	registerHandlers(b, logger)

	logger.Info("telegram bot initialized successfully")

	return b, nil
}

// registerHandlers registers all handlers with the bot
func registerHandlers(b *bot.Bot, logger *zap.Logger) {
	// Get bot dependencies
	api := b.GetAPI()
	stateManager := b.GetStateManager()
	sessionUC := b.GetSessionUsecase()
	projectUC := b.GetProjectUsecase()
	keyboard := b.GetKeyboard()
	cfg := b.GetConfig()
	contextQuestions := b.GetContextQuestions()

	// Register callback handler (handles all button clicks)
	callbackHandler := handlers.NewCallbackHandler(api, stateManager, sessionUC, projectUC, contextQuestions, keyboard, logger)
	b.RegisterHandler(callbackHandler)

	// Register goal handler (ASK_USER_GOAL state)
	goalHandler := handlers.NewGoalHandler(api, stateManager, sessionUC, projectUC, keyboard, logger)
	b.RegisterHandler(goalHandler)

	// Register questions handler (WAITING_FOR_ANSWERS state)
	questionsHandler := handlers.NewQuestionsHandler(api, stateManager, sessionUC, projectUC, keyboard, logger)
	b.RegisterHandler(questionsHandler)

	// Register draft handler (DRAFT_COLLECTING state)
	draftHandler := handlers.NewDraftHandler(api, stateManager, sessionUC, keyboard, logger, cfg.MaxDraftMessages)
	b.RegisterHandler(draftHandler)

	// Register context handler (ASK_USER_CONTEXT state)
	contextHandler := handlers.NewContextHandler(api, stateManager, sessionUC, contextQuestions, keyboard, logger)
	b.RegisterHandler(contextHandler)

	// Register project name handler (ASK_PROJECT_NAME state)
	projectNameHandler := handlers.NewProjectNameHandler(api, stateManager, sessionUC, logger)
	b.RegisterHandler(projectNameHandler)

	// Register project description handler (ASK_PROJECT_DESCRIPTION state)
	projectDescriptionHandler := handlers.NewProjectDescriptionHandler(api, stateManager, sessionUC, projectUC, keyboard, logger)
	b.RegisterHandler(projectDescriptionHandler)

	logger.Info("telegram handlers registered",
		zap.Int("handler_count", 7),
	)

	// TODO: Optional handlers to implement:
	// - ProjectHandler (SELECT_OR_CREATE_PROJECT state)
	// - ContextHandler (ASK_USER_CONTEXT state)
	// - ResultHandler (DONE state) - for displaying results
}
