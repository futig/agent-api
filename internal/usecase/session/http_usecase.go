package session

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/google/uuid"
)

// StartHTTPSession aggregates session creation, context generation, and question loading
func (uc *SessionUsecase) StartHTTPSession(
	ctx context.Context,
	req *entity.StartSessionRequest,
) (*entity.IterationWithQuestions, error) {
	session := &entity.Session{
		ID:     uuid.New().String(),
		Status: entity.SessionStatusWaitingForAnswers,
	}

	sessionType := entity.SessionTypeInterview
	session.Type = &sessionType
	session.UserGoal = &req.UserGoal

	var projectContext string
	var projectDescription *string

	if req.ProjectID != nil {
		session.ProjectID = req.ProjectID

		project, err := uc.projectRepo.Get(ctx, *req.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("get project: %w", err)
		}

		projectDescription = &project.Description

		projectContext, err = uc.ragConnector.GetContext(ctx, &entity.RAGGetContextRequest{
			ProjectID:    *req.ProjectID,
			UserGoal:     *session.UserGoal,
			TopK:         5,
			MaxQuestions: 10,
		})
		if err != nil {
			return nil, fmt.Errorf("get RAG context: %w", err)
		}
	} else {
		projectContext = uc.formatManualContext(req.ContextQuestions)
	}

	session.ProjectContext = &projectContext

	session, err := uc.sessionRepo.CreateFilledSession(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("create filled session: %w", err)
	}

	blocks, err := uc.generateQuestionsBlocks(ctx, req.UserGoal, projectContext, projectDescription)
	if err != nil {
		return nil, fmt.Errorf("generate questions: %w", err)
	}

	savedIterations, err := uc.saveQuestionsToDatabase(ctx, session.ID, blocks)
	if err != nil || len(savedIterations) == 0 {
		return nil, fmt.Errorf("save questions: %w", err)
	}

	return savedIterations[0], nil
}

func (uc *SessionUsecase) SubmitHTTPAudioAnswer(
	ctx context.Context, sessionID, questionID string, audioFile *multipart.FileHeader,
) (*entity.IterationWithQuestions, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusWaitingForAnswers {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	file, err := audioFile.Open()
	if err != nil {
		return nil, fmt.Errorf("open audio file: %w", err)
	}
	defer file.Close()

	audioData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read audio file: %w", err)
	}

	file.Close()

	return uc.SubmitAudioAnswer(ctx, sessionID, questionID, audioData)
}
