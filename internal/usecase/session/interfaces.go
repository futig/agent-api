package session

import (
	"context"

	"github.com/futig/agent-backend/internal/entity"
)

type RagConnector interface {
	GetContext(ctx context.Context, req *entity.RAGGetContextRequest) (string, error)
}

type LLMConnector interface {
	GenerateQuestions(ctx context.Context, req *entity.LLMGenerateQuestionsRequest) (*entity.LLMGenerateQuestionsResponse, error)
	GenerateSummary(ctx context.Context, req *entity.LLMGenerateSummaryRequest) (string, error)
	ValidateAnswers(ctx context.Context, req *entity.LLMValidateAnswersRequest) (*entity.LLMValidateAnswersResponse, error)
	ValidateDraft(ctx context.Context, req *entity.LLMValidateDraftRequest) (*entity.LLMValidateAnswersResponse, error)
	GenerateDraftSummary(ctx context.Context, req *entity.LLMGenerateDraftSummaryRequest) (string, error)
}

type ASRConnector interface {
	TranscribeBytes(ctx context.Context, audioData []byte, filename string) (string, error)
}
