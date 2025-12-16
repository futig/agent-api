package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/telegram/keyboard"
	"github.com/futig/agent-backend/internal/telegram/state"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// ProjectNameHandler handles ASK_PROJECT_NAME state
type ProjectNameHandler struct {
	BaseHandler
	bot          *tgbotapi.BotAPI
	stateManager *state.Manager
	sessionUC    SessionUsecase
	logger       *zap.Logger
}

// NewProjectNameHandler creates a new project name handler
func NewProjectNameHandler(
	bot *tgbotapi.BotAPI,
	stateManager *state.Manager,
	sessionUC SessionUsecase,
	logger *zap.Logger,
) *ProjectNameHandler {
	return &ProjectNameHandler{
		BaseHandler: BaseHandler{
			stateName:     HandlerStateAskProjectName,
			messageSender: NewMessageSender(bot, logger),
		},
		bot:          bot,
		stateManager: stateManager,
		sessionUC:    sessionUC,
		logger:       logger,
	}
}

// Handle processes project name input
func (h *ProjectNameHandler) Handle(ctx context.Context, msg *Message) error {
	if msg.Text == "" {
		h.sendMessage(msg.ChatID, "❌ Пожалуйста, введите название проекта текстом.", nil)
		return nil
	}

	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get telegram session: %w", err)
	}

	// Save project name in state data
	stateData, err := h.stateManager.GetStateData(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get state data: %w", err)
	}

	stateData.ProjectName = msg.Text

	if err := h.stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
		return fmt.Errorf("update state data: %w", err)
	}

	// Change session status to ask for project description
	if _, err = h.sessionUC.UpdateSessionStatus(ctx, telegramSession.SessionID, entity.SessionStatusAskProjectDescription); err != nil {
		ctxzap.Error(ctx, "failed to update session status",
			zap.Error(err),
			zap.String("session_id", telegramSession.SessionID),
		)
		h.HandleError(ctx, msg.ChatID, err)
		return nil
	}

	h.sendMessage(msg.ChatID, "📝 Введите описание проекта:", nil)
	return nil
}

// ProjectDescriptionHandler handles ASK_PROJECT_DESCRIPTION state
type ProjectDescriptionHandler struct {
	BaseHandler
	bot          *tgbotapi.BotAPI
	stateManager *state.Manager
	sessionUC    SessionUsecase
	projectUC    ProjectUsecase
	keyboard     *keyboard.Builder
	logger       *zap.Logger
}

// NewProjectDescriptionHandler creates a new project description handler
func NewProjectDescriptionHandler(
	bot *tgbotapi.BotAPI,
	stateManager *state.Manager,
	sessionUC SessionUsecase,
	projectUC ProjectUsecase,
	kb *keyboard.Builder,
	logger *zap.Logger,
) *ProjectDescriptionHandler {
	return &ProjectDescriptionHandler{
		BaseHandler: BaseHandler{
			stateName:     HandlerStateAskProjectDescription,
			messageSender: NewMessageSender(bot, logger),
		},
		bot:          bot,
		stateManager: stateManager,
		sessionUC:    sessionUC,
		projectUC:    projectUC,
		keyboard:     kb,
		logger:       logger,
	}
}

// Handle processes project description input and creates the project
func (h *ProjectDescriptionHandler) Handle(ctx context.Context, msg *Message) error {
	if msg.Text == "" {
		h.sendMessage(msg.ChatID, "❌ Пожалуйста, введите описание проекта текстом.", nil)
		return nil
	}

	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get telegram session: %w", err)
	}

	sessionID := telegramSession.SessionID
	if sessionID == "" {
		return fmt.Errorf("session ID not found in telegram session")
	}

	// Get session to retrieve result
	session, err := h.sessionUC.GetSession(ctx, sessionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get session",
			zap.Error(err),
			zap.String("session_id", sessionID),
		)
		h.HandleError(ctx, msg.ChatID, err)
		return nil
	}

	if session.Result == nil || *session.Result == "" {
		h.sendMessage(msg.ChatID, "❌ Бизнес-требования еще не сформированы.", nil)
		return nil
	}

	// Get project name from state data
	stateData, err := h.stateManager.GetStateData(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get state data: %w", err)
	}

	if stateData.ProjectName == "" {
		h.sendMessage(msg.ChatID, "❌ Название проекта не найдено. Пожалуйста, начните сначала.", nil)
		return nil
	}

	h.sendMessage(msg.ChatID, fmt.Sprintf("💾 Создаю проект '%s'...", stateData.ProjectName), nil)

	// Start typing indicator
	typing := NewTypingNotifier(h.bot, msg.ChatID, h.logger)
	typing.Start(ctx)
	defer typing.Stop()

	// Create project with requirements file (indexed in RAG)
	fileName := fmt.Sprintf("requirements_%d.md", time.Now().Unix())
	project, err := h.projectUC.CreateProjectFromContent(
		ctx,
		stateData.ProjectName,
		msg.Text,
		fileName,
		[]byte(*session.Result),
		"text/markdown",
	)
	if err != nil {
		ctxzap.Error(ctx, "failed to create project with requirements",
			zap.Error(err),
			zap.String("title", stateData.ProjectName),
		)
		h.sendMessage(msg.ChatID, "❌ Не удалось создать проект.", nil)
		return nil
	}

	ctxzap.Info(ctx, "project created from telegram bot",
		zap.String("project_id", project.ID),
		zap.String("title", project.Title),
	)

	// Update session with new project ID
	session.ProjectID = &project.ID
	if _, err = h.sessionUC.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusDone); err != nil {
		ctxzap.Warn(ctx, "failed to update session status to done",
			zap.Error(err),
			zap.String("session_id", sessionID),
		)
	}

	// Clear state data
	stateData.ProjectName = ""
	if err := h.stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
		ctxzap.Warn(ctx, "failed to clear project name from state",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
	}

	typing.Stop()

	// Check if there are skipped questions
	hasSkipped, err := h.sessionUC.HasSkippedQuestions(ctx, sessionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to check skipped questions",
			zap.Error(err),
			zap.String("session_id", sessionID),
		)
	}

	// Show success message with download buttons
	successMsg := fmt.Sprintf("✅ Проект '%s' создан и требования сохранены!\n\nМожешь скачать их в удобном формате:", project.Title)
	h.sendMessage(msg.ChatID, successMsg, h.keyboard.ResultDownloadOnlyKeyboard(hasSkipped))
	return nil
}
