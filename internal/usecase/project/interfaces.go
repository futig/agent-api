package project

import (
	"context"

	"github.com/futig/agent-backend/internal/entity"
)

type RagConnector interface {
	GetContext(ctx context.Context, req *entity.RAGGetContextRequest) (string, error)
	IndexFiles(ctx context.Context, projectID string, files []entity.FileData) error
	DeleteIndex(ctx context.Context, projectID string) error
}
