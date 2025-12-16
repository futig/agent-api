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

// QuestionsHandler handles WAITING_FOR_ANSWERS state (Q&A loop)
type QuestionsHandler struct {
	BaseHandler
	bot          *tgbotapi.BotAPI
	stateManager *state.Manager
	sessionUC    SessionUsecase
	projectUC    ProjectUsecase
	keyboard     *keyboard.Builder
	logger       *zap.Logger
}

// NewQuestionsHandler creates a new questions handler
func NewQuestionsHandler(
	bot *tgbotapi.BotAPI,
	stateManager *state.Manager,
	sessionUC SessionUsecase,
	projectUC ProjectUsecase,
	kb *keyboard.Builder,
	logger *zap.Logger,
) *QuestionsHandler {
	return &QuestionsHandler{
		BaseHandler: BaseHandler{
			stateName:     HandlerStateWaitingAnswers,
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

// Handle processes answer submissions (text or voice)
func (h *QuestionsHandler) Handle(ctx context.Context, msg *Message) error {
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

	currentQuestionID := stateData.CurrentQuestionID
	if currentQuestionID == "" {
		h.sendMessage(msg.ChatID, "❌ Текущий вопрос не найден. Нажмите /start", nil)
		return nil
	}

	var nextIteration *entity.IterationWithQuestions

	// Handle voice message
	if msg.Voice != nil {
		ctxzap.Info(ctx, "processing voice answer",
			zap.Int64("user_id", msg.UserID),
			zap.String("question_id", currentQuestionID),
		)

		// Download voice file
		audioData, err := downloadVoiceFile(ctx, h.bot, msg.Voice.FileID)
		if err != nil {
			ctxzap.Error(ctx, "failed to download voice file",
				zap.Error(err),
			)
			h.sendMessage(msg.ChatID, render.ErrTranscription, nil)
			return nil
		}

		// Send processing message
		h.sendMessage(msg.ChatID, "🎤 Расшифровываю...", nil)

		// Start progress notifier for long operation
		progress := NewProgressNotifier(h.bot, msg.ChatID)
		progress.Start(ctx)
		defer progress.Stop()

		// Submit audio answer
		nextIteration, err = h.sessionUC.SubmitAudioAnswer(ctx, sessionID, currentQuestionID, audioData)
		if err != nil {
			ctxzap.Error(ctx, "failed to submit audio answer",
				zap.Error(err),
			)
			h.sendMessage(msg.ChatID, render.ErrTranscription, nil)
			return nil
		}
	} else if msg.Text != "" {
		// Handle text message
		ctxzap.Info(ctx, "processing text answer",
			zap.Int64("user_id", msg.UserID),
			zap.String("question_id", currentQuestionID),
		)

		nextIteration, err = h.sessionUC.SubmitTextAnswer(ctx, sessionID, currentQuestionID, msg.Text)
		if err != nil {
			h.HandleError(ctx, msg.ChatID, err)
			return nil
		}
	} else {
		h.sendMessage(msg.ChatID, "❌ Пожалуйста, отправьте текст или голосовое сообщение", nil)
		return nil
	}

	// Send acknowledgment (critical - must be delivered)
	sendCriticalMessage(h.bot, msg.ChatID, "✅ Принял ответ", nil, h.logger)

	// Defensive check: if AnsweringSkipped is true but TotalSkippedQuestions is 0,
	// we're not really in the skipped flow, so reset the flag
	if stateData.AnsweringSkipped && stateData.TotalSkippedQuestions == 0 {
		ctxzap.Warn(ctx, "AnsweringSkipped was true but TotalSkippedQuestions is 0, resetting flag",
			zap.Int64("user_id", msg.UserID),
			zap.String("session_id", sessionID),
		)
		stateData.AnsweringSkipped = false
		if err := h.stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
			ctxzap.Error(ctx, "failed to reset AnsweringSkipped flag",
				zap.Error(err),
				zap.Int64("user_id", msg.UserID),
			)
		}
	}

	// If we are in "answer skipped" flow, move to the next skipped/unanswered question
	if stateData.AnsweringSkipped {
		// Clear forward navigation - not applicable when answering skipped questions
		if len(stateData.NextQuestionIDs) > 0 {
			stateData.NextQuestionIDs = []string{}
			if err := h.stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
				ctxzap.Error(ctx, "failed to clear NextQuestionIDs",
					zap.Error(err),
					zap.Int64("user_id", msg.UserID),
				)
			}
		}

		_, err := handleNextSkippedQuestion(
			ctx,
			msg,
			sessionID,
			h.sessionUC,
			h.projectUC,
			h.stateManager,
			h.keyboard,
			h.bot,
			h.logger,
			h.sendMessage,
		)
		if err != nil {
			ctxzap.Error(ctx, "failed to handle next skipped question",
				zap.Error(err),
				zap.String("session_id", sessionID),
			)
			h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		}

		return nil
	}

	// Check if we need to return to a question in forward navigation stack
	if len(stateData.NextQuestionIDs) > 0 {
		// Pop the next question from forward stack
		nextQuestionID := stateData.NextQuestionIDs[len(stateData.NextQuestionIDs)-1]
		stateData.NextQuestionIDs = stateData.NextQuestionIDs[:len(stateData.NextQuestionIDs)-1]

		// Get question details
		question, err := h.sessionUC.GetQuestionByID(ctx, nextQuestionID)
		if err != nil {
			ctxzap.Error(ctx, "failed to get next question from forward stack",
				zap.Error(err),
				zap.String("question_id", nextQuestionID),
			)
			// If we can't get the question, clear the forward stack and continue normally
			stateData.NextQuestionIDs = []string{}
		} else {
			// Get iteration to show question index
			iteration, err := h.sessionUC.GetIterationByID(ctx, question.IterationID)
			if err != nil {
				ctxzap.Error(ctx, "failed to get iteration",
					zap.Error(err),
					zap.String("iteration_id", question.IterationID),
				)
			} else {
				// Find question index in iteration
				questionIndex := 0
				for i, q := range iteration.Questions {
					if q.ID == nextQuestionID {
						questionIndex = i + 1
						break
					}
				}

				title := ""
				if questionIndex == 1 {
					title = iteration.Title
				}

				questionText := render.RenderQuestion(
					title,
					questionIndex,
					len(iteration.Questions),
					question.Question,
				)

				// Track question history for back navigation (only one level)
				if stateData.CurrentQuestionID != "" {
					stateData.PreviousQuestionID = stateData.CurrentQuestionID
				}

				// Update state
				stateData.CurrentIterationID = question.IterationID
				stateData.CurrentQuestionID = nextQuestionID

				if err := h.stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
					ctxzap.Error(ctx, "failed to update state data",
						zap.Error(err),
						zap.Int64("user_id", msg.UserID),
					)
				}

				hasPrevious := stateData.PreviousQuestionID != ""
				h.sendMessage(msg.ChatID, questionText, h.keyboard.QuestionNavigationKeyboard(nextQuestionID, hasPrevious))

				return nil
			}
		}
	}

	// Regular flow: Check if there are more questions
	if nextIteration == nil || len(nextIteration.Questions) == 0 {
		ctxzap.Info(ctx, "no more questions, moving to validation",
			zap.String("session_id", sessionID),
		)

		h.sendMessage(msg.ChatID, render.MsgValidating, nil)

		if err := handleValidationAndSummaryCommon(
			ctx,
			msg,
			sessionID,
			h.sessionUC,
			h.projectUC,
			h.stateManager,
			h.keyboard,
			h.bot,
			h.logger,
			h.sendMessage,
		); err != nil {
			ctxzap.Error(ctx, "failed to validate answers or generate summary",
				zap.Error(err),
				zap.String("session_id", sessionID),
			)
			h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		}

		return nil
	}

	// Find first unanswered question
	var nextQuestion entity.QuestionDTO
	var found bool
	questionIndex := 0

	for i, q := range nextIteration.Questions {
		if q.Status == entity.AnswerStatusUnanswered {
			nextQuestion = q
			found = true
			questionIndex = i + 1
			break
		}
	}

	if !found {
		// All questions in this iteration are answered, trigger validation
		ctxzap.Warn(ctx, "all questions answered but iteration returned, running validation",
			zap.String("iteration_id", nextIteration.IterationID),
		)

		// Inform user that validation may take some time
		h.sendMessage(msg.ChatID, render.MsgValidating, nil)

		if err := handleValidationAndSummaryCommon(
			ctx,
			msg,
			sessionID,
			h.sessionUC,
			h.projectUC,
			h.stateManager,
			h.keyboard,
			h.bot,
			h.logger,
			h.sendMessage,
		); err != nil {
			ctxzap.Error(ctx, "failed to validate answers or generate summary",
				zap.Error(err),
				zap.String("session_id", sessionID),
			)
			h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		}

		return nil
	}
	title := ""
	if questionIndex == 1 {
		title = nextIteration.Title
	}

	questionText := render.RenderQuestion(
		title,
		questionIndex,
		len(nextIteration.Questions),
		nextQuestion.Question,
	)

	// Track question history for back navigation (only one level)
	if stateData.CurrentQuestionID != "" {
		// Save current question as previous before moving to next
		stateData.PreviousQuestionID = stateData.CurrentQuestionID
	}

	// Clear forward navigation stack since we're moving forward naturally
	stateData.NextQuestionIDs = []string{}

	// Update state data with new current question
	stateData.CurrentIterationID = nextIteration.IterationID
	stateData.CurrentQuestionID = nextQuestion.ID
	h.stateManager.UpdateStateData(ctx, msg.UserID, stateData)

	// Check if there is a previous question to show back button
	hasPrevious := stateData.PreviousQuestionID != ""
	h.sendMessage(msg.ChatID, questionText, h.keyboard.QuestionNavigationKeyboard(nextQuestion.ID, hasPrevious))

	return nil
}
