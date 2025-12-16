package session

import (
	"context"
	"mime/multipart"

	"github.com/futig/agent-backend/internal/entity"
)

type SessionUsecase interface {
	StartHTTPSession(ctx context.Context, req *entity.StartSessionRequest) (*entity.IterationWithQuestions, error)
	LoadSessionQuestions(ctx context.Context, sessionID string) ([]*entity.IterationWithQuestions, error)
	SkipAnswer(ctx context.Context, sessionID, questionID string) (*entity.IterationWithQuestions, error)
	SubmitTextAnswer(ctx context.Context, sessionID, questionID, answer string) (*entity.IterationWithQuestions, error)
	SubmitHTTPAudioAnswer(ctx context.Context, sessionID, questionID string, audioFile *multipart.FileHeader) (*entity.IterationWithQuestions, error)
	ValidateAnswers(ctx context.Context, sessionID string) (*entity.IterationWithQuestions, error)
	GenerateSummary(ctx context.Context, sessionID string) (*entity.Session, error)
	GetSession(ctx context.Context, sessionID string) (*entity.Session, error)
	GetSessionResult(ctx context.Context, sessionID string) (string, error)
	CancelSession(ctx context.Context, sessionID string) error
}

type CallbackConnector interface {
	SendError(ctx context.Context, callbackURL string, requestID string, message string, details map[string]any)
	SendQuestions(ctx context.Context, callbackURL string, requestID string, data *entity.IterationWithQuestions)
	SendFinalResult(ctx context.Context, callbackURL string, requestID string, data *entity.SessionDTO)
}
