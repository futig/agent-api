package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/futig/agent-backend/internal/telegram/keyboard"
	"github.com/futig/agent-backend/internal/telegram/render"
	"github.com/futig/agent-backend/internal/telegram/state"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// ContextHandler handles ASK_USER_CONTEXT state (manual project context)
type ContextHandler struct {
	BaseHandler
	bot          *tgbotapi.BotAPI
	stateManager *state.Manager
	sessionUC    SessionUsecase
	questions    []string
	keyboard     *keyboard.Builder
	logger       *zap.Logger
}

// NewContextHandler creates a new context handler
func NewContextHandler(
	bot *tgbotapi.BotAPI,
	stateManager *state.Manager,
	sessionUC SessionUsecase,
	questions []string,
	kb *keyboard.Builder,
	logger *zap.Logger,
) *ContextHandler {
	return &ContextHandler{
		BaseHandler: BaseHandler{
			stateName:     HandlerStateAskContext,
			messageSender: NewMessageSender(bot, logger),
		},
		bot:          bot,
		stateManager: stateManager,
		sessionUC:    sessionUC,
		questions:    questions,
		keyboard:     kb,
		logger:       logger,
	}
}

// Handle processes manual project context input (text or voice)
func (h *ContextHandler) Handle(ctx context.Context, msg *Message) error {
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get telegram session: %w", err)
	}

	sessionID := telegramSession.SessionID
	if sessionID == "" {
		return fmt.Errorf("session ID not found in telegram session")
	}

	if len(h.questions) == 0 {
		ctxzap.Error(ctx, "context questions not configured")
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	questionsText := formatContextQuestions(h.questions)

	// Handle voice message
	if msg.Voice != nil {
		ctxzap.Info(ctx, "processing voice project context",
			zap.Int64("user_id", msg.UserID),
			zap.String("session_id", sessionID),
		)

		audioData, err := downloadVoiceFile(ctx, h.bot, msg.Voice.FileID)
		if err != nil {
			ctxzap.Error(ctx, "failed to download context voice file",
				zap.Error(err),
				zap.String("file_id", msg.Voice.FileID),
			)
			h.sendMessage(msg.ChatID, render.ErrTranscription, nil)
			return nil
		}

		h.sendMessage(msg.ChatID, "🎤 Расшифровываю ответы о проекте...", nil)

		// Start progress notifier for long operation
		progress := NewProgressNotifier(h.bot, msg.ChatID)
		progress.Start(ctx)
		defer progress.Stop()

		if _, err := h.sessionUC.SubmitAudioUserProjectContext(ctx, sessionID, questionsText, audioData); err != nil {
			ctxzap.Error(ctx, "failed to submit audio project context",
				zap.Error(err),
				zap.String("session_id", sessionID),
			)
			h.sendMessage(msg.ChatID, render.ErrTranscription, nil)
			return nil
		}
	} else if msg.Text != "" {
		// Handle text message
		ctxzap.Info(ctx, "processing text project context",
			zap.Int64("user_id", msg.UserID),
			zap.String("session_id", sessionID),
			zap.String("answers", msg.Text),
		)

		if _, err := h.sessionUC.SubmitTextUserProjectContext(ctx, sessionID, questionsText, msg.Text); err != nil {
			h.HandleError(ctx, msg.ChatID, err)
			return nil
		}
	} else {
		h.sendMessage(msg.ChatID, "❌ Пожалуйста, отправьте текст или голосовое сообщение", nil)
		return nil
	}

	// After context is set, move to mode selection
	h.sendMessage(msg.ChatID, render.MsgChooseMode, h.keyboard.ModeSelectionKeyboard())

	return nil
}

func formatContextQuestions(questions []string) string {
	var b strings.Builder
	for i, q := range questions {
		b.WriteString(fmt.Sprintf("%d) %s\n\n", i+1, q))
	}
	return b.String()
}
