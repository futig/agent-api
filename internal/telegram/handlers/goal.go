package handlers

import (
	"context"
	"fmt"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/telegram/keyboard"
	"github.com/futig/agent-backend/internal/telegram/render"
	"github.com/futig/agent-backend/internal/telegram/state"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// GoalHandler handles ASK_USER_GOAL state
type GoalHandler struct {
	BaseHandler
	bot          *tgbotapi.BotAPI
	stateManager *state.Manager
	sessionUC    SessionUsecase
	projectUC    ProjectUsecase
	keyboard     *keyboard.Builder
	logger       *zap.Logger
}

// NewGoalHandler creates a new goal handler
func NewGoalHandler(
	bot *tgbotapi.BotAPI,
	stateManager *state.Manager,
	sessionUC SessionUsecase,
	projectUC ProjectUsecase,
	kb *keyboard.Builder,
	logger *zap.Logger,
) *GoalHandler {
	return &GoalHandler{
		BaseHandler: BaseHandler{
			stateName:     HandlerStateAskGoal,
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

// Handle processes user goal input (text or voice)
func (h *GoalHandler) Handle(ctx context.Context, msg *Message) error {
	// Get telegram session
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get telegram session: %w", err)
	}

	sessionID := telegramSession.SessionID
	if sessionID == "" {
		return fmt.Errorf("session ID not found in telegram session")
	}

	// Handle voice message
	if msg.Voice != nil {
		ctxzap.Info(ctx, "processing voice goal",
			zap.Int64("user_id", msg.UserID),
			zap.String("session_id", sessionID),
		)

		// Download voice file
		audioData, err := downloadVoiceFile(ctx, h.bot, msg.Voice.FileID)
		if err != nil {
			ctxzap.Error(ctx, "failed to download voice file",
				zap.Error(err),
				zap.String("file_id", msg.Voice.FileID),
			)
			h.sendMessage(msg.ChatID, render.ErrTranscription, nil)
			return nil
		}

		// Send processing message
		h.sendMessage(msg.ChatID, "🎤 Расшифровываю голосовое сообщение...", nil)

		// Start progress notifier for long operation
		progress := NewProgressNotifier(h.bot, msg.ChatID)
		progress.Start(ctx)
		defer progress.Stop()

		// Submit audio goal
		_, err = h.sessionUC.SubmitAudioUserGoal(ctx, sessionID, audioData)
		if err != nil {
			ctxzap.Error(ctx, "failed to submit audio goal",
				zap.Error(err),
				zap.String("session_id", sessionID),
			)
			h.sendMessage(msg.ChatID, render.ErrTranscription, nil)
			return nil
		}
	} else if msg.Text != "" {
		// Handle text message
		ctxzap.Info(ctx, "processing text goal",
			zap.Int64("user_id", msg.UserID),
			zap.String("session_id", sessionID),
		)

		_, err = h.sessionUC.SubmitTextUserGoal(ctx, sessionID, msg.Text)
		if err != nil {
			h.HandleError(ctx, msg.ChatID, err)
			return nil
		}
	} else {
		h.sendMessage(msg.ChatID, "❌ Пожалуйста, отправьте текст или голосовое сообщение", nil)
		return nil
	}

	// After goal is set, move to project selection stage
	if err := h.showProjectSelection(ctx, msg.UserID, msg.ChatID); err != nil {
		h.HandleError(ctx, msg.ChatID, err)
		return nil
	}

	return nil
}

// showProjectSelection lists projects with pagination and shows selection keyboard
func (h *GoalHandler) showProjectSelection(ctx context.Context, userID int64, chatID int64) error {
	const pageSize = 10

	// Get state data to get current page
	stateData, err := h.stateManager.GetStateData(ctx, userID)
	if err != nil {
		return fmt.Errorf("get state data: %w", err)
	}

	// Calculate offset
	page := stateData.ProjectListPage
	offset := page * pageSize

	// Fetch projects with one extra to check if there are more
	projects, err := h.projectUC.ListProjects(ctx, &entity.ListProjectsRequest{
		Skip:  offset,
		Limit: pageSize + 1, // Fetch one extra to check if there are more pages
	})
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	// Check if there are more pages
	hasNextPage := len(projects) > pageSize
	if hasNextPage {
		projects = projects[:pageSize] // Trim to page size
	}

	kbProjects := make([]keyboard.Project, 0, len(projects))
	for _, p := range projects {
		kbProjects = append(kbProjects, keyboard.Project{
			ID:    p.ID,
			Title: p.Title,
		})
	}

	hasPrevPage := page > 0
	h.sendMessage(chatID, render.MsgSelectProject, h.keyboard.ProjectSelectionKeyboardWithPagination(kbProjects, hasPrevPage, hasNextPage))
	return nil
}
