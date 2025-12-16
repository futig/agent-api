package project

import (
	"context"

	"github.com/futig/agent-backend/internal/entity"
)

type ProjectUsecase interface {
	CreateProject(ctx context.Context, req *entity.CreateProjectRequest) (*entity.Project, error)
	ListProjects(ctx context.Context, req *entity.ListProjectsRequest) ([]*entity.Project, error)
	GetProject(ctx context.Context, id string) (*entity.Project, error)
	DeleteProject(ctx context.Context, id string) error
	AddFiles(ctx context.Context, req *entity.AddFilesRequest) ([]*entity.File, error)
	ListFiles(ctx context.Context, projectID string) ([]*entity.File, error)
}

type CallbackConnector interface {
	SendError(ctx context.Context, callbackURL string, requestID string, message string, details map[string]any)
	SendProjectUpdated(ctx context.Context, callbackURL string, requestID string, data *entity.CallbackProjectUpdatedData)
}
