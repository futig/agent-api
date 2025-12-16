package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/pkg/formatter"
	"github.com/futig/agent-backend/internal/telegram/keyboard"
	"github.com/futig/agent-backend/internal/telegram/render"
	"github.com/futig/agent-backend/internal/telegram/state"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// CallbackHandler handles all callback button clicks
type CallbackHandler struct {
	BaseHandler
	bot          *tgbotapi.BotAPI
	stateManager *state.Manager
	sessionUC    SessionUsecase
	projectUC    ProjectUsecase
	keyboard     *keyboard.Builder
	logger       *zap.Logger
	questions    []string
}

// NewCallbackHandler creates a new callback handler
func NewCallbackHandler(
	bot *tgbotapi.BotAPI,
	stateManager *state.Manager,
	sessionUC SessionUsecase,
	projectUC ProjectUsecase,
	questions []string,
	kb *keyboard.Builder,
	logger *zap.Logger,
) *CallbackHandler {
	return &CallbackHandler{
		BaseHandler: BaseHandler{
			stateName:     HandlerStateCallback, // Special state for callbacks
			messageSender: NewMessageSender(bot, logger),
		},
		bot:          bot,
		stateManager: stateManager,
		sessionUC:    sessionUC,
		projectUC:    projectUC,
		keyboard:     kb,
		logger:       logger,
		questions:    questions,
	}
}

// Handle routes callback queries to appropriate actions
func (h *CallbackHandler) Handle(ctx context.Context, msg *Message) error {
	// Parse callback data
	data, err := keyboard.ParseCallback(msg.CallbackData)
	if err != nil {
		ctxzap.Error(ctx, "failed to parse callback",
			zap.Error(err),
			zap.String("data", msg.CallbackData),
		)
		return fmt.Errorf("parse callback: %w", err)
	}

	ctxzap.Info(ctx, "handling callback",
		zap.String("action", data.Action),
		zap.String("value", data.Value),
		zap.Int64("user_id", msg.UserID),
	)

	// Route based on action
	switch data.Action {
	case "action":
		return h.handleAction(ctx, msg, data.Value)
	case "mode":
		return h.handleModeSelection(ctx, msg, data.Value)
	case "proj":
		return h.handleProjectSelection(ctx, msg, data.Value)
	case "skip":
		return h.handleSkipQuestion(ctx, msg, data.Value)
	case "prev":
		return h.handlePreviousQuestion(ctx, msg, data.Value)
	case "explain":
		return h.handleExplainQuestion(ctx, msg, data.Value)
	case "dl":
		return h.handleDownload(ctx, msg, data.Value)
	case "confirm":
		return h.handleConfirmation(ctx, msg, data.Value)
	case "page":
		return h.handlePageNavigation(ctx, msg, data.Value)
	default:
		ctxzap.Warn(ctx, "unknown callback action",
			zap.String("action", data.Action),
		)
		return fmt.Errorf("unknown action: %s", data.Action)
	}
}

// handleAction handles general actions
func (h *CallbackHandler) handleAction(ctx context.Context, msg *Message, value string) error {
	switch value {
	case "start":
		// Start button clicked
		return h.handleStart(ctx, msg)
	case "start_interview":
		// Begin interview
		return h.handleStartInterview(ctx, msg)
	case "start_draft":
		// Begin draft mode
		return h.handleStartDraft(ctx, msg)
	case "choose_mode":
		// Return to mode selection
		return h.handleChooseMode(ctx, msg)
	case "generate":
		// Force generate requirements
		return h.handleGenerate(ctx, msg)
	case "finish":
		// Finish session
		return h.handleFinish(ctx, msg)
	case "change_project":
		// Change project selection
		return h.handleChangeProject(ctx, msg)
	case "answer_skipped":
		// Return to skipped questions
		return h.handleAnswerSkipped(ctx, msg)
	case "save_new_project":
		// Save requirements to a new project
		return h.handleSaveNewProject(ctx, msg)
	case "save_to_project":
		// Save requirements to existing project
		return h.handleSaveToProject(ctx, msg)
	default:
		return fmt.Errorf("unknown action value: %s", value)
	}
}

// handleModeSelection handles Interview/Draft mode selection
func (h *CallbackHandler) handleModeSelection(ctx context.Context, msg *Message, value string) error {
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get user state: %w", err)
	}

	var sessionType entity.SessionType
	switch value {
	case "interview":
		sessionType = entity.SessionTypeInterview
	case "draft":
		sessionType = entity.SessionTypeDraft
	default:
		return fmt.Errorf("invalid mode: %s", value)
	}

	// Set session type
	_, err = h.sessionUC.SetSessionType(ctx, telegramSession.SessionID, sessionType)
	if err != nil {
		h.HandleError(ctx, msg.ChatID, err)
		return nil
	}

	// Send appropriate info message
	if sessionType == entity.SessionTypeInterview {
		// Show interview info
		infoText := render.RenderInterviewInfo(15, 3, 10) // Example values
		h.sendMessage(msg.ChatID, infoText, h.keyboard.InterviewInfoKeyboard())
	} else {
		// Show draft info
		infoText := render.RenderDraftInfo(30) // Example value for max draft messages
		h.sendMessage(msg.ChatID, infoText, h.keyboard.DraftInfoKeyboard())
	}

	return nil
}

// handleStartInterview handles starting the interview
func (h *CallbackHandler) handleStartInterview(ctx context.Context, msg *Message) error {
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get user state: %w", err)
	}

	// Send processing message
	h.sendMessage(msg.ChatID, "⏳ Генерирую вопросы...", nil)

	// Start progress notifier for long operation
	progress := NewProgressNotifier(h.bot, msg.ChatID)
	progress.Start(ctx)
	defer progress.Stop()

	// Load questions
	iterations, err := h.sessionUC.LoadSessionQuestions(ctx, telegramSession.SessionID)
	if err != nil {
		h.HandleError(ctx, msg.ChatID, err)
		return nil
	}

	// If no questions generated, inform user
	if len(iterations) == 0 {
		h.sendMessage(msg.ChatID, "❌ Не удалось сгенерировать вопросы. Попробуйте ещё раз.", nil)
		return nil
	}

	// Calculate total questions and blocks
	totalQuestions := 0
	for _, it := range iterations {
		totalQuestions += len(it.Questions)
	}
	blockCount := len(iterations)

	// Inform user about total questions and blocks
	summaryText := fmt.Sprintf(
		"🧩 Я подготовил для тебя %d вопросов в %d блоках.",
		totalQuestions,
		blockCount,
	)
	h.sendMessage(msg.ChatID, summaryText, nil)

	// Send first question
	firstIteration := iterations[0]
	if len(firstIteration.Questions) > 0 {
		firstQuestion := firstIteration.Questions[0]
		questionText := render.RenderQuestion(
			firstIteration.Title,
			1,
			len(firstIteration.Questions),
			firstQuestion.Question,
		)

		// Get existing state data to preserve history
		stateData, err := h.stateManager.GetStateData(ctx, msg.UserID)
		if err != nil {
			ctxzap.Error(ctx, "failed to get state data",
				zap.Error(err),
				zap.Int64("user_id", msg.UserID),
			)
			h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
			return nil
		}

		// Clear previous history and skipped questions state when starting new interview
		stateData.PreviousQuestionID = ""
		stateData.NextQuestionIDs = []string{}
		stateData.AnsweringSkipped = false
		stateData.TotalSkippedQuestions = 0
		stateData.CurrentSkippedQuestionNumber = 0
		stateData.SkippedQuestionIDs = nil
		stateData.CurrentSkippedQuestionIndex = 0
		stateData.CurrentIterationID = iterations[0].IterationID
		stateData.CurrentQuestionID = firstQuestion.ID

		h.stateManager.UpdateStateData(ctx, msg.UserID, stateData)

		// First question has no previous
		h.sendMessage(msg.ChatID, questionText, h.keyboard.QuestionNavigationKeyboard(firstQuestion.ID, false))
	}

	return nil
}

// handleStartDraft handles starting draft mode
func (h *CallbackHandler) handleStartDraft(ctx context.Context, msg *Message) error {
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get user state: %w", err)
	}

	// Switch backend session status to DRAFT_COLLECTING
	if _, err := h.sessionUC.StartDraftCollecting(ctx, telegramSession.SessionID); err != nil {
		ctxzap.Error(ctx, "failed to start draft collecting",
			zap.Error(err),
			zap.String("session_id", telegramSession.SessionID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// Initialize draft message counter
	stateData := &state.StateData{
		DraftMessageCount: 0,
	}
	h.stateManager.UpdateStateData(ctx, msg.UserID, stateData)

	// Начальное сообщение без кнопок — просто просим присылать материалы
	h.sendMessage(msg.ChatID, "📄 Отлично! Начинай присылать материалы.", nil)

	return nil
}

// handleSkipQuestion handles skipping a question
func (h *CallbackHandler) handleSkipQuestion(ctx context.Context, msg *Message, questionID string) error {
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get user state: %w", err)
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

	// If we are answering previously skipped questions, move to the next skipped one
	if stateData.AnsweringSkipped {
		_, err := handleSkipCurrentQuestion(
			ctx,
			msg,
			telegramSession.SessionID,
			currentQuestionID,
			h.sessionUC,
			h.projectUC,
			h.stateManager,
			h.keyboard,
			h.bot,
			h.logger,
			h.sendMessage,
		)
		if err != nil {
			ctxzap.Error(ctx, "failed to handle skip in answering skipped mode",
				zap.Error(err),
				zap.String("session_id", telegramSession.SessionID),
			)
			h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		}

		return nil
	}

	// Skip the question
	nextIteration, err := h.sessionUC.SkipAnswer(ctx, telegramSession.SessionID, questionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to skip question",
			zap.Error(err),
			zap.String("question_id", questionID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// If no more questions, move to validation
	if nextIteration == nil || len(nextIteration.Questions) == 0 {
		h.sendMessage(msg.ChatID, render.MsgValidating, nil)

		if err := handleValidationAndSummaryCommon(
			ctx,
			msg,
			telegramSession.SessionID,
			h.sessionUC,
			h.projectUC,
			h.stateManager,
			h.keyboard,
			h.bot,
			h.logger,
			h.sendMessage,
		); err != nil {
			ctxzap.Error(ctx, "failed to validate answers or generate summary after skip",
				zap.Error(err),
				zap.String("session_id", telegramSession.SessionID),
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
			telegramSession.SessionID,
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
				zap.String("session_id", telegramSession.SessionID),
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
		stateData.PreviousQuestionID = stateData.CurrentQuestionID
	}

	// Clear forward navigation stack since we're skipping forward
	stateData.NextQuestionIDs = []string{}

	// Update state data with new current question
	stateData.CurrentIterationID = nextIteration.IterationID
	stateData.CurrentQuestionID = nextQuestion.ID
	h.stateManager.UpdateStateData(ctx, msg.UserID, stateData)

	hasPrevious := stateData.PreviousQuestionID != ""
	h.sendMessage(msg.ChatID, questionText, h.keyboard.QuestionNavigationKeyboard(nextQuestion.ID, hasPrevious))

	return nil
}

// handleExplainQuestion shows question explanation
func (h *CallbackHandler) handleExplainQuestion(ctx context.Context, msg *Message, questionID string) error {
	explanation, err := h.sessionUC.GetQuestionExplanation(ctx, questionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get question explanation",
			zap.Error(err),
			zap.String("question_id", questionID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	if explanation == "" {
		h.sendMessage(msg.ChatID, "💡 К этому вопросу пока нет отдельного пояснения. Ответь как можно подробнее.", nil)
		return nil
	}

	text := fmt.Sprintf("💡 Пояснение к вопросу:\n\n%s", explanation)
	h.sendMessage(msg.ChatID, text, nil)
	return nil
}

// handlePreviousQuestion navigates back to the previous question
func (h *CallbackHandler) handlePreviousQuestion(ctx context.Context, msg *Message, questionID string) error {
	stateData, err := h.stateManager.GetStateData(ctx, msg.UserID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get state data",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// Check if there is a previous question
	if stateData.PreviousQuestionID == "" {
		h.sendMessage(msg.ChatID, "❌ Нет предыдущего вопроса", nil)
		return nil
	}

	previousQuestionID := stateData.PreviousQuestionID

	// Push current question to forward navigation stack
	if stateData.CurrentQuestionID != "" {
		stateData.NextQuestionIDs = append(stateData.NextQuestionIDs, stateData.CurrentQuestionID)
	}

	// Clear previous question (can't go back further)
	stateData.PreviousQuestionID = ""

	// Get question details
	question, err := h.sessionUC.GetQuestionByID(ctx, previousQuestionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get question",
			zap.Error(err),
			zap.String("question_id", previousQuestionID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// Get iteration to show question index
	iteration, err := h.sessionUC.GetIterationByID(ctx, question.IterationID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get iteration",
			zap.Error(err),
			zap.String("iteration_id", question.IterationID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// Find question index in iteration
	questionIndex := 0
	for i, q := range iteration.Questions {
		if q.ID == previousQuestionID {
			questionIndex = i + 1
			break
		}
	}

	// Determine if we're in skipped questions flow and format accordingly
	var questionText string
	if stateData.AnsweringSkipped {
		// Decrement skipped question index and counter when going back
		if stateData.CurrentSkippedQuestionIndex > 0 {
			stateData.CurrentSkippedQuestionIndex--
		}
		if stateData.CurrentSkippedQuestionNumber > 1 {
			stateData.CurrentSkippedQuestionNumber--
		}

		questionText = render.RenderSkippedQuestion(
			stateData.CurrentSkippedQuestionNumber,
			stateData.TotalSkippedQuestions,
			question.Question,
		)
	} else {
		// Regular question format
		title := ""
		if questionIndex == 1 {
			title = iteration.Title
		}

		questionText = render.RenderQuestion(
			title,
			questionIndex,
			len(iteration.Questions),
			question.Question,
		)
	}

	// Show current answer if exists
	if question.Answer != nil && *question.Answer != "" {
		questionText += fmt.Sprintf("\n\n📝 Текущий ответ:\n%s\n\nМожешь изменить ответ, отправив новый.", *question.Answer)
	}

	// Update state
	stateData.CurrentIterationID = question.IterationID
	stateData.CurrentQuestionID = previousQuestionID

	if err := h.stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
		ctxzap.Error(ctx, "failed to update state data",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	hasPrevious := stateData.PreviousQuestionID != ""
	h.sendMessage(msg.ChatID, questionText, h.keyboard.QuestionNavigationKeyboard(previousQuestionID, hasPrevious))

	return nil
}

// handleDownload handles result download
func (h *CallbackHandler) handleDownload(ctx context.Context, msg *Message, format string) error {
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get user state: %w", err)
	}

	// Validate and normalize format
	resultFormat := entity.ResultFormat(format)
	if !resultFormat.IsValid() {
		ctxzap.Warn(ctx, "invalid download format parameter", zap.String("format", format))
		h.sendMessage(msg.ChatID, "❌ Неверный формат. Доступны: markdown, docx, pdf", nil)
		return nil
	}

	// Get plain text result
	result, err := h.sessionUC.GetSessionResult(ctx, telegramSession.SessionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get result",
			zap.Error(err),
			zap.String("session_id", telegramSession.SessionID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// Create formatter and format result
	factory := formatter.NewFactory()
	fmtr, err := factory.Create(resultFormat)
	if err != nil {
		ctxzap.Error(ctx, "format not implemented", zap.Error(err))
		h.sendMessage(msg.ChatID, "❌ Формат не поддерживается", nil)
		return nil
	}

	formattedResult, err := fmtr.Format(result)
	if err != nil {
		ctxzap.Error(ctx, "failed to format result", zap.Error(err))
		h.sendMessage(msg.ChatID, "❌ Не удалось подготовить файл", nil)
		return nil
	}

	// Send as document
	filename := fmt.Sprintf("requirements-%s%s", telegramSession.SessionID, fmtr.FileExtension())
	doc := tgbotapi.FileBytes{
		Name:  filename,
		Bytes: formattedResult,
	}

	docMsg := tgbotapi.NewDocument(msg.ChatID, doc)
	if _, err := h.bot.Send(docMsg); err != nil {
		ctxzap.Error(ctx, "failed to send document",
			zap.Error(err),
		)
		h.sendMessage(msg.ChatID, "❌ Не удалось отправить файл", nil)
	}

	return nil
}

// handleGenerate forces requirement generation
func (h *CallbackHandler) handleGenerate(ctx context.Context, msg *Message) error {
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get user state: %w", err)
	}

	// Check if already processing (idempotency)
	stateData, err := h.stateManager.GetStateData(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get state data: %w", err)
	}

	if stateData.IsProcessing {
		elapsed := time.Since(stateData.ProcessingStarted)
		if elapsed < 5*time.Minute {
			// Still processing, ignore duplicate request
			h.sendMessage(msg.ChatID, "⏳ Уже обрабатываю запрос, подождите немного...", nil)
			ctxzap.Info(ctx, "duplicate generate request ignored",
				zap.Int64("user_id", msg.UserID),
				zap.Duration("elapsed", elapsed),
			)
			return nil
		}
		// Processing timeout exceeded (>5 min), allow retry
		ctxzap.Warn(ctx, "processing timeout exceeded, allowing retry",
			zap.Int64("user_id", msg.UserID),
			zap.Duration("elapsed", elapsed),
		)
	}

	// Set processing flag
	stateData.IsProcessing = true
	stateData.ProcessingStarted = time.Now()
	if err := h.stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
		ctxzap.Error(ctx, "failed to set processing flag", zap.Error(err))
	}

	// Ensure flag is cleared on exit
	defer func() {
		stateData.IsProcessing = false
		if err := h.stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
			ctxzap.Error(ctx, "failed to clear processing flag", zap.Error(err))
		}
	}()

	h.sendMessage(msg.ChatID, render.MsgProcessing, nil)

	// Decide flow based on session type
	session, err := h.sessionUC.GetSession(ctx, telegramSession.SessionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get session before generate",
			zap.Error(err),
			zap.String("session_id", telegramSession.SessionID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	if session.Type != nil && *session.Type == entity.SessionTypeDraft {
		return h.handleGenerateDraft(ctx, msg, telegramSession.SessionID)
	}

	return h.handleGenerateInterview(ctx, msg, telegramSession.SessionID)
}

// handleGenerateInterview handles final generation for interview mode
func (h *CallbackHandler) handleGenerateInterview(ctx context.Context, msg *Message, sessionID string) error {
	// Start typing indicator during summary generation
	typing := NewTypingNotifier(h.bot, msg.ChatID, h.logger)
	typing.Start(ctx)
	defer typing.Stop()

	// Generate summary
	session, err := h.sessionUC.GenerateSummary(ctx, sessionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to generate interview summary",
			zap.Error(err),
			zap.String("session_id", sessionID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	ctxzap.Info(ctx, "interview requirements generated successfully",
		zap.String("session_id", sessionID),
		zap.String("status", string(session.Status)),
	)

	hasSkipped, err := h.sessionUC.HasSkippedQuestions(ctx, sessionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to check skipped questions",
			zap.Error(err),
			zap.String("session_id", sessionID),
		)
	}

	h.sendMessage(msg.ChatID, render.MsgResultReady, h.keyboard.ResultDownloadKeyboard(hasSkipped))

	return nil
}

// handleGenerateDraft handles validation + generation for draft mode
func (h *CallbackHandler) handleGenerateDraft(ctx context.Context, msg *Message, sessionID string) error {
	// If мы уже вышли из этапа сбора драфта (например, отвечаем на дополнительные вопросы),
	// повторно валидировать драфт не нужно — просто генерируем требования как в интервью-режиме.
	session, err := h.sessionUC.GetSession(ctx, sessionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get session before draft generate",
			zap.Error(err),
			zap.String("session_id", sessionID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// Start typing indicator during validation
	typing := NewTypingNotifier(h.bot, msg.ChatID, h.logger)
	typing.Start(ctx)
	defer typing.Stop()

	var additionalIteration *entity.IterationWithQuestions

	if session.Status == entity.SessionStatusDraftCollecting {
		// Validate draft messages
		additionalIteration, err = h.sessionUC.ValidateDraftMessages(ctx, sessionID)
		if err != nil {
			ctxzap.Error(ctx, "failed to validate draft messages",
				zap.Error(err),
				zap.String("session_id", sessionID),
			)
			h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
			return nil
		}
	}

	// If additional questions are needed, send them as regular interview questions
	if additionalIteration != nil && len(additionalIteration.Questions) > 0 {
		ctxzap.Info(ctx, "additional questions needed for draft",
			zap.Int("count", len(additionalIteration.Questions)),
			zap.String("session_id", sessionID),
		)

		questionText := render.RenderQuestion(
			additionalIteration.Title,
			1,
			len(additionalIteration.Questions),
			additionalIteration.Questions[0].Question,
		)

		// Get existing state data to preserve history
		stateData, err := h.stateManager.GetStateData(ctx, msg.UserID)
		if err != nil {
			ctxzap.Error(ctx, "failed to get state data",
				zap.Error(err),
				zap.Int64("user_id", msg.UserID),
			)
			h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
			return nil
		}

		// Clear previous history when transitioning from draft to questions
		stateData.PreviousQuestionID = ""
		stateData.CurrentIterationID = additionalIteration.IterationID
		stateData.CurrentQuestionID = additionalIteration.Questions[0].ID

		h.stateManager.UpdateStateData(ctx, msg.UserID, stateData)

		// First question has no previous
		h.sendMessage(msg.ChatID, questionText, h.keyboard.QuestionNavigationKeyboard(additionalIteration.Questions[0].ID, false))

		return nil
	}

	// No additional questions - generate draft summary
	session, err = h.sessionUC.GenerateDraftSummary(ctx, sessionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to generate draft summary",
			zap.Error(err),
			zap.String("session_id", sessionID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	ctxzap.Info(ctx, "draft requirements generated successfully",
		zap.String("session_id", sessionID),
		zap.String("status", string(session.Status)),
	)

	hasSkipped, err := h.sessionUC.HasSkippedQuestions(ctx, sessionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to check skipped questions",
			zap.Error(err),
			zap.String("session_id", sessionID),
		)
	}

	h.sendMessage(msg.ChatID, render.MsgResultReady, h.keyboard.ResultDownloadKeyboard(hasSkipped))

	return nil
}

// handleFinish finishes the session
func (h *CallbackHandler) handleFinish(ctx context.Context, msg *Message) error {
	// Get state data to check for pending confirmation
	stateData, err := h.stateManager.GetStateData(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get state data: %w", err)
	}

	// If not already pending confirmation, ask for it
	if stateData.PendingConfirmation != "finish" {
		stateData.PendingConfirmation = "finish"
		if err := h.stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
			ctxzap.Error(ctx, "failed to set pending confirmation",
				zap.Error(err),
				zap.Int64("user_id", msg.UserID),
			)
			h.HandleError(ctx, msg.ChatID, err)
			return nil
		}

		// Show confirmation keyboard
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Да, завершить", "confirm:finish"),
				tgbotapi.NewInlineKeyboardButtonData("❌ Нет, продолжить", "confirm:continue"),
			),
		)
		h.sendMessage(msg.ChatID, "⚠️ Вы уверены? Весь прогресс будет потерян.", keyboard)
		return nil
	}

	// User already confirmed (this shouldn't happen normally, confirmation goes through handleConfirmation)
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get user state: %w", err)
	}

	// Cancel session
	if err := h.sessionUC.CancelSession(ctx, telegramSession.SessionID); err != nil {
		ctxzap.Error(ctx, "failed to cancel session",
			zap.Error(err),
			zap.String("session_id", telegramSession.SessionID),
		)
	}

	// Delete user state
	if err := h.stateManager.DeleteSession(ctx, msg.UserID); err != nil {
		ctxzap.Error(ctx, "failed to delete state",
			zap.Error(err),
		)
	}

	h.sendMessage(msg.ChatID, render.MsgSessionFinished, nil)

	return nil
}

// handleStart handles start action
func (h *CallbackHandler) handleStart(ctx context.Context, msg *Message) error {
	// Create a new backend session when the user explicitly starts the flow
	session, err := h.sessionUC.StartSession(ctx)
	if err != nil {
		ctxzap.Error(ctx, "failed to start session",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// Create/update telegram session mapping
	if err := h.stateManager.CreateOrUpdateSession(ctx, msg.UserID, session.ID); err != nil {
		ctxzap.Error(ctx, "failed to create telegram session",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// Ask for user goal
	h.sendMessage(msg.ChatID, render.MsgAskGoal, nil)
	return nil
}

// handleChooseMode returns to mode selection
func (h *CallbackHandler) handleChooseMode(ctx context.Context, msg *Message) error {
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get user state: %w", err)
	}

	if telegramSession.SessionID == "" {
		h.sendMessage(msg.ChatID, render.ErrSessionNotFound, nil)
		return nil
	}

	// Move backend session back to CHOOSE_MODE so that user can change mode
	if _, err := h.sessionUC.RestartModeSelection(ctx, telegramSession.SessionID); err != nil {
		ctxzap.Error(ctx, "failed to restart mode selection",
			zap.Error(err),
			zap.String("session_id", telegramSession.SessionID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	h.sendMessage(msg.ChatID, render.MsgChooseMode, h.keyboard.ModeSelectionKeyboard())

	return nil
}

// handleChangeProject handles project change
func (h *CallbackHandler) handleChangeProject(ctx context.Context, msg *Message) error {
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get user state: %w", err)
	}

	if telegramSession.SessionID == "" {
		h.sendMessage(msg.ChatID, render.ErrSessionNotFound, nil)
		return nil
	}

	// Move backend session back to SELECT_OR_CREATE_PROJECT so that
	// project selection and context flow can be started again.
	if _, err := h.sessionUC.RestartProjectSelection(ctx, telegramSession.SessionID); err != nil {
		ctxzap.Error(ctx, "failed to restart project selection",
			zap.Error(err),
			zap.String("session_id", telegramSession.SessionID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	const pageSize = 10

	// Get state data to get current page
	stateData, err := h.stateManager.GetStateData(ctx, msg.UserID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get state data",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// Reset page when changing project
	stateData.ProjectListPage = 0
	if err := h.stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
		ctxzap.Error(ctx, "failed to update state data",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
	}

	// Fetch projects with one extra to check if there are more
	projects, err := h.projectUC.ListProjects(ctx, &entity.ListProjectsRequest{
		Skip:  0,
		Limit: pageSize + 1,
	})
	if err != nil {
		ctxzap.Error(ctx, "failed to list projects",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// Check if there are more pages
	hasNextPage := len(projects) > pageSize
	if hasNextPage {
		projects = projects[:pageSize]
	}

	kbProjects := make([]keyboard.Project, 0, len(projects))
	for _, p := range projects {
		kbProjects = append(kbProjects, keyboard.Project{
			ID:    p.ID,
			Title: p.Title,
		})
	}

	h.sendMessage(msg.ChatID, render.MsgSelectProject, h.keyboard.ProjectSelectionKeyboardWithPagination(kbProjects, false, hasNextPage))

	return nil
}

// handleProjectSelection handles project selection
func (h *CallbackHandler) handleProjectSelection(ctx context.Context, msg *Message, projectID string) error {
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get user state: %w", err)
	}

	if projectID == "none" {
		// No project - switch to manual context mode
		if _, err := h.sessionUC.StartManualContext(ctx, telegramSession.SessionID); err != nil {
			ctxzap.Error(ctx, "failed to start manual context",
				zap.Error(err),
				zap.String("session_id", telegramSession.SessionID),
			)
			h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
			return nil
		}

		if len(h.questions) == 0 {
			ctxzap.Error(ctx, "context questions not configured")
			h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
			return nil
		}

		// Send all context questions in a single message
		text := "Ответь, пожалуйста, на несколько вопросов о проекте:\n\n"
		for i, q := range h.questions {
			text += fmt.Sprintf("%d) %s\n\n", i+1, q)
		}
		text += "Ответь одним сообщением — текстом или голосом."

		h.sendMessage(msg.ChatID, text, nil)
		return nil
	}

	// Inform user and submit RAG project context (potentially slow)
	h.sendMessage(msg.ChatID, "⏳ Получаю контекст проекта...", nil)

	// Submit RAG project context
	_, err = h.sessionUC.SubmitRAGProjectContext(ctx, telegramSession.SessionID, projectID)
	if err != nil {
		ctxzap.Error(ctx, "failed to submit project context",
			zap.Error(err),
			zap.String("project_id", projectID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// Show mode selection
	h.sendMessage(msg.ChatID, render.MsgChooseMode, h.keyboard.ModeSelectionKeyboard())

	return nil
}

// handleAnswerSkipped returns to skipped questions
func (h *CallbackHandler) handleAnswerSkipped(ctx context.Context, msg *Message) error {
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get user state: %w", err)
	}

	unanswered, err := h.sessionUC.GetUnansweredQuestions(ctx, telegramSession.SessionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get unanswered questions",
			zap.Error(err),
			zap.String("session_id", telegramSession.SessionID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	if len(unanswered) == 0 {
		h.sendMessage(msg.ChatID, "📝 Пропущенных вопросов нет.", nil)
		return nil
	}

	err = h.sessionUC.SetWaitingForAnswersStatus(ctx, telegramSession.SessionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to update status",
			zap.Error(err),
			zap.String("session_id", telegramSession.SessionID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	q := unanswered[0]

	stateData, err := h.stateManager.GetStateData(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get state data: %w", err)
	}

	// Initialize skipped questions tracking
	stateData.TotalSkippedQuestions = len(unanswered)
	stateData.CurrentSkippedQuestionNumber = 1
	stateData.CurrentSkippedQuestionIndex = 0

	// Build list of skipped question IDs
	stateData.SkippedQuestionIDs = make([]string, len(unanswered))
	for i, uq := range unanswered {
		stateData.SkippedQuestionIDs[i] = uq.ID
	}

	questionText := render.RenderSkippedQuestion(
		stateData.CurrentSkippedQuestionNumber,
		stateData.TotalSkippedQuestions,
		q.Question,
	)

	// Clear previous history when starting to answer skipped questions (new flow)
	stateData.PreviousQuestionID = ""
	stateData.NextQuestionIDs = []string{} // Clear forward navigation from previous interview
	stateData.CurrentIterationID = q.IterationID
	stateData.CurrentQuestionID = q.ID
	stateData.CurrentQuestionIndex = 1
	stateData.AnsweringSkipped = true

	if err := h.stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
		ctxzap.Error(ctx, "failed to update state data",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// First skipped question has no previous
	h.sendMessage(msg.ChatID, questionText, h.keyboard.QuestionNavigationKeyboard(q.ID, false))

	return nil
}

// handleConfirmation handles confirmation callbacks for destructive actions
func (h *CallbackHandler) handleConfirmation(ctx context.Context, msg *Message, value string) error {
	stateData, err := h.stateManager.GetStateData(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get state data: %w", err)
	}

	switch value {
	case "cancel", "finish":
		// User confirmed cancellation or finish
		if stateData.PendingConfirmation == "cancel" || stateData.PendingConfirmation == "finish" {
			telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
			if err != nil {
				h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
				return nil
			}

			// Cancel session
			if telegramSession.SessionID != "" {
				if err := h.sessionUC.CancelSession(ctx, telegramSession.SessionID); err != nil {
					ctxzap.Error(ctx, "failed to cancel session",
						zap.Error(err),
						zap.String("session_id", telegramSession.SessionID),
					)
				}
			}

			// Delete telegram session
			if err := h.stateManager.DeleteSession(ctx, msg.UserID); err != nil {
				ctxzap.Error(ctx, "failed to delete telegram session",
					zap.Error(err),
					zap.Int64("user_id", msg.UserID),
				)
			}

			h.sendMessage(msg.ChatID, render.MsgSessionFinished, nil)
		}

	case "continue":
		// User cancelled the destructive action
		stateData.PendingConfirmation = ""
		h.stateManager.UpdateStateData(ctx, msg.UserID, stateData)
		h.sendMessage(msg.ChatID, "✅ Продолжаем работу", nil)

	default:
		return fmt.Errorf("unknown confirmation value: %s", value)
	}

	return nil
}

// handlePageNavigation handles pagination navigation (prev/next)
func (h *CallbackHandler) handlePageNavigation(ctx context.Context, msg *Message, direction string) error {
	const pageSize = 10

	// Get state data
	stateData, err := h.stateManager.GetStateData(ctx, msg.UserID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get state data",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// Update page number
	if direction == "next" {
		stateData.ProjectListPage++
	} else if direction == "prev" && stateData.ProjectListPage > 0 {
		stateData.ProjectListPage--
	}

	// Calculate offset
	offset := stateData.ProjectListPage * pageSize

	// Save updated state
	if err := h.stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
		ctxzap.Error(ctx, "failed to update state data",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
	}

	// Fetch projects with one extra to check if there are more
	projects, err := h.projectUC.ListProjects(ctx, &entity.ListProjectsRequest{
		Skip:  offset,
		Limit: pageSize + 1,
	})
	if err != nil {
		ctxzap.Error(ctx, "failed to list projects",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
		h.sendMessage(msg.ChatID, render.ClassifyError(err), nil)
		return nil
	}

	// Check if there are more pages
	hasNextPage := len(projects) > pageSize
	if hasNextPage {
		projects = projects[:pageSize]
	}

	kbProjects := make([]keyboard.Project, 0, len(projects))
	for _, p := range projects {
		kbProjects = append(kbProjects, keyboard.Project{
			ID:    p.ID,
			Title: p.Title,
		})
	}

	hasPrevPage := stateData.ProjectListPage > 0
	h.sendMessage(msg.ChatID, render.MsgSelectProject, h.keyboard.ProjectSelectionKeyboardWithPagination(kbProjects, hasPrevPage, hasNextPage))

	return nil
}

// handleSaveNewProject initiates flow for saving requirements to a new project
func (h *CallbackHandler) handleSaveNewProject(ctx context.Context, msg *Message) error {
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get user state: %w", err)
	}

	// Change session status to ask for project name
	if _, err = h.sessionUC.UpdateSessionStatus(ctx, telegramSession.SessionID, entity.SessionStatusAskProjectName); err != nil {
		ctxzap.Error(ctx, "failed to update session status",
			zap.Error(err),
			zap.String("session_id", telegramSession.SessionID),
		)
		h.HandleError(ctx, msg.ChatID, err)
		return nil
	}

	h.sendMessage(msg.ChatID, "📝 Введите название нового проекта:", nil)
	return nil
}

// handleSaveToProject saves requirements to existing project
func (h *CallbackHandler) handleSaveToProject(ctx context.Context, msg *Message) error {
	telegramSession, err := h.stateManager.GetSession(ctx, msg.UserID)
	if err != nil {
		return fmt.Errorf("get user state: %w", err)
	}

	session, err := h.sessionUC.GetSession(ctx, telegramSession.SessionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get session",
			zap.Error(err),
			zap.String("session_id", telegramSession.SessionID),
		)
		h.HandleError(ctx, msg.ChatID, err)
		return nil
	}

	if session.ProjectID == nil || *session.ProjectID == "" {
		h.sendMessage(msg.ChatID, "❌ Проект не выбран. Используйте 'Сохранить в новый проект'.", nil)
		return nil
	}

	if session.Result == nil || *session.Result == "" {
		h.sendMessage(msg.ChatID, "❌ Бизнес-требования еще не сформированы.", nil)
		return nil
	}

	// Get project title for display
	project, err := h.projectUC.GetProject(ctx, *session.ProjectID)
	if err != nil {
		ctxzap.Error(ctx, "failed to get project",
			zap.Error(err),
			zap.String("project_id", *session.ProjectID),
		)
		h.HandleError(ctx, msg.ChatID, err)
		return nil
	}

	// Send progress message
	h.sendMessage(msg.ChatID, fmt.Sprintf("💾 Сохраняю требования в проект '%s'...", project.Title), nil)

	// Start typing indicator and progress notifier
	typing := NewTypingNotifier(h.bot, msg.ChatID, h.logger)
	typing.Start(ctx)
	defer typing.Stop()

	// Save requirements as a file to the project
	fileName := fmt.Sprintf("requirements_%d.md", time.Now().Unix())
	_, err = h.projectUC.AddFileFromContent(
		ctx,
		*session.ProjectID,
		fileName,
		[]byte(*session.Result),
		"text/markdown",
	)
	if err != nil {
		ctxzap.Error(ctx, "failed to save requirements to project",
			zap.Error(err),
			zap.String("project_id", *session.ProjectID),
		)
		h.sendMessage(msg.ChatID, "❌ Не удалось сохранить требования в проект.", nil)
		return nil
	}

	typing.Stop()

	// Check if there are skipped questions
	hasSkipped, err := h.sessionUC.HasSkippedQuestions(ctx, telegramSession.SessionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to check skipped questions",
			zap.Error(err),
			zap.String("session_id", telegramSession.SessionID),
		)
	}

	// Show success message with download buttons
	successMsg := fmt.Sprintf("✅ Требования успешно сохранены в проект '%s'!\n\nМожешь скачать их в удобном формате:", project.Title)
	h.sendMessage(msg.ChatID, successMsg, h.keyboard.ResultDownloadOnlyKeyboard(hasSkipped))
	return nil
}
