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

// DraftHandler handles DRAFT_COLLECTING state (free-form draft messages)
type DraftHandler struct {
	BaseHandler
	bot              *tgbotapi.BotAPI
	stateManager     *state.Manager
	sessionUC        SessionUsecase
	keyboard         *keyboard.Builder
	logger           *zap.Logger
	maxDraftMessages int
}

// NewDraftHandler creates a new draft handler
func NewDraftHandler(
	bot *tgbotapi.BotAPI,
	stateManager *state.Manager,
	sessionUC SessionUsecase,
	kb *keyboard.Builder,
	logger *zap.Logger,
	maxDraftMessages int,
) *DraftHandler {
	return &DraftHandler{
		BaseHandler: BaseHandler{
			stateName:     HandlerStateDraftCollecting,
			messageSender: NewMessageSender(bot, logger),
		},
		bot:              bot,
		stateManager:     stateManager,
		sessionUC:        sessionUC,
		keyboard:         kb,
		logger:           logger,
		maxDraftMessages: maxDraftMessages,
	}
}

// Handle processes draft messages (text or voice) in DRAFT_COLLECTING state
func (h *DraftHandler) Handle(ctx context.Context, msg *Message) error {
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get telegram session: %w", err)
	}

	sessionID := telegramSession.SessionID
	if sessionID == "" {
		return fmt.Errorf("session ID not found in telegram session")
	}

	stateData, err := h.stateManager.GetStateData(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get state data: %w", err)
	}

	// Enforce max draft messages
	maxMessages := h.maxDraftMessages
	if maxMessages <= 0 {
		maxMessages = 10
	}

	if stateData.DraftMessageCount >= maxMessages {
		h.sendMessage(msg.ChatID, render.RenderMaxDraftMessagesError(maxMessages), h.keyboard.DraftCollectionKeyboard())
		return nil
	}

	var createdMsg *entity.SessionMessage

	// Voice draft message
	if msg.Voice != nil {
		ctxzap.Info(ctx, "processing draft voice message",
			zap.Int64("user_id", msg.UserID),
			zap.String("session_id", sessionID),
		)

		audioData, err := downloadVoiceFile(ctx, h.bot, msg.Voice.FileID)
		if err != nil {
			ctxzap.Error(ctx, "failed to download draft voice file",
				zap.Error(err),
				zap.String("file_id", msg.Voice.FileID),
			)
			h.sendMessage(msg.ChatID, render.ErrTranscription, nil)
			return nil
		}

		h.sendMessage(msg.ChatID, "🎤 Расшифровываю голосовое сообщение...", nil)

		// Start progress notifier for long operation
		progress := NewProgressNotifier(h.bot, msg.ChatID)
		progress.Start(ctx)
		defer progress.Stop()

		createdMsg, err = h.sessionUC.AddAudioDraftMessage(ctx, sessionID, audioData)
		if err != nil {
			ctxzap.Error(ctx, "failed to add audio draft message",
				zap.Error(err),
				zap.String("session_id", sessionID),
			)
			h.sendMessage(msg.ChatID, render.ErrTranscription, nil)
			return nil
		}
	} else if msg.Text != "" {
		// Text draft message
		ctxzap.Info(ctx, "processing draft text message",
			zap.Int64("user_id", msg.UserID),
			zap.String("session_id", sessionID),
		)

		createdMsg, err = h.sessionUC.AddDraftMessage(ctx, sessionID, msg.Text)
		if err != nil {
			h.HandleError(ctx, msg.ChatID, err)
			return nil
		}
	} else {
		h.sendMessage(msg.ChatID, "❌ Пожалуйста, отправьте текст или голосовое сообщение", nil)
		return nil
	}

	if createdMsg == nil {
		ctxzap.Warn(ctx, "draft message created is nil",
			zap.String("session_id", sessionID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// Update draft counters in state
	stateData.DraftMessageCount++

	if err := h.stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
		ctxzap.Error(ctx, "failed to update draft state data",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
	}

	h.sendMessage(
		msg.ChatID,
		render.RenderDraftProgress(stateData.DraftMessageCount, maxMessages),
		h.keyboard.DraftCollectionKeyboard(),
	)

	return nil
}
