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

// handleValidationAndSummaryCommon runs validation and, if needed, generates final summary.
func handleValidationAndSummaryCommon(
	ctx context.Context,
	msg *Message,
	sessionID string,
	sessionUC SessionUsecase,
	projectUC ProjectUsecase,
	stateManager *state.Manager,
	kb *keyboard.Builder,
	bot *tgbotapi.BotAPI,
	logger *zap.Logger,
	send func(chatID int64, text string, replyMarkup interface{}),
) error {
	// Get session to determine its type
	session, err := sessionUC.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	// Start typing indicator during validation
	typing := NewTypingNotifier(bot, msg.ChatID, logger)
	typing.Start(ctx)
	defer typing.Stop()

	var additionalIteration *entity.IterationWithQuestions

	// Call appropriate validation method based on session type
	if session.Type != nil && *session.Type == entity.SessionTypeDraft {
		additionalIteration, err = sessionUC.ValidateDraftMessages(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("validate draft messages: %w", err)
		}
	} else {
		additionalIteration, err = sessionUC.ValidateAnswers(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("validate answers: %w", err)
		}
	}

	// Additional questions are needed
	if additionalIteration != nil && len(additionalIteration.Questions) > 0 {
		ctxzap.Info(ctx, "additional questions needed",
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
		stateData, err := stateManager.GetStateData(ctx, msg.UserID)
		if err != nil {
			return fmt.Errorf("get state data: %w", err)
		}

		// Track question history for back navigation (only one level)
		if stateData.CurrentQuestionID != "" {
			stateData.PreviousQuestionID = stateData.CurrentQuestionID
		}

		stateData.CurrentIterationID = additionalIteration.IterationID
		stateData.CurrentQuestionID = additionalIteration.Questions[0].ID

		if err := stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
			return fmt.Errorf("update state data: %w", err)
		}

		hasPrevious := stateData.PreviousQuestionID != ""
		send(msg.ChatID, questionText, kb.QuestionNavigationKeyboard(additionalIteration.Questions[0].ID, hasPrevious))

		return nil
	}

	// No additional questions - generate summary
	sessionTypeStr := "unknown"
	if session.Type != nil {
		sessionTypeStr = string(*session.Type)
	}
	ctxzap.Info(ctx, "validation passed, generating requirements",
		zap.String("session_id", sessionID),
		zap.String("session_type", sessionTypeStr),
	)

	// Stop typing indicator before starting progress notifier
	typing.Stop()

	// Inform user that summary generation may take some time
	send(msg.ChatID, render.MsgProcessing, nil)

	// Start progress notifier for long-running summary generation
	progress := NewProgressNotifier(bot, msg.ChatID)
	progress.Start(ctx)
	defer progress.Stop()

	// Call appropriate summary generation method based on session type
	var finalSession *entity.Session
	if session.Type != nil && *session.Type == entity.SessionTypeDraft {
		finalSession, err = sessionUC.GenerateDraftSummary(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("generate draft summary: %w", err)
		}
	} else {
		finalSession, err = sessionUC.GenerateSummary(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("generate summary: %w", err)
		}
	}

	ctxzap.Info(ctx, "requirements generated successfully",
		zap.String("session_id", sessionID),
		zap.String("status", string(finalSession.Status)),
	)

	hasSkipped, err := sessionUC.HasSkippedQuestions(ctx, sessionID)
	if err != nil {
		ctxzap.Error(ctx, "failed to check skipped questions",
			zap.Error(err),
			zap.String("session_id", sessionID),
		)
	}

	// Get project title if session has a project
	projectTitle := ""
	if projectUC != nil && finalSession.ProjectID != nil && *finalSession.ProjectID != "" {
		project, err := projectUC.GetProject(ctx, *finalSession.ProjectID)
		if err != nil {
			ctxzap.Warn(ctx, "failed to get project for keyboard",
				zap.Error(err),
				zap.String("project_id", *finalSession.ProjectID),
			)
		} else {
			projectTitle = project.Title
		}
	}

	// Show result and save/download buttons
	send(msg.ChatID, render.MsgResultReady, kb.ResultSaveKeyboard(hasSkipped, projectTitle))

	return nil
}

// handleNextSkippedQuestion processes the next skipped/unanswered question
// Returns true if there are more skipped questions to answer, false otherwise
func handleNextSkippedQuestion(
	ctx context.Context,
	msg *Message,
	sessionID string,
	sessionUC SessionUsecase,
	projectUC ProjectUsecase,
	stateManager *state.Manager,
	kb *keyboard.Builder,
	bot *tgbotapi.BotAPI,
	logger *zap.Logger,
	send func(chatID int64, text string, replyMarkup interface{}),
) (bool, error) {
	stateData, err := stateManager.GetStateData(ctx, msg.UserID)
	if err != nil {
		return false, fmt.Errorf("get state data: %w", err)
	}

	// Initialize skipped questions list on first entry
	if len(stateData.SkippedQuestionIDs) == 0 {
		// Get list of unanswered questions
		skippedQuestions, err := sessionUC.GetUnansweredQuestions(ctx, sessionID)
		if err != nil {
			return false, fmt.Errorf("get unanswered questions: %w", err)
		}

		// If no unanswered questions at all, trigger validation
		if len(skippedQuestions) == 0 {
			stateData.AnsweringSkipped = false
			stateData.TotalSkippedQuestions = 0
			stateData.CurrentSkippedQuestionNumber = 0
			stateData.SkippedQuestionIDs = nil
			stateData.CurrentSkippedQuestionIndex = 0
			if err := stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
				ctxzap.Error(ctx, "failed to clear answering skipped flag",
					zap.Error(err),
					zap.Int64("user_id", msg.UserID),
				)
				return false, fmt.Errorf("update state data: %w", err)
			}

			ctxzap.Info(ctx, "no more skipped questions, moving to validation",
				zap.String("session_id", sessionID),
			)

			send(msg.ChatID, render.MsgValidating, nil)

			// Run validation
			if err := handleValidationAndSummaryCommon(ctx, msg, sessionID, sessionUC, projectUC, stateManager, kb, bot, logger, send); err != nil {
				return false, fmt.Errorf("handle validation: %w", err)
			}

			return false, nil
		}

		// Initialize the list
		stateData.SkippedQuestionIDs = make([]string, len(skippedQuestions))
		for i, q := range skippedQuestions {
			stateData.SkippedQuestionIDs[i] = q.ID
		}
		stateData.TotalSkippedQuestions = len(skippedQuestions)
		stateData.CurrentSkippedQuestionIndex = 0
		stateData.CurrentSkippedQuestionNumber = 1
	} else {
		// Move to next question in the list
		stateData.CurrentSkippedQuestionIndex++
		stateData.CurrentSkippedQuestionNumber++

		// Check if we've reached the end
		if stateData.CurrentSkippedQuestionIndex >= len(stateData.SkippedQuestionIDs) {
			// All skipped questions answered, clear state and trigger validation
			stateData.AnsweringSkipped = false
			stateData.TotalSkippedQuestions = 0
			stateData.CurrentSkippedQuestionNumber = 0
			stateData.SkippedQuestionIDs = nil
			stateData.CurrentSkippedQuestionIndex = 0
			if err := stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
				ctxzap.Error(ctx, "failed to clear answering skipped flag",
					zap.Error(err),
					zap.Int64("user_id", msg.UserID),
				)
				return false, fmt.Errorf("update state data: %w", err)
			}

			ctxzap.Info(ctx, "completed all skipped questions, moving to validation",
				zap.String("session_id", sessionID),
			)

			send(msg.ChatID, render.MsgValidating, nil)

			// Run validation
			if err := handleValidationAndSummaryCommon(ctx, msg, sessionID, sessionUC, projectUC, stateManager, kb, bot, logger, send); err != nil {
				return false, fmt.Errorf("handle validation: %w", err)
			}

			return false, nil
		}
	}

	// Get the question at current index
	nextQuestionID := stateData.SkippedQuestionIDs[stateData.CurrentSkippedQuestionIndex]
	nextQuestion, err := sessionUC.GetQuestionByID(ctx, nextQuestionID)
	if err != nil {
		return false, fmt.Errorf("get question by id: %w", err)
	}

	questionText := render.RenderSkippedQuestion(
		stateData.CurrentSkippedQuestionNumber,
		stateData.TotalSkippedQuestions,
		nextQuestion.Question,
	)

	// Track question history for back navigation (only one level)
	if stateData.CurrentQuestionID != "" {
		stateData.PreviousQuestionID = stateData.CurrentQuestionID
	}

	stateData.CurrentIterationID = nextQuestion.IterationID
	stateData.CurrentQuestionID = nextQuestion.ID
	stateData.AnsweringSkipped = true

	if err := stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
		ctxzap.Error(ctx, "failed to update state data for next skipped question",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
		return false, fmt.Errorf("update state data: %w", err)
	}

	hasPrevious := stateData.PreviousQuestionID != ""
	send(msg.ChatID, questionText, kb.QuestionNavigationKeyboard(nextQuestion.ID, hasPrevious))

	return true, nil
}

// handleSkipCurrentQuestion skips the current question and processes the next skipped question
// Returns true if there are more skipped questions to answer, false otherwise
func handleSkipCurrentQuestion(
	ctx context.Context,
	msg *Message,
	sessionID string,
	currentQuestionID string,
	sessionUC SessionUsecase,
	projectUC ProjectUsecase,
	stateManager *state.Manager,
	kb *keyboard.Builder,
	bot *tgbotapi.BotAPI,
	logger *zap.Logger,
	send func(chatID int64, text string, replyMarkup interface{}),
) (bool, error) {
	// Skip current question in the backend
	_, err := sessionUC.SkipSkipedQuestion(ctx, sessionID, currentQuestionID)
	if err != nil {
		return false, fmt.Errorf("skip question: %w", err)
	}

	stateData, err := stateManager.GetStateData(ctx, msg.UserID)
	if err != nil {
		return false, fmt.Errorf("get state data: %w", err)
	}

	// Move to next question in our saved list
	stateData.CurrentSkippedQuestionIndex++
	stateData.CurrentSkippedQuestionNumber++

	// Check if we've reached the end
	if stateData.CurrentSkippedQuestionIndex >= len(stateData.SkippedQuestionIDs) {
		// All skipped questions processed, clear state and trigger validation
		stateData.AnsweringSkipped = false
		stateData.TotalSkippedQuestions = 0
		stateData.CurrentSkippedQuestionNumber = 0
		stateData.SkippedQuestionIDs = nil
		stateData.CurrentSkippedQuestionIndex = 0
		if err := stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
			ctxzap.Error(ctx, "failed to clear answering skipped flag on skip",
				zap.Error(err),
				zap.Int64("user_id", msg.UserID),
			)
			return false, fmt.Errorf("update state data: %w", err)
		}

		send(msg.ChatID, render.MsgValidating, nil)

		// Run validation
		if err := handleValidationAndSummaryCommon(ctx, msg, sessionID, sessionUC, projectUC, stateManager, kb, bot, logger, send); err != nil {
			return false, fmt.Errorf("handle validation: %w", err)
		}

		return false, nil
	}

	// Get next question from our saved list
	nextQuestionID := stateData.SkippedQuestionIDs[stateData.CurrentSkippedQuestionIndex]
	nextQuestion, err := sessionUC.GetQuestionByID(ctx, nextQuestionID)
	if err != nil {
		return false, fmt.Errorf("get question by id: %w", err)
	}

	questionText := render.RenderSkippedQuestion(
		stateData.CurrentSkippedQuestionNumber,
		stateData.TotalSkippedQuestions,
		nextQuestion.Question,
	)

	// Track question history for back navigation (only one level)
	if stateData.CurrentQuestionID != "" {
		stateData.PreviousQuestionID = stateData.CurrentQuestionID
	}

	stateData.CurrentIterationID = nextQuestion.IterationID
	stateData.CurrentQuestionID = nextQuestion.ID
	stateData.AnsweringSkipped = true

	if err := stateManager.UpdateStateData(ctx, msg.UserID, stateData); err != nil {
		ctxzap.Error(ctx, "failed to update state data for next skipped question",
			zap.Error(err),
			zap.Int64("user_id", msg.UserID),
		)
		return false, fmt.Errorf("update state data: %w", err)
	}

	hasPrevious := stateData.PreviousQuestionID != ""
	send(msg.ChatID, questionText, kb.QuestionNavigationKeyboard(nextQuestion.ID, hasPrevious))

	return true, nil
}
