package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/google/uuid"
)

// generateQuestionsBlocks calls LLM to generate question blocks
func (uc *SessionUsecase) generateQuestionsBlocks(
	ctx context.Context,
	userGoal string,
	projectContext string,
	projectDescription *string,
) ([]entity.QuestionsBlock, error) {
	req := &entity.LLMGenerateQuestionsRequest{
		UserGoal:           userGoal,
		ProjectContext:     projectContext,
		ProjectDescription: projectDescription,
	}

	response, err := uc.llmConnector.GenerateQuestions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("generate questions: %w", err)
	}

	if len(response.Iterations) == 0 {
		return nil, fmt.Errorf("no questions generated")
	}

	return response.Iterations, nil
}

// saveQuestionsToDatabase saves question blocks as iterations + questions
func (uc *SessionUsecase) saveQuestionsToDatabase(
	ctx context.Context, sessionID string, blocks []entity.QuestionsBlock,
) ([]*entity.IterationWithQuestions, error) {
	// Get the maximum iteration number for this session
	maxIterationNumber, err := uc.iterationRepo.GetMaxIterationNumber(ctx, sessionID)
	if err != nil {
		// If no iterations exist yet, start from 0
		maxIterationNumber = 0
	}

	iterations := make([]*entity.IterationWithQuestions, 0, len(blocks))

	for idx, block := range blocks {
		// Start from max + 1 to avoid conflicts
		iterationNumber := maxIterationNumber + idx + 1

		iteration := entity.Iteration{
			ID:              uuid.New().String(),
			SessionID:       sessionID,
			IterationNumber: iterationNumber,
			Title:           block.Title,
		}

		savedIteration, err := uc.iterationRepo.CreateIteration(ctx, iteration)
		if err != nil {
			return nil, fmt.Errorf("create iteration %d: %w", iterationNumber, err)
		}

		questions := make([]*entity.Question, 0, len(block.Questions))

		for qIdx, q := range block.Questions {
			question := entity.Question{
				ID:             uuid.New().String(),
				IterationID:    savedIteration.ID,
				QuestionNumber: qIdx + 1,
				Status:         entity.AnswerStatusUnanswered,
				Question:       q.Text,
				Explanation:    q.Explanation,
			}

			if _, err := uc.questionRepo.CreateQuestion(ctx, question); err != nil {
				return nil, fmt.Errorf("create question: %w", err)
			}

			questions = append(questions, &question)
		}

		iterations = append(iterations, questionsToIterationDTO(savedIteration, questions))
	}

	return iterations, nil
}

func (uc *SessionUsecase) getCurrentIteration(ctx context.Context, sessionID string) (*entity.IterationWithQuestions, error) {
	currentIteration, err := uc.iterationRepo.GetCurrentIteration(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get current iteration: %w", err)
	}

	curentQuestion, err := uc.questionRepo.ListQuestionsByIteration(ctx, currentIteration.ID)
	if err != nil || len(curentQuestion) == 0 {
		return nil, fmt.Errorf("list questions by iteration: %w", err)
	}

	hasUnansweredQuestions := curentQuestion[len(curentQuestion)-1].Status == entity.AnswerStatusUnanswered

	if hasUnansweredQuestions {
		return questionsToIterationDTO(currentIteration, curentQuestion), nil
	}

	nextIteration, err := uc.iterationRepo.GetNextIteration(ctx, sessionID)
	if err != nil {
		return nil, nil
	}

	_, err = uc.sessionRepo.UpdateSessionIteration(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("update iteration: %w", err)
	}

	curentQuestion, err = uc.questionRepo.ListQuestionsByIteration(ctx, nextIteration.ID)
	if err != nil || len(curentQuestion) == 0 {
		return nil, fmt.Errorf("list questions by iteration: %w", err)
	}

	return questionsToIterationDTO(nextIteration, curentQuestion), nil
}

// formatManualContext formats context questions into a string
func (uc *SessionUsecase) formatManualContext(questions []entity.QuestionWithAnswer) string {
	var sb strings.Builder
	for i, qa := range questions {
		sb.WriteString(fmt.Sprintf("Вопрос %d: %s\n", i+1, qa.Question))
		sb.WriteString(fmt.Sprintf("Пользователь ответил: %s\n\n", qa.Answer))
	}
	return sb.String()
}

// transcribeAudio transcribes audio file to text
func (uc *SessionUsecase) transcribeAudio(ctx context.Context, filename string, audioData []byte) (string, error) {
	transcript, err := uc.asrConnector.TranscribeBytes(ctx, audioData, filename)
	if err != nil {
		return "", fmt.Errorf("transcribe audio: %w", err)
	}

	if transcript == "" {
		return "", fmt.Errorf("transcription is empty")
	}

	return transcript, nil
}

// collectAllAnswers collects all answered questions from all iterations
func (uc *SessionUsecase) collectAllAnswers(ctx context.Context, sessionID string) ([]entity.QuestionWithAnswer, error) {
	questions, err := uc.questionRepo.ListQuestionsBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list iterations: %w", err)
	}

	// Initialize as empty slice instead of nil to ensure JSON serialization as [] not null
	allAnswers := make([]entity.QuestionWithAnswer, 0)

	for _, question := range questions {
		if question.Status == entity.AnswerStatusAnswered {
			allAnswers = append(allAnswers, entity.QuestionWithAnswer{
				Question: question.Question,
				Answer:   *question.Answer,
			})
		}
	}

	return allAnswers, nil
}

// HasSkippedQuestions checks if there are any skipped questions in the session
func (uc *SessionUsecase) HasSkippedQuestions(ctx context.Context, sessionID string) (bool, error) {
	questions, err := uc.questionRepo.GetUnansweredQuestions(ctx, sessionID)
	if err != nil {
		return false, fmt.Errorf("list questions: %w", err)
	}

	return len(questions) > 0, nil
}
