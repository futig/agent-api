package handlers

import (
	"context"

	"github.com/futig/agent-backend/internal/entity"
)

// SessionUsecase defines the interface for session business logic operations
// Used by the Telegram bot handlers to orchestrate the interview workflow
type SessionUsecase interface {
	// Bot methods - granular operations for Telegram bot workflow
	StartSession(ctx context.Context) (*entity.Session, error)
	SubmitTextUserGoal(ctx context.Context, sessionID, goal string) (*entity.Session, error)
	SubmitAudioUserGoal(ctx context.Context, sessionID string, audioGoal []byte) (*entity.Session, error)
	SubmitRAGProjectContext(ctx context.Context, sessionID, projectID string) (*entity.Session, error)
	SubmitTextUserProjectContext(ctx context.Context, sessionID, questions, answers string) (*entity.Session, error)
	SubmitAudioUserProjectContext(ctx context.Context, sessionID, questions string, audioAnswers []byte) (*entity.Session, error)
	SetSessionType(ctx context.Context, sessionID string, sessionType entity.SessionType) (*entity.Session, error)
	StartManualContext(ctx context.Context, sessionID string) (*entity.Session, error)
	RestartModeSelection(ctx context.Context, sessionID string) (*entity.Session, error)
	RestartProjectSelection(ctx context.Context, sessionID string) (*entity.Session, error)
	StartDraftCollecting(ctx context.Context, sessionID string) (*entity.Session, error)
	LoadSessionQuestions(ctx context.Context, sessionID string) ([]*entity.IterationWithQuestions, error)
	SkipAnswer(ctx context.Context, sessionID, questionID string) (*entity.IterationWithQuestions, error)
	SubmitTextAnswer(ctx context.Context, sessionID, questionID, answer string) (*entity.IterationWithQuestions, error)
	SubmitAudioAnswer(ctx context.Context, sessionID, questionID string, audioAnswer []byte) (*entity.IterationWithQuestions, error)
	HasSkippedQuestions(ctx context.Context, sessionID string) (bool, error)
	SetWaitingForAnswersStatus(ctx context.Context, sessionID string) error
	SkipSkipedQuestion(ctx context.Context, sessionID, questionID string) ([]*entity.Question, error)
	GetUnansweredQuestions(ctx context.Context, sessionID string) ([]*entity.Question, error)
	GetQuestionExplanation(ctx context.Context, questionID string) (string, error)
	GetQuestionByID(ctx context.Context, questionID string) (*entity.Question, error)
	GetIterationByID(ctx context.Context, iterationID string) (*entity.IterationWithQuestions, error)
	ValidateAnswers(ctx context.Context, sessionID string) (*entity.IterationWithQuestions, error)
	GenerateSummary(ctx context.Context, sessionID string) (*entity.Session, error)
	// Draft mode methods
	AddDraftMessage(ctx context.Context, sessionID, messageText string) (*entity.SessionMessage, error)
	AddAudioDraftMessage(ctx context.Context, sessionID string, audioData []byte) (*entity.SessionMessage, error)
	ValidateDraftMessages(ctx context.Context, sessionID string) (*entity.IterationWithQuestions, error)
	GenerateDraftSummary(ctx context.Context, sessionID string) (*entity.Session, error)
	// Common methods
	GetSession(ctx context.Context, sessionID string) (*entity.Session, error)
	GetSessionResult(ctx context.Context, sessionID string) (string, error)
	CancelSession(ctx context.Context, sessionID string) error
	UpdateSessionStatus(ctx context.Context, sessionID string, status entity.SessionStatus) (*entity.Session, error)
}

// ProjectUsecase defines the subset of project operations needed by Telegram handlers
type ProjectUsecase interface {
	ListProjects(ctx context.Context, req *entity.ListProjectsRequest) ([]*entity.Project, error)
	GetProject(ctx context.Context, projectID string) (*entity.Project, error)
	CreateProject(ctx context.Context, req *entity.CreateProjectRequest) (*entity.Project, error)
	CreateProjectFromContent(ctx context.Context, title, description, filename string, content []byte, contentType string) (*entity.Project, error)
	AddFiles(ctx context.Context, req *entity.AddFilesRequest) ([]*entity.File, error)
	AddFileFromContent(ctx context.Context, projectID, filename string, content []byte, contentType string) (*entity.File, error)
}
