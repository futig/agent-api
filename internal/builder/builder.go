package builder

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/futig/agent-backend/internal/api"
	projectapi "github.com/futig/agent-backend/internal/api/project"
	sessionapi "github.com/futig/agent-backend/internal/api/session"
	"github.com/futig/agent-backend/internal/config"
	"github.com/futig/agent-backend/internal/integration/asr"
	"github.com/futig/agent-backend/internal/integration/callback"
	"github.com/futig/agent-backend/internal/integration/llm"
	"github.com/futig/agent-backend/internal/integration/rag"
	"github.com/futig/agent-backend/internal/pkg/validator"
	"github.com/futig/agent-backend/internal/repository"
	"github.com/futig/agent-backend/internal/telegram"
	"github.com/futig/agent-backend/internal/usecase/project"
	"github.com/futig/agent-backend/internal/usecase/session"
	"go.uber.org/zap"
)

func Build() (*App, error) {
	ctx := context.Background()

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	logger, err := setupLogger(cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("setup logger: %w", err)
	}

	logger.Info("Building application",
		zap.String("environment", cfg.Environment),
		zap.String("server_addr", cfg.ServerAddr),
	)

	// Setup database connection
	db, err := setupDatabase(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("setup database: %w", err)
	}

	// Run database migrations
	logger.Info("Running database migrations")
	if err := repository.RunMigrations(cfg.DatabaseURL); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	logger.Info("Database migrations completed successfully")

	// Initialize repositories
	projectRepo := repository.NewProjectPostgres(db)
	projectFileRepo := repository.NewProjectFilePostgres(db)
	sessionRepo := repository.NewSessionPostgres(db)
	iterationRepo := repository.NewIterationPostgres(db)
	questionRepo := repository.NewQuestionPostgres(db)
	sessionMessageRepo := repository.NewSessionMessagePostgres(db)
	logger.Info("Repositories initialized")

	// Initialize connectors
	callbackConnector := callback.NewConnector(cfg.CallbackConnectorCfg, logger)

	// Initialize external service connectors (with mock support)
	var ragConnector project.RagConnector
	var llmConnector session.LLMConnector
	var asrConnector session.ASRConnector

	if cfg.EnableMocks {
		logger.Info("Using mock connectors for external services")
		ragConnector = rag.NewMockConnector(logger)
		llmConnector = llm.NewMockConnector(logger)
		asrConnector = asr.NewMockConnector(logger)
	} else {
		logger.Info("Using real connectors for external services")
		ragConnector = rag.NewConnector(cfg.RAGConnectorCfg, logger)
		llmConnector = llm.NewConnector(cfg.LLMConnectorCfg, logger)
		asrConnector = asr.NewConnector(cfg.ASRConnectorCfg, logger)
	}

	// Initialize validators
	fileValidator := validator.NewFileValidator(cfg.FileUploadCfg)
	logger.Info("Validators initialized")

	// Initialize use cases
	projectUC := project.NewUsecase(
		projectRepo,
		projectFileRepo,
		fileValidator,
		ragConnector,
		logger,
	)

	sessionUC := session.NewUsecase(
		sessionRepo,
		iterationRepo,
		questionRepo,
		projectRepo,
		sessionMessageRepo,
		fileValidator,
		ragConnector,
		llmConnector,
		asrConnector,
		logger,
	)
	logger.Info("Use cases initialized")

	// Setup API handlers
	projectHandler := projectapi.NewHandler(projectUC, cfg.FileUploadCfg, callbackConnector, fileValidator)
	sessionHandler := sessionapi.NewHandler(sessionUC, fileValidator, callbackConnector)
	logger.Info("API handlers initialized")

	// Setup router
	router := api.SetupRouter(projectHandler, sessionHandler, logger)
	logger.Info("HTTP router configured")

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("Application built successfully",
		zap.String("environment", cfg.Environment),
	)

	return &App{
		server: server,
		db:     db,
		logger: logger,
	}, nil
}

// BuildTelegramBot creates and initializes the Telegram bot
func BuildTelegramBot() (telegram.Bot, *zap.Logger, error) {
	ctx := context.Background()

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	logger, err := setupLogger(cfg.LogLevel)
	if err != nil {
		return nil, nil, fmt.Errorf("setup logger: %w", err)
	}

	logger.Info("Building Telegram bot",
		zap.String("environment", cfg.Environment),
	)

	// Setup database connection
	db, err := setupDatabase(ctx, cfg, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("setup database: %w", err)
	}

	// Run database migrations
	logger.Info("Running database migrations")
	if err := repository.RunMigrations(cfg.DatabaseURL); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("run migrations: %w", err)
	}
	logger.Info("Database migrations completed successfully")

	// Initialize repositories
	projectRepo := repository.NewProjectPostgres(db)
	projectFileRepo := repository.NewProjectFilePostgres(db)
	sessionRepo := repository.NewSessionPostgres(db)
	iterationRepo := repository.NewIterationPostgres(db)
	questionRepo := repository.NewQuestionPostgres(db)
	sessionMessageRepo := repository.NewSessionMessagePostgres(db)
	telegramStateRepo := repository.NewTelegramStateRepository(db)
	logger.Info("Repositories initialized")

	// Initialize connectors
	var ragConnector project.RagConnector
	var llmConnector session.LLMConnector
	var asrConnector session.ASRConnector

	if cfg.EnableMocks {
		logger.Info("Using mock connectors for external services")
		ragConnector = rag.NewMockConnector(logger)
		llmConnector = llm.NewMockConnector(logger)
		asrConnector = asr.NewMockConnector(logger)
	} else {
		logger.Info("Using real connectors for external services")
		ragConnector = rag.NewConnector(cfg.RAGConnectorCfg, logger)
		llmConnector = llm.NewConnector(cfg.LLMConnectorCfg, logger)
		asrConnector = asr.NewConnector(cfg.ASRConnectorCfg, logger)
	}

	// Initialize validators
	fileValidator := validator.NewFileValidator(cfg.FileUploadCfg)
	logger.Info("Validators initialized")

	// Initialize use cases
	projectUC := project.NewUsecase(
		projectRepo,
		projectFileRepo,
		fileValidator,
		ragConnector,
		logger,
	)

	sessionUC := session.NewUsecase(
		sessionRepo,
		iterationRepo,
		questionRepo,
		projectRepo,
		sessionMessageRepo,
		fileValidator,
		ragConnector,
		llmConnector,
		asrConnector,
		logger,
	)
	logger.Info("Use cases initialized")

	// Initialize Telegram bot
	bot, err := telegram.NewBot(&cfg.TelegramCfg, cfg.ContextQuestions, telegramStateRepo, sessionUC, projectUC, logger)
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("initialize telegram bot: %w", err)
	}

	logger.Info("Telegram bot built successfully",
		zap.String("environment", cfg.Environment),
	)

	return bot, logger, nil
}
